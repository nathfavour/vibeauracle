package brain

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/cenkalti/backoff/v4"
	"github.com/nathfavour/vibeauracle/auth"
	vcontext "github.com/nathfavour/vibeauracle/context"
	"github.com/nathfavour/vibeauracle/copilot"
	"github.com/nathfavour/vibeauracle/model"
	"github.com/nathfavour/vibeauracle/prompt"
	"github.com/nathfavour/vibeauracle/sys"
	"github.com/nathfavour/vibeauracle/tooling"
	"github.com/nathfavour/vibeauracle/vault"
)

// Request represents a user request or system trigger
type Request struct {
	ID      string
	Content string
}

// Response represents the brain's output
type Response struct {
	Content string
	Error   error
}

// Brain is the cognitive orchestrator
type Brain struct {
	model    *model.Model
	monitor  *sys.Monitor
	fs       sys.FS
	config   *sys.Config
	cm       *sys.ConfigManager
	auth     *auth.Handler
	vault    *vault.Vault
	memory   *vcontext.Memory
	prompts  *prompt.System
	tools    *tooling.Registry
	security *tooling.SecurityGuard
	sessions map[string]*tooling.Session

	// Copilot SDK integration
	copilotProvider *copilot.Provider
	usingCopilotSDK bool

	// Loop Detection
	detector *LoopDetector

	// Callbacks
	OnStreamDelta func(delta string)
	OnStreamDone  func(full string)
}

// LoopDetector tracks agent actions to detect infinite loops
type LoopDetector struct {
	lastActions []string
	maxHistory  int
}

func NewLoopDetector(maxHistory int) *LoopDetector {
	return &LoopDetector{
		lastActions: make([]string, 0, maxHistory),
		maxHistory:  maxHistory,
	}
}

func (ld *LoopDetector) AddAction(action string) bool {
	// Normalize action string (trim whitespace, etc)
	action = strings.TrimSpace(action)

	// Check for repetition
	repeatCount := 0
	for _, a := range ld.lastActions {
		if a == action {
			repeatCount++
		}
	}

	// If we see the exact same response + tool result sequence 3 times, it's a loop
	if repeatCount >= 3 {
		return true
	}

	ld.lastActions = append(ld.lastActions, action)
	if len(ld.lastActions) > ld.maxHistory {
		ld.lastActions = ld.lastActions[1:]
	}
	return false
}

func New() *Brain {
	// ... (existing New logic)
	cm, _ := sys.NewConfigManager()
	cfg, _ := cm.Load()
	v, _ := vault.New("vibeauracle", cfg.DataDir)
	guard := tooling.NewSecurityGuard()

	b := &Brain{
		monitor:  sys.NewMonitor(),
		config:   cfg,
		cm:       cm,
		auth:     auth.NewHandler(),
		vault:    v,
		memory:   vcontext.NewMemory(),
		security: guard,
		sessions: make(map[string]*tooling.Session),
		detector: NewLoopDetector(10),
	}

	// Prompt system is modular and configurable.
	b.prompts = prompt.New(cfg, b.memory, &prompt.NoopRecommender{})

	// Seamless GitHub Onboarding:
	// If project is fresh (default provider is ollama/empty) and gh token is found,
	// immediately promote to github-copilot for a zero-config experience.
	if (cfg.Model.Provider == "ollama" || cfg.Model.Provider == "") && (cfg.Model.Name == "llama3" || cfg.Model.Name == "") {
		if token, _ := auth.GetGithubCLIToken(); token != "" {
			// Prefer Copilot SDK if copilot CLI is available
			if copilot.IsAvailable() {
				cfg.Model.Provider = "copilot-sdk"
				cfg.Model.Name = "gpt-4o"
			} else {
				cfg.Model.Provider = "github-copilot"
				cfg.Model.Name = "gpt-4o"
			}
			_ = cm.Save(cfg) // Persist the zero-config win
		}
	}

	b.initProvider()

	// Proactive Autofix: If the configured model is missing or it's the first run,
	// try to autodetect what's available on the system.
	go b.autodetectBestModel()

	b.fs = sys.NewLocalFS("")
	b.tools = tooling.Setup(b.fs, b.monitor, b.security)

	// Register VibeAuracle tools with Copilot SDK if active
	if b.usingCopilotSDK && b.copilotProvider != nil {
		b.registerToolsWithCopilot()
	}

	return b
}

