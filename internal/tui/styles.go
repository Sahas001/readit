package tui

import "github.com/charmbracelet/lipgloss"

// Theme colours.
var (
	colorPrimary   = lipgloss.Color("#FF4500") // Reddit orange
	colorSecondary = lipgloss.Color("#5A5A5A")
	colorAccent    = lipgloss.Color("#00D1B2")
	colorText      = lipgloss.Color("#E0E0E0")
	colorMuted     = lipgloss.Color("#808080")
	colorBg        = lipgloss.Color("#1A1A2E")
	colorUpvote    = lipgloss.Color("#FF8B60")
	colorDownvote  = lipgloss.Color("#7193FF")
)

// Reusable styles.
var (
	styleLogo = lipgloss.NewStyle().
		Foreground(colorPrimary).
		Bold(true).
		Padding(0, 1)

	styleTitle = lipgloss.NewStyle().
		Foreground(colorText).
		Bold(true)

	styleSubtitle = lipgloss.NewStyle().
		Foreground(colorMuted).
		Italic(true)

	styleScore = lipgloss.NewStyle().
		Foreground(colorUpvote).
		Bold(true).
		Width(5).
		Align(lipgloss.Right)

	styleSelectedItem = lipgloss.NewStyle().
		Foreground(colorPrimary).
		Bold(true).
		BorderLeft(true).
		BorderStyle(lipgloss.ThickBorder()).
		BorderForeground(colorPrimary).
		PaddingLeft(1)

	styleNormalItem = lipgloss.NewStyle().
		Foreground(colorText).
		PaddingLeft(2)

	styleStatusBar = lipgloss.NewStyle().
		Foreground(colorMuted).
		Background(lipgloss.Color("#2A2A3E")).
		Padding(0, 1)

	stylePrompt = lipgloss.NewStyle().
		Foreground(colorAccent).
		Bold(true)

	styleError = lipgloss.NewStyle().
		Foreground(lipgloss.Color("#FF0000")).
		Bold(true)
)
