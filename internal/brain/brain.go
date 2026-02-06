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
	"github.com/nathfavour/vibeauracle/internal/doctor"
	"github.com/nathfavour/vibeauracle/internal/vibe"
	"github.com/nathfavour/vibeauracle/model"
	"github.com/nathfavour/vibeauracle/prompt"
	"github.com/nathfavour/vibeauracle/sys"
	"github.com/nathfavour/vibeauracle/tooling"
	"github.com/nathfavour/vibeauracle/vault"
)

// Request represents a user request or system trigger
func New() *Brain {
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
		security: guard,
		sessions: make(map[string]*tooling.Session),
		detector: NewLoopDetector(10),
		extMgr:   vibe.NewManager(cfg.DataDir),
	}

	_ = b.extMgr.LoadAll()
	_ = b.extMgr.InitializeDefaults()

	b.fs = sys.NewLocalFS("")
	b.tools = tooling.Setup(b.fs, b.monitor, b.security)
	vibe.RegisterExtensions(context.Background(), b.extMgr, b.tools)

	// 1. Initialize Provider first so we have an embedder
	b.initProvider()

	// 2. Initialize Memory with the provider
	b.memory = vcontext.NewMemory(b.model.Provider())

	// 3. Initialize Prompts with the model and memory
	b.prompts = prompt.New(cfg, b.memory, &prompt.NoopRecommender{}, b.model) // ...

	// Seamless GitHub Onboarding & Auto-Switch:
	// Automatically promote to copilot-sdk/sdk mode if detected and not manually overridden.
	if copilot.IsAvailable() {
		changed := false
		if !cfg.Model.UserConfigured && cfg.Model.Provider != "copilot-sdk" {
			cfg.Model.Provider = "copilot-sdk"
			cfg.Model.Name = "gpt-4o"
			changed = true
		}
		if !cfg.Agent.UserConfigured && cfg.Agent.Mode != "sdk" {
			cfg.Agent.Mode = "sdk"
			changed = true
		}
		if changed {
			_ = cm.Save(cfg)
		}
	} else if (cfg.Model.Provider == "ollama" || cfg.Model.Provider == "") && (cfg.Model.Name == "llama3" || cfg.Model.Name == "") && !cfg.Model.UserConfigured {
		// Fallback onboarding for standard github-copilot if SDK is missing but gh token is found
		if token, _ := auth.GetGithubCLIToken(); token != "" {
			cfg.Model.Provider = "github-copilot"
			cfg.Model.Name = "gpt-4o"
			_ = cm.Save(cfg)
		}
	}

	b.initProvider()

	// Proactive Autofix: If the configured model is missing or it's the first run,
	// try to autodetect what's available on the system.
	go b.autodetectBestModel()

	// Register healer for autonomous recovery
		doctor.RegisterHealer(func(issue string) {
			go b.Heal(context.Background(), issue)
		})
	
		return b
	}
	
