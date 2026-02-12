package main

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/nathfavour/vibeauracle/brain"
	"github.com/nathfavour/vibeauracle/internal/doctor"
	"github.com/nathfavour/vibeauracle/sys"
)

func (m *model) handleChatKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	// Global overrides
	if msg.String() == "ctrl+c" {
		m.saveState()
		return m, tea.Quit
	}

	if msg.String() == "ctrl+a" {
		m.messages = append(m.messages, systemStyle.Render(" COMMIT ")+subtleStyle.Render(" Triggering autocommit..."))
		m.isThinking = true
		return m, tea.Batch(
			m.asyncRender(),
			func() tea.Msg {
				ctx := context.Background()
				// We call scm_commit without a message to trigger autocommiter/AI logic
				args := json.RawMessage(`{"all": true}`)
				res, err := m.brain.ExecuteTool(ctx, "scm_commit", args)
				return interventionResultMsg{result: res, err: err}
			},
		)
	}

	if msg.String() == "ctrl+y" {
		m.isAuracleMode = !m.isAuracleMode
		status := "DISABLED"
		if m.isAuracleMode {
			status = "ENABLED"
			m.isThinking = true
			return m, tea.Batch(
				m.asyncRender(),
				m.processRequest("AURACLE_MODE: Start autonomous project analysis and improvement loop. Be cheeky and efficient."),
			)
		}
		m.messages = append(m.messages, auracleStyle.Render(" AURACLE MODE ")+subtleStyle.Render(" "+status))
		return m, m.asyncRender()
	}

	// Disable all other interactions in Auracle mode
	if m.isAuracleMode {
		return m, nil
	}

	// Suggestion navigation
	if len(m.suggestions) > 0 {
		switch msg.String() {
		case "down":
			m.suggestionIdx = (m.suggestionIdx + 1) % len(m.suggestions)
			return m, nil
		case "up":
			m.suggestionIdx = (m.suggestionIdx - 1 + len(m.suggestions)) % len(m.suggestions)
			return m, nil
		case "enter":
			return m.applySuggestion()
		case "esc":
			m.suggestions = nil
			return m, nil
		}
	}

	// Arrow up/down when textarea is empty: cycle through prompt history
	if m.textarea.Value() == "" || m.historyIndex >= 0 {
		switch msg.String() {
		case "up":
			if len(m.promptHistory) > 0 {
				if m.historyIndex < 0 {
					// First time pressing up, save current input
					m.tempPrompt = m.textarea.Value()
					m.historyIndex = len(m.promptHistory) - 1
				} else if m.historyIndex > 0 {
					m.historyIndex--
				}
				m.textarea.SetValue(m.promptHistory[m.historyIndex])
				m.textarea.SetCursor(len(m.textarea.Value()))
				return m, nil
			}
			return m, nil
		case "down":
			if m.historyIndex >= 0 {
				if m.historyIndex < len(m.promptHistory)-1 {
					m.historyIndex++
					m.textarea.SetValue(m.promptHistory[m.historyIndex])
				} else {
					// Back to current input
					m.historyIndex = -1
					m.textarea.SetValue(m.tempPrompt)
				}
				m.textarea.SetCursor(len(m.textarea.Value()))
				return m, nil
			}
			return m, nil
		}
	}

	switch msg.String() {
	case "pgup":
		m.viewport.ViewUp()
		return m, nil
	case "pgdown":
		m.viewport.ViewDown()
		return m, nil
	case "shift+up":
		m.viewport.LineUp(1)
		return m, nil
	case "shift+down":
		m.viewport.LineDown(1)
		return m, nil
	case "esc":
		m.textarea.Reset()
		m.suggestions = nil
		m.historyIndex = -1
		m.tempPrompt = ""
		m.textarea.FocusedStyle.Text = lipgloss.NewStyle()
		return m, nil
	case "enter":
		v := m.textarea.Value()
		if strings.TrimSpace(v) == "" {
			return m, nil
		}
		if strings.HasPrefix(strings.TrimSpace(v), "/") {
			return m.handleSlashCommand(v)
		}
		// Save to prompt history
		m.promptHistory = append(m.promptHistory, v)
		if len(m.promptHistory) > 50 { // Keep last 50 prompts
			m.promptHistory = m.promptHistory[1:]
		}
		m.historyIndex = -1 // Reset history navigation
		m.tempPrompt = ""

		// Label: User
		m.messages = append(m.messages, userStyle.Render("User ")+m.styleMessage(v))
		m.textarea.Reset()
		m.textarea.FocusedStyle.Text = lipgloss.NewStyle()
		m.suggestions = nil

		m.saveState()
		m.isThinking = true
		m.wasStreaming = false // Reset streaming flag for new turn
		return m, tea.Batch(m.asyncRender(), m.processRequest(v))
	default:
		val := m.textarea.Value()
		m.updateSuggestions(val)

		// If we just typed /models /use, trigger model discovery if empty
		if strings.HasSuffix(val, "/models /use ") && len(m.allModelDiscoveries) == 0 {
			return m, m.discoverModels()
		}

		if strings.HasPrefix(val, "/") {
			m.textarea.FocusedStyle.Text = systemStyle
		} else {
			m.textarea.FocusedStyle.Text = lipgloss.NewStyle()
		}
	}
	return m, nil
}