// registerToolsWithCopilot bridges VibeAuracle tools to the Copilot SDK.
func (b *Brain) registerToolsWithCopilot() {
	bridge := copilot.NewToolBridge()

	// Get core tools from the registry
	for _, toolName := range tooling.CoreTools() {
		tool, found := b.tools.Get(toolName)
		if !found {
			continue
		}
		meta := tool.Metadata()
		bridge.AddTool(copilot.VibeToolDefinition{
			Name:        meta.Name,
			Description: meta.Description,
			Parameters:  meta.Parameters,
			Execute: func(ctx context.Context, args json.RawMessage) (string, error) {
				result, err := tool.Execute(ctx, args)
				if err != nil {
					return "", err
				}
				return result.Content, nil
			},
		})
	}

	b.copilotProvider.RegisterTools(bridge)
}

func (b *Brain) initProvider() {
	configMap := map[string]string{
		"endpoint": b.config.Model.Endpoint,
		"model":    b.config.Model.Name,
		"base_url": b.config.Model.Endpoint, // Map endpoint to base_url for OpenAI/Others
	}

	// Fetch credentials from vault
	if b.vault != nil {
		if token, err := b.vault.Get("github_models_pat"); err == nil {
			configMap["token"] = token
		}
		if key, err := b.vault.Get("openai_api_key"); err == nil && key != "" {
			configMap["api_key"] = key
			configMap["provider_type"] = "openai"
		} else if key, err := b.vault.Get("anthropic_api_key"); err == nil && key != "" {
			configMap["api_key"] = key
			configMap["provider_type"] = "anthropic"
		}
	}

	// Auto-login fallback: Use gh CLI token if still empty for GitHub-based providers
	if configMap["token"] == "" && (b.config.Model.Provider == "github-models" || b.config.Model.Provider == "github-copilot") {
		if token, _ := auth.GetGithubCLIToken(); token != "" {
			configMap["token"] = token
		}
	}

	// Initialize the provider
	p, err := model.GetProvider(b.config.Model.Provider, configMap)
	if err != nil {
		fmt.Printf("Error initializing provider %s: %v\n", b.config.Model.Provider, err)
		// Fallback if copilot-sdk fails
		if b.config.Model.Provider == "copilot-sdk" {
			tooling.ReportStatus("⚠️", "copilot", fmt.Sprintf("SDK unavailable: %v, falling back", err))
			b.config.Model.Provider = "github-copilot"
			p, _ = model.GetProvider("github-copilot", configMap)
		}
	}

	b.model = model.New(p)
	b.usingCopilotSDK = false
	b.copilotProvider = nil

	// Check if we are using the Copilot SDK provider to enable SDK-specific features
	if sdkP, ok := p.(*model.CopilotSDKProvider); ok {
		b.copilotProvider = sdkP.GetSDKProvider()
		b.usingCopilotSDK = true
		tooling.ReportStatus("🚀", "copilot", "Using native Copilot SDK")

		// Set streaming callbacks
		b.copilotProvider.SetStreamCallbacks(func(delta string) {
			if b.OnStreamDelta != nil {
				b.OnStreamDelta(delta)
			}
		}, func(full string) {
			if b.OnStreamDone != nil {
				b.OnStreamDone(full)
			}
		})

		// Re-register tools if SDK is active
		b.registerToolsWithCopilot()
	}

	// Update the prompt system's recommender to use the newly initialized model.
	if b.prompts != nil && b.model != nil {
		b.prompts.SetRecommender(prompt.NewModelRecommender(b.model))
	}
}

// Shutdown gracefully stops all resources including Copilot SDK.
func (b *Brain) Shutdown() error {
	if b.copilotProvider != nil {
		return b.copilotProvider.Stop()
	}
	return nil
}

// ModelDiscovery represents a discovered model with its provider
type ModelDiscovery struct {
	Name     string
	Provider string
}

