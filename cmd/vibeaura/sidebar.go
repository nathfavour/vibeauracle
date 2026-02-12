package main

import (
	"fmt"
	"os"
	"sort"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

// AuraComponent is a dynamic widget in the smart sidebar.
type AuraComponent interface {
	Name() string
	Update(tea.Msg) tea.Cmd
	View(width, height int, m *model) string
	Relevance(m *model) float64 // 0.0 to 1.0
}

// SidebarManager manages the fluid layout of AuraComponents.
type SidebarManager struct {
	components []AuraComponent
	width      int
	height     int
}

func NewSidebarManager() *SidebarManager {
	return &SidebarManager{
		components: []AuraComponent{
			&ExplorerAura{},
			&FileFocusAura{},
			&TodoAura{},
			&VitalsAura{},
		},
	}
}

func (s *SidebarManager) Update(msg tea.Msg) tea.Cmd {
	var cmds []tea.Cmd
	for _, c := range s.components {
		cmds = append(cmds, c.Update(msg))
	}
	return tea.Batch(cmds...)
}

func (s *SidebarManager) View(width, height int, m *model) string {
	s.width = width
	s.height = height

	// 1. Calculate relevance scores
	type scored struct {
		c     AuraComponent
		score float64
	}
	var scoredList []scored
	totalScore := 0.0

	for _, c := range s.components {
		score := c.Relevance(m)
		if score > 0 {
			scoredList = append(scoredList, scored{c, score})
			totalScore += score
		}
	}

	// 2. Sort by score descending
	sort.Slice(scoredList, func(i, j int) bool {
		return scoredList[i].score > scoredList[j].score
	})

	// Focus Override: If top item has very high score, give it 100% height
	if len(scoredList) > 0 && scoredList[0].score >= 0.9 {
		return scoredList[0].c.View(width, height, m)
	}

	// 3. Allocate height
	if totalScore == 0 {
		return lipgloss.NewStyle().
			Width(width).
			Height(height).
			Align(lipgloss.Center, lipgloss.Center).
			Render(subtleStyle.Render("System Idle\nListening for vibes..."))
	}

	var sections []string
	remainingHeight := height - (len(scoredList) - 1) // Subtract space for separators

	for i, item := range scoredList {
		h := int(float64(remainingHeight) * (item.score / totalScore))
		if i == len(scoredList)-1 {
			// Ensure we use all available height in the last item
			currentUsed := 0
			for _, s := range sections {
				currentUsed += lipgloss.Height(s) + 1
			}
			h = height - currentUsed
		}

		if h > 0 {
			view := item.c.View(width, h, m)
			if view != "" {
				sections = append(sections, view)
			}
		}
	}

	return lipgloss.JoinVertical(lipgloss.Left, sections...)
}

// --- Concrete Components ---

// VitalsAura shows system stats.
type VitalsAura struct{}

func (v *VitalsAura) Name() string { return "vitals" }
func (v *VitalsAura) Update(msg tea.Msg) tea.Cmd { return nil }
func (v *VitalsAura) Relevance(m *model) float64 { return 0.2 }
func (v *VitalsAura) View(w, h int, m *model) string {
	if h < 2 {
		return ""
	}
	title := lipgloss.NewStyle().Foreground(lipgloss.Color("#00FF00")).Bold(true).Render("◆ VITALS")
	return lipgloss.NewStyle().Width(w).Height(h).Border(lipgloss.NormalBorder(), false, false, false, true).BorderForeground(lipgloss.Color("#444444")).PaddingLeft(1).Render(title + "\n" + subtleStyle.Render("CPU/MEM Stable"))
}

// TodoAura parses TODO files.
type TodoAura struct{}

func (t *TodoAura) Name() string { return "todos" }
func (t *TodoAura) Update(msg tea.Msg) tea.Cmd { return nil }
func (t *TodoAura) Relevance(m *model) float64 {
	if len(m.focusScores) == 0 {
		return 0.8
	}
	return 0.3
}
func (t *TodoAura) View(w, h int, m *model) string {
	if h < 2 {
		return ""
	}
	title := lipgloss.NewStyle().Foreground(lipgloss.Color("#FFD700")).Bold(true).Render("◆ TASKS")

	// Look for TODO files
	var todos []string
	todoFiles := []string{"TODO.md", "TODO", "VIBES.md", "TODO.prod.md"}
	for _, f := range todoFiles {
		if data, err := os.ReadFile(f); err == nil {
			lines := strings.Split(string(data), "\n")
			for _, line := range lines {
				line = strings.TrimSpace(line)
				if strings.Contains(line, "[ ]") {
					task := strings.TrimSpace(strings.Replace(line, "- [ ]", "", 1))
					task = strings.TrimSpace(strings.Replace(task, "* [ ]", "", 1))
					glow := lipgloss.NewStyle().Foreground(lipgloss.Color("#FFD700")).Render("○ ")
					todos = append(todos, glow+task)
				} else if strings.Contains(line, "[x]") || strings.Contains(line, "[X]") {
					task := strings.TrimSpace(strings.Replace(line, "- [x]", "", 1))
					task = strings.TrimSpace(strings.Replace(task, "- [X]", "", 1))
					done := lipgloss.NewStyle().Foreground(lipgloss.Color("#444444")).Strikethrough(true).Render("✓ " + task)
					todos = append(todos, done)
				}
			}
		}
	}

	if len(todos) == 0 {
		return lipgloss.NewStyle().Width(w).Height(h).Border(lipgloss.NormalBorder(), false, false, false, true).BorderForeground(lipgloss.Color("#444444")).PaddingLeft(1).Render(title + "\n" + subtleStyle.Render("No pending tasks in TODO.md"))
	}

	var sb strings.Builder
	sb.WriteString(title + "\n")
	for i, todo := range todos {
		if i >= h-2 {
			break
		}
		sb.WriteString(todo + "\n")
	}

	return lipgloss.NewStyle().Width(w).Height(h).Border(lipgloss.NormalBorder(), false, false, false, true).BorderForeground(lipgloss.Color("#444444")).PaddingLeft(1).Render(sb.String())
}

// ExplorerAura is the smart file list.
type ExplorerAura struct{}

func (e *ExplorerAura) Name() string { return "explorer" }
func (e *ExplorerAura) Update(msg tea.Msg) tea.Cmd { return nil }
func (e *ExplorerAura) Relevance(m *model) float64 {
	if m.focus == focusTree {
		return 1.0
	}
	return 0.4
}
func (e *ExplorerAura) View(w, h int, m *model) string {
	if h < 2 {
		return ""
	}
	title := lipgloss.NewStyle().Foreground(lipgloss.Color("#04D9FF")).Bold(true).Render("◆ EXPLORER")

	var sb strings.Builder
	sb.WriteString(title + "\n")

	// If we have recent command output (like ls), show that instead of raw tree
	if m.lastCmdOutput != "" {
		lines := strings.Split(m.lastCmdOutput, "\n")
		for i, line := range lines {
			if i >= h-3 {
				sb.WriteString(subtleStyle.Render("..."))
				break
			}
			if strings.TrimSpace(line) == "" { continue }
			sb.WriteString("  " + line + "\n")
		}
		return lipgloss.NewStyle().Width(w).Height(h).Border(lipgloss.NormalBorder(), false, false, false, true).BorderForeground(lipgloss.Color("#04D9FF")).PaddingLeft(1).Render(sb.String())
	}

	maxEntries := h - 2
	for i, entry := range m.treeEntries {
		if i >= maxEntries {
			break
		}
		cursor := "  "
		style := lipgloss.NewStyle()
		if i == m.treeCursor && m.focus == focusTree {
			cursor = "> "
			style = suggestionStyle
		}
		icon := "📄 "
		if entry.IsDir() {
			icon = "📁 "
		}
		sb.WriteString(style.Render(cursor+icon+entry.Name()) + "\n")
	}

	return lipgloss.NewStyle().Width(w).Height(h).Border(lipgloss.NormalBorder(), false, false, false, true).BorderForeground(lipgloss.Color("#444444")).PaddingLeft(1).Render(sb.String())
}

// FileFocusAura handles targeted file previews via # command.
type FileFocusAura struct {
	lastFile string
	content  string
}

func (f *FileFocusAura) Name() string { return "focus" }
func (f *FileFocusAura) Update(msg tea.Msg) tea.Cmd { return nil }
func (f *FileFocusAura) Relevance(m *model) float64 {
	max := 0.0
	for _, score := range m.focusScores {
		if score > max {
			max = score
		}
	}
	return max
}
func (f *FileFocusAura) View(w, h int, m *model) string {
	if h < 2 {
		return ""
	}

	var topFile string
	var topScore float64
	for file, score := range m.focusScores {
		if score > topScore {
			topScore = score
			topFile = file
		}
	}

	if topFile == "" {
		return ""
	}

	if topFile != f.lastFile {
		data, err := os.ReadFile(topFile)
		if err == nil {
			f.content = string(data)
			f.lastFile = topFile
		}
	}

	// Check for line range in textarea
	fileRange := ""
	val := m.textarea.Value()
	tag := "#" + topFile
	if idx := strings.Index(val, tag+":"); idx != -1 {
		rest := val[idx+len(tag)+1:]
		if end := strings.Index(rest, " "); end != -1 {
			fileRange = rest[:end]
		} else {
			fileRange = rest
		}
	}

	title := lipgloss.NewStyle().Foreground(lipgloss.Color("#FF00D7")).Bold(true).Render("◆ " + topFile)
	if fileRange != "" {
		title += lipgloss.NewStyle().Foreground(lipgloss.Color("#7D56F4")).Render(" :" + fileRange)
	}

	lines := strings.Split(f.content, "\n")
	startLine := 0
	highlightStart := -1
	highlightEnd := -1
	
	// Handle range parsing like "10-20" or "45"
	if fileRange != "" {
		if strings.Contains(fileRange, "-") {
			parts := strings.Split(fileRange, "-")
			fmt.Sscanf(parts[0], "%d", &highlightStart)
			fmt.Sscanf(parts[1], "%d", &highlightEnd)
			startLine = highlightStart - 3 // Show some context above
		} else {
			fmt.Sscanf(fileRange, "%d", &highlightStart)
			highlightEnd = highlightStart
			startLine = highlightStart - (h / 2) // Center the line
		}
	}
	
	if startLine < 0 { startLine = 0 }
	if startLine >= len(lines) { startLine = len(lines) - 1 }

	var sb strings.Builder
	sb.WriteString(title + "\n")
	for i := startLine; i < len(lines) && i < startLine+(h-2); i++ {
		isHighlighted := (i+1 >= highlightStart && i+1 <= highlightEnd)
		
		lineNumStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("#444444"))
		contentStyle := lipgloss.NewStyle()
		
		if isHighlighted {
			lineNumStyle = lineNumStyle.Foreground(lipgloss.Color("#FF00D7")).Bold(true)
			contentStyle = contentStyle.Foreground(lipgloss.Color("#FAFAFA")).Background(lipgloss.Color("#5F00FF"))
		}

		lineNum := lineNumStyle.Render(fmt.Sprintf("%3d ", i+1))
		lineContent := lines[i]
		if len(lineContent) > w-5 {
			lineContent = lineContent[:w-8] + "..."
		}
		sb.WriteString(lineNum + contentStyle.Render(lineContent) + "\n")
	}

	return lipgloss.NewStyle().Width(w).Height(h).Border(lipgloss.NormalBorder(), false, false, false, true).BorderForeground(lipgloss.Color("#FF00D7")).PaddingLeft(1).Render(sb.String())
}

