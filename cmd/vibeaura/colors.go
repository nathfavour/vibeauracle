package main

import (
	"fmt"

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

// ============================================================================
// MODULAR OUTPUT FUNCTIONS - Use these everywhere for consistent styling
// ============================================================================

// printTitle prints a styled section title with emoji
func printTitle(emoji, title string) {
	fmt.Println()
	fmt.Println(cliTitle.Render(emoji + " " + title))
	fmt.Println(cliMuted.Render("─────────────────────────────────────────────"))
}

// printKeyValue prints a labeled value (key: value format)
func printKeyValue(key, value string) {
	fmt.Printf("%s %s\n", cliLabel.Render(key+":"), cliValue.Render(value))
}

// printKeyValueHighlight prints a labeled value with highlighted value
func printKeyValueHighlight(key, value string) {
	fmt.Printf("%s %s\n", cliLabel.Render(key+":"), cliHighlight.Render(value))
}

// printSuccess prints a success message with badge
func printSuccess(message string) {
	fmt.Println(cliBadgeSuccess.Render("SUCCESS") + " " + cliSuccess.Render(message))
}

// printError prints an error message with badge
func printError(message string) {
	fmt.Println(cliBadgeError.Render("ERROR") + " " + cliError.Render(message))
}

// printInfo prints an info message
func printInfo(message string) {
	fmt.Println(cliInfo.Render("ℹ️  " + message))
}

// printWarning prints a warning message
func printWarning(message string) {
	fmt.Println(cliWarning.Render("⚠️  " + message))
}

// printBullet prints a bulleted list item
func printBullet(text string) {
	fmt.Println(cliBullet.Render("●") + " " + cliValue.Render(text))
}

// printBulletWithMeta prints a bullet with additional metadata
func printBulletWithMeta(text, meta string) {
	fmt.Printf("%s %s %s\n",
		cliBullet.Render("●"),
		cliValue.Render(text),
		cliMuted.Render("("+meta+")"),
	)
}

// printCommand prints a command hint
func printCommand(prefix, cmd, suffix string) {
	fmt.Println(cliInfo.Render(prefix) + " " + cliCommand.Render(cmd) + " " + cliInfo.Render(suffix))
}

// printStatus prints a status with badge
func printStatus(badge, message string) {
	fmt.Println(cliBadgeInfo.Render(badge) + " " + cliValue.Render(message))
}

// printDone prints completion message
func printDone() {
	fmt.Println()
	fmt.Println(cliSuccess.Render("✓ Done"))
}

// printNewline prints an empty line
func printNewline() {
	fmt.Println()
}