func (m *model) handleConvoKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "up", "k":
		m.viewport.LineUp(1)
	case "down", "j":
		m.viewport.LineDown(1)
	case "pgup":
		m.viewport.ViewUp()
	case "pgdown":
		m.viewport.ViewDown()
	case "home":
		m.viewport.GotoTop()
	case "end":
		m.viewport.GotoBottom()
	}
	return m, nil
}

func (m *model) handlePerusalKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	if m.isFileOpen {
		switch msg.String() {
		case "up", "k":
			m.perusalVp.LineUp(1)
			return m, nil
		case "down", "j":
			m.perusalVp.LineDown(1)
			return m, nil
		}
	}

	switch msg.String() {
	case "up", "k":
		if m.treeCursor > 0 {
			m.treeCursor--
			m.updatePerusalContent()
		}
	case "down", "j":
		if m.treeCursor < len(m.treeEntries)-1 {
			m.treeCursor++
			m.updatePerusalContent()
		}
	case "enter":
		if len(m.treeEntries) == 0 {
			return m, nil
		}
		entry := m.treeEntries[m.treeCursor]
		path := filepath.Join(m.currentPath, entry.Name())
		if entry.IsDir() {
			m.currentPath = path
			m.treeCursor = 0
			m.loadTree(path)
		} else {
			m.openFile(path)
		}
	case "backspace":
		parent := filepath.Dir(m.currentPath)
		m.currentPath = parent
		m.treeCursor = 0
		m.loadTree(parent)
	case ":":
		// Quick command mode if needed, but for now just :i
	case "i":
		if m.isFileOpen {
			m.focus = focusEdit
			m.editArea.Focus()
		}
	}
	return m, nil
}

func (m *model) handleEditKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	if msg.String() == "ctrl+s" {
		content := m.editArea.Value()
		os.WriteFile(m.currentPath, []byte(content), 0644)
		m.focus = focusTree
		m.openFile(m.currentPath) // Refresh view
		return m, nil
	}
	return m, nil
}

