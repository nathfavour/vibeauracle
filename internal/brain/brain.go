package brain

import (
	"context"
	"fmt"
	"os"
	"strings"

	"github.com/cenkalti/backoff/v4"
	"github.com/nathfavour/vibeauracle/auth"
	"github.com/nathfavour/vibeauracle/connect"
	vcontext "github.com/nathfavour/vibeauracle/context"
	"github.com/nathfavour/vibeauracle/copilot"
	"github.com/nathfavour/vibeauracle/internal/doctor"
	"github.com/nathfavour/vibeauracle/internal/vibe"
	"github.com/nathfavour/vibeauracle/internal/watcher"
	"github.com/nathfavour/vibeauracle/model"
	"github.com/nathfavour/vibeauracle/prompt"
	"github.com/nathfavour/vibeauracle/sys"
	"github.com/nathfavour/vibeauracle/tooling"
	"github.com/nathfavour/vibeauracle/vault"
)

func New() *Brain {
	cm, err := sys.NewConfigManager()
	if err != nil {
		// Fatal error if we can't load config
		panic(fmt.Sprintf("failed to initialize config manager: %v", err))
	}
	cfg, _ := cm.Load()
	v, _ := vault.New("vibeauracle", cfg.DataDir)
	guard := tooling.NewSecurityGuard()

	w, _ := watcher.New()

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
		watcher:  w,
	}

	if w != nil {
		cwd, _ := os.Getwd()
		_ = w.AddRoot(cwd)
		w.SubscribeFunc(func(e watcher.Event) {
			if b.OnFilesystemEvent != nil {
				b.OnFilesystemEvent(e)
			}
		})
		w.Start()
	}

	conn, _ := connect.NewConnector(context.Background())
	b.connector = conn

	_ = b.extMgr.LoadAll()
	_ = b.extMgr.InitializeDefaults()

	b.fs = sys.NewLocalFS("")
	b.skillDirectories = b.DiscoverSkills()
	b.tools = tooling.Setup(b.fs, b.monitor, b.security)
	vibe.RegisterExtensions(context.Background(), b.extMgr, b.tools)

	b.initProvider()
	var provider model.Provider
	if b.model != nil {
		provider = b.model.Provider()
	}
	b.memory = vcontext.NewMemory(provider)
	b.prompts = prompt.New(cfg, b.memory, &prompt.NoopRecommender{}, b.model)

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
		if token, _ := auth.GetGithubCLIToken(); token != "" {
			cfg.Model.Provider = "github-copilot"
			cfg.Model.Name = "gpt-4o"
			_ = cm.Save(cfg)
		}
	}

	b.initProvider()
	go b.autodetectBestModel()

	doctor.RegisterHealer(func(issue string) {
		go b.Heal(context.Background(), issue)
	})

	return b
}

func (b *Brain) Shutdown() error {
	if b.connector != nil {
		_ = b.connector.Close()
	}
	if b.copilotProvider != nil {
		return b.copilotProvider.Stop()
	}
	return nil
}

func (b *Brain) StartConnector() string {
	if b.connector == nil {
		return ""
	}
	return b.connector.GetAddress()
}

func (b *Brain) GetConnectorAddress() string {
	if b.connector == nil {
		return ""
	}
	return b.connector.GetAddress()
}

func (b *Brain) ShareSession(sessionType, permissions, targetUser string, allowedClients []string) (string, error) {
	if b.connector == nil {
		return "", fmt.Errorf("connector not initialized")
	}
	sess := b.connector.CreateSharedSession(sessionType, permissions, targetUser, allowedClients)
	return sess.ID, nil
}

