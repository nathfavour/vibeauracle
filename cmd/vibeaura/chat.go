package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"strings"

	"github.com/charmbracelet/bubbles/textarea"
	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/google/uuid"
	"github.com/nathfavour/vibeauracle/brain"
	vmodel "github.com/nathfavour/vibeauracle/model"
	"github.com/nathfavour/vibeauracle/prompt"
	"github.com/nathfavour/vibeauracle/reactor"
	"github.com/nathfavour/vibeauracle/sys"
	"github.com/nathfavour/vibeauracle/tooling"
)

func initialModel(b *brain.Brain) *model {
	// Initialize native clipboard
	_ = initClipboard()

	ta := textarea.New()
	ta.Placeholder = "Send a message or type / for commands..."
	ta.Focus()
	ta.Prompt = "│ "
	ta.CharLimit = 5000
	ta.SetWidth(60)
	ta.SetHeight(3)
	ta.FocusedStyle.CursorLine = lipgloss.NewStyle()
	ta.ShowLineNumbers = false
	ta.FocusedStyle.Prompt = lipgloss.NewStyle().Foreground(lipgloss.Color("#FF00D7")).Bold(true).SetString("┃ ")
	ta.FocusedStyle.CursorLine = lipgloss.NewStyle()
	ta.ShowLineNumbers = false
	ta.FocusedStyle.Text = lipgloss.NewStyle()

	ea := textarea.New()
	ea.Placeholder = "Edit file... (Esc to cancel, Ctrl+S to save)"
	ea.ShowLineNumbers = true
	ea.SetWidth(60)
	ea.SetHeight(20)

	vp := viewport.New(60, 15)
	pvp := viewport.New(60, 15)

	cwd, _ := os.Getwd()

	banner := buildBanner(vp.Width)

	m := &model{
		textarea:    ta,
		editArea:    ea,
		viewport:    vp,
		perusalVp:   pvp,
		messages:    []string{},
		brain:       b,
		focus:       focusInput,
		currentPath: cwd,
		showTree:    true, // Show tree by default
		banner:      banner,

		// Thinking / Agentic Process State
		thinkingLog: []StatusEvent{},
		isThinking:  false,

		updater: NewAsyncUpdateManager(),

		// Prompt History
		promptHistory: []string{},
		historyIndex:  -1, // -1 means not browsing history

		// Dynamic Commands from Extensions
		dynamicCommands: make(map[string]brain.CLICommand),

		// Non-blocking Engine
		reactor: reactor.New(),
		md:      reactor.NewMarkdownRenderer(vp.Width, b.Config().(*sys.Config).UI.Theme),

		// Anyisland Management
		isManaged: sys.IsManagedByAnyisland(),
	}

	m.loadDynamicCommands()
	// Load initial tree
	m.loadTree(cwd)

	// Attempt to restore state
	// Priority 1: Hot-Swap State (explicit file path)
	if resumeStateFile != "" {
		content, err := os.ReadFile(resumeStateFile)
		if err == nil {
			var state chatState
			if json.Unmarshal(content, &state) == nil {
				m.messages = state.Messages
				m.textarea.SetValue(state.Input)
				// Clean up the temp file
				os.Remove(resumeStateFile)

				// Append a system note about the update
				ensureBanner(&m.messages, banner)

				updateMsg := lipgloss.NewStyle().
					Border(lipgloss.RoundedBorder()).
					BorderForeground(lipgloss.Color("62")).
					Padding(0, 1).
					Foreground(lipgloss.Color("10")).
					Render(fmt.Sprintf("⚡ UPDATED TO %s", "LATEST"))

				m.messages = append(m.messages, updateMsg)

				m.viewport.SetContent(m.renderMessages())
				m.viewport.GotoBottom()
				return m
			}
		}
	}

	// Priority 2: Persistent Session State (Brain Memory)
	var state chatState
	sessionID := b.GetSessionID()
	if err := b.RecallState(sessionID, &state); err == nil && len(state.Messages) > 0 {
		m.messages = state.Messages
		m.promptHistory = state.PromptHistory
		m.showTree = state.ShowSidebar
		ensureBanner(&m.messages, banner)
		m.textarea.SetValue(state.Input)
		m.viewport.SetContent(m.renderMessages())
		if m.viewport.TotalLineCount() <= m.viewport.Height {
			m.viewport.GotoTop()
		} else {
			m.viewport.GotoBottom()
		}
	} else {
		m.messages = append(m.messages, banner)

		// Seamless Welcome for configured AI providers
		provider := b.Config().(*sys.Config).Model.Provider
		switch provider {
		case "copilot-sdk":
			welcome := lipgloss.NewStyle().
				Foreground(lipgloss.Color("#FAFAFA")).
				Background(lipgloss.Color("#7D56F4")). // VibeAuracle Purple
				Padding(0, 1).
				Bold(true).
				Render(" 🚀 COPILOT SDK ACTIVE ")

			user := b.GetIdentity()
			identity := ""
			if user != "" {
				identity = subtleStyle.Render("Logged in as ") + aiStyle.Render(user)
			}

			m.messages = append(m.messages, welcome+" "+identity)
			m.messages = append(m.messages, subtleStyle.Render("Powered by GitHub Copilot SDK. Tool-intimacy enabled."))
		case "github-copilot", "github-models":
			user := b.GetIdentity()
			welcome := lipgloss.NewStyle().
				Foreground(lipgloss.Color("#FAFAFA")).
				Background(lipgloss.Color("#24292e")). // GitHub Dark Gray
				Padding(0, 1).
				Bold(true).
				Render(" 🐙 GITHUB COPILOT ACTIVE ")

			identity := ""
			if user != "" {
				identity = subtleStyle.Render("Logged in as ") + aiStyle.Render(user)
			}

			m.messages = append(m.messages, welcome+" "+identity)
			m.messages = append(m.messages, subtleStyle.Render("Zero-config integration successful. I'm ready to assist."))
		case "openai":
			welcome := lipgloss.NewStyle().
				Foreground(lipgloss.Color("#FAFAFA")).
				Background(lipgloss.Color("#10a37f")). // OpenAI Green
				Padding(0, 1).
				Bold(true).
				Render(" 🤖 OPENAI CONNECTED ")
			m.messages = append(m.messages, welcome)
			m.messages = append(m.messages, subtleStyle.Render("Using OpenAI API. Model: "+b.Config().(*sys.Config).Model.Name))
		case "anthropic":
			welcome := lipgloss.NewStyle().
				Foreground(lipgloss.Color("#FAFAFA")).
				Background(lipgloss.Color("#CC785C")). // Anthropic Orange
				Padding(0, 1).
				Bold(true).
				Render(" 🧠 ANTHROPIC CONNECTED ")
			m.messages = append(m.messages, welcome)
			m.messages = append(m.messages, subtleStyle.Render("Using Anthropic API. Model: "+b.Config().(*sys.Config).Model.Name))
		case "ollama":
			welcome := lipgloss.NewStyle().
				Foreground(lipgloss.Color("#FAFAFA")).
				Background(lipgloss.Color("#7D56F4")). // VibeAuracle Purple
				Padding(0, 1).
				Bold(true).
				Render(" 🦙 LOCAL OLLAMA ")
			m.messages = append(m.messages, welcome)
			m.messages = append(m.messages, subtleStyle.Render("Running locally. Model: "+b.Config().(*sys.Config).Model.Name))
		default:
			m.messages = append(m.messages, "Type "+systemStyle.Render("/help")+" to see available commands.")
		}
		m.messages = append(m.messages, subtleStyle.Render("Session: ")+aiStyle.Render(m.brain.GetSessionPath()))
		m.renderMessages()
		m.updateViewport()
		m.viewport.GotoTop()
	}

	return m
}

