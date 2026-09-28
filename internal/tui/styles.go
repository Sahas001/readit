package tui

import "github.com/charmbracelet/lipgloss"

// Styles holds all session-scoped Lipgloss styling rules derived from a Theme palette.
type Styles struct {
	// Typography and Branding Styles
	Logo       lipgloss.Style
	LogoBadge  lipgloss.Style
	Tagline    lipgloss.Style
	Title      lipgloss.Style
	Subtitle   lipgloss.Style
	Meta       lipgloss.Style
	MetaAuthor lipgloss.Style
	Prompt     lipgloss.Style
	Rule       lipgloss.Style

	// Board and Post List Styles
	Score             lipgloss.Style
	VoteNeutral       lipgloss.Style
	VoteUp            lipgloss.Style
	VoteDown          lipgloss.Style
	SelectedItem      lipgloss.Style
	NormalItem        lipgloss.Style
	PostTitle         lipgloss.Style
	PostTitleSelected lipgloss.Style
	PostCardSelected  lipgloss.Style
	PostCardNormal    lipgloss.Style
	LinkBadge         lipgloss.Style
	SortPill          lipgloss.Style
	SearchBar         lipgloss.Style
	SearchBarActive   lipgloss.Style
	SearchBarFiltered lipgloss.Style
	FilterPrompt      lipgloss.Style
	FilterQuery       lipgloss.Style
	FilterHint        lipgloss.Style

	// Threaded Comment Styles
	PostBody       lipgloss.Style
	Branch         lipgloss.Style
	SelectedBranch lipgloss.Style
	Author         lipgloss.Style
	OpBadge        lipgloss.Style
	Badge          lipgloss.Style

	// Modal Form & Dialog Styles
	InputFocused lipgloss.Style
	InputBlurred lipgloss.Style
	ModalCard    lipgloss.Style
	CharCount    lipgloss.Style

	// Status & Navigation Bar Styles
	StatusBar   lipgloss.Style
	StatusBadge lipgloss.Style
	StatusKey   lipgloss.Style
	StatusDesc  lipgloss.Style
	StatusFlash lipgloss.Style
	StatusDot   lipgloss.Style

	// Notification, Error, and Empty State Styles
	Error     lipgloss.Style
	ErrorCard lipgloss.Style
	EmptyCard lipgloss.Style
}

// NewStyles constructs a new Styles bundle parameterized strictly by the given Theme palette.
func NewStyles(t Theme) Styles {
	return Styles{
		Logo: lipgloss.NewStyle().
			Foreground(lipgloss.Color("#FF4500")).
			Bold(true),

		LogoBadge: lipgloss.NewStyle().
			Foreground(lipgloss.Color("#FFFFFF")).
			Background(lipgloss.Color("#FF4500")).
			Bold(true).
			Padding(0, 1),

		Tagline: lipgloss.NewStyle().
			Foreground(t.TextMuted).
			Italic(true),

		Title: lipgloss.NewStyle().
			Foreground(t.Text).
			Bold(true),

		Subtitle: lipgloss.NewStyle().
			Foreground(t.TextMuted),

		Meta: lipgloss.NewStyle().
			Foreground(t.TextMuted),

		MetaAuthor: lipgloss.NewStyle().
			Foreground(t.Text).
			Bold(true),

		Prompt: lipgloss.NewStyle().
			Foreground(t.Accent).
			Bold(true),

		Rule: lipgloss.NewStyle().
			Foreground(t.Border),

		Score: lipgloss.NewStyle().
			Foreground(t.Upvote).
			Bold(true).
			Width(5).
			Align(lipgloss.Center),

		VoteNeutral: lipgloss.NewStyle().
			Foreground(t.TextDim).
			Bold(true),

		VoteUp: lipgloss.NewStyle().
			Foreground(t.Upvote).
			Bold(true),

		VoteDown: lipgloss.NewStyle().
			Foreground(t.Downvote).
			Bold(true),

		SelectedItem: lipgloss.NewStyle().
			Foreground(t.Primary).
			Bold(true).
			BorderLeft(true).
			BorderStyle(lipgloss.ThickBorder()).
			BorderForeground(t.Primary).
			PaddingLeft(1),

		NormalItem: lipgloss.NewStyle().
			Foreground(t.Text).
			PaddingLeft(2),

		PostTitle: lipgloss.NewStyle().
			Foreground(t.Text).
			Bold(true),

		PostTitleSelected: lipgloss.NewStyle().
			Foreground(lipgloss.Color("#FFFFFF")).
			Bold(true),

		PostCardSelected: lipgloss.NewStyle().
			BorderLeft(true).
			BorderStyle(lipgloss.ThickBorder()).
			BorderForeground(t.Primary).
			PaddingLeft(1),

		PostCardNormal: lipgloss.NewStyle().
			BorderLeft(true).
			BorderStyle(lipgloss.Border{Left: " "}).
			PaddingLeft(1),

		LinkBadge: lipgloss.NewStyle().
			Foreground(t.Accent).
			Bold(true),

		SortPill: lipgloss.NewStyle().
			Foreground(t.TextDim).
			Italic(true),

		SearchBar: lipgloss.NewStyle().
			Foreground(t.TextMuted),

		SearchBarActive: lipgloss.NewStyle().
			Foreground(t.Primary).
			Bold(true),

		SearchBarFiltered: lipgloss.NewStyle().
			Foreground(t.Accent).
			Bold(true),

		FilterPrompt: lipgloss.NewStyle().
			Foreground(t.Primary).
			Bold(true),

		FilterQuery: lipgloss.NewStyle().
			Foreground(lipgloss.Color("#FFFFFF")).
			Bold(true),

		FilterHint: lipgloss.NewStyle().
			Foreground(t.TextDim),

		PostBody: lipgloss.NewStyle().
			Foreground(t.Text).
			Padding(1, 0),

		Branch: lipgloss.NewStyle().
			Foreground(t.TextDim),

		SelectedBranch: lipgloss.NewStyle().
			Foreground(t.Primary).
			Bold(true),

		Author: lipgloss.NewStyle().
			Foreground(t.Accent).
			Bold(true),

		OpBadge: lipgloss.NewStyle().
			Foreground(lipgloss.Color("#FFFFFF")).
			Background(t.Primary).
			Padding(0, 1).
			Bold(true),

		Badge: lipgloss.NewStyle().
			Foreground(lipgloss.Color("#FFFFFF")).
			Background(t.Primary).
			Padding(0, 1).
			Bold(true),

		InputFocused: lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(t.Primary).
			Padding(0, 1),

		InputBlurred: lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(t.Border).
			Padding(0, 1),

		ModalCard: lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(t.Border).
			Padding(1, 2),

		CharCount: lipgloss.NewStyle().
			Foreground(t.TextDim),

		StatusBar: lipgloss.NewStyle().
			Foreground(t.TextMuted),

		StatusBadge: lipgloss.NewStyle().
			Foreground(lipgloss.Color("#FFFFFF")).
			Background(t.Primary).
			Bold(true).
			Padding(0, 1),

		StatusKey: lipgloss.NewStyle().
			Foreground(t.Text).
			Bold(true),

		StatusDesc: lipgloss.NewStyle().
			Foreground(t.TextMuted),

		StatusFlash: lipgloss.NewStyle().
			Foreground(t.Upvote).
			Bold(true),

		StatusDot: lipgloss.NewStyle().
			Foreground(t.Positive).
			Bold(true),

		Error: lipgloss.NewStyle().
			Foreground(t.Negative).
			Bold(true),

		ErrorCard: lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(t.Negative).
			Padding(1, 2),

		EmptyCard: lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(t.Border).
			Padding(1, 3).
			Align(lipgloss.Center),
	}
}