func (m *model) handleSlashCommand(cmd string) (tea.Model, tea.Cmd) {
	m.textarea.Reset()
	m.suggestions = nil

	// Normalize: handle both "/models/list" and "/models /list"
	raw := strings.TrimSpace(cmd)
	var parts []string

	if strings.Contains(raw, " ") {
		parts = strings.Fields(raw)
	} else if strings.Count(raw, "/") > 1 {
		segments := strings.Split(strings.TrimPrefix(raw, "/"), "/")
		for _, s := range segments {
			if s != "" {
				parts = append(parts, "/"+s)
			}
		}
	} else {
		parts = []string{raw}
	}

	if len(parts) == 0 {
		return m, nil
	}

	// Guardrail: Ensure it's a top-level command
	isTopLevel := false
	for _, c := range allCommands {
		if c == parts[0] {
			isTopLevel = true
			break
		}
	}

	if !isTopLevel {
		// Check if it's a known subcommand run out of context
		isSub := false
		for _, subs := range subCommands {
			for _, s := range subs {
				if s == parts[0] {
					isSub = true
					break
				}
			}
			if isSub {
				break
			}
		}

		if isSub {
			m.messages = append(m.messages,
				systemStyle.Render(" COMMAND ")+"\n"+
					helpStyle.Render("That is a subcommand and cannot be run alone.")+"\n"+
					helpStyle.Render(fmt.Sprintf("Usage: %s %s", "parent", parts[0])),
			)
		} else {
			m.messages = append(m.messages, errorStyle.Render(" Unknown Command: ")+parts[0])
		}
		m.viewport.SetContent(m.renderMessages())
		m.viewport.GotoBottom()
		return m, nil
	}

	switch parts[0] {
	case "/help":
		m.messages = append(m.messages, systemStyle.Render(" COMMANDS ")+"\n"+helpStyle.Render("• /help    - Show this list\n• /status  - System resource snapshot\n• /mcp     - Manage MCP tools & servers\n• /skill   - Manage agentic vibes/skills\n• /sys     - Hardware & system details\n• /auth    - Manage AI provider credentials\n• /agent   - Select agentic runtime engine\n• /session - Manage directory-aware sessions\n• /sidebar - Toggle right sidebar visibility\n• /copy    - Copy last Q&A block to clipboard\n• /shot    - Take a beautiful TUI screenshot\n• /record  - Start/stop high-quality TUI recording\n• /cwd     - Show current directory\n• /version - Show version info\n• /update  - Check for updates immediately\n• /restart - Restart vibeauracle\n• /clear   - Clear chat history\n• /auracle - Toggle autonomous agent loop\n• /exit    - Quit vibeauracle"))
	case "/status":
		res, _ := m.brain.GetSnapshot()
		snapshot := res.(sys.Snapshot)
		status := fmt.Sprintf(systemStyle.Render(" SYSTEM ")+"\n"+helpStyle.Render("CPU: %.1f%% | Mem: %.1f%%"), snapshot.CPUUsage, snapshot.MemoryUsage)

		if m.lastUsage.TotalTokens > 0 {
			status += "\n" + systemStyle.Render(" LAST USAGE ") + "\n" + helpStyle.Render(fmt.Sprintf("Tokens: %d (In: %d, Out: %d)", m.lastUsage.TotalTokens, m.lastUsage.InputTokens, m.lastUsage.OutputTokens))
			if m.lastUsage.Cost > 0 {
				status += helpStyle.Render(fmt.Sprintf("\nCost: $%.4f", m.lastUsage.Cost))
			}
		}
		m.messages = append(m.messages, status)

	case "/cwd":
		res, _ := m.brain.GetSnapshot()
		snapshot := res.(sys.Snapshot)
		m.messages = append(m.messages, systemStyle.Render(" CWD ")+" "+helpStyle.Render(snapshot.WorkingDir))

	case "/version":
		m.messages = append(m.messages, systemStyle.Render(" VERSION ")+"\n"+helpStyle.Render(fmt.Sprintf("App: %s\nCommit: %s\nCompiler: %s", Version, Commit, runtime.Version())))
	case "/auth":
		return m.handleAuthCommand(parts)
	case "/models":
		return m.handleModelsCommand(parts)
	case "/agent":
		return m.handleAgentCommand(parts)
	case "/session":
		return m.handleSessionCommand(parts)
	case "/connect":
		return m.handleConnectCommand(parts)
	case "/share":
		return m.handleShareCommand(parts)
	case "/mcp":
		return m.handleMcpCommand(parts)
	case "/sys":
		return m.handleSysCommand(parts)
	case "/skill":
		return m.handleSkillCommand(parts)
	case "/copy":
		var lastUser, lastAI string
		// Find last AI response
		for i := len(m.messages) - 1; i >= 0; i-- {
			msg := m.messages[i]
			if strings.Contains(msg, "Auracle: ") {
				// Extract content part after label
				parts := strings.SplitN(msg, "Auracle: ", 2)
				if len(parts) == 2 {
					lastAI = stripANSI(parts[1])
				} else {
					lastAI = stripANSI(msg)
				}

				// Find last User question before this AI response
				for j := i - 1; j >= 0; j-- {
					uMsg := m.messages[j]
					if strings.Contains(uMsg, "User ") {
						uParts := strings.SplitN(uMsg, "User ", 2)
						if len(uParts) == 2 {
							lastUser = stripANSI(uParts[1])
						} else {
							lastUser = stripANSI(uMsg)
						}
						break
					}
				}
				break
			}
		}

		if lastUser != "" && lastAI != "" {
			formatted := fmt.Sprintf("Question: %s\n\nAnswer: %s", strings.TrimSpace(lastUser), strings.TrimSpace(lastAI))
			// 1. Try native Go clipboard (talks to X11/Wayland/Win/Mac APIs directly)
			writeToClipboard(formatted)

			m.messages = append(m.messages, subtleStyle.Render("✓ Copied Q&A block to clipboard"))
		} else {
			m.messages = append(m.messages, errorStyle.Render(" COPY ERROR ")+"\nNo Q&A block found to copy.")
		}
		return m, m.asyncRender()

	case "/shot":
		return m.takeScreenshot()

	case "/record":
		return m.toggleRecording()

	case "/show-tree", "/sidebar":
		m.showTree = !m.showTree
		// trigger resize
		return m, func() tea.Msg { return tea.WindowSizeMsg{Width: m.width, Height: m.height} }

	case "/clear":
		m.messages = []string{}
		m.historyRendered = ""
		ensureBanner(&m.messages, m.banner)
		m.messages = append(m.messages, "Type "+systemStyle.Render("/help")+" to see available commands.")
		m.saveState()
		return m, m.asyncRenderWithPos(true, false, 0)

	case "/heal":
		issue := "Analyze and fix current project failures"
		if len(parts) > 1 {
			issue = strings.Join(parts[1:], " ")
		}
		m.messages = append(m.messages, systemStyle.Render(" HEALING ")+" "+helpStyle.Render(issue))
		m.isThinking = true
		return m, tea.Batch(
			m.asyncRender(),
			func() tea.Msg {
				ctx := context.Background()
				resp, err := m.brain.Heal(ctx, issue)
				if err != nil {
					resp.Error = err
				}
				return resp
			},
		)
	case "/exit":
		return m, tea.Quit
	case "/update":
		m.messages = append(m.messages, systemStyle.Render(" UPDATE ")+"\n"+helpStyle.Render("Checking for latest release..."))
		return m, tea.Batch(m.asyncRender(), m.updater.CheckUpdateCmd(true))
	case "/restart":
		m.saveState()
		restartSelf()
		return m, tea.Quit // Fallback if restartSelf doesn't exec
	case "/auracle":
		m.isAuracleMode = !m.isAuracleMode
		status := "DISABLED"
		if m.isAuracleMode {
			status = "ENABLED"
			m.isThinking = true
			prompt := "AURACLE_MODE: Start autonomous project analysis and improvement loop. Be cheeky and efficient."
			if len(parts) > 1 {
				prompt = "AURACLE_MODE: " + strings.Join(parts[1:], " ")
			}
			m.messages = append(m.messages, auracleStyle.Render(" AURACLE MODE ")+subtleStyle.Render(" ENABLED"))
			return m, tea.Batch(
				m.asyncRender(),
				m.processRequest(prompt),
			)
		}
		m.messages = append(m.messages, auracleStyle.Render(" AURACLE MODE ")+subtleStyle.Render(" "+status))
		return m, m.asyncRender()
	default:
		// Check for dynamic commands
		if cmd, ok := m.dynamicCommands[parts[0]]; ok {
			// For TUI slash commands, we simulate the CLI behavior
			// Find the extension that owns this command
			for _, ext := range m.brain.Extensions() {
				if !ext.Enabled || ext.Manifest == nil {
					continue
				}
				for _, c := range ext.Manifest.CLICommands {
					if "/"+c.Name == parts[0] {
						m.messages = append(m.messages, systemStyle.Render(" EXTENSION ")+" "+helpStyle.Render(fmt.Sprintf("Executing %s %s...", ext.Name, cmd.Action)))
						m.viewport.SetContent(m.renderMessages())
						m.viewport.GotoBottom()

						// Execute and return result as a message
						execCmd := exec.Command(ext.Manifest.Command, cmd.Action)
						out, err := execCmd.CombinedOutput()
						if err != nil {
							m.messages = append(m.messages, errorStyle.Render(" ERROR ")+"\n"+err.Error())
						} else {
							m.messages = append(m.messages, aiStyle.Render(ext.Name+": ")+"\n"+string(out))
						}
						m.viewport.SetContent(m.renderMessages())
						m.viewport.GotoBottom()
						return m, nil
					}
				}
			}
		}
		m.messages = append(m.messages, errorStyle.Render(" Unknown Command: ")+parts[0])
	}
	m.viewport.SetContent(m.renderMessages())
	m.viewport.GotoBottom()
	return m, nil
}