// DiscoverModels fetches available models from all configured providers
func (b *Brain) DiscoverModels(ctx context.Context) ([]ModelDiscovery, error) {
	var discoveries []ModelDiscovery

	// List of potential providers to check
	providersToCheck := []string{"ollama", "openai", "github-models", "github-copilot", "copilot-sdk"}

	for _, pName := range providersToCheck {
		configMap := map[string]string{
			"endpoint": b.config.Model.Endpoint,
			"base_url": b.config.Model.Endpoint,
		}

		// Hydrate with credentials
		if b.vault != nil {
			switch pName {
			case "github-models", "github-copilot":
				if token, err := b.vault.Get("github_models_pat"); err == nil {
					configMap["token"] = token
				} else {
					// Fallback to CLI token
					if ghToken, _ := auth.GetGithubCLIToken(); ghToken != "" {
						configMap["token"] = ghToken
					} else {
						continue // Still no token, skip
					}
				}
			case "openai":
				if key, err := b.vault.Get("openai_api_key"); err == nil {
					configMap["api_key"] = key
				} else {
					continue // No key, skip
				}
			case "ollama":
				// Usually no auth needed for local ollama
			}
		}

		p, err := model.GetProvider(pName, configMap)
		if err != nil {
			continue
		}

		models, err := p.ListModels(ctx)
		if err != nil {
			continue
		}

		for _, m := range models {
			discoveries = append(discoveries, ModelDiscovery{
				Name:     m,
				Provider: pName,
			})
		}
	}

	return discoveries, nil
}

// SetModel updates the active model and provider
func (b *Brain) SetModel(provider, name string) error {
	b.config.Model.Provider = provider
	b.config.Model.Name = name

	// If provider is ollama, we might need to handle endpoint too,
	// but for now we keep the existing one or reset to default if changed.
	if provider == "ollama" && b.config.Model.Endpoint == "" {
		b.config.Model.Endpoint = "http://localhost:11434"
	}

	if err := b.cm.Save(b.config); err != nil {
		return fmt.Errorf("saving config: %w", err)
	}

	b.initProvider()
	return nil
}

// SetAgentMode switches between 'vibe', 'sdk', and 'custom' agentic runtimes
func (b *Brain) SetAgentMode(mode string) error {
	if mode != "vibe" && mode != "sdk" && mode != "custom" {
		return fmt.Errorf("invalid agent mode: %s (must be 'vibe', 'sdk', or 'custom')", mode)
	}
	b.config.Agent.Mode = mode
	return b.cm.Save(b.config)
}

// RegisterCustomAgent adds or updates a user-defined agent
func (b *Brain) RegisterCustomAgent(agent sys.CustomAgent) error {
	for i, a := range b.config.Agent.CustomAgents {
		if a.Name == agent.Name {
			b.config.Agent.CustomAgents[i] = agent
			return b.cm.Save(b.config)
		}
	}
	b.config.Agent.CustomAgents = append(b.config.Agent.CustomAgents, agent)
	return b.cm.Save(b.config)
}

// GetCustomAgents returns the list of registered custom agents
func (b *Brain) GetCustomAgents() []sys.CustomAgent {
	return b.config.Agent.CustomAgents
}

// SetActiveCustomAgent sets the active custom agent by name
func (b *Brain) SetActiveCustomAgent(name string) error {
	for _, a := range b.config.Agent.CustomAgents {
		if a.Name == name {
			b.config.Agent.ActiveCustom = name
			b.config.Agent.Mode = "custom"
			return b.cm.Save(b.config)
		}
	}
	return fmt.Errorf("custom agent '%s' not found", name)
}

