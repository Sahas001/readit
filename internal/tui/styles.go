package tui

import "github.com/charmbracelet/lipgloss"

// Convenience color references from currentTheme for backwards compatibility.
var (
	colorPrimary   = currentTheme.Primary
	colorSecondary = currentTheme.Secondary
	colorAccent    = currentTheme.Accent
	colorText      = currentTheme.Text
	colorMuted     = currentTheme.TextMuted
	colorBg        = currentTheme.Background
	colorCardBg    = currentTheme.CardBg
	colorUpvote    = currentTheme.Upvote
	colorDownvote  = currentTheme.Downvote
	colorBorder    = currentTheme.Border
)

// Typography and Branding Styles
var (
	styleLogo = lipgloss.NewStyle().
			Foreground(currentTheme.Primary).
			Bold(true)

	styleLogoBadge = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#FFFFFF")).
			Background(currentTheme.Primary).
			Bold(true).
			Padding(0, 1)

	styleTagline = lipgloss.NewStyle().
			Foreground(currentTheme.TextMuted).
			Italic(true)

	styleTitle = lipgloss.NewStyle().
			Foreground(currentTheme.Text).
			Bold(true)

	styleSubtitle = lipgloss.NewStyle().
			Foreground(currentTheme.TextMuted)

	styleMeta = lipgloss.NewStyle().
			Foreground(currentTheme.TextMuted)

	styleMetaAuthor = lipgloss.NewStyle().
			Foreground(currentTheme.Text).
			Bold(true)

	stylePrompt = lipgloss.NewStyle().
			Foreground(currentTheme.Accent).
			Bold(true)

	styleRule = lipgloss.NewStyle().
			Foreground(currentTheme.Border)
)

// Board and Post List Styles
var (
	styleScore = lipgloss.NewStyle().
			Foreground(currentTheme.Upvote).
			Bold(true).
			Width(5).
			Align(lipgloss.Right)

	styleVoteNeutral = lipgloss.NewStyle().
				Foreground(currentTheme.TextDim).
				Bold(true)

	styleVoteUp = lipgloss.NewStyle().
			Foreground(currentTheme.Upvote).
			Bold(true)

	styleVoteDown = lipgloss.NewStyle().
			Foreground(currentTheme.Downvote).
			Bold(true)

	styleSelectedItem = lipgloss.NewStyle().
				Foreground(currentTheme.Primary).
				Bold(true).
				BorderLeft(true).
				BorderStyle(lipgloss.ThickBorder()).
				BorderForeground(currentTheme.Primary).
				PaddingLeft(1)

	styleNormalItem = lipgloss.NewStyle().
			Foreground(currentTheme.Text).
			PaddingLeft(2)

	stylePostTitle = lipgloss.NewStyle().
			Foreground(currentTheme.Text).
			Bold(true)

	stylePostTitleSelected = lipgloss.NewStyle().
				Foreground(lipgloss.Color("#FFFFFF")).
				Bold(true)

	stylePostCardSelected = lipgloss.NewStyle().
				Background(currentTheme.CardBgHover).
				BorderLeft(true).
				BorderStyle(lipgloss.ThickBorder()).
				BorderForeground(currentTheme.Primary).
				Padding(0, 1)

	stylePostCardNormal = lipgloss.NewStyle().
				Padding(0, 1).
				PaddingLeft(2)

	styleLinkBadge = lipgloss.NewStyle().
			Foreground(currentTheme.Accent).
			Background(currentTheme.CardBg).
			Padding(0, 1)

	styleSortPill = lipgloss.NewStyle().
			Foreground(currentTheme.TextDim).
			Italic(true)
)

// Threaded Comment Styles
var (
	stylePostBody = lipgloss.NewStyle().
			Foreground(currentTheme.Text).
			Padding(1, 0)

	styleBranch = lipgloss.NewStyle().
			Foreground(currentTheme.TextDim)

	styleSelectedBranch = lipgloss.NewStyle().
				Foreground(currentTheme.Primary).
				Bold(true)

	styleAuthor = lipgloss.NewStyle().
			Foreground(currentTheme.Accent).
			Bold(true)

	styleOpBadge = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#FFFFFF")).
			Background(currentTheme.Primary).
			Padding(0, 1).
			Bold(true)

	styleBadge = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#FFFFFF")).
			Background(currentTheme.Primary).
			Padding(0, 1).
			Bold(true)
)

// Modal Form & Dialog Styles
var (
	styleInputFocused = lipgloss.NewStyle().
				Border(lipgloss.RoundedBorder()).
				BorderForeground(currentTheme.Primary).
				Padding(0, 1)

	styleInputBlurred = lipgloss.NewStyle().
				Border(lipgloss.RoundedBorder()).
				BorderForeground(currentTheme.Border).
				Padding(0, 1)

	styleModalCard = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(currentTheme.Border).
			Padding(1, 2).
			Background(currentTheme.CardBg)

	styleCharCount = lipgloss.NewStyle().
			Foreground(currentTheme.TextDim)
)

// Status & Navigation Bar Styles
var (
	styleStatusBar = lipgloss.NewStyle().
			Foreground(currentTheme.TextMuted).
			Background(currentTheme.CardBg)

	styleStatusBadge = lipgloss.NewStyle().
				Foreground(lipgloss.Color("#FFFFFF")).
				Background(currentTheme.Primary).
				Bold(true).
				Padding(0, 1)

	styleStatusKey = lipgloss.NewStyle().
			Foreground(currentTheme.Text).
			Bold(true)

	styleStatusDesc = lipgloss.NewStyle().
			Foreground(currentTheme.TextMuted)

	styleStatusFlash = lipgloss.NewStyle().
				Foreground(currentTheme.Upvote).
				Bold(true)

	styleStatusDot = lipgloss.NewStyle().
			Foreground(currentTheme.Positive).
			Bold(true)
)

// Notification, Error, and Empty State Styles
var (
	styleError = lipgloss.NewStyle().
			Foreground(currentTheme.Negative).
			Bold(true)

	styleErrorCard = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(currentTheme.Negative).
			Padding(1, 2)

	styleEmptyCard = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(currentTheme.Border).
			Padding(1, 3).
			Align(lipgloss.Center)
)