func (m *model) handleAuthCommand(parts []string) (tea.Model, tea.Cmd) {
	if len(parts) < 2 {
		m.messages = append(m.messages, systemStyle.Render(" AUTH ")+"\n"+helpStyle.Render("Manage your AI provider credentials.\n\nUsage: /auth <provider> [key/endpoint]\nProviders: /ollama, /github-models, /github-copilot, /copilot-sdk, /openai, /anthropic"))
		return m, nil
	}

	provider := strings.ToLower(parts[1])
	switch provider {
	case "/copilot-sdk", "copilot-sdk":
		m.messages = append(m.messages, systemStyle.Render(" COPILOT SDK ")+"\n"+helpStyle.Render("Using GitHub Copilot SDK. No token required if 'gh' CLI is authenticated.\nTo use BYOK (OpenAI/Anthropic), provide the key for the respective provider (e.g. /auth /openai <key>)"))
	case "/ollama", "ollama":
		if len(parts) > 2 {
			endpoint := parts[2]
			cfg := m.brain.Config().(*sys.Config)
			cfg.Model.Endpoint = endpoint
			if err := m.brain.UpdateConfig(cfg); err != nil {
				m.messages = append(m.messages, errorStyle.Render(" CONFIG ERROR ")+"\n"+err.Error())
			} else {
				m.messages = append(m.messages, systemStyle.Render(" OLLAMA ")+"\n"+helpStyle.Render(fmt.Sprintf("Ollama endpoint set to: %s", endpoint)))
			}
		} else {
			m.messages = append(m.messages, systemStyle.Render(" OLLAMA ")+"\n"+helpStyle.Render("Ollama is usually active on http://localhost:11434.\nTo use a custom host: /auth /ollama <endpoint>"))
		}

	case "/github-models", "github-models":
		if len(parts) > 2 {
			err := m.brain.StoreSecret("github_models_pat", parts[2])
			if err != nil {
				m.messages = append(m.messages, errorStyle.Render(" VAULT ERROR ")+"\n"+err.Error())
			} else {
				m.messages = append(m.messages, systemStyle.Render(" GITHUB MODELS ")+"\n"+helpStyle.Render("GitHub Models PAT received and stored securely."))
			}
		} else {
			m.messages = append(m.messages, systemStyle.Render(" GITHUB MODELS ")+"\n"+helpStyle.Render("Special BYOK method for GitHub AI Models.\nUsage: /auth /github-models <your-pat-token>"))
		}
	case "/github-copilot", "github-copilot":
		m.messages = append(m.messages, systemStyle.Render(" GITHUB COPILOT ")+"\n"+errorStyle.Render(" Not yet integrated "))
	case "/openai", "openai", "/anthropic", "anthropic":
		if len(parts) > 2 {
			providerName := strings.TrimPrefix(provider, "/")
			err := m.brain.StoreSecret(providerName+"_api_key", parts[2])
			if err != nil {
				m.messages = append(m.messages, errorStyle.Render(" VAULT ERROR ")+"\n"+err.Error())
			} else {
				m.messages = append(m.messages, systemStyle.Render(strings.ToUpper(providerName))+"\n"+helpStyle.Render(fmt.Sprintf("%s API key received and stored securely.", strings.Title(providerName))))
			}

			// Optional: set custom endpoint if provided as 3rd arg
			if len(parts) > 3 {
				endpoint := parts[3]
				cfg := m.brain.Config().(*sys.Config)
				cfg.Model.Endpoint = endpoint
				if err := m.brain.UpdateConfig(cfg); err == nil {
					m.messages = append(m.messages, helpStyle.Render("Endpoint set to: "+endpoint))
				}
			}
		} else {
			providerTitle := strings.Title(strings.TrimPrefix(provider, "/"))
			m.messages = append(m.messages, systemStyle.Render(strings.ToUpper(providerTitle))+"\n"+helpStyle.Render(fmt.Sprintf("Usage: /auth %s <api-key> [endpoint]", provider)))
		}
	default:
		m.messages = append(m.messages, systemStyle.Render(" AUTH ")+"\n"+errorStyle.Render(fmt.Sprintf(" Provider '%s' not yet integrated ", provider)))
	}

	m.viewport.SetContent(m.renderMessages())
	m.viewport.GotoBottom()
	return m, nil
}

