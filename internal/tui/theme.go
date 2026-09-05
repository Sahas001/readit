package tui

import "github.com/charmbracelet/lipgloss"

// Spacing constants for consistent vertical and horizontal rhythm.
const (
	SpaceXS = 1
	SpaceSM = 1
	SpaceMD = 2
	SpaceLG = 3
	SpaceXL = 4
)

// Theme defines the complete semantic color palette for ReadIT.
type Theme struct {
	Primary     lipgloss.Color // Reddit brand orange
	PrimaryDark lipgloss.Color // Deep brick orange
	Secondary   lipgloss.Color // Cool slate gray
	Accent      lipgloss.Color // Vibrant cyan / teal
	Text        lipgloss.Color // Crisp soft white
	TextMuted   lipgloss.Color // Metadata gray
	TextDim     lipgloss.Color // Subdued background gray
	Background  lipgloss.Color // Terminal charcoal canvas
	CardBg      lipgloss.Color // Elevated panel background
	CardBgHover lipgloss.Color // Selected item background
	Border      lipgloss.Color // Subtle line border
	BorderFocus lipgloss.Color // Highlighted / focused border
	Upvote      lipgloss.Color // Warm orange vote
	Downvote    lipgloss.Color // Cool blue vote
	Positive    lipgloss.Color // Success / OP green
	Negative    lipgloss.Color // Danger / Error red
	Selection   lipgloss.Color // Active selection indicator
}

// DefaultTheme returns the production Reddit/developer-tool dark theme.
func DefaultTheme() Theme {
	return Theme{
		Primary:     lipgloss.Color("#FF4500"), // Reddit orange
		PrimaryDark: lipgloss.Color("#CC3700"), // Deep orange
		Secondary:   lipgloss.Color("#5A5A72"), // Cool slate
		Accent:      lipgloss.Color("#00D1B2"), // Cyan/Teal
		Text:        lipgloss.Color("#F0F0F5"), // Soft white
		TextMuted:   lipgloss.Color("#888899"), // Metadata gray
		TextDim:     lipgloss.Color("#4E4E62"), // Dark gray
		Background:  lipgloss.Color("#12121A"), // Dark canvas
		CardBg:      lipgloss.Color("#181824"), // Slate panel
		CardBgHover: lipgloss.Color("#222234"), // Selected item background
		Border:      lipgloss.Color("#2C2C3E"), // Subtle border
		BorderFocus: lipgloss.Color("#FF4500"), // Active orange border
		Upvote:      lipgloss.Color("#FF8B60"), // Upvote orange
		Downvote:    lipgloss.Color("#7193FF"), // Downvote blue
		Positive:    lipgloss.Color("#48C774"), // Green
		Negative:    lipgloss.Color("#FF3860"), // Red
		Selection:   lipgloss.Color("#FF4500"), // Selection indicator
	}
}

// Global active theme.
var currentTheme = DefaultTheme()
