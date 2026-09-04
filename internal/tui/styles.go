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
	colorCardBg    = lipgloss.Color("#222238")
	colorUpvote    = lipgloss.Color("#FF8B60")
	colorDownvote  = lipgloss.Color("#7193FF")
	colorBorder    = lipgloss.Color("#3A3A52")
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
			Background(colorCardBg)

	styleStatusBadge = lipgloss.NewStyle().
				Foreground(lipgloss.Color("#FFFFFF")).
				Background(colorPrimary).
				Bold(true).
				Padding(0, 1)

	styleStatusKey = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#FFFFFF")).
			Bold(true)

	styleStatusDesc = lipgloss.NewStyle().
			Foreground(colorMuted)

	styleStatusFlash = lipgloss.NewStyle().
				Foreground(colorUpvote).
				Bold(true)

	stylePrompt = lipgloss.NewStyle().
			Foreground(colorAccent).
			Bold(true)

	styleError = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#FF4545")).
			Bold(true)

	// Post detail and comment styles
	stylePostBody = lipgloss.NewStyle().
			Foreground(colorText).
			Padding(1, 0)

	styleBranch = lipgloss.NewStyle().
			Foreground(colorMuted)

	styleSelectedBranch = lipgloss.NewStyle().
				Foreground(colorPrimary).
				Bold(true)

	styleAuthor = lipgloss.NewStyle().
			Foreground(colorAccent).
			Bold(true)

	styleBadge = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#FFFFFF")).
			Background(colorPrimary).
			Padding(0, 1).
			Bold(true)

	styleOpBadge = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#FFFFFF")).
			Background(colorPrimary).
			Padding(0, 1).
			Bold(true)

	styleInputFocused = lipgloss.NewStyle().
				Border(lipgloss.RoundedBorder()).
				BorderForeground(colorPrimary).
				Padding(0, 1)

	styleInputBlurred = lipgloss.NewStyle().
				Border(lipgloss.RoundedBorder()).
				BorderForeground(colorBorder).
				Padding(0, 1)

	styleHeaderBox = lipgloss.NewStyle().
			Border(lipgloss.NormalBorder(), false, false, true, false).
			BorderForeground(colorBorder).
			PaddingBottom(1)

	styleCharCount = lipgloss.NewStyle().
			Foreground(colorMuted)

	styleRule = lipgloss.NewStyle().
			Foreground(colorBorder)
)