// Process handles the "Plan-Execute-Reflect" loop
func (b *Brain) Process(ctx context.Context, req Request) (Response, error) {
	tooling.ReportStatus("🧠", "think", "Processing request...")

	// Early check for model or Copilot SDK
	if b.model == nil && !b.usingCopilotSDK {
		tooling.ReportStatus("❌", "error", "No AI model configured")
		return Response{}, fmt.Errorf("no AI model configured. Run 'vibeaura auth' to set up a provider")
	}

	// 1. Session & Thread Management
	sessionID := "default" // In a real app, this would come from the request
	session, ok := b.sessions[sessionID]
	if !ok {
		session = tooling.NewSession(sessionID)
		b.sessions[sessionID] = session
	}

	// 2. Perceive: Receive request + SystemSnapshot
	snapshot, _ := b.monitor.GetSnapshot()
	tooling.ReportStatus("👁️", "perceive", fmt.Sprintf("CWD: %s", snapshot.WorkingDir))

	// 3. Tool Awareness (Smart Handshake)
	toolDefs := b.tools.GetPromptDefinitions(tooling.CoreTools())
	tooling.ReportStatus("🔧", "tools", fmt.Sprintf("Loaded %d core tools", len(tooling.CoreTools())))

	// 4. Update Rolling Context Window
	b.memory.AddToWindow(req.ID, req.Content, "user_prompt")
	tooling.ReportStatus("🧠", "memory", "Analyzing conversation context...")

	// 5. Prompt System: classify + layer instructions + inject recall + build final prompt
	augmentedPrompt := ""
	var recs []prompt.Recommendation
	var promptIntent prompt.Intent

	if b.config.Prompt.Enabled && b.prompts != nil {
		tooling.ReportStatus("📝", "prompt", "Selecting prompt strategy...")
		env, builtRecs, err := b.prompts.Build(ctx, req.Content, snapshot, toolDefs)
		if err != nil {
			tooling.ReportStatus("❌", "error", fmt.Sprintf("Prompt build failed: %v", err))
			return Response{}, fmt.Errorf("building prompt: %w", err)
		}
		if ignored, ok := env.Metadata["ignored"].(bool); ok && ignored {
			tooling.ReportStatus("⏭️", "skip", "Empty/invalid prompt ignored")
			return Response{Content: "(ignored empty/invalid prompt)"}, nil
		}
		augmentedPrompt = env.Prompt
		recs = builtRecs
		promptIntent = env.Intent
		tooling.ReportStatus("✅", "prompt", fmt.Sprintf("Strategy: %s", promptIntent))
	} else {
		// Fallback...
		tooling.ReportStatus("📝", "prompt", "Using fallback prompt builder")
		snippets, _ := b.memory.Recall(req.Content)
		contextStr := strings.Join(snippets, "\n")
		// ... (rest of fallback)
		augmentedPrompt = fmt.Sprintf(`System Context:
%s

System CWD: %s
Available Tools (JSON-RPC 2.0 Style):
%s

User Request (Thread ID: %s):
%s`, contextStr, snapshot.WorkingDir, toolDefs, req.ID, req.Content)
	}

	// MODE: SDK AGENT
	// If agent mode is 'sdk' and we are using the SDK provider, delegate the entire loop.
	if b.config.Agent.Mode == "sdk" && b.usingCopilotSDK && b.copilotProvider != nil {
		tooling.ReportStatus("🚀", "agent-sdk", "Delegating task to native Copilot SDK runtime...")
		resp, err := b.copilotProvider.Generate(ctx, augmentedPrompt)
		if err != nil {
			tooling.ReportStatus("❌", "error", fmt.Sprintf("SDK Agent error: %v", err))
			return Response{}, fmt.Errorf("sdk agent execution: %w", err)
		}
		tooling.ReportStatus("✅", "done", "SDK Agent completed task")
		_ = b.memory.Store(req.ID, resp)
		return Response{Content: resp}, nil
	}

	// MODE: VIBE AGENT (Internal Loop)
	tooling.ReportStatus("🎨", "agent-vibe", "Executing via internal Vibe Agent...")
	// EXECUTION LOOP (Agentic) - allow up to 10 turns for complex tasks
	maxTurns := 10
	history := augmentedPrompt
	b.detector = NewLoopDetector(10) // Reset for each new process

	for i := 0; i < maxTurns; i++ {
		tooling.ReportStatus("🔄", "loop", fmt.Sprintf("Turn %d/%d: Thinking...", i+1, maxTurns))

		// ... (Generation logic)
		var resp string
		var generateErr error

		if b.usingCopilotSDK && b.copilotProvider != nil {
			// Use Copilot SDK for generation
			generateErr = backoff.Retry(func() error {
				var err error
				resp, err = b.copilotProvider.Generate(ctx, history)
				if err != nil {
					if ctx.Err() != nil {
						return backoff.Permanent(err)
					}
					tooling.ReportStatus("⏳", "retry", fmt.Sprintf("Retrying (SDK)... (%v)", err))
					return err
				}
				return nil
			}, backoff.WithContext(backoff.NewExponentialBackOff(), ctx))
		} else {
			// Use standard model provider
			generateErr = backoff.Retry(func() error {
				var err error
				resp, err = b.model.Generate(ctx, history)
				if err != nil {
					if ctx.Err() != nil {
						return backoff.Permanent(err)
					}
					tooling.ReportStatus("⏳", "retry", fmt.Sprintf("Retrying thinking... (%v)", err))
					return err
				}
				return nil
			}, backoff.WithContext(backoff.NewExponentialBackOff(), ctx))
		}

		if generateErr != nil {
			tooling.ReportStatus("❌", "error", fmt.Sprintf("Model error: %v", generateErr))
			return Response{}, fmt.Errorf("generating response: %w", generateErr)
		}

		// Loop Detection: If model response is identical and we already tried tools, it might be stuck.
		if b.detector.AddAction(resp) {
			tooling.ReportStatus("🛑", "loop-detected", "Agent stuck in a repetitive loop. Halting.")
			return Response{Content: resp + "\n\n(Stopped: Loop detected)"}, nil
		}

		// Show first 100 chars of response
		preview := resp
		if len(preview) > 100 {
			preview = preview[:100] + "..."
		}
		tooling.ReportStatus("💬", "response", preview)

		tooling.ReportStatus("🔎", "parsing", "Analyzing response for tool calls...")

		// 2. Parse & Execute Tools
		executed, resultVal, interventionErr, execErr := b.executeToolCalls(ctx, resp)

		// Bubble up intervention immediately so UI can handle it
		if interventionErr != nil {
			tooling.ReportStatus("⚠️", "intervention", "User approval required")
			return Response{}, interventionErr
		}

		// Add tool results to loop detection too
		if executed && b.detector.AddAction(resultVal) {
			tooling.ReportStatus("🛑", "loop-detected", "Tool results are repetitive. Halting.")
			return Response{Content: "Agent halted due to repetitive tool output: " + resultVal}, nil
		}

		if !executed {
			tooling.ReportStatus("✅", "done", "Task complete")
			// ...
			session.AddThread(&tooling.Thread{
				ID:       req.ID,
				Prompt:   req.Content,
				Response: resp,
				Metadata: map[string]interface{}{
					"prompt_intent":    promptIntent,
					"recommendations":  recs,
					"response_raw_len": len(resp),
				},
			})
			_ = b.memory.Store(req.ID, resp)
			return Response{Content: resp}, nil
		}

		// 3. Observation (feed back into history) - prompt to continue with remaining tasks
		if execErr != nil {
			tooling.ReportStatus("❌", "tool", fmt.Sprintf("Tool error: %v", execErr))
			history += fmt.Sprintf("\n\nTool execution failed: %v\n\nContinue executing the remaining steps. Output the next tool call.\nAssistant:", execErr)
		} else {
			resultPreview := resultVal
			if len(resultPreview) > 80 {
				resultPreview = resultPreview[:80] + "..."
			}
			tooling.ReportStatus("✅", "tool", fmt.Sprintf("Result: %s", resultPreview))
			history += fmt.Sprintf("\n\nTool output:\n%s\n\nOriginal request: %s\n\nIf there are more steps to complete, output the next tool call now. Only provide a summary when ALL tasks are done.\nAssistant:", resultVal, req.Content)
		}

		// 4. Record intermediate step
		_ = b.memory.Store(req.ID+"_step_"+fmt.Sprint(i), resultVal)
	}

	tooling.ReportStatus("⚠️", "limit", "Agent loop limit reached")
	return Response{Content: "Agent loop limit reached. Some tasks may not have completed."}, nil
}