func (m *model) handleModelsCommand(parts []string) (tea.Model, tea.Cmd) {
	if len(parts) < 2 || parts[1] == "/list" || parts[1] == "list" {
		m.messages = append(m.messages, systemStyle.Render(" DISCOVERING MODELS ")+"\n"+subtleStyle.Render("Querying active providers..."))
		m.viewport.SetContent(m.renderMessages())
		m.viewport.GotoBottom()

		return m, func() tea.Msg {
			discoveries, err := m.brain.DiscoverModels(context.Background())
			if err != nil {
				return brain.Response{Error: err}
			}

			var sb strings.Builder
			sb.WriteString(systemStyle.Render(" AVAILABLE MODELS ") + "\n")
			if len(discoveries) == 0 {
				sb.WriteString(helpStyle.Render("No models found. Check /auth to configure providers."))
			} else {
				for _, d := range discoveries {
					sb.WriteString(fmt.Sprintf("%s %s\n",
						aiStyle.Render("• "+d.Name),
						subtleStyle.Render("("+d.Provider+")")))
				}
				sb.WriteString("\n" + helpStyle.Render("Use /models /use <provider> <model> to switch."))
			}
			return brain.Response{Content: sb.String()}
		}
	}

	sub := strings.ToLower(parts[1])
	if (sub == "/use" || sub == "use") && len(parts) >= 4 {
		provider := parts[2]
		modelName := parts[3]
		err := m.brain.SetModel(provider, modelName)
		if err != nil {
			m.messages = append(m.messages, errorStyle.Render(" SWITCH ERROR ")+"\n"+err.Error())
		} else {
			m.messages = append(m.messages, systemStyle.Render(" MODEL SWITCHED ")+"\n"+helpStyle.Render(fmt.Sprintf("Now using %s via %s", modelName, provider)))
		}
	} else if sub == "/use" || sub == "use" {
		m.messages = append(m.messages, systemStyle.Render(" MODELS ")+"\n"+helpStyle.Render("Usage: /models /use <provider> <model_name>")+"\n"+subtleStyle.Render("Tip: Use the interactive selector by typing '/models /use ' and scrolling."))
	} else if sub == "/pull" || sub == "pull" {
		if len(parts) >= 3 {
			modelName := parts[2]
			m.messages = append(m.messages, systemStyle.Render(" OLLAMA PULL ")+"\n"+helpStyle.Render("Requesting pull for: "+modelName))
			return m, m.pullOllamaModel(modelName)
		}
		m.messages = append(m.messages, systemStyle.Render(" MODELS ")+"\n"+helpStyle.Render("Usage: /models /pull <model_name>")+"\n"+subtleStyle.Render("Example: /models /pull llama3.2"))
	} else {
		m.messages = append(m.messages, errorStyle.Render(" Unknown MODELS subcommand: ")+sub)
	}

	m.viewport.SetContent(m.renderMessages())
	m.viewport.GotoBottom()
	return m, nil
}

func (m *model) handleAgentCommand(parts []string) (tea.Model, tea.Cmd) {
	if len(parts) < 2 {
		cfg := m.brain.Config().(*sys.Config).Agent
		msg := systemStyle.Render(" AGENT MODE ") + "\n"
		msg += helpStyle.Render(fmt.Sprintf("Current engine: %s", cfg.Mode))
		if cfg.Mode == "custom" {
			msg += helpStyle.Render(fmt.Sprintf(" (%s)", cfg.ActiveCustom))
		}
		msg += "\n\n" + helpStyle.Render("Usage: /agent <mode>\nModes: /auracle, /sdk, /custom (Agentic Vibes)")
		msg += "\n\n" + helpStyle.Render("Subcommands for /custom (Agentic Vibes):\n• /agent /custom /list\n• /agent /custom /use <name>\n• /agent /custom /add <name> <prompt>")
		m.messages = append(m.messages, msg)
		m.viewport.SetContent(m.renderMessages())
		m.viewport.GotoBottom()
		return m, nil
	}

	sub := strings.ToLower(parts[1])
	if sub == "/custom" {
		if len(parts) < 3 || parts[2] == "/list" {
			agents := m.brain.GetCustomAgents()
			var sb strings.Builder
			sb.WriteString(systemStyle.Render(" AGENTIC VIBES ") + "\n")
			if len(agents) == 0 {
				sb.WriteString(helpStyle.Render("No agentic vibes registered. Use /agent /custom /add to create one."))
			} else {
				for _, a := range agents {
					sb.WriteString(fmt.Sprintf("%s %s\n", aiStyle.Render("• "+a.Name), helpStyle.Render(a.Description)))
				}
			}
			m.messages = append(m.messages, sb.String())
		} else if parts[2] == "/use" && len(parts) >= 4 {
			name := parts[3]
			if err := m.brain.SetActiveCustomAgent(name); err != nil {
				m.messages = append(m.messages, errorStyle.Render(" VIBE ERROR ")+"\n"+err.Error())
			} else {
				m.messages = append(m.messages, systemStyle.Render(" VIBE SWITCHED ")+"\n"+helpStyle.Render(fmt.Sprintf("🌌 Now using agentic vibe: %s", name)))
			}
		} else if parts[2] == "/add" && len(parts) >= 5 {
			name := parts[3]
			prompt := strings.Join(parts[4:], " ")
			err := m.brain.RegisterCustomAgent(sys.CustomAgent{
				Name:   name,
				Prompt: prompt,
			})
			if err != nil {
				m.messages = append(m.messages, errorStyle.Render(" VIBE ERROR ")+"\n"+err.Error())
			} else {
				m.messages = append(m.messages, systemStyle.Render(" VIBE ADDED ")+"\n"+helpStyle.Render(fmt.Sprintf("🌌 Agentic vibe '%s' registered.", name)))
			}
		} else {
			m.messages = append(m.messages, errorStyle.Render(" Unknown custom subcommand: ")+parts[2])
		}
		m.viewport.SetContent(m.renderMessages())
		m.viewport.GotoBottom()
		return m, nil
	}

	mode := strings.TrimPrefix(sub, "/")
	err := m.brain.SetAgentMode(mode)
	if err != nil {
		m.messages = append(m.messages, errorStyle.Render(" AGENT ERROR ")+"\n"+err.Error())
	} else {
		icon := "🔮"
		if mode == "sdk" {
			icon = "🚀"
		} else if mode == "custom" {
			icon = "🌌"
		}
		m.messages = append(m.messages, systemStyle.Render(" AGENT SWITCHED ")+"\n"+helpStyle.Render(fmt.Sprintf("%s Now using %s agentic runtime engine.", icon, strings.ToUpper(mode))))
	}

	m.viewport.SetContent(m.renderMessages())
	m.viewport.GotoBottom()
	return m, nil
}

