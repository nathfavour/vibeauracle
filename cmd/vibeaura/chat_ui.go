package main

import (
	"fmt"
	"path/filepath"
	"regexp"
	"strings"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/nathfavour/vibeauracle/brain"
	"github.com/nathfavour/vibeauracle/sys"
)

func buildBanner(width int) string {
	if width <= 0 {
		width = 60
	}

	// Wide terminals/panes get the big ASCII banner.
	ascii := []string{
		lipgloss.NewStyle().Foreground(lipgloss.Color("#FF00D7")).Bold(true).Render("       _ _                                  _"),
		lipgloss.NewStyle().Foreground(lipgloss.Color("#D700FF")).Bold(true).Render(" __   _(_) |__   ___  __ _ _   _ _ __ __ _  ___| | ___"),
		lipgloss.NewStyle().Foreground(lipgloss.Color("#AF00FF")).Bold(true).Render(" \\ \\ / / | '_ \\ / _ \\/ _` | | | | '__/ _` |/ __| |/ _ \\"),
		lipgloss.NewStyle().Foreground(lipgloss.Color("#8700FF")).Bold(true).Render("  \\ V /| | |_) |  __/ (_| | |_| | | | (_| | (__| |  __/"),
		lipgloss.NewStyle().Foreground(lipgloss.Color("#5F00FF")).Bold(true).Render("   \\_/ |_|_.__/ \\___|\\__,_|\\__,_|_|  \\__,_|\\___|_|\\___|"),
	}

	maxASCII := 0
	for _, l := range ascii {
		w := lipgloss.Width(l)
		if w > maxASCII {
			maxASCII = w
		}
	}

	tagline := helpStyle.Render("Distributed, System-Intimate AI Engineering Ecosystem")
	if width >= maxASCII {
		return strings.Join(append(append(ascii, ""), tagline), "\n") + "\n"
	}

	// Compact banner for narrow panes: multicolor title + tagline.
	word := "vibeauracle"
	colors := []lipgloss.Color{
		lipgloss.Color("#FF00D7"),
		lipgloss.Color("#D700FF"),
		lipgloss.Color("#AF00FF"),
		lipgloss.Color("#8700FF"),
		lipgloss.Color("#5F00FF"),
		lipgloss.Color("#7D56F4"),
		lipgloss.Color("#04D9FF"),
	}

	spaced := width >= (len(word)*2 - 1)
	title := gradientWord(word, colors, spaced)
	if lipgloss.Width(title) > width {
		// Fall back if spacing makes it too wide.
		title = gradientWord(word, colors, false)
	}

	// Keep tagline only if it fits reasonably.
	if width < 44 {
		return title + "\n" + helpStyle.Render("System-Intimate AI") + "\n"
	}
	return title + "\n" + tagline + "\n"
}

func gradientWord(word string, colors []lipgloss.Color, spaced bool) string {
	var b strings.Builder
	colorIdx := 0
	for _, r := range word {
		style := lipgloss.NewStyle().Foreground(colors[colorIdx%len(colors)]).Bold(true)
		b.WriteString(style.Render(string(r)))
		colorIdx++
		if spaced {
			b.WriteString(" ")
		}
	}
	return strings.TrimRight(b.String(), " ")
}

func isBannerMessage(msg string) bool {
	// This substring exists in both the wide and compact banner variants.
	return strings.Contains(msg, "System-Intimate") || strings.Contains(msg, "_(_) |__")
}

func ensureBanner(messages *[]string, banner string) {
	if messages == nil {
		return
	}
	if len(*messages) == 0 {
		*messages = append(*messages, banner)
		return
	}
	if isBannerMessage((*messages)[0]) {
		(*messages)[0] = banner
		return
	}
	*messages = append([]string{banner}, *messages...)
}

func stripANSI(str string) string {
	const ansi = "[\u001B\u009B][[()#;?]*(?:[0-9]{1,4}(?:;[0-9]{0,4})*)?[0-9A-ORZcf-nqry=><]"
	re := regexp.MustCompile(ansi)
	return re.ReplaceAllString(str, "")
}

func shortenModelName(name string) string {
	return brain.ShortenModelName(name)
}

func waitForUpdateTick() tea.Cmd {
	return tea.Tick(30*time.Minute, func(t time.Time) tea.Msg {
		return checkUpdateTickMsg(t)
	})
}

