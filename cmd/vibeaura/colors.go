package main

import (
	"github.com/charmbracelet/lipgloss"
)

// Vibeauracle Color Palette - A vibrant, modern theme
var (
	// Primary accents
	ColorPrimary   = lipgloss.Color("#7C3AED") // Violet
	ColorSecondary = lipgloss.Color("#06B6D4") // Cyan
	ColorAccent    = lipgloss.Color("#F59E0B") // Amber

	// Status colors
	ColorSuccess = lipgloss.Color("#10B981") // Emerald
	ColorWarning = lipgloss.Color("#F59E0B") // Amber
	ColorError   = lipgloss.Color("#EF4444") // Red
	ColorInfo    = lipgloss.Color("#3B82F6") // Blue

	// Neutral tones
	ColorMuted = lipgloss.Color("#6B7280") // Gray
	ColorDim   = lipgloss.Color("#9CA3AF") // Light Gray
	ColorBold  = lipgloss.Color("#F3F4F6") // Almost White

	// Special
	ColorMagic   = lipgloss.Color("#EC4899") // Pink
	ColorNeon    = lipgloss.Color("#22D3EE") // Bright Cyan
	ColorSunrise = lipgloss.Color("#FB923C") // Orange
)

// CLI Styles - for colorful command-line output
var (
	// Headers and titles
	cliTitle = lipgloss.NewStyle().
			Bold(true).
			Foreground(ColorPrimary)

	cliSubtitle = lipgloss.NewStyle().
			Italic(true).
			Foreground(ColorSecondary)

	// Status messages
	cliSuccess = lipgloss.NewStyle().
			Bold(true).
			Foreground(ColorSuccess)

	cliError = lipgloss.NewStyle().
			Bold(true).
			Foreground(ColorError)

	cliWarning = lipgloss.NewStyle().
			Foreground(ColorWarning)

	cliInfo = lipgloss.NewStyle().
		Foreground(ColorInfo)

	// Labels and values
	cliLabel = lipgloss.NewStyle().
			Foreground(ColorNeon).
			Bold(true)

	cliValue = lipgloss.NewStyle().
			Foreground(ColorBold)

	cliMuted = lipgloss.NewStyle().
			Foreground(ColorMuted)

	// Special elements
	cliBullet = lipgloss.NewStyle().
			Foreground(ColorMagic).
			Bold(true)

	cliCommand = lipgloss.NewStyle().
			Foreground(ColorSunrise).
			Bold(true)

	cliHighlight = lipgloss.NewStyle().
			Foreground(ColorAccent).
			Bold(true)

	// Badges
	cliBadgeSuccess = lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color("#000")).
			Background(ColorSuccess).
			Padding(0, 1)

	cliBadgeError = lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color("#FFF")).
			Background(ColorError).
			Padding(0, 1)

	cliBadgeInfo = lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color("#FFF")).
			Background(ColorInfo).
			Padding(0, 1)

	cliBadgeWarning = lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color("#000")).
			Background(ColorWarning).
			Padding(0, 1)
)

// Gradient helpers
func gradientText(text string) string {
	// Simple gradient effect using alternating colors
	colors := []lipgloss.Color{ColorPrimary, ColorSecondary, ColorMagic, ColorNeon}
	result := ""
	for i, char := range text {
		style := lipgloss.NewStyle().Foreground(colors[i%len(colors)])
		result += style.Render(string(char))
	}
	return result
}