func (m *model) handleSessionCommand(parts []string) (tea.Model, tea.Cmd) {
	if len(parts) < 2 {
		path := m.brain.GetSessionPath()
		msg := systemStyle.Render(" SESSION ") + "\n"
		msg += helpStyle.Render(fmt.Sprintf("Current Path: %s", path))
		msg += "\n" + helpStyle.Render(fmt.Sprintf("ID: %s", m.brain.GetSessionID()))
		msg += "\n\n" + helpStyle.Render("Usage: /session <subcommand>\nSubcommands: /list, /clear")
		m.messages = append(m.messages, msg)
		m.viewport.SetContent(m.renderMessages())
		m.viewport.GotoBottom()
		return m, nil
	}

	sub := strings.ToLower(parts[1])
	switch sub {
	case "/list", "list":
		sessions, err := m.brain.ListSessions()
		if err != nil {
			m.messages = append(m.messages, errorStyle.Render(" SESSION ERROR ")+"\n"+err.Error())
		} else {
			var sb strings.Builder
			sb.WriteString(systemStyle.Render(" STORED SESSIONS ") + "\n")
			if len(sessions) == 0 {
				sb.WriteString(helpStyle.Render("No stored sessions found."))
			} else {
				for _, s := range sessions {
					sb.WriteString(fmt.Sprintf("%s %s\n", aiStyle.Render("•"), helpStyle.Render(s)))
				}
				sb.WriteString("\n" + helpStyle.Render("Sessions are identified by directory hash."))
			}
			m.messages = append(m.messages, sb.String())
		}
	case "/clear", "clear":
		sessionID := m.brain.GetSessionID()
		if err := m.brain.ClearState(sessionID); err != nil {
			m.messages = append(m.messages, errorStyle.Render(" SESSION ERROR ")+"\n"+err.Error())
		} else {
			m.messages = append(m.messages, systemStyle.Render(" SESSION CLEARED ")+"\n"+helpStyle.Render(fmt.Sprintf("Cleared history for current directory.")))
			// Reset current UI state too
			m.messages = []string{}
			ensureBanner(&m.messages, m.banner)
			m.messages = append(m.messages, subtleStyle.Render("Session: ")+aiStyle.Render(m.brain.GetSessionPath()))
			m.viewport.SetContent(m.renderMessages())
			m.viewport.GotoTop()
		}
	default:
		m.messages = append(m.messages, errorStyle.Render(" Unknown SESSION subcommand: ")+sub)
	}

	m.viewport.SetContent(m.renderMessages())
	m.viewport.GotoBottom()
	return m, nil
}

func (m *model) handleMcpCommand(parts []string) (tea.Model, tea.Cmd) {
	if len(parts) < 2 {
		m.messages = append(m.messages, systemStyle.Render(" MCP ")+"\n"+helpStyle.Render("Manage Model Context Protocol servers.\n\nUsage: /mcp <subcommand>\nSubcommands: /list, /add, /logs, /call"))
		return m, nil
	}

	sub := strings.ToLower(parts[1])
	switch sub {
	case "/list", "list":
		m.messages = append(m.messages, systemStyle.Render(" MCP SERVERS ")+"\n"+helpStyle.Render("• github (stdio) - tools: github_query\n• postgres (stdio) - tools: postgres_exec"))
	case "/add", "add":
		m.messages = append(m.messages, systemStyle.Render(" MCP ")+"\n"+helpStyle.Render("Usage: /mcp /add <name> <command> [args...]"))
	case "/logs", "logs":
		m.messages = append(m.messages, systemStyle.Render(" MCP LOGS ")+"\n"+subtleStyle.Render("Waiting for MCP traffic..."))
	case "/call", "call":
		m.messages = append(m.messages, systemStyle.Render(" MCP CALL ")+"\n"+helpStyle.Render("Usage: /mcp /call <tool_name> <json_args>"))
	default:
		m.messages = append(m.messages, errorStyle.Render(" Unknown MCP subcommand: ")+sub)
	}

	m.viewport.SetContent(m.renderMessages())
	m.viewport.GotoBottom()
	return m, nil
}