func (b *Brain) Process(ctx context.Context, reqObj interface{}) (interface{}, error) {
	var req Request

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

	if b.model == nil && !b.usingCopilotSDK {
		tooling.ReportStatus("❌", "error", "No AI model configured")
		return Response{}, fmt.Errorf("no AI model configured. Run 'vibeaura auth' to set up a provider")
	}

	sessionID := b.GetSessionID()
	session, ok := b.sessions[sessionID]
	if !ok {
		var storedSession tooling.Session
		if err := b.RecallState(sessionID+"_obj", &storedSession); err == nil {
			session = &storedSession
		} else {
			session = tooling.NewSession(sessionID)
		}
		b.sessions[sessionID] = session
	}

	snapshot, _ := b.monitor.GetSnapshot()
	tooling.ReportStatus("👁️", "perceive", fmt.Sprintf("CWD: %s", snapshot.WorkingDir))

	toolDefs := b.tools.GetPromptDefinitions(nil)
	tooling.ReportStatus("🔧", "tools", fmt.Sprintf("Loaded %d tools", len(b.tools.List())))

	b.memory.AddToWindow(req.ID, req.Content, "user_prompt")
	tooling.ReportStatus("🧠", "memory", "Analyzing conversation context...")

	recentHistory := ""
	if len(session.Threads) > 0 {
		var hb strings.Builder
		hb.WriteString("\nRECENT CONVERSATION HISTORY:\n")
		start := 0
		if len(session.Threads) > 5 {
			start = len(session.Threads) - 5
		}
		for _, t := range session.Threads[start:] {
			hb.WriteString(fmt.Sprintf("User: %s\nAssistant: %s\n", t.Prompt, t.Response))
		}
		recentHistory = hb.String()
	}

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

		if req.Intent != "" {
			promptIntent = req.Intent
		}

		tooling.ReportStatus("✅", "prompt", fmt.Sprintf("Strategy: %s", promptIntent))
	} else {
		tooling.ReportStatus("📝", "prompt", "Using fallback prompt builder")
		snippets, _ := b.memory.Recall(ctx, req.Content, snapshot.WorkingDir)
		contextStr := strings.Join(snippets, "\n")
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

	if b.config.Agent.Mode == "auracle" {
		tooling.ReportStatus("🔮", "agent-auracle", "Executing via internal Auracle...")
	}

	if b.config.Agent.Mode == "custom" {
		var activeAgent *sys.CustomAgent
		for _, a := range b.config.Agent.CustomAgents {
			if a.Name == b.config.Agent.ActiveCustom {
				activeAgent = &a
				break
			}
		}

		if activeAgent != nil {
			tooling.ReportStatus("🌌", "vibe-agent", fmt.Sprintf("Executing via Agentic Vibe: %s", activeAgent.Name))
			augmentedPrompt = fmt.Sprintf("Agentic Vibe Instructions: %s\n\n%s", activeAgent.Prompt, augmentedPrompt)

			if len(activeAgent.Tools) > 0 {
				toolDefs = b.tools.GetPromptDefinitions(activeAgent.Tools)
			}
		}
	}

	maxTurns := 10
	history := augmentedPrompt
	var fullResponse strings.Builder
	var totalUsage model.Usage
	b.detector = NewLoopDetector(10)

	for i := 0; i < maxTurns; i++ {
		tooling.ReportStatus("🔄", "loop", fmt.Sprintf("Turn %d/%d: Thinking...", i+1, maxTurns))

		var resp string
		var usage model.Usage
		var generateErr error

		if b.usingCopilotSDK && b.copilotProvider != nil {
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

		totalUsage.InputTokens += usage.InputTokens
		totalUsage.OutputTokens += usage.OutputTokens
		totalUsage.TotalTokens += usage.TotalTokens
		totalUsage.Cost += usage.Cost

		if b.detector.AddAction(resp) {
			tooling.ReportStatus("🛑", "loop-detected", "Agent stuck in a repetitive loop. Halting.")
			doctor.Send("brain", "warning", "Loop detected", map[string]any{"response": resp})
			finalContent := fullResponse.String() + "\n" + resp + "\n\n(Stopped: Loop detected)"
			return Response{Content: finalContent}, nil
		}

		if resp != "" {
			if fullResponse.Len() > 0 {
				fullResponse.WriteString("\n\n")
			}
			fullResponse.WriteString(resp)
		}

		preview := resp
		if len(preview) > 100 {
			preview = preview[:100] + "..."
		}
		tooling.ReportStatus("💬", "response", preview)

		executed, resultVal, interventionErr, execErr := b.executeToolCalls(ctx, resp, promptIntent)

		if interventionErr != nil {
			tooling.ReportStatus("⚠️", "intervention", "User approval required")
			return Response{}, interventionErr
		}

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

		_ = b.memory.Store(req.ID+"_step_"+fmt.Sprint(i), resultVal)
	}

	tooling.ReportStatus("⚠️", "limit", "Agent loop limit reached")
	finalContent := fullResponse.String() + "\n\n(Stopped: Agent loop limit reached)"
	return Response{Content: finalContent}, nil
}