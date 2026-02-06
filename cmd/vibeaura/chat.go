package main

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/charmbracelet/bubbles/textarea"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"sort"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/google/uuid"
	"github.com/nathfavour/vibeauracle/brain"
	"github.com/nathfavour/vibeauracle/internal/doctor"
	vmodel "github.com/nathfavour/vibeauracle/model"
	"github.com/nathfavour/vibeauracle/prompt"
	"github.com/nathfavour/vibeauracle/reactor"
	"github.com/nathfavour/vibeauracle/sys"
	"github.com/nathfavour/vibeauracle/tooling"
)

func (m *model) loadDynamicCommands() {
	m.dynamicCommands = make(map[string]brain.CLICommand)
	for _, ext := range m.brain.Extensions() {
		if !ext.Enabled || ext.Manifest == nil {
			continue
		}
		for _, cmd := range ext.Manifest.CLICommands {
			slashName := "/" + cmd.Name
			m.dynamicCommands[slashName] = cmd
			// Add to auto-complete
			found := false
			for _, c := range allCommands {
				if c == slashName {
					found = true
					break
				}
			}
			if !found {
				allCommands = append(allCommands, slashName)
			}
		}
	}
}

func initialModel(b *brain.Brain) *model {

	// Initialize native clipboard
	_ = initClipboard()

	ta := textarea.New()
	ta.Placeholder = "Send a message or type / for commands..."
	ta.Focus()
	ta.Prompt = "┃ "
	ta.CharLimit = 2000
	ta.SetWidth(60)
	ta.SetHeight(3)
	ta.FocusedStyle.CursorLine = lipgloss.NewStyle()
	ta.ShowLineNumbers = false

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

		historyIndex: -1, // -1 means not browsing history

		// Dynamic Commands from Extensions
		dynamicCommands: make(map[string]brain.CLICommand),

		// Non-blocking Engine

		reactor: reactor.New(),

		md: reactor.NewMarkdownRenderer(vp.Width, b.Config().(*sys.Config).UI.Theme),
	}

	m.loadDynamicCommands()
	// Load initial tree

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
					Render(fmt.Sprintf("⚡ UPDATED TO %s", "LATEST")) // We don't have the hash here easily unless we passed it.

				// Better: We can pass the new version in the state file too!

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
	return tea.Batch(
		textarea.Blink,
		m.updater.CheckUpdateCmd(false), // Initial check
		waitForUpdateTick(),             // Schedule next check
	)
}
func (m *model) saveState() {
	state := chatState{
		Messages:      m.messages,
		Input:         m.textarea.Value(),
		PromptHistory: m.promptHistory,
		ShowSidebar:   m.showTree,
	}
	m.brain.StoreState(m.brain.GetSessionID(), state)
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

	switch msg.(type) {

	case tea.KeyMsg:

		switch m.focus {

		case focusInput:

			m.textarea, tiCmd = m.textarea.Update(msg)

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

				content: content,

				wasAtBottom: wasAtBottom,

				wasAtTop: wasAtTop,

				prevOffset: prevYOffset,
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
				// ... (intervention handling remains the same)
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
			m.messages = append(m.messages, aiStyle.Render("VibeAuracle: ")+m.styleMessage(msg.Content))

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
		return m, m.asyncRender()

	case checkUpdateTickMsg:
		return m, tea.Batch(
			m.updater.CheckUpdateCmd(false),
			waitForUpdateTick(),
		)

	case statusMsg:

		m.lastStatus = StatusEvent(msg)

		// Map step/type to block header

		header := ""

		switch msg.Step {

		case "think":

			header = thinkHeaderStyle.Render(" 🧠 THINKING ")

		case "plan":

			header = thinkHeaderStyle.Render(" 📝 PLANNING ")

		case "exec", "tool":

			header = modificationHeaderStyle.Render(" 🔧 EXECUTING ")

		case "done":

			header = decisionHeaderStyle.Render(" ✅ COMPLETE ")

		case "delegation":

			header = delegationHeaderStyle.Render(" 🚀 DELEGATING ")

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
		m.lastStreamContent = aiStyle.Render("VibeAuracle: ") + m.styleMessage(m.streamingContent.String()) + subtleStyle.Render("▌")

		// Immediate O(1) viewport update (bypassing full re-render)
		m.updateViewport()
		return m, nil
	case streamDoneMsg:
		m.isStreaming = false
		full := aiStyle.Render("VibeAuracle: ") + m.styleMessage(msg.FullContent)
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
		// Start download immediately
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

	// 5. Check for Hot-Swap Opportunity
	// DISABLED: Hot-swap is disabled until a release binary with --resume-state support is published.
	// The user will be prompted to restart manually after an update is downloaded.
	if m.updateReady && !m.isThinking {
		// No-op for now; the "Update ready" message is already shown.
		// User can restart manually.
	}

	return m, tea.Batch(tiCmd, vpCmd, eaCmd, pvCmd)
}

func (m *model) loadTree(path string) {
	entries, _ := os.ReadDir(path)
	m.treeEntries = nil
	for _, e := range entries {
		if !strings.HasPrefix(e.Name(), ".") || e.Name() == ".env" {
			m.treeEntries = append(m.treeEntries, e)
		}
	}
	m.isFileOpen = false
	m.updatePerusalContent()
}

func (m *model) openFile(path string) {
	content, err := os.ReadFile(path)
	if err == nil {
		m.isFileOpen = true
		m.currentPath = path
		m.editArea.SetValue(string(content))
		m.perusalVp.SetContent(string(content))
	}
}

func (m *model) updatePerusalContent() {
	if m.isFileOpen {
		return
	}
	var sb strings.Builder
	sb.WriteString(systemStyle.Render(" EXPLORER: "+m.currentPath) + "\n\n")
	for i, entry := range m.treeEntries {
		cursor := "  "
		if i == m.treeCursor {
			cursor = "> "
		}
		icon := "📄 "
		if entry.IsDir() {
			icon = "📁 "
		}
		line := cursor + icon + entry.Name()
		if i == m.treeCursor {
			sb.WriteString(suggestionStyle.Render(line) + "\n")
		} else {
			sb.WriteString(line + "\n")
		}
	}
	m.perusalVp.SetContent(sb.String())
}

func (m *model) updateSuggestions(val string) {
	m.suggestions = nil
	m.suggestionIdx = 0
	m.triggerChar = ""
	m.isFilteringModels = false

	if val == "" {
		return
	}

	if strings.Contains(val, "/models /use") {
		m.isFilteringModels = true
		if len(m.allModelDiscoveries) == 0 {
			// Trigger discovery
			go func() {
				// We can't return Cmd here, so we'll just wait for the next Update cycle
				// if we were in a proper Msg flow, but here we are in a helper.
				// Better to trigger this from handleChatKey or applySuggestion.
			}()
		}

		// Everything after "/models /use " is the filter
		parts := strings.Split(val, "/models /use")
		filter := ""
		if len(parts) > 1 {
			filter = strings.TrimSpace(parts[1])
		}
		m.suggestionFilter = filter

		for _, d := range m.allModelDiscoveries {
			display := fmt.Sprintf("%s (%s)", shortenModelName(d.Name), d.Provider)
			if filter == "" || strings.Contains(strings.ToLower(display), strings.ToLower(filter)) {
				// We store the full identifier for applySuggestion, but display it nicely
				m.suggestions = append(m.suggestions, fmt.Sprintf("%s|%s", d.Provider, d.Name))
			}
		}
		return
	}

	// Handle trailing space for subcommand triggering
	if strings.HasSuffix(val, " ") {
		parts := strings.Fields(val)
		if len(parts) == 1 {
			if subs, ok := subCommands[parts[0]]; ok {
				m.suggestions = subs
				m.triggerChar = "" // Already has / in the subCommand string
				sort.Strings(m.suggestions)
				return
			}
		}
	}

	words := strings.Fields(val)
	if len(words) == 0 {
		if strings.HasSuffix(val, "/") {
			m.triggerChar = "/"
			m.suggestions = append([]string{}, allCommands...)
			sort.Strings(m.suggestions)
		} else if strings.HasSuffix(val, "#") {
			m.triggerChar = "#"
			m.suggestions = m.getFileSuggestions("")
		}
		return
	}

	lastWord := words[len(words)-1]

	// Check if we are typing a subcommand
	if len(words) > 1 {
		parentCmd := words[0]
		if subs, ok := subCommands[parentCmd]; ok {
			m.triggerChar = "" // Subcommands already have slashes
			for _, sub := range subs {
				if strings.HasPrefix(sub, lastWord) {
					m.suggestions = append(m.suggestions, sub)
				}
			}
			sort.Strings(m.suggestions)
			if len(m.suggestions) > 0 {
				return
			}
		}
	}

	if strings.HasPrefix(lastWord, "/") {
		m.triggerChar = "/"
		for _, cmd := range allCommands {
			if strings.HasPrefix(cmd, lastWord) {
				m.suggestions = append(m.suggestions, cmd)
			}
		}
		sort.Strings(m.suggestions)
	} else if strings.HasPrefix(lastWord, "#") {
		m.triggerChar = "#"
		m.suggestions = m.getFileSuggestions(lastWord[1:])
	}
}

func (m *model) getFileSuggestions(prefix string) []string {
	var suggestions []string
	root, _ := os.Getwd()

	filepath.WalkDir(root, func(path string, d os.DirEntry, err error) error {
		if err != nil || len(suggestions) > 30 {
			return nil
		}

		name := d.Name()
		if d.IsDir() {
			if name == ".git" || name == "node_modules" || name == "vendor" || name == "bin" || name == "dist" {
				return filepath.SkipDir
			}
			if prefix != "" && !strings.HasPrefix(name, prefix) && !strings.HasPrefix(path, prefix) {
				return nil
			}
		}

		rel, _ := filepath.Rel(root, path)
		if rel == "." {
			return nil
		}

		if prefix == "" || strings.HasPrefix(rel, prefix) || strings.HasPrefix(name, prefix) {
			suggestions = append(suggestions, rel)
		}

		return nil
	})

	sort.Strings(suggestions)
	return suggestions
}

func (m *model) applySuggestion() (tea.Model, tea.Cmd) {
	if len(m.suggestions) == 0 {
		return m, nil
	}

	val := m.textarea.Value()
	suggestion := m.suggestions[m.suggestionIdx]

	// Handle model selection specialized format: provider|name
	if m.isFilteringModels && strings.Contains(suggestion, "|") {
		parts := strings.Split(suggestion, "|")
		provider := parts[0]
		modelName := parts[1]
		fullCmd := fmt.Sprintf("/models /use %s %s", provider, modelName)
		m.textarea.SetValue(fullCmd)
		m.textarea.SetCursor(len(m.textarea.Value()))
		m.suggestions = nil
		return m.handleSlashCommand(fullCmd)
	}

	// Determine if we are completing a subcommand or a top-level command
	words := strings.Fields(val)
	if len(words) == 0 {
		m.textarea.SetValue(suggestion)
	} else {
		// If the last word is what we're completing
		lastWord := words[len(words)-1]

		if strings.HasSuffix(val, " ") {
			// Context: User just typed a space, we are appending a new part
			m.textarea.SetValue(strings.TrimRight(val, " ") + " " + suggestion)
		} else if strings.HasPrefix(suggestion, lastWord) || (strings.HasPrefix(lastWord, "/") && strings.HasPrefix(suggestion, "/")) {
			// Context: User is partially typing the suggestion, replace the partial part
			words[len(words)-1] = suggestion
			m.textarea.SetValue(strings.Join(words, " "))
		} else {
			// Context: Unclear, safest to append with space
			m.textarea.SetValue(strings.TrimRight(val, " ") + " " + suggestion)
		}
	}

	m.textarea.SetCursor(len(m.textarea.Value()))
	m.suggestions = nil

	currentVal := strings.TrimSpace(m.textarea.Value())
	parts := strings.Fields(currentVal)

	// If we just completed a top-level command that has subcommands, add a space and show them
	if len(parts) == 1 {
		if _, ok := subCommands[parts[0]]; ok {
			m.textarea.SetValue(parts[0] + " ")
			m.textarea.SetCursor(len(m.textarea.Value()))
			m.updateSuggestions(m.textarea.Value())
			return m, nil
		}
	}

	// Auto-execute logic for no-arg commands/subcommands
	noArgSubs := map[string]map[string]bool{
		"/models":  {"/list": true},
		"/sys":     {"/stats": true, "/env": true, "/update": true, "/logs": true},
		"/mcp":     {"/list": true, "/logs": true},
		"/skill":   {"/list": true},
		"/agent":   {"/vibe": true, "/sdk": true},
		"/session": {"/list": true, "/clear": true},
	}

	if len(parts) == 1 {
		if _, hasSubs := subCommands[parts[0]]; !hasSubs {
			return m.handleSlashCommand(currentVal)
		}
	} else if len(parts) == 2 {
		if subs, ok := noArgSubs[parts[0]]; ok {
			if subs[parts[1]] {
				return m.handleSlashCommand(currentVal)
			}
		}
	}

	// Otherwise, add a trailing space for the next argument
	m.textarea.SetValue(currentVal + " ")
	m.textarea.SetCursor(len(m.textarea.Value()))
	return m, nil
}

func (m *model) processRequest(content string) tea.Cmd {
	return func() tea.Msg {
		ctx := context.Background()
		req := brain.Request{
			ID:      uuid.NewString(),
			Content: content,
		}
		res, err := m.brain.Process(ctx, req)
		var resp brain.Response
		if err != nil {
			resp.Error = err
		} else {
			resp = res.(brain.Response)
		}
		return resp

	}
}

func (m *model) takeScreenshot() (tea.Model, tea.Cmd) {

	config := m.brain.GetConfig()

	dir := config.UI.ScreenshotDir

	if err := os.MkdirAll(dir, 0755); err != nil {
		m.messages = append(m.messages, errorStyle.Render(" Screenshot Error: ")+err.Error())
		return m, nil
	}

	timestamp := time.Now().Format("2006-01-02_150405")
	filename := fmt.Sprintf("vibeaura_%s", timestamp)

	basePath := filepath.Join(dir, filename)
	ansiPath := basePath + ".ansi"
	svgPath := basePath + ".svg"
	pngPath := basePath + ".png"

	// Use current layout but ensure it's rendered for capture
	m.isCapturing = true
	rawView := m.View()
	m.isCapturing = false

	// Tier 2: Generate SVG but don't save yet if targeting PNG
	svgContent := convertAnsiToSVG(rawView)
	_ = os.WriteFile(svgPath, []byte(svgContent), 0644)

	// Tier 1: Try PNG
	err := convertToPNG(svgPath, pngPath)

	msg := systemStyle.Render(" SCREENSHOT CAPTURED ") + "\n"

	if err == nil {
		// Highest Tier: PNG only
		_ = os.Remove(svgPath)
		msg += helpStyle.Render("🖼️ Saved PNG: " + pngPath)
	} else if svgContent != "" {
		// Middle Tier: SVG only
		msg += helpStyle.Render("📍 Saved SVG: " + svgPath)
		msg += "\n" + errorStyle.Render(" PNG fail: ") + helpStyle.Render("install ffmpeg/rsvg")
	} else {
		msg += helpStyle.Render("📄 Saved ANSI: " + ansiPath)
	}

	m.messages = append(m.messages, msg)
	return m, m.asyncRender()
}
func (m *model) toggleRecording() (tea.Model, tea.Cmd) {
	if m.isRecording {
		m.isRecording = false
		msg := systemStyle.Render(" RECORDING STOPPED ") + "\n" + helpStyle.Render("Processing frames in background...")
		m.messages = append(m.messages, msg)

		// Deep copy frames to avoid race conditions during background processing
		frames := make([]recordedFrame, len(m.recordedFrames))
		copy(frames, m.recordedFrames)
		m.recordedFrames = nil

		// Start encoding state
		m.isEncoding = true
		m.encodingCurrent = 0
		m.encodingTotal = len(frames)
		m.recordingErr = nil

		// Capture program and config for background use
		p := m.getProgram()
		outDir := m.brain.GetConfig().UI.ScreenshotDir

		go m.processRecording(m.recordingID, frames, p, outDir)
		return m, m.asyncRender()
	}

	// Dependency Check
	if err := checkRecordingDependencies(); err != nil {
		m.messages = append(m.messages, errorStyle.Render(" RECORDING UNAVAILABLE ")+"\n"+helpStyle.Render(err.Error()))
		return m, m.asyncRender()
	}

	m.isRecording = true
	m.isDirty = true // Force capture of the first frame
	m.recordingID = uuid.New().String()
	m.recordedFrames = nil
	msg := systemStyle.Render(" RECORDING STARTED ") + "\n" + helpStyle.Render("Capture interval: 41ms")
	m.messages = append(m.messages, msg)
	return m, tea.Batch(m.asyncRender(), recordTick())
}
func (m *model) getProgram() *tea.Program {
	return uiProgram
}

func getBestEncoder() string {
	if runtime.GOOS == "darwin" {
		return "h264_videotoolbox"
	}
	if _, err := exec.LookPath("nvidia-smi"); err == nil {
		cmd := exec.Command("ffmpeg", "-hide_banner", "-encoders")
		if out, err := cmd.Output(); err == nil && strings.Contains(string(out), "h264_nvenc") {
			return "h264_nvenc"
		}
	}
	return "libx264"
}

func (m *model) processRecording(id string, frames []recordedFrame, p *tea.Program, outDir string) {
	if len(frames) == 0 {
		if p != nil {
			p.Send(recordingErrorMsg{Err: fmt.Errorf("no frames recorded")})
		}
		return
	}

	_ = os.MkdirAll(outDir, 0755)

	// 1. Parallelized rendering to memory with deduplication
	numFrames := len(frames)
	rgbDatas := make([][]byte, numFrames)
	var width, height int

	type renderResult struct {
		data []byte
		w, h int
		err  error
	}
	cache := make(map[string]*renderResult)
	var cacheMu sync.Mutex

	var wg sync.WaitGroup
	sem := make(chan struct{}, runtime.NumCPU())
	var processedCount int32

	for i := range frames {
		wg.Add(1)
		go func(idx int) {
			defer wg.Done()
			sem <- struct{}{}
			defer func() { <-sem }()

			ansi := frames[idx].content
			cacheMu.Lock()
			res, ok := cache[ansi]
			cacheMu.Unlock()

			if !ok {
				data, w, h, err := renderAnsiToRGB(ansi)
				res = &renderResult{data, w, h, err}
				cacheMu.Lock()
				cache[ansi] = res
				cacheMu.Unlock()
			}

			if res.err == nil {
				rgbDatas[idx] = res.data
				cacheMu.Lock()
				if width == 0 {
					width = res.w
					height = res.h
				}
				cacheMu.Unlock()
			}

			newCount := atomic.AddInt32(&processedCount, 1)
			if p != nil && newCount%10 == 0 {
				p.Send(recordingProgressMsg{Current: int(newCount), Total: numFrames})
			}
		}(i)
	}
	wg.Wait()

	if p != nil {
		p.Send(recordingProgressMsg{Current: numFrames, Total: numFrames})
	}

	// 2. Assemble with FFmpeg using rawvideo pipe and GPU acceleration
	timestamp := time.Now().Format("2006-01-02_150405")
	finalPath := filepath.Join(outDir, fmt.Sprintf("vibeaura_rec_%s.mp4", timestamp))

	encoder := getBestEncoder()
	args := []string{
		"-y",
		"-framerate", "24",
		"-f", "rawvideo",
		"-pixel_format", "rgb24",
		"-video_size", fmt.Sprintf("%dx%d", width, height),
		"-i", "-",
		"-c:v", encoder,
	}

	if encoder == "libx264" {
		args = append(args, "-preset", "slower", "-crf", "17", "-tune", "stillimage")
	} else if encoder == "h264_nvenc" {
		args = append(args, "-preset", "slow", "-cq", "17", "-rc", "vbr")
	} else if encoder == "h264_videotoolbox" {
		args = append(args, "-realtime", "false", "-q:v", "90")
	}

	args = append(args,
		"-pix_fmt", "yuv420p",
		"-movflags", "+faststart",
		"-vf", "scale='min(1920,iw)':-2:force_original_aspect_ratio=decrease:flags=lanczos,pad=ceil(iw/2)*2:ceil(ih/2)*2",
		finalPath,
	)

	cmd := exec.Command("ffmpeg", args...)
	stdin, err := cmd.StdinPipe()
	if err != nil {
		if p != nil {
			p.Send(recordingErrorMsg{Err: fmt.Errorf("ffmpeg pipe failed: %w", err)})
		}
		return
	}

	if err := cmd.Start(); err != nil {
		if p != nil {
			p.Send(recordingErrorMsg{Err: fmt.Errorf("ffmpeg start failed: %w", err)})
		}
		return
	}

	// Feed the pipe with frames
	for i, frame := range frames {
		data := rgbDatas[i]
		if data == nil {
			continue
		}
		for j := 0; j < frame.ticks; j++ {
			_, _ = stdin.Write(data)
		}
	}
	stdin.Close()
	_ = cmd.Wait()

	if p != nil {
		p.Send(recordingFinishedMsg{Path: finalPath})
	}
}
func (m *model) discoverModels() tea.Cmd {
	return func() tea.Msg {
		discoveries, err := m.brain.DiscoverModels(context.Background())
		if err != nil {
			return brain.Response{Error: err}
		}
		return discoveries
	}
}

func (m *model) pullOllamaModel(name string) tea.Cmd {
	return func() tea.Msg {
		err := m.brain.PullModel(context.Background(), name)
		if err != nil {
			return brain.Response{Error: err}
		}
		return brain.Response{Content: "Successfully pulled " + name + ". You can now use it with /models /use ollama " + name}
	}
}

func (m *model) resumeIntervention(resumeFn func(string) (interface{}, error), choice string) tea.Cmd {
	return func() tea.Msg {
		result, err := resumeFn(choice)
		return interventionResultMsg{result: result, err: err}
	}
}