func recordTick() tea.Cmd {
	return tea.Tick(time.Millisecond*41, func(t time.Time) tea.Msg {
		return recordTickMsg(t)
	})
}

func waitForStatus() tea.Cmd {
	return func() tea.Msg {
		return statusMsg(<-StatusStream) // Update to use exposed channel
	}
}

func (m *model) updateViewport() {
	var full strings.Builder
	full.WriteString(m.historyRendered)

	// Append active thinking/action block
	if m.activeBlock != "" {
		if full.Len() > 0 {
			full.WriteString("\n\n")
		}
		full.WriteString(m.activeBlock)
	}

	// Append active streaming content
	if m.lastStreamContent != "" {
		if full.Len() > 0 {
			full.WriteString("\n\n")
		}
		full.WriteString(lipgloss.NewStyle().Width(m.viewport.Width).Render(m.lastStreamContent))
	}

	m.viewport.SetContent(full.String())
	m.viewport.GotoBottom()
}

func (m *model) styleMessage(v string) string {
	if strings.TrimSpace(v) == "" {
		return ""
	}

	// If it's a multi-line message (likely markdown), don't style parts
	if strings.Contains(v, "\n") {
		return v
	}

	parts := strings.Split(v, " ")
	for i, p := range parts {
		if strings.HasPrefix(p, "/") {
			// Check if it's a known command or subcommand
			isKnown := false
			for _, c := range allCommands {
				if c == p {
					isKnown = true
					break
				}
			}
			if !isKnown {
				for _, subs := range subCommands {
					for _, s := range subs {
						if s == p {
							isKnown = true
							break
						}
					}
					if isKnown {
						break
					}
				}
			}

			if isKnown {
				parts[i] = systemStyle.Render(p)
			} else {
				parts[i] = errorStyle.Render(p)
			}
		} else if strings.HasPrefix(p, "#") {
			parts[i] = tagStyle.Render(p)
		}
	}
	return strings.Join(parts, " ")
}

func (m *model) renderMessages() string {
	// Sync renderer width with viewport
	m.md.SetWidth(m.viewport.Width)

	// 1. If width changed, we MUST re-render EVERYTHING (unavoidable O(N), but rare)
	if m.lastViewportWidth != m.viewport.Width {
		var sb strings.Builder
		for i, msg := range m.messages {
			content := m.renderSingleMessage(msg)
			sb.WriteString(content)
			if i < len(m.messages)-1 {
				sb.WriteString("\n\n")
			}
		}
		m.historyRendered = sb.String()
		m.lastViewportWidth = m.viewport.Width
		m.lastMessageCount = len(m.messages)
	}

	// 2. Incremental update: If messages were added, only render the new ones
	if len(m.messages) > m.lastMessageCount {
		var sb strings.Builder
		sb.WriteString(m.historyRendered)
		for i := m.lastMessageCount; i < len(m.messages); i++ {
			if sb.Len() > 0 {
				sb.WriteString("\n\n")
			}
			content := m.renderSingleMessage(m.messages[i])
			sb.WriteString(content)
		}
		m.historyRendered = sb.String()
		m.lastMessageCount = len(m.messages)
	}

	// 3. Return the stable history.
	return m.historyRendered
}

func (m *model) renderSingleMessage(raw string) string {
	content := raw
	width := m.viewport.Width - 4 // Account for padding/borders

	// If it's a special block message, render it raw (it's already styled)
	if strings.HasPrefix(raw, "BLOCK:") {
		return lipgloss.NewStyle().Width(m.viewport.Width).Render(strings.TrimPrefix(raw, "BLOCK:"))
	}

	if strings.HasPrefix(raw, userStyle.Render("User ")) {
		inner := strings.TrimPrefix(raw, userStyle.Render("User "))
		rendered := userLabelStyle.Render("YOU") + "\n" +
			userBubbleStyle.Width(width).Render(inner)
		return rendered
	}

	if strings.HasPrefix(raw, aiStyle.Render("Auracle: ")) {
		inner := strings.TrimPrefix(raw, aiStyle.Render("Auracle: "))
		// Only render markdown if it's not currently streaming
		if !strings.HasSuffix(inner, subtleStyle.Render("▌")) {
			rendered := aiLabelStyle.Render("AURACLE") + "\n" +
				aiBubbleStyle.Width(width).Render(m.md.Render(inner, width))
			return rendered
		}
		// Streaming content already has Auracle: prefix from update
		return aiBubbleStyle.Width(width).Render(raw)
	}

	return lipgloss.NewStyle().Width(m.viewport.Width).Render(content)
}