// executeToolCalls parses the response for JSON tool invocations and executes ALL of them.
func (b *Brain) executeToolCalls(ctx context.Context, input string) (bool, string, error, error) {
	var results []string
	var lastErr error
	var interventionErr error
	executed := false
	remaining := input

	// Find and execute ALL tool calls in the response
	for {
		start := strings.Index(remaining, "```json")
		if start == -1 {
			break
		}

		contentStart := start + 7
		blockContent := remaining[contentStart:]

		end := strings.Index(blockContent, "```")
		if end == -1 {
			break
		}

		jsonStr := strings.TrimSpace(blockContent[:end])
		remaining = blockContent[end+3:] // Move past this block

		// Attempt to parse tool call
		var call struct {
			Tool string          `json:"tool"`
			Args json.RawMessage `json:"parameters"`
		}
		if err := json.Unmarshal([]byte(jsonStr), &call); err != nil {
			continue // Not a valid tool call, skip
		}

		if call.Tool == "" {
			continue
		}

		// Found a tool call!
		executed = true
		tooling.ReportStatus("🔧", "tool", fmt.Sprintf("Executing: %s", call.Tool))

		t, found := b.tools.Get(call.Tool)
		if !found {
			lastErr = fmt.Errorf("tool '%s' not found", call.Tool)
			results = append(results, fmt.Sprintf("Error: tool '%s' not found", call.Tool))
			continue
		}

		res, err := t.Execute(ctx, call.Args)
		if err != nil {
			// Check for intervention error
			if strings.Contains(err.Error(), "intervention required") {
				interventionErr = err
				break // Stop processing, need user input
			}
			lastErr = err
			results = append(results, fmt.Sprintf("Error executing %s: %v", call.Tool, err))
			continue
		}

		results = append(results, fmt.Sprintf("[%s]: %s", call.Tool, res.Content))
	}

	if interventionErr != nil {
		return executed, strings.Join(results, "\n"), interventionErr, nil
	}

	return executed, strings.Join(results, "\n"), nil, lastErr
}