func (m *model) handleSysCommand(parts []string) (tea.Model, tea.Cmd) {
	if len(parts) < 2 {
		m.messages = append(m.messages, systemStyle.Render(" SYS ")+"\n"+helpStyle.Render("System and hardware intimacy controls.\n\nUsage: /sys <subcommand>\nSubcommands: /stats, /env, /update, /logs"))
		return m, nil
	}

	sub := strings.ToLower(parts[1])
	switch sub {
	case "/stats", "stats":
		res, _ := m.brain.GetSnapshot()
		snapshot := res.(sys.Snapshot)
		stats := fmt.Sprintf(systemStyle.Render(" POWER SNAPSHOT ")+"\n"+
			helpStyle.Render("OS: %s | Arch: %s\nCPU: %.1f%% | Mem: %.1f%%\nGoroutines: %d"),
			runtime.GOOS, runtime.GOARCH, snapshot.CPUUsage, snapshot.MemoryUsage, runtime.NumGoroutine())

		m.messages = append(m.messages, stats)
	case "/env", "env":
		m.messages = append(m.messages, systemStyle.Render(" ENVIRONMENT ")+"\n"+helpStyle.Render("Limited view (Filtered for security)\nSHELL: %s\nPATH: %s..."), os.Getenv("SHELL"), os.Getenv("PATH")[:30])
	case "/update", "update":
		// This uses the logic from update.go
		m.messages = append(m.messages, systemStyle.Render(" UPDATE ")+"\n"+helpStyle.Render("Checking for latest release on GitHub..."))
		// In a real implementation, we would return a Cmd here to run the update check
	case "/logs", "logs":
		recent := doctor.GetRecentLogs(20)
		var sb strings.Builder
		sb.WriteString(systemStyle.Render(" RECENT LOGS ") + "\n")
		if len(recent) == 0 {
			sb.WriteString(helpStyle.Render("No recent logs found."))
		} else {
			for _, log := range recent {
				icon := "ℹ️"
				switch log.Type {
				case "error":
					icon = "❌"
				case "warning":
					icon = "⚠️"
				case "init":
					icon = "🚀"
				}
				sb.WriteString(fmt.Sprintf("%s %s: %s\n", icon, aiStyle.Render(log.Source), log.Message))
				if log.Extra != nil {
					extraBytes, _ := json.Marshal(log.Extra)
					sb.WriteString(subtleStyle.Render(fmt.Sprintf("   %s", string(extraBytes))) + "\n")
				}
			}
		}
		m.messages = append(m.messages, sb.String())
	default:
		m.messages = append(m.messages, errorStyle.Render(" Unknown SYS subcommand: ")+sub)
	}

	m.viewport.SetContent(m.renderMessages())
	m.viewport.GotoBottom()
	return m, nil
}

func (m *model) handleSkillCommand(parts []string) (tea.Model, tea.Cmd) {
	if len(parts) < 2 {
		m.messages = append(m.messages, systemStyle.Render(" SKILL ")+"\n"+helpStyle.Render("Manage Brain capabilities (Vibes).\n\nUsage: /skill <subcommand>\nSubcommands: /list, /info, /load, /disable"))
		return m, nil
	}

	sub := strings.ToLower(parts[1])
	switch sub {
	case "/list", "list":
		skills := m.brain.GetSkills()
		if len(skills) == 0 {
			m.messages = append(m.messages, systemStyle.Render(" ACTIVE SKILLS ")+"\n"+helpStyle.Render("No localized skills (.agent/skills) discovered in this project."))
		} else {
			var sb strings.Builder
			sb.WriteString(systemStyle.Render(" DISCOVERED PROJECT SKILLS "))
			sb.WriteString("\n")
			for _, s := range skills {
				sb.WriteString(helpStyle.Render(fmt.Sprintf("• %s (%s)", s.Name, s.Path)))
				sb.WriteString("\n")
			}
			m.messages = append(m.messages, sb.String())
		}
	case "/info", "info":
		m.messages = append(m.messages, systemStyle.Render(" SKILL INFO ")+"\n"+helpStyle.Render("Usage: /skill /info <skill_id>"))
	case "/load", "load":
		m.messages = append(m.messages, systemStyle.Render(" LOAD SKILL ")+"\n"+helpStyle.Render("Usage: /skill /load <path_or_url>"))
	case "/disable", "disable":
		m.messages = append(m.messages, systemStyle.Render(" DISABLE SKILL ")+"\n"+helpStyle.Render("Usage: /skill /disable <skill_id>"))
	default:
		m.messages = append(m.messages, errorStyle.Render(" Unknown SKILL subcommand: ")+sub)
	}

	m.viewport.SetContent(m.renderMessages())
	m.viewport.GotoBottom()
	return m, nil
}