func (m *model) View() string {
	header := titleStyle.Render(" vibeauracle ") + " " + helpStyle.Render("v"+Version)

	if m.isManaged {
		header += lipgloss.NewStyle().
			Foreground(lipgloss.Color("#FAFAFA")).
			Background(lipgloss.Color("#00AFD7")).
			Padding(0, 1).
			Bold(true).
			MarginLeft(1).
			Render(" MANAGED ")
	}
	
	// Create a nice glow effect for the header on wider terminals
	if m.width > 80 {
		header = lipgloss.JoinHorizontal(lipgloss.Center,
			lipgloss.NewStyle().Foreground(lipgloss.Color("#FF00D7")).Render("◆"),
			header,
			lipgloss.NewStyle().Foreground(lipgloss.Color("#04D9FF")).Render("◆"),
		)
	}

	borderWidth := m.width
	if borderWidth > 20 {
		borderWidth--
	}
	border := strings.Repeat("─", borderWidth)

	// 1. Conversation Viewport
	chatView := m.viewport.View()
	if m.focus == focusConvo {
		chatView = activeBorder.Width(m.viewport.Width).Render(chatView)
	} else {
		chatView = inactiveBorder.Width(m.viewport.Width).Render(chatView)
	}

	// 2. Side Pane (Fluid Sidebar)
	mainContent := chatView
	if m.showTree {
		sidebarView := m.sidebar.View(m.perusalVp.Width, m.perusalVp.Height, m)
		mainContent = lipgloss.JoinHorizontal(lipgloss.Top,
			chatView,
			sidebarView,
		)
	}

	// 3. Status Bar (Dynamic & Reactive)
	cfg := m.brain.Config().(*sys.Config)

	// Colorful Chips for Env Bar
	chipStyle := lipgloss.NewStyle().Padding(0, 1).Bold(true).Foreground(lipgloss.Color("#FAFAFA"))
	providerChip := chipStyle.Background(lipgloss.Color("#7D56F4")).Render(strings.ToUpper(cfg.Model.Provider))
	modelChip := chipStyle.Background(lipgloss.Color("#04D9FF")).Render(shortenModelName(cfg.Model.Name))
	agentChip := chipStyle.Background(lipgloss.Color("#FF00D7")).Render(strings.ToUpper(cfg.Agent.Mode))

	envBar := lipgloss.JoinHorizontal(lipgloss.Center,
		subtleStyle.Render(" PROVIDER: "), providerChip,
		subtleStyle.Render("  MODEL: "), modelChip,
		subtleStyle.Render("  AGENT: "), agentChip,
	)

	// System Vitals (Simple Pulse)
	res, _ := m.brain.GetSnapshot()
	snapshot, _ := res.(sys.Snapshot)
	vitals := fmt.Sprintf("%s %.1f%%  %s %.1f%%  %s %s",
		subtleStyle.Render("CPU"), snapshot.CPUUsage,
		subtleStyle.Render("MEM"), snapshot.MemoryUsage,
		subtleStyle.Render("DIR"), filepath.Base(snapshot.WorkingDir),
	)

	statusBar := ""
	if m.isAuracleMode {
		label := auracleStyle.Render(" 🔮 AURACLE MODE ")
		msg := statusMessageStyle.Render(" Running autonomously... (Ctrl+Y to stop)")
		statusBar = "\n" + label + msg + "\n"
	} else if m.isRecording {
		label := lipgloss.NewStyle().Foreground(lipgloss.Color("#FAFAFA")).Background(lipgloss.Color("#FF0000")).Padding(0, 1).Bold(true).Render(" ● REC ")
		msg := statusMessageStyle.Render(fmt.Sprintf(" Recording... (%d frames)", len(m.recordedFrames)))
		statusBar = "\n" + label + msg + "\n"
	} else if m.isEncoding {
		pct := 0
		if m.encodingTotal > 0 {
			pct = (m.encodingCurrent * 100) / m.encodingTotal
		}
		label := statusLabelStyle.Render(" 🎬 ENCODING ")
		msg := statusMessageStyle.Render(fmt.Sprintf(" Processing frames: %d/%d (%d%%)", m.encodingCurrent, m.encodingTotal, pct))

		// Small progress bar
		barWidth := 20
		filled := (pct * barWidth) / 100
		bar := lipgloss.NewStyle().Foreground(lipgloss.Color("#7D56F4")).Render(strings.Repeat("█", filled)) +
			lipgloss.NewStyle().Foreground(lipgloss.Color("#444444")).Render(strings.Repeat("░", barWidth-filled))

		statusBar = "\n" + label + msg + "\n" + " " + bar + "\n"
	} else if m.isThinking || m.isStreaming {
		statusIcon := m.lastStatus.Icon
		if statusIcon == "" {
			statusIcon = "⏳"
		}
		if m.isStreaming {
			statusIcon = "📡"
		}

		step := m.lastStatus.Step
		if step == "" && m.isStreaming {
			step = "STREAMING"
		}

		label := statusLabelStyle.Render(fmt.Sprintf(" %s %s ", statusIcon, strings.ToUpper(step)))
		msg := statusMessageStyle.Render(" " + m.lastStatus.Message)
		if m.isStreaming {
			msg = statusMessageStyle.Render(" Receiving response...")
		}
		statusBar = "\n" + label + msg + "\n"
	} else if m.lastUsage.TotalTokens > 0 {
		usageInfo := fmt.Sprintf(" 🪙  Tokens: %d (In: %d, Out: %d)",
			m.lastUsage.TotalTokens, m.lastUsage.InputTokens, m.lastUsage.OutputTokens)
		if m.lastUsage.Cost > 0 {
			usageInfo += fmt.Sprintf(" | 💵 Cost: $%.4f", m.lastUsage.Cost)
		}
		statusBar = "\n" + subtleStyle.Render(usageInfo) + "\n"
	}
	
	// 4. Input Box (Adaptive Border)
	inputView := m.textarea.View()
	inputStyle := inactiveBorder
	if m.focus == focusInput {
		inputStyle = activeBorder
		if len(m.textarea.Value()) > 0 {
			// Change border color as user types
			inputStyle = inputStyle.BorderForeground(lipgloss.Color("#FF00D7"))
		}
	}
	inputView = inputStyle.Width(m.width - 2).Render(inputView)

	view := fmt.Sprintf(
		"%s\n%s\n%s\n%s\n%s\n%s\n%s%s\n%s",
		header,
		lipgloss.NewStyle().Foreground(lipgloss.Color("#444444")).Render(border),
		mainContent,
		lipgloss.NewStyle().MarginLeft(2).Render(vitals),
		lipgloss.NewStyle().Foreground(lipgloss.Color("#444444")).Render(border),
		lipgloss.NewStyle().Align(lipgloss.Center).Width(m.width).Render(envBar),
		lipgloss.NewStyle().Foreground(lipgloss.Color("#444444")).Render(border),
		statusBar,
		inputView,
	)
	if !m.isCapturing {
		if suggs := m.renderSuggestions(); suggs != "" {
			view += "\n" + suggs
		}
	}

	return view + "\n"
}