func (m *model) Init() tea.Cmd {
	cmds := []tea.Cmd{
		textarea.Blink,
		waitForStatus(),
	}

	if !m.isManaged {
		cmds = append(cmds,
			m.updater.CheckUpdateCmd(false), // Initial check
			waitForUpdateTick(),             // Schedule next check
		)
	}

	return tea.Batch(cmds...)
}

func (m *model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var (
		tiCmd tea.Cmd
		vpCmd tea.Cmd
		eaCmd tea.Cmd
		pvCmd tea.Cmd
	)

	// Mark state as dirty for any message that isn't a recording tick
	if _, ok := msg.(recordTickMsg); !ok {
		m.isDirty = true
	}

	// Update components based on focus and message type
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch m.focus {
		case focusInput:
			m.textarea, tiCmd = m.textarea.Update(msg)
			
			// Adaptive Height: Adjust textarea height based on content
			lines := strings.Count(m.textarea.Value(), "\n") + 1
			newHeight := lines
			if newHeight < 3 {
				newHeight = 3
			}
			if newHeight > 10 {
				newHeight = 10
			}
			if newHeight != m.textarea.Height() {
				m.textarea.SetHeight(newHeight)
				// Trigger a resize-like logic to recompute viewport height
				m.viewport.Height = m.height - m.textarea.Height() - 8
				m.perusalVp.Height = m.viewport.Height
			}
		case focusConvo:
			m.viewport, vpCmd = m.viewport.Update(msg)
		case focusTree:
			m.perusalVp, pvCmd = m.perusalVp.Update(msg)
		case focusEdit:
			m.editArea, eaCmd = m.editArea.Update(msg)
		}
	default:
		// Always update for non-key messages (Resize, Blink, etc.)
		m.textarea, tiCmd = m.textarea.Update(msg)
		m.editArea, eaCmd = m.editArea.Update(msg)
		m.viewport, vpCmd = m.viewport.Update(msg)
		m.perusalVp, pvCmd = m.perusalVp.Update(msg)
	}

	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		wasAtTop := m.viewport.AtTop()
		wasAtBottom := m.viewport.AtBottom()
		prevYOffset := m.viewport.YOffset

		m.width = msg.Width
		m.height = msg.Height

		if m.showTree {
			m.viewport.Width = (msg.Width / 2) - 2
			m.perusalVp.Width = msg.Width - m.viewport.Width - 4
		} else {
			m.viewport.Width = msg.Width - 2
		}

		m.textarea.SetWidth(m.viewport.Width + 2)
		m.editArea.SetWidth(m.perusalVp.Width)
		m.viewport.Height = msg.Height - m.textarea.Height() - 8
		m.perusalVp.Height = m.viewport.Height
		m.editArea.SetHeight(m.perusalVp.Height - 2)

		// Defer expensive rendering to a command to avoid hanging the UI
		return m, func() tea.Msg {
			content := m.renderMessages()
			return layoutMsg{
				content:     content,
				wasAtBottom: wasAtBottom,
				wasAtTop:    wasAtTop,
				prevOffset:  prevYOffset,
			}
		}

	case layoutMsg:
		m.historyRendered = msg.content
		m.updateViewport()
		if msg.wasAtBottom {
			m.viewport.GotoBottom()
		} else if msg.wasAtTop {
			m.viewport.GotoTop()
		} else {
			m.viewport.SetYOffset(msg.prevOffset)
			if m.viewport.PastBottom() {
				m.viewport.GotoBottom()
			}
		}
		return m, nil

	case tea.KeyMsg:
		// Universal focus switcher: Tab cycles Input → Convo → Tree → Input
		if msg.String() == "tab" && m.focus != focusEdit {
			switch m.focus {
			case focusInput:
				m.focus = focusConvo
				m.textarea.Blur()
			case focusConvo:
				m.focus = focusTree
			case focusTree:
				m.focus = focusInput
				m.textarea.Focus()
			}
			return m, nil
		}

		if msg.String() == "esc" {
			if m.focus == focusEdit {
				m.focus = focusTree
				return m, nil
			}
			m.focus = focusInput
			m.textarea.Focus()
			m.suggestions = nil
			return m, nil
		}

		// Handle active focus
		var cmd tea.Cmd
		switch m.focus {
		case focusInput:
			// Intervention handling takes priority
			if m.pendingIntervention != nil {
				return m.handleInterventionKey(msg)
			}
			return m.handleChatKey(msg)
		case focusConvo:
			return m.handleConvoKey(msg)
		case focusTree:
			return m.handlePerusalKey(msg)
		case focusEdit:
			return m.handleEditKey(msg)
		}
		return m, cmd

	case recordTickMsg:
		if m.isRecording {
			// Efficiency: If nothing has changed since the last tick, just increment the counter
			// of the last frame instead of re-rendering the entire view.
			if !m.isDirty && len(m.recordedFrames) > 0 {
				m.recordedFrames[len(m.recordedFrames)-1].ticks++
				return m, recordTick()
			}

			m.isCapturing = true
			currentView := m.View()
			m.isCapturing = false
			m.isDirty = false // Reset dirty flag after capture

			if len(m.recordedFrames) > 0 && m.recordedFrames[len(m.recordedFrames)-1].content == currentView {
				m.recordedFrames[len(m.recordedFrames)-1].ticks++
			} else {
				m.recordedFrames = append(m.recordedFrames, recordedFrame{
					content: currentView,
					ticks:   1,
				})
			}
			return m, recordTick()
		}
		return m, nil

	case recordingProgressMsg:
		m.encodingCurrent = msg.Current
		m.encodingTotal = msg.Total
		return m, nil

	case recordingFinishedMsg:
		m.isEncoding = false
		m.messages = append(m.messages, systemStyle.Render(" RECORDING COMPLETE ")+"\n"+helpStyle.Render("🎬 Saved to: "+msg.Path))
		return m, m.asyncRender()

	case recordingErrorMsg:
		m.isEncoding = false
		m.recordingErr = msg.Err
		m.messages = append(m.messages, errorStyle.Render(" RECORDING FAILED ")+"\n"+helpStyle.Render(msg.Err.Error()))
		return m, m.asyncRender()

	case brain.Response:
		m.isThinking = false
		if msg.Error != nil {
			// Check if this is an intervention request
			var interventionErr *tooling.InterventionError
			if errors.As(msg.Error, &interventionErr) {
				m.pendingIntervention = &interventionState{
					title:    interventionErr.Title,
					choices:  interventionErr.Choices,
					selected: 0,
					resume: func(choice string) (interface{}, error) {
						return interventionErr.Resume(choice)
					},
					requestID: uuid.NewString(),
				}
				m.messages = append(m.messages, m.renderInterventionSelector())
				m.viewport.SetContent(m.renderMessages())
				m.viewport.GotoBottom()
				return m, nil // Wait for user input
			}
			m.messages = append(m.messages, errorStyle.Render(" BRAIN ERROR ")+"\n"+msg.Error.Error())
		} else if m.wasStreaming {
			// Response already handled by streaming, just reset flag
			m.wasStreaming = false
			// Persist thinking trace if any
			if len(m.thinkingLog) > 0 {
				var b strings.Builder
				for _, log := range m.thinkingLog {
					b.WriteString(fmt.Sprintf("  %s %s\n", log.Icon, log.Message))
				}
				m.messages = append(m.messages, subtleStyle.Render(b.String()))
			}

			// Proactive Recommendations UI (for streaming case)
			if meta, ok := msg.Metadata["recommendations"].([]prompt.Recommendation); ok && len(meta) > 0 {
				var rb strings.Builder
				rb.WriteString("\n" + lipgloss.NewStyle().Foreground(highlight).Render("💡 Recommended Actions:") + "\n")
				for _, r := range meta {
					rb.WriteString(fmt.Sprintf("  %s %s\n", aiStyle.Render("• "+r.Title), helpStyle.Render(r.Description)))
				}
				m.messages = append(m.messages, rb.String())
			}
		} else {
			// Persist the thinking trace faintly
			if len(m.thinkingLog) > 0 {
				var b strings.Builder
				for _, log := range m.thinkingLog {
					b.WriteString(fmt.Sprintf("  %s %s\n", log.Icon, log.Message))
				}
				m.messages = append(m.messages, subtleStyle.Render(b.String()))
			}

			// Label: Auracle
			m.messages = append(m.messages, aiStyle.Render("Auracle: ")+m.styleMessage(msg.Content))

			// Proactive Recommendations UI
			if meta, ok := msg.Metadata["recommendations"].([]prompt.Recommendation); ok && len(meta) > 0 {
				var rb strings.Builder
				rb.WriteString("\n" + lipgloss.NewStyle().Foreground(highlight).Render("💡 Recommended Actions:") + "\n")
				for _, r := range meta {
					rb.WriteString(fmt.Sprintf("  %s %s\n", aiStyle.Render("• "+r.Title), helpStyle.Render(r.Description)))
				}
				m.messages = append(m.messages, rb.String())
			}
		}

		// Archive the final active block into permanent history
		if m.activeBlock != "" {
			m.messages = append(m.messages, "BLOCK:"+m.activeBlock)
			m.activeBlock = ""
		}

		m.saveState()
		// Auto-focus back to input and clear thinking log for next turn
		m.focus = focusInput
		m.textarea.Focus()
		m.thinkingLog = nil

		if m.isAuracleMode {
			// Sophisticated completion check: only stop if manually toggled or 
			// the model explicitly indicates it has reached a true perfection state 
			// (which the system prompt now defines as a 5-turn counter)
			if strings.Contains(msg.Content, "\"is_project_perfect\": true") && strings.Contains(msg.Content, "\"no_more_work_counter\": 5") {
				m.isAuracleMode = false
				m.messages = append(m.messages, auracleStyle.Render(" AURACLE MODE ")+subtleStyle.Render(" COMPLETED (PROJECT ACHIEVED STASIS)"))
				return m, m.asyncRender()
			}
			m.isThinking = true
			return m, tea.Batch(
				m.asyncRender(),
				m.processRequest("AURACLE_MODE: Maintain drift. Analyze the self_audit and identified_gaps. Carry out the next_steps. If nothing is left, find something creative to add or improve. Increment counter only if absolutely zero value left to add."),
			)
		}

		return m, m.asyncRender()

	case checkUpdateTickMsg:
		return m, tea.Batch(
			m.updater.CheckUpdateCmd(false),
			waitForUpdateTick(),
		)

	case statusMsg:
		m.lastStatus = StatusEvent(msg)
		// Map step/type to block header with more color
		header := ""
		switch msg.Step {
		case "think":
			header = thinkHeaderStyle.Background(lipgloss.Color("#5F00FF")).Render(" 🧠 THINKING ")
		case "plan":
			header = thinkHeaderStyle.Background(lipgloss.Color("#AF00FF")).Render(" 📝 PLANNING ")
		case "exec", "tool":
			header = modificationHeaderStyle.Background(lipgloss.Color("#D700FF")).Render(" 🔧 EXECUTING ")
		case "done":
			header = decisionHeaderStyle.Background(lipgloss.Color("#00AF00")).Render(" ✅ COMPLETE ")
		case "delegation":
			header = delegationHeaderStyle.Background(lipgloss.Color("#00AFD7")).Render(" 🚀 DELEGATING ")
		default:
			header = thinkHeaderStyle.Render(" ◆ " + strings.ToUpper(msg.Step) + " ")
		}

		// Format block body
		body := msg.Message
		if msg.Extra != "" {
			if strings.HasPrefix(msg.Extra, "cmd:") {
				cmd := strings.TrimPrefix(msg.Extra, "cmd:")
				body += "\n\n" + lipgloss.NewStyle().Foreground(lipgloss.Color("#00FF00")).Render("$ "+cmd)
			} else if strings.HasPrefix(msg.Extra, "file:") {
				parts := strings.SplitN(strings.TrimPrefix(msg.Extra, "file:"), "\n", 2)
				if len(parts) == 2 {
					body += fmt.Sprintf("\n\n📄 Writing to %s:\n```\n%s\n```",
						lipgloss.NewStyle().Bold(true).Render(parts[0]),
						parts[1])
				}
			} else if strings.HasPrefix(msg.Extra, "patch:") {
				parts := strings.SplitN(strings.TrimPrefix(msg.Extra, "patch:"), "\n", 2)
				if len(parts) == 2 {
					body += fmt.Sprintf("\n\n🩹 Patching %s:\n```diff\n%s\n```",
						lipgloss.NewStyle().Bold(true).Render(parts[0]),
						parts[1])
				}
			}
		}

		// Update active block
		m.activeBlock = fmt.Sprintf("%s\n%s", header, blockBodyStyle.Render(body))
		// Immediate O(1) viewport update
		m.updateViewport()
		return m, waitForStatus()

	case usageMsg:
		m.lastUsage = vmodel.Usage(msg)
		return m, nil

	case streamDeltaMsg:
		if !m.isStreaming {
			m.isStreaming = true
			m.wasStreaming = true
		}
		m.streamingContent.WriteString(msg.Delta)
		m.lastStreamContent = aiStyle.Render("Auracle: ") + m.styleMessage(m.streamingContent.String()) + subtleStyle.Render("▌")
		// Immediate O(1) viewport update (bypassing full re-render)
		m.updateViewport()
		return m, nil

	case streamDoneMsg:
		m.isStreaming = false
		full := aiStyle.Render("Auracle: ") + m.styleMessage(msg.FullContent)
		m.messages = append(m.messages, full)
		m.wasStreaming = true
		m.streamingContent.Reset()
		m.lastStreamContent = ""
		m.focus = focusInput
		m.textarea.Focus()
		return m, m.asyncRender()

	case []brain.ModelDiscovery:
		m.allModelDiscoveries = msg
		// If we are currently typing /models /use, refresh suggestions
		val := m.textarea.Value()
		if strings.Contains(val, "/models /use") {
			m.updateSuggestions(val)
		}

	case UpdateAvailableMsg:
		m.updateVersion = msg.Latest.TagName
		m.messages = append(m.messages, systemStyle.Render(" UPDATE FOUND ")+"\n"+helpStyle.Render(fmt.Sprintf("Version %s is available. Downloading and installing now...", m.updateVersion)))
		return m, tea.Batch(m.asyncRender(), m.updater.DownloadUpdateCmd(msg.Latest))

	case UpdateReadyMsg:
		m.updateReady = true
		m.messages = append(m.messages, systemStyle.Render(" UPDATE READY ")+"\n"+helpStyle.Render("A new version has been downloaded. Please restart vibeaura to apply."))
		return m, m.asyncRender()

	case UpdateNoUpdateMsg:
		m.messages = append(m.messages, subtleStyle.Render("✅  Vibeauracle is already up to date."))
		return m, m.asyncRender()

	case interventionResultMsg:
		m.isThinking = false
		if msg.err != nil {
			m.messages = append(m.messages, errorStyle.Render(" ACTION ERROR ")+"\n"+msg.err.Error())
		} else if result, ok := msg.result.(*tooling.ToolResult); ok {
			if result.Error != nil {
				m.messages = append(m.messages, errorStyle.Render(" TOOL ERROR ")+"\n"+result.Error.Error())
			} else {
				m.messages = append(m.messages, aiStyle.Render("Tool: ")+m.styleMessage(result.Content))
			}
		} else if msg.result != nil {
			m.messages = append(m.messages, aiStyle.Render("Result: ")+fmt.Sprintf("%v", msg.result))
		} else {
			m.messages = append(m.messages, subtleStyle.Render("✓ Action completed"))
		}
		m.saveState()
		return m, m.asyncRender()
	}

	return m, tea.Batch(tiCmd, vpCmd, eaCmd, pvCmd)
}