// Package default styles for fallback and backwards compatibility.
var defaultStyles = NewStyles(DefaultTheme())

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

// Typography and Branding Styles (backwards compatibility).
var (
	styleLogo       = defaultStyles.Logo
	styleLogoBadge  = defaultStyles.LogoBadge
	styleTagline    = defaultStyles.Tagline
	styleTitle      = defaultStyles.Title
	styleSubtitle   = defaultStyles.Subtitle
	styleMeta       = defaultStyles.Meta
	styleMetaAuthor = defaultStyles.MetaAuthor
	stylePrompt     = defaultStyles.Prompt
	styleRule       = defaultStyles.Rule
)

// Board and Post List Styles (backwards compatibility).
var (
	styleScore             = defaultStyles.Score
	styleVoteNeutral       = defaultStyles.VoteNeutral
	styleVoteUp            = defaultStyles.VoteUp
	styleVoteDown          = defaultStyles.VoteDown
	styleSelectedItem      = defaultStyles.SelectedItem
	styleNormalItem        = defaultStyles.NormalItem
	stylePostTitle         = defaultStyles.PostTitle
	stylePostTitleSelected = defaultStyles.PostTitleSelected
	stylePostCardSelected  = defaultStyles.PostCardSelected
	stylePostCardNormal    = defaultStyles.PostCardNormal
	styleLinkBadge         = defaultStyles.LinkBadge
	styleSortPill          = defaultStyles.SortPill
	styleSearchBar         = defaultStyles.SearchBar
	styleSearchBarActive   = defaultStyles.SearchBarActive
	styleSearchBarFiltered = defaultStyles.SearchBarFiltered
	styleFilterPrompt      = defaultStyles.FilterPrompt
	styleFilterQuery       = defaultStyles.FilterQuery
	styleFilterHint        = defaultStyles.FilterHint
)

// Threaded Comment Styles (backwards compatibility).
var (
	stylePostBody       = defaultStyles.PostBody
	styleBranch         = defaultStyles.Branch
	styleSelectedBranch = defaultStyles.SelectedBranch
	styleAuthor         = defaultStyles.Author
	styleOpBadge        = defaultStyles.OpBadge
	styleBadge          = defaultStyles.Badge
)

// Modal Form & Dialog Styles (backwards compatibility).
var (
	styleInputFocused = defaultStyles.InputFocused
	styleInputBlurred = defaultStyles.InputBlurred
	styleModalCard    = defaultStyles.ModalCard
	styleCharCount    = defaultStyles.CharCount
)

// Status & Navigation Bar Styles (backwards compatibility).
var (
	styleStatusBar   = defaultStyles.StatusBar
	styleStatusBadge = defaultStyles.StatusBadge
	styleStatusKey   = defaultStyles.StatusKey
	styleStatusDesc  = defaultStyles.StatusDesc
	styleStatusFlash = defaultStyles.StatusFlash
	styleStatusDot   = defaultStyles.StatusDot
)

// Notification, Error, and Empty State Styles (backwards compatibility).
var (
	styleError     = defaultStyles.Error
	styleErrorCard = defaultStyles.ErrorCard
	styleEmptyCard = defaultStyles.EmptyCard
)