func (m *model) renderSuggestions() string {
	if len(m.suggestions) == 0 {
		return ""
	}

	maxItems := 10
	items := m.suggestions
	if len(items) > maxItems {
		items = items[:maxItems]
	}

	width := 50
	if m.width-10 < width {
		width = m.width - 4
	}

	var rows []string

	// Header/Filter input for model selector
	if m.isFilteringModels {
		filterHeader := lipgloss.NewStyle().
			Foreground(lipgloss.Color("#7D56F4")).
			Bold(true).
			Padding(0, 1).
			Render("🔍 Filter: " + m.suggestionFilter + "█")
		rows = append(rows, filterHeader)
		rows = append(rows, lipgloss.NewStyle().Foreground(lipgloss.Color("#444444")).Render(strings.Repeat("─", width)))

		if len(m.allModelDiscoveries) == 0 {
			rows = append(rows, subtleStyle.Width(width).Render("  Discovering models..."))
		}
	}

	for i, s := range items {
		selected := i == m.suggestionIdx

		style := suggestionStyle
		if selected {
			style = selectedSuggestionStyle
		}

		name := filepath.Base(s)
		dir := filepath.Dir(s)

		if strings.Contains(s, "|") && m.isFilteringModels {
			parts := strings.Split(s, "|")
			provider := parts[0]
			modelName := parts[1]
			name = shortenModelName(modelName)
			dir = provider

			// Shorten provider names for UI
			if dir == "github-models" {
				dir = "github"
			}
		} else {
			if m.triggerChar == "/" {
				name = s
			}
			if dir == "." || m.triggerChar == "/" {
				dir = ""
			}
		}

		// Truncate name if path is too long
		namePart := name
		if len(namePart) > 25 {
			namePart = namePart[:22] + "..."
		}

		dirPart := dir
		if len(dirPart) > width-25 {
			dirPart = "..." + dirPart[len(dirPart)-(width-28):]
		}

		// Calculate spacing for right alignment
		spacing := width - lipgloss.Width(namePart) - lipgloss.Width(dirPart) - 2
		if spacing < 1 {
			spacing = 1
		}

		row := fmt.Sprintf(" %s%s%s ", namePart, strings.Repeat(" ", spacing), dirPart)
		rows = append(rows, style.Width(width).Render(row))
	}

	return lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(highlight).
		MarginLeft(2).
		Render(lipgloss.JoinVertical(lipgloss.Left, rows...))
}

