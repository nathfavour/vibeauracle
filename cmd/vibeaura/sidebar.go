package main

import (
	"fmt"
	"sort"
	"strings"
	"time"

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

	// 3. Allocate height
	if totalScore == 0 {
		return lipgloss.NewStyle().
			Width(width).
			Height(height).
			Align(lipgloss.Center, lipgloss.Center).
			Render(subtleStyle.Render("System Idle
Listening for vibes..."))
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
			sections = append(sections, view)
		}
	}

	return lipgloss.JoinVertical(lipgloss.Left, sections...)
}

// --- Concrete Components ---

// VitalsAura shows system stats.
type VitalsAura struct{}

func (v *VitalsAura) Name() string { return "vitals" }
func (v *VitalsAura) Update(msg tea.Msg) tea.Cmd { return nil }
func (v *VitalsAura) Relevance(m *model) float64 { return 0.2 } // Always present but low priority
func (v *VitalsAura) View(w, h int, m *model) string {
	if h < 2 { return "" }
	title := lipgloss.NewStyle().Foreground(lipgloss.Color("#00FF00")).Bold(true).Render("◆ VITALS")
	// Simplified view for sidebar
	return lipgloss.NewStyle().Width(w).Height(h).Border(lipgloss.NormalBorder(), false, false, false, true).BorderForeground(lipgloss.Color("#444444")).PaddingLeft(1).Render(title + "
" + subtleStyle.Render("CPU/MEM Stable"))
}

// TodoAura parses TODO files.
type TodoAura struct{}

func (t *TodoAura) Name() string { return "todos" }
func (t *TodoAura) Update(msg tea.Msg) tea.Cmd { return nil }
func (t *TodoAura) Relevance(m *model) float64 {
	// High relevance if no active focus or new session
	if len(m.activeFiles) == 0 {
		return 0.8
	}
	return 0.3
}
func (t *TodoAura) View(w, h int, m *model) string {
	if h < 2 { return "" }
	title := lipgloss.NewStyle().Foreground(lipgloss.Color("#FFD700")).Bold(true).Render("◆ TASKS")
	return lipgloss.NewStyle().Width(w).Height(h).Border(lipgloss.NormalBorder(), false, false, false, true).BorderForeground(lipgloss.Color("#444444")).PaddingLeft(1).Render(title + "
" + subtleStyle.Render("Reading TODO.md..."))
}

// ExplorerAura is the smart file list.
type ExplorerAura struct{}

func (e *ExplorerAura) Name() string { return "explorer" }
func (e *ExplorerAura) Update(msg tea.Msg) tea.Cmd { return nil }
func (e *ExplorerAura) Relevance(m *model) float64 {
	if m.focus == focusTree { return 1.0 }
	return 0.5
}
func (e *ExplorerAura) View(w, h int, m *model) string {
	if h < 2 { return "" }
	title := lipgloss.NewStyle().Foreground(lipgloss.Color("#04D9FF")).Bold(true).Render("◆ EXPLORER")
	return lipgloss.NewStyle().Width(w).Height(h).Border(lipgloss.NormalBorder(), false, false, false, true).BorderForeground(lipgloss.Color("#444444")).PaddingLeft(1).Render(title + "
" + subtleStyle.Render(m.currentPath))
}

// FileFocusAura handles targeted file previews via # command.
type FileFocusAura struct{}

func (f *FileFocusAura) Name() string { return "focus" }
func (f *FileFocusAura) Update(msg tea.Msg) tea.Cmd { return nil }
func (f *FileFocusAura) Relevance(m *model) float64 {
	max := 0.0
	for _, score := range m.focusScores {
		if score > max { max = score }
	}
	return max
}
func (f *FileFocusAura) View(w, h int, m *model) string {
	if h < 2 { return "" }
	
	// Find top active file
	var topFile string
	var topScore float64
	for file, score := range m.focusScores {
		if score > topScore {
			topScore = score
			topFile = file
		}
	}

	if topFile == "" { return "" }

	title := lipgloss.NewStyle().Foreground(lipgloss.Color("#FF00D7")).Bold(true).Render("◆ FOCUS: " + topFile)
	return lipgloss.NewStyle().Width(w).Height(h).Border(lipgloss.NormalBorder(), false, false, false, true).BorderForeground(lipgloss.Color("#FF00D7")).PaddingLeft(1).Render(title + "
" + subtleStyle.Render("Lines targeted..."))
}
