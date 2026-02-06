package main

import (
	"regexp"
	"strings"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/nathfavour/vibeauracle/brain"
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