func (m *model) handleInterventionKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	if m.pendingIntervention == nil {
		return m, nil
	}

	switch msg.String() {
	case "up", "k":
		m.pendingIntervention.selected--
		if m.pendingIntervention.selected < 0 {
			m.pendingIntervention.selected = len(m.pendingIntervention.choices) - 1
		}
		// Re-render the selector in place
		m.updateInterventionDisplay()
		return m, nil

	case "down", "j":
		m.pendingIntervention.selected++
		if m.pendingIntervention.selected >= len(m.pendingIntervention.choices) {
			m.pendingIntervention.selected = 0
		}
		m.updateInterventionDisplay()
		return m, nil

	case "enter":
		// User confirmed their choice
		choice := m.pendingIntervention.choices[m.pendingIntervention.selected]
		resumeFn := m.pendingIntervention.resume
		m.pendingIntervention = nil

		// Remove the intervention UI from messages
		if len(m.messages) > 0 {
			m.messages = m.messages[:len(m.messages)-1]
		}

		// Show what the user chose
		m.messages = append(m.messages, subtleStyle.Render("→ "+choice))

		// Resume the agent loop
		m.isThinking = true
		return m, tea.Batch(m.asyncRender(), m.resumeIntervention(resumeFn, choice))

	case "esc":
		// User cancelled
		m.pendingIntervention = nil
		if len(m.messages) > 0 {
			m.messages = m.messages[:len(m.messages)-1]
		}
		m.messages = append(m.messages, subtleStyle.Render("→ Action cancelled"))
		return m, m.asyncRender()
	}

	return m, nil
}

func (m *model) updateInterventionDisplay() {
	if len(m.messages) > 0 {
		m.messages[len(m.messages)-1] = m.renderInterventionSelector()
		m.updateViewport()
	}
}

func (m *model) renderInterventionSelector() string {
	if m.pendingIntervention == nil {
		return ""
	}

	var lines []string
	lines = append(lines, interventionTitleStyle.Render("⚠️  "+m.pendingIntervention.title))
	lines = append(lines, "")
	lines = append(lines, helpStyle.Render("Use ↑/↓ to navigate, Enter to confirm, Esc to cancel"))
	lines = append(lines, "")

	for i, choice := range m.pendingIntervention.choices {
		prefix := "  "
		style := interventionChoiceStyle
		if i == m.pendingIntervention.selected {
			prefix = "▶ "
			style = interventionSelectedStyle
		}
		lines = append(lines, style.Render(prefix+choice))
	}

	return interventionBoxStyle.Render(strings.Join(lines, "\n"))
}

func (m *model) asyncRender() tea.Cmd {
	return m.asyncRenderWithPos(m.viewport.AtTop(), m.viewport.AtBottom(), m.viewport.YOffset)
}

func (m *model) asyncRenderWithPos(wasAtTop, wasAtBottom bool, prevOffset int) tea.Cmd {
	return func() tea.Msg {
		content := m.renderMessages()
		return layoutMsg{
			content:     content,
			wasAtBottom: wasAtBottom,
			wasAtTop:    wasAtTop,
			prevOffset:  prevOffset,
		}
	}
}