func (m *model) handleConnectCommand(parts []string) (tea.Model, tea.Cmd) {
	if len(parts) < 2 {
		m.messages = append(m.messages, systemStyle.Render(" CONNECT ") + "\n" + helpStyle.Render("Remote access and P2P tunneling.\n\nUsage: /connect <subcommand>\nSubcommands: /list, /new, /join, /close, /clients"))
		return m, nil
	}

	sub := strings.ToLower(parts[1])
	switch sub {
	case "/list", "list":
		m.messages = append(m.messages, systemStyle.Render(" CONNECTIONS ") + "\n" + helpStyle.Render("No active P2P connections."))
	case "/new", "new":
		m.messages = append(m.messages, systemStyle.Render(" CONNECT ") + "\n" + helpStyle.Render("Starting new P2P listener..."))
		addr := m.brain.StartConnector()
		if addr == "" {
			m.messages = append(m.messages, errorStyle.Render(" ERROR ") + " Failed to start connector.")
		} else {
			m.messages = append(m.messages, subtleStyle.Render("Listening on: ")+aiStyle.Render(addr))
		}
	case "/join", "join":
		if len(parts) < 3 {
			m.messages = append(m.messages, systemStyle.Render(" CONNECT ") + "\n" + errorStyle.Render(" Missing address. Usage: /connect /join <address> "))
		} else {
			m.messages = append(m.messages, systemStyle.Render(" CONNECT ") + "\n" + helpStyle.Render(fmt.Sprintf("Attempting to join: %s", parts[2])))
		}
	case "/close", "close":
		m.messages = append(m.messages, systemStyle.Render(" CONNECT ") + "\n" + helpStyle.Render("Closing all connections."))
	case "/clients", "clients":
		m.messages = append(m.messages, systemStyle.Render(" CLIENTS ") + "\n" + helpStyle.Render("Manage authorized clients and registration.\nUsage: /connect /clients <subcommand>\nSubcommands: /list, /reg, /rev"))
	default:
		m.messages = append(m.messages, errorStyle.Render(" Unknown CONNECT subcommand: ")+sub)
	}

	m.viewport.SetContent(m.renderMessages())
	m.viewport.GotoBottom()
	return m, nil
}

func (m *model) handleShareCommand(parts []string) (tea.Model, tea.Cmd) {
	if len(parts) < 2 {
		m.messages = append(m.messages, systemStyle.Render(" SHARE ") + "\n" + helpStyle.Render("Share your session with others.\n\nUsage: /share <subcommand> [flags]\nSubcommands: /browser, /tui, /stop\nFlags: --ro (Read-Only), --rw (Read-Write), --to <userID>"))
		return m, nil
	}

	const DefaultBaseURI = "https://example.com" // Undecided base URI

	permissions := "ro" // Default to Read-Only
	targetUser := ""
	var allowedClients []string

	// Basic flag parsing
	for i, p := range parts {
		if p == "--rw" {
			permissions = "rw"
		} else if p == "--ro" {
			permissions = "ro"
		} else if p == "--to" && i+1 < len(parts) {
			targetUser = parts[i+1]
		}
	}

	sub := strings.ToLower(parts[1])
	switch sub {
	case "/browser", "browser":
		id, err := m.brain.ShareSession("browser", permissions, targetUser, allowedClients)
		if err != nil {
			m.messages = append(m.messages, errorStyle.Render(" ERROR ") + " " + err.Error())
		} else {
			shareUrl := fmt.Sprintf("%s/shared/%s", DefaultBaseURI, id)
			permDesc := "Read-Only"
			if permissions == "rw" {
				permDesc = "Read-Write"
			}
			msg := fmt.Sprintf("Session shared to browser (%s)!", permDesc)
			if targetUser != "" {
				msg = fmt.Sprintf("Session shared to user %s (%s)!", targetUser, permDesc)
			}
			m.messages = append(m.messages, systemStyle.Render(" SHARE ") + "\n" + helpStyle.Render(msg) + "\n" + aiStyle.Render(shareUrl))
		}
	case "/tui", "tui":
		id, err := m.brain.ShareSession("tui", permissions, targetUser, allowedClients)
		if err != nil {
			m.messages = append(m.messages, errorStyle.Render(" ERROR ") + " " + err.Error())
		} else {
			permDesc := "Read-Only"
			if permissions == "rw" {
				permDesc = "Read-Write"
			}
			msg := fmt.Sprintf("Session shared for TUI clients (%s)!", permDesc)
			if targetUser != "" {
				msg = fmt.Sprintf("Session shared for user %s (%s)!", targetUser, permDesc)
			}
			m.messages = append(m.messages, systemStyle.Render(" SHARE ") + "\n" + helpStyle.Render(msg) + "\n" + subtleStyle.Render(fmt.Sprintf("Recipients can run: /connect /join auracle://%s", id)))
		}
	case "/stop", "stop":
		m.messages = append(m.messages, systemStyle.Render(" SHARE ") + "\n" + helpStyle.Render("Sharing stopped."))
	default:
		m.messages = append(m.messages, errorStyle.Render(" Unknown SHARE subcommand: ")+sub)
	}

	m.viewport.SetContent(m.renderMessages())
	m.viewport.GotoBottom()
	return m, nil
}