// registerToolsWithCopilot bridges VibeAuracle tools to the Copilot SDK.
func (b *Brain) registerToolsWithCopilot() {
	bridge := copilot.NewToolBridge()

	// Get core tools from the registry
	for _, t := range b.tools.List() {
		tool := t
		if false {
			continue
		}
		meta := t.Metadata()
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
		"model": b.config.Model.Name,
	}

	// Only include endpoint/base_url if it's not the default Ollama one when using copilot-sdk,
	// or if it's a non-SDK provider where we always need the endpoint (like Ollama/OpenAI).
	isSDK := b.config.Model.Provider == "copilot-sdk"
	isDefaultOllama := b.config.Model.Endpoint == "http://localhost:11434"

	if !isSDK || !isDefaultOllama {
		configMap["endpoint"] = b.config.Model.Endpoint
		configMap["base_url"] = b.config.Model.Endpoint
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

	// Wire usage callback
	b.model.SetUsageCallback(func(u model.Usage) {
		if b.OnUsage != nil {
			b.OnUsage(u)
		}
	})

	// Wire streaming callbacks globally for all providers
	b.model.SetStreamCallbacks(func(delta string) {
		if b.OnStreamDelta != nil {
			b.OnStreamDelta(delta)
		}
	}, func(full string) {
		if b.OnStreamDone != nil {
			b.OnStreamDone(full)
		}
	})

	// Check if we are using the Copilot SDK provider to enable SDK-specific features
	if sdkP, ok := p.(*model.CopilotSDKProvider); ok {
		b.copilotProvider = sdkP.GetSDKProvider()
		b.usingCopilotSDK = true
		tooling.ReportStatus("🚀", "copilot", "Using native Copilot SDK")

		b.copilotProvider.SetStatusCallback(func(icon, step, message string) {
			tooling.ReportStatus(icon, step, message)
		})

		// Re-register tools if SDK is active
		b.registerToolsWithCopilot()
	}

		// Synchronize all cognitive components with the new provider
		if b.model != nil {
			if b.prompts != nil {
				b.prompts.SetRecommender(prompt.NewModelRecommender(b.model))
				b.prompts.SetModel(b.model)
			}
			if b.memory != nil {
				b.memory.SetEmbedder(b.model.Provider())
			}
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
// SetAgentMode switches between 'vibe', 'sdk', and 'custom' agentic runtimes
func (b *Brain) SetAgentMode(mode string) error {
	if mode != "vibe" && mode != "sdk" && mode != "custom" {
		return fmt.Errorf("invalid agent mode: %s (must be 'vibe', 'sdk', or 'custom')", mode)
	}
	b.config.Agent.Mode = mode
	b.config.Agent.UserConfigured = true
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
func (b *Brain) Process(ctx context.Context, reqObj interface{}) (interface{}, error) {
	var req Request

	// Handle different request types (Map from Daemon, or direct Request struct)
	switch r := reqObj.(type) {
	case Request:
		req = r
	case *Request:
		req = *r
	case map[string]interface{}:
		req.ID, _ = r["id"].(string)
		req.Content, _ = r["content"].(string)
		if intent, ok := r["intent"].(string); ok {
			req.Intent = Intent(intent)
		}
	default:
		return Response{}, fmt.Errorf("unsupported request type: %T", reqObj)
	}

	tooling.ReportStatus("🧠", "think", "Processing request...")

	// Early check for model or Copilot SDK
	if b.model == nil && !b.usingCopilotSDK {
		tooling.ReportStatus("❌", "error", "No AI model configured")
		return Response{}, fmt.Errorf("no AI model configured. Run 'vibeaura auth' to set up a provider")
	}

	// 1. Session & Thread Management
	sessionID := b.GetSessionID()
	session, ok := b.sessions[sessionID]
	if !ok {
		// Attempt to restore session object from memory
		var storedSession tooling.Session
		if err := b.RecallState(sessionID+"_obj", &storedSession); err == nil {
			session = &storedSession
		} else {
			session = tooling.NewSession(sessionID)
		}
		b.sessions[sessionID] = session
	}

	// 2. Perceive: Receive request + SystemSnapshot
	snapshot, _ := b.monitor.GetSnapshot()
	tooling.ReportStatus("👁️", "perceive", fmt.Sprintf("CWD: %s", snapshot.WorkingDir))

	// 3. Tool Awareness (Smart Handshake)
	toolDefs := b.tools.GetPromptDefinitions(nil)
	tooling.ReportStatus("🔧", "tools", fmt.Sprintf("Loaded %d tools", len(b.tools.List())))

	// 4. Update Rolling Context Window
	b.memory.AddToWindow(req.ID, req.Content, "user_prompt")
	tooling.ReportStatus("🧠", "memory", "Analyzing conversation context...")

	// Provide recent history to prompt builder
	recentHistory := ""
	if len(session.Threads) > 0 {
		var hb strings.Builder
		hb.WriteString("\nRECENT CONVERSATION HISTORY:\n")
		start := 0
		if len(session.Threads) > 5 { // Last 5 turns
			start = len(session.Threads) - 5
		}
		for _, t := range session.Threads[start:] {
			hb.WriteString(fmt.Sprintf("User: %s\nAssistant: %s\n", t.Prompt, t.Response))
		}
		recentHistory = hb.String()
	}

	// 5. Prompt System: classify + layer instructions + inject recall + build final prompt
	augmentedPrompt := ""
	var recs []prompt.Recommendation
	var promptIntent prompt.Intent

	if b.config.Prompt.Enabled && b.prompts != nil {
		tooling.ReportStatus("📝", "prompt", "Selecting prompt strategy...")

		env, builtRecs, err := b.prompts.Build(ctx, req.Content, snapshot, toolDefs, recentHistory)
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

		// Manual override from request
		if req.Intent != "" {
			promptIntent = req.Intent
		}

		tooling.ReportStatus("✅", "prompt", fmt.Sprintf("Strategy: %s", promptIntent))
	} else {
		// Fallback...
		tooling.ReportStatus("📝", "prompt", "Using fallback prompt builder")
		snippets, _ := b.memory.Recall(ctx, req.Content, snapshot.WorkingDir)
		contextStr := strings.Join(snippets, "\n") // ... (rest of fallback)
		augmentedPrompt = fmt.Sprintf(`System Context:
%s

RECENT CONVERSATION HISTORY:
%s

System CWD: %s
Available Tools (JSON-RPC 2.0 Style):
%s

User Request (Thread ID: %s):
%s`, contextStr, recentHistory, snapshot.WorkingDir, toolDefs, req.ID, req.Content)
	}

	// MODE: SDK AGENT
	// If agent mode is 'sdk' and we are using the SDK provider, delegate the entire loop.
	if b.config.Agent.Mode == "sdk" && b.usingCopilotSDK && b.copilotProvider != nil {
		tooling.ReportStatus("🚀", "agent-sdk", "Delegating task to native Copilot SDK runtime...")
		resp, cUsage, err := b.copilotProvider.Generate(ctx, augmentedPrompt, true)
		usage := model.Usage{
			InputTokens:  cUsage.InputTokens,
			OutputTokens: cUsage.OutputTokens,
			TotalTokens:  cUsage.TotalTokens,
			Cost:         cUsage.Cost,
		}
		if err != nil {
			tooling.ReportStatus("❌", "error", fmt.Sprintf("SDK Agent error: %v", err))
			return Response{}, fmt.Errorf("sdk agent execution: %w", err)
		}
		tooling.ReportStatus("✅", "done", "SDK Agent completed task")
		_ = b.memory.Store(req.ID, resp)
		_ = b.StoreState(sessionID+"_obj", session)
		return Response{
			Content: resp,
			Metadata: map[string]interface{}{
				"recommendations": recs,
				"usage":           usage,
			},
		}, nil
	}

	// MODE: CUSTOM AGENT
	if b.config.Agent.Mode == "custom" {
		var activeAgent *sys.CustomAgent
		for _, a := range b.config.Agent.CustomAgents {
			if a.Name == b.config.Agent.ActiveCustom {
				activeAgent = &a
				break
			}
		}

		if activeAgent != nil {
			tooling.ReportStatus("👤", "agent-custom", fmt.Sprintf("Executing via Custom Agent: %s", activeAgent.Name))
			// Inject custom prompt
			augmentedPrompt = fmt.Sprintf("Custom Agent Instructions: %s\n\n%s", activeAgent.Prompt, augmentedPrompt)

			// Restrict tools if specified
			if len(activeAgent.Tools) > 0 {
				toolDefs = b.tools.GetPromptDefinitions(activeAgent.Tools)
			}
		}
	}

	// MODE: VIBE AGENT (Internal Loop)
	if b.config.Agent.Mode == "vibe" {
		tooling.ReportStatus("🎨", "agent-vibe", "Executing via internal Vibe Agent...")
	}
	// EXECUTION LOOP (Agentic) - allow up to 10 turns for complex tasks
	maxTurns := 10
	history := augmentedPrompt
	var fullResponse strings.Builder
	var totalUsage model.Usage
	b.detector = NewLoopDetector(10) // Reset for each new process

	for i := 0; i < maxTurns; i++ {
		tooling.ReportStatus("🔄", "loop", fmt.Sprintf("Turn %d/%d: Thinking...", i+1, maxTurns))

		// 1. Generation
		var resp string
		var usage model.Usage
		var generateErr error

		if b.usingCopilotSDK && b.copilotProvider != nil {
			// Use Copilot SDK for generation
			generateErr = backoff.Retry(func() error {
				var err error
				var cUsage copilot.Usage
				resp, cUsage, err = b.copilotProvider.Generate(ctx, history, true)
				usage = model.Usage{
					InputTokens:  cUsage.InputTokens,
					OutputTokens: cUsage.OutputTokens,
					TotalTokens:  cUsage.TotalTokens,
					Cost:         cUsage.Cost,
				}
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
				resp, usage, err = b.model.Generate(ctx, history)
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
			doctor.Send("brain", "error", "Generation failed", map[string]any{"error": generateErr.Error(), "turn": i})
			return Response{}, fmt.Errorf("generating response: %w", generateErr)
		}

		// Accumulate usage
		totalUsage.InputTokens += usage.InputTokens
		totalUsage.OutputTokens += usage.OutputTokens
		totalUsage.TotalTokens += usage.TotalTokens
		totalUsage.Cost += usage.Cost

		// Loop Detection: If model response is identical and we already tried tools, it might be stuck.
		if b.detector.AddAction(resp) {
			tooling.ReportStatus("🛑", "loop-detected", "Agent stuck in a repetitive loop. Halting.")
			doctor.Send("brain", "warning", "Loop detected", map[string]any{"response": resp})
			finalContent := fullResponse.String() + "\n" + resp + "\n\n(Stopped: Loop detected)"
			return Response{Content: finalContent}, nil
		}

		// Accumulate response
		if resp != "" {
			if fullResponse.Len() > 0 {
				fullResponse.WriteString("\n\n")
			}
			fullResponse.WriteString(resp)
		}

		// Show first 100 chars of response
		preview := resp
		if len(preview) > 100 {
			preview = preview[:100] + "..."
		}
		tooling.ReportStatus("💬", "response", preview)

		tooling.ReportStatus("🔎", "parsing", "Analyzing response for tool calls...")

		// 2. Parse & Execute Tools
		executed, resultVal, interventionErr, execErr := b.executeToolCalls(ctx, resp, promptIntent)

		// Bubble up intervention immediately so UI can handle it
		if interventionErr != nil {
			tooling.ReportStatus("⚠️", "intervention", "User approval required")
			return Response{}, interventionErr
		}

		// Add tool results to loop detection too
		if executed && b.detector.AddAction(resultVal) {
			tooling.ReportStatus("🛑", "loop-detected", "Tool results are repetitive. Halting.")
			finalContent := fullResponse.String() + "\n\n(Stopped: Loop detected in tool results)"
			return Response{Content: finalContent}, nil
		}

		if !executed {
			tooling.ReportStatus("✅", "done", "Task complete")
			finalContent := fullResponse.String()
			session.AddThread(&tooling.Thread{
				ID:       req.ID,
				Prompt:   req.Content,
				Response: finalContent,
				Metadata: map[string]interface{}{
					"prompt_intent":    promptIntent,
					"recommendations":  recs,
					"response_raw_len": len(finalContent),
					"usage":            totalUsage,
				},
			})
			_ = b.memory.Store(req.ID, finalContent)
			_ = b.StoreState(sessionID+"_obj", session)
			return Response{
				Content: finalContent,
				Metadata: map[string]interface{}{
					"recommendations": recs,
					"usage":           totalUsage,
				},
			}, nil
		}

		// 3. Observation (feed back into history) - prompt to continue with remaining tasks
		history += "\n" + resp

		if execErr != nil {
			tooling.ReportStatus("❌", "tool", fmt.Sprintf("Tool error: %v", execErr))
			history += fmt.Sprintf("\n\nObservation: Tool execution failed: %v\n\nContinue executing the remaining steps. Output the next tool call.", execErr)
		} else {
			resultPreview := resultVal
			if len(resultPreview) > 80 {
				resultPreview = resultPreview[:80] + "..."
			}
			tooling.ReportStatus("✅", "tool", fmt.Sprintf("Result: %s", resultPreview))
			history += fmt.Sprintf("\n\nObservation: Tool output:\n%s\n\nOriginal request: %s\n\nIf there are more steps to complete, output the next tool call now. Only provide a summary when ALL tasks are done.", resultVal, req.Content)
		}

		// 4. Record intermediate step
		_ = b.memory.Store(req.ID+"_step_"+fmt.Sprint(i), resultVal)
	}

	tooling.ReportStatus("⚠️", "limit", "Agent loop limit reached")
	finalContent := fullResponse.String() + "\n\n(Stopped: Agent loop limit reached)"
	return Response{Content: finalContent}, nil
}

// executeToolCalls parses the response for JSON tool invocations and executes ALL of them.
func (b *Brain) executeToolCalls(ctx context.Context, input string, intent prompt.Intent) (bool, string, error, error) {
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

		// Security: Block tool execution if intent is CHAT or ASK (unless specifically authorized).
		// This prevents the model from "hallucinating" tool calls during normal chat or Q&A.
		if intent == prompt.IntentChat || intent == prompt.IntentAsk {
			tooling.ReportStatus("🛡️", "security", fmt.Sprintf("Blocked tool call '%s' in %s mode", call.Tool, intent))
			results = append(results, fmt.Sprintf("Error: tool execution is disabled in %s mode. Please use '/do' or 'implement:' if you want me to take action.", intent))
			executed = true // Mark as executed so the loop can handle the "result"
			continue
		}

				// Found a tool call!
				executed = true
				
				// Detailed status reporting for specific tools
				extra := ""
				switch call.Tool {
				case "shell_exec":
					var args struct{ Command string `json:"command"` }
					if err := json.Unmarshal(call.Args, &args); err == nil {
						extra = "cmd:" + args.Command
					}
				case "sys_write_file":
					var args struct{ Path string `json:"path"`; Content string `json:"content"` }
					if err := json.Unmarshal(call.Args, &args); err == nil {
						extra = "file:" + args.Path + "\n" + args.Content
					}
				case "sys_patch":
					var args struct{ Path string `json:"path"`; Patch string `json:"patch"` }
					if err := json.Unmarshal(call.Args, &args); err == nil {
						extra = "patch:" + args.Path + "\n" + args.Patch
					}
				}
		
				if extra != "" {
					tooling.ReportStatus("🔧", "exec", fmt.Sprintf("Executing %s", call.Tool), extra)
				} else {
					tooling.ReportStatus("🔧", "tool", fmt.Sprintf("Executing: %s", call.Tool))
				}
		t, found := b.tools.Get(call.Tool)
		if !found {
			lastErr = fmt.Errorf("tool '%s' not found", call.Tool)
			doctor.Send("brain", "error", "Tool not found", map[string]any{"tool": call.Tool})
			results = append(results, fmt.Sprintf("Error: tool '%s' not found", call.Tool))
			continue
		}

		res, err := t.Execute(ctx, call.Args)
		if err != nil {
			// Check for intervention error
			if strings.Contains(err.Error(), "intervention required") {
				interventionErr = err
				doctor.Send("brain", "intervention", "Intervention required", map[string]any{"tool": call.Tool})
				break // Stop processing, need user input
			}
			lastErr = err
			doctor.Send("brain", "error", "Tool execution failed", map[string]any{"tool": call.Tool, "error": err.Error()})
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
// GetSnapshot returns a current snapshot of system resources via the monitor
func (b *Brain) GetSnapshot() (interface{}, error) {
	return b.monitor.GetSnapshot()
}

// GetIdentity returns the current user identity if available
func (b *Brain) GetIdentity() string {
	if b.config.Model.Provider == "github-copilot" || b.config.Model.Provider == "github-models" {
		return auth.GetGithubUser()
	}
	return ""
}

// Extensions returns the list of loaded extensions
func (b *Brain) Extensions() []*vibe.Extension {
	return b.extMgr.List()
}

// RegisterExtension registers a new extension
func (b *Brain) RegisterExtension(name, desc string) (*vibe.Extension, error) {
	return b.extMgr.Register(name, desc)
}

func (b *Brain) SetExtensionEnabled(id string, enabled bool) error {
	return b.extMgr.SetEnabled(id, enabled)
}