// PullModel requests a model download (currently only supported by Ollama)
func (b *Brain) PullModel(ctx context.Context, name string) error {
	// Re-initialize provider to ensure we have the latest endpoint
	configMap := map[string]string{
		"endpoint": b.config.Model.Endpoint,
		"model":    name,
	}

	p, err := model.GetProvider("ollama", configMap)
	if err != nil {
		return err
	}

	// Dynamic check for PullModel capability
	if puller, ok := p.(interface {
		PullModel(ctx context.Context, name string, cb func(any)) error
	}); ok {
		return puller.PullModel(ctx, name, nil)
	}

	return fmt.Errorf("provider '%s' does not support pulling models", p.Name())
}

// StoreState persists application state
func (b *Brain) StoreState(id string, state interface{}) error {
	return b.memory.SaveState(id, state)
}

// RecallState retrieves application state
func (b *Brain) RecallState(id string, target interface{}) error {
	return b.memory.LoadState(id, target)
}

// ClearState removes application state
func (b *Brain) ClearState(id string) error {
	return b.memory.ClearState(id)
}

// GetConfig returns the brain's configuration
func (b *Brain) GetConfig() *sys.Config {
	return b.config
}

// Config is an alias for GetConfig
func (b *Brain) Config() *sys.Config {
	return b.config
}

// UpdateConfig updates the brain's configuration and persists it
func (b *Brain) UpdateConfig(cfg *sys.Config) error {
	b.config = cfg
	if err := b.cm.Save(b.config); err != nil {
		return fmt.Errorf("saving config: %w", err)
	}
	b.initProvider()
	return nil
}

// GetSnapshot returns a current snapshot of system resources via the monitor
func (b *Brain) GetSnapshot() (sys.Snapshot, error) {
	return b.monitor.GetSnapshot()
}

// StoreSecret saves a secret in the vault
func (b *Brain) StoreSecret(key, value string) error {
	if b.vault == nil {
		return fmt.Errorf("vault not initialized")
	}
	return b.vault.Set(key, value)
}

func (b *Brain) autodetectBestModel() {
	// Only autodetect if we are using the default "llama3" which might not exist,
	// or if the model name is empty/none.
	// If we've already promoted to github-copilot, skip autodetection unless it fails.
	if b.config.Model.Provider == "github-copilot" {
		return
	}
	if b.config.Model.Name != "llama3" && b.config.Model.Name != "" && b.config.Model.Name != "none" {
		return
	}

	ctx := context.Background()
	discoveries, err := b.DiscoverModels(ctx)
	if err != nil || len(discoveries) == 0 {
		return
	}

	// 1. Try to find if LLAMA-3 or 3.2 is actually there (better matching than just 'llama3')
	for _, d := range discoveries {
		name := strings.ToLower(d.Name)
		if strings.Contains(name, "llama") || strings.Contains(name, "gpt-4o") || strings.Contains(name, "phi-3") {
			b.SetModel(d.Provider, d.Name)
			return
		}
	}

	// 2. Fallback to the first available model from any provider
	if len(discoveries) > 0 {
		b.SetModel(discoveries[0].Provider, discoveries[0].Name)
	}
}

// GetSecret retrieves a secret from the vault
func (b *Brain) GetSecret(key string) (string, error) {
	if b.vault == nil {
		return "", fmt.Errorf("vault not initialized")
	}
	return b.vault.Get(key)
}

// GetIdentity returns the current user identity if available
func (b *Brain) GetIdentity() string {
	if b.config.Model.Provider == "github-copilot" || b.config.Model.Provider == "github-models" {
		return auth.GetGithubUser()
	}
	return ""
}
