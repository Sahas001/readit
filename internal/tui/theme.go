package tui

import (
	"strings"

	"github.com/charmbracelet/lipgloss"
)

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
	ID          string         // Unique identifier ("readit", "catppuccin", "nord", "dracula", "gruvbox", "tokyonight")
	Name        string         // Display name ("ReadIT Dark", "Catppuccin Mocha", etc.)
	Primary     lipgloss.Color // Main brand / accent color
	PrimaryDark lipgloss.Color // Deep / darkened brand accent
	Secondary   lipgloss.Color // Secondary accent / slate
	Accent      lipgloss.Color // Vibrant highlight / cyan / teal
	Text        lipgloss.Color // Foreground text
	TextMuted   lipgloss.Color // Metadata / secondary text
	TextDim     lipgloss.Color // Subdued background / border text
	Background  lipgloss.Color // Terminal canvas background
	CardBg      lipgloss.Color // Elevated panel / card background
	CardBgHover lipgloss.Color // Selected item background
	Border      lipgloss.Color // Subtle line border
	BorderFocus lipgloss.Color // Highlighted / focused border
	Upvote      lipgloss.Color // Upvote color
	Downvote    lipgloss.Color // Downvote color
	Positive    lipgloss.Color // Success / OP green
	Negative    lipgloss.Color // Danger / Error red
	Selection   lipgloss.Color // Active selection indicator
}

// ThemeReadIT returns the default Reddit/developer-tool dark theme.
func ThemeReadIT() Theme {
	return Theme{
		ID:          "readit",
		Name:        "ReadIT Dark",
		Primary:     lipgloss.Color("#FF4500"), // Reddit orange
		PrimaryDark: lipgloss.Color("#CC3700"), // Deep brick orange
		Secondary:   lipgloss.Color("#5A5A72"), // Cool slate gray
		Accent:      lipgloss.Color("#00D1B2"), // Cyan / Teal
		Text:        lipgloss.Color("#F0F0F5"), // Crisp soft white
		TextMuted:   lipgloss.Color("#888899"), // Metadata gray
		TextDim:     lipgloss.Color("#4E4E62"), // Subdued dark gray
		Background:  lipgloss.Color("#12121A"), // Dark canvas
		CardBg:      lipgloss.Color("#181824"), // Slate panel
		CardBgHover: lipgloss.Color("#222234"), // Selected item background
		Border:      lipgloss.Color("#2C2C3E"), // Subtle border
		BorderFocus: lipgloss.Color("#FF4500"), // Active orange border
		Upvote:      lipgloss.Color("#FF8B60"), // Upvote orange
		Downvote:    lipgloss.Color("#7193FF"), // Downvote blue
		Positive:    lipgloss.Color("#48C774"), // OP green
		Negative:    lipgloss.Color("#FF3860"), // Danger red
		Selection:   lipgloss.Color("#FF4500"), // Selection indicator
	}
}

// ThemeCatppuccinMocha returns the soothing pastel Catppuccin Mocha theme.
func ThemeCatppuccinMocha() Theme {
	return Theme{
		ID:          "catppuccin",
		Name:        "Catppuccin Mocha",
		Primary:     lipgloss.Color("#CBA6F7"), // Mauve
		PrimaryDark: lipgloss.Color("#A6ADC8"), // Subtext0
		Secondary:   lipgloss.Color("#89B4FA"), // Blue
		Accent:      lipgloss.Color("#94E2D5"), // Teal
		Text:        lipgloss.Color("#CDD6F4"), // Foreground Text
		TextMuted:   lipgloss.Color("#A6ADC8"), // Subtext0
		TextDim:     lipgloss.Color("#585B70"), // Surface2
		Background:  lipgloss.Color("#1E1E2E"), // Base
		CardBg:      lipgloss.Color("#181825"), // Mantle
		CardBgHover: lipgloss.Color("#313244"), // Surface0
		Border:      lipgloss.Color("#45475A"), // Surface1
		BorderFocus: lipgloss.Color("#CBA6F7"), // Mauve focus
		Upvote:      lipgloss.Color("#FAB387"), // Peach
		Downvote:    lipgloss.Color("#89B4FA"), // Blue
		Positive:    lipgloss.Color("#A6E3A1"), // Green
		Negative:    lipgloss.Color("#F38BA8"), // Red
		Selection:   lipgloss.Color("#CBA6F7"), // Mauve
	}
}

// ThemeNord returns an arctic, north-bluish clean theme.
func ThemeNord() Theme {
	return Theme{
		ID:          "nord",
		Name:        "Nord",
		Primary:     lipgloss.Color("#88C0D0"), // Frost Cyan (nord8)
		PrimaryDark: lipgloss.Color("#81A1C1"), // Frost Blue (nord9)
		Secondary:   lipgloss.Color("#5E81AC"), // Polar Blue (nord10)
		Accent:      lipgloss.Color("#8FBCBB"), // Frost Teal (nord7)
		Text:        lipgloss.Color("#ECEFF4"), // Snow Storm 3 (nord6)
		TextMuted:   lipgloss.Color("#D8DEE9"), // Snow Storm 1 (nord4)
		TextDim:     lipgloss.Color("#4C566A"), // Polar Night 4 (nord3)
		Background:  lipgloss.Color("#2E3440"), // Polar Night 1 (nord0)
		CardBg:      lipgloss.Color("#3B4252"), // Polar Night 2 (nord1)
		CardBgHover: lipgloss.Color("#434C5E"), // Polar Night 3 (nord2)
		Border:      lipgloss.Color("#4C566A"), // Polar Night 4 (nord3)
		BorderFocus: lipgloss.Color("#88C0D0"), // Frost Cyan
		Upvote:      lipgloss.Color("#D08770"), // Aurora Orange (nord12)
		Downvote:    lipgloss.Color("#81A1C1"), // Frost Blue (nord9)
		Positive:    lipgloss.Color("#A3BE8C"), // Aurora Green (nord14)
		Negative:    lipgloss.Color("#BF616A"), // Aurora Red (nord11)
		Selection:   lipgloss.Color("#88C0D0"), // Frost Cyan
	}
}

// ThemeDracula returns the iconic high-contrast vampire dark theme.
func ThemeDracula() Theme {
	return Theme{
		ID:          "dracula",
		Name:        "Dracula",
		Primary:     lipgloss.Color("#BD93F9"), // Purple
		PrimaryDark: lipgloss.Color("#9872D4"), // Deep Purple
		Secondary:   lipgloss.Color("#6272A4"), // Comment Gray
		Accent:      lipgloss.Color("#8BE9FD"), // Cyan
		Text:        lipgloss.Color("#F8F8F2"), // Foreground White
		TextMuted:   lipgloss.Color("#99A2C2"), // Light Comment
		TextDim:     lipgloss.Color("#6272A4"), // Comment Slate
		Background:  lipgloss.Color("#282A36"), // Canvas Background
		CardBg:      lipgloss.Color("#21222C"), // Deep Dark Background
		CardBgHover: lipgloss.Color("#44475A"), // Current Line
		Border:      lipgloss.Color("#44475A"), // Subtle Border
		BorderFocus: lipgloss.Color("#BD93F9"), // Purple Focus
		Upvote:      lipgloss.Color("#FFB86C"), // Orange
		Downvote:    lipgloss.Color("#8BE9FD"), // Cyan
		Positive:    lipgloss.Color("#50FA7B"), // Green
		Negative:    lipgloss.Color("#FF5555"), // Red
		Selection:   lipgloss.Color("#BD93F9"), // Purple
	}
}

// ThemeGruvboxDark returns the warm retro groove dark theme.
func ThemeGruvboxDark() Theme {
	return Theme{
		ID:          "gruvbox",
		Name:        "Gruvbox Dark",
		Primary:     lipgloss.Color("#FE8019"), // Bright Orange
		PrimaryDark: lipgloss.Color("#D65D0E"), // Dark Orange
		Secondary:   lipgloss.Color("#83A598"), // Bright Blue
		Accent:      lipgloss.Color("#8EC07C"), // Bright Aqua
		Text:        lipgloss.Color("#EBDBB2"), // Light Fore
		TextMuted:   lipgloss.Color("#A89984"), // Gray
		TextDim:     lipgloss.Color("#665C54"), // Dark Gray
		Background:  lipgloss.Color("#1D2021"), // Hard Dark Bg
		CardBg:      lipgloss.Color("#282828"), // Dark Bg
		CardBgHover: lipgloss.Color("#3C3836"), // Bg1
		Border:      lipgloss.Color("#504945"), // Bg2
		BorderFocus: lipgloss.Color("#FE8019"), // Orange Focus
		Upvote:      lipgloss.Color("#FE8019"), // Orange Upvote
		Downvote:    lipgloss.Color("#83A598"), // Blue Downvote
		Positive:    lipgloss.Color("#B8BB26"), // Green
		Negative:    lipgloss.Color("#FB4934"), // Red
		Selection:   lipgloss.Color("#FE8019"), // Orange
	}
}

// ThemeTokyoNight returns the sleek neon-inspired Tokyo Night theme.
func ThemeTokyoNight() Theme {
	return Theme{
		ID:          "tokyonight",
		Name:        "Tokyo Night",
		Primary:     lipgloss.Color("#7AA2F7"), // Tokyo Blue
		PrimaryDark: lipgloss.Color("#3D59A1"), // Deep Blue
		Secondary:   lipgloss.Color("#BB9AF7"), // Magenta / Purple
		Accent:      lipgloss.Color("#7DCFFF"), // Cyan / Ice
		Text:        lipgloss.Color("#C0CAF5"), // Foreground
		TextMuted:   lipgloss.Color("#9AA5CE"), // Subtext
		TextDim:     lipgloss.Color("#565F89"), // Comment Slate
		Background:  lipgloss.Color("#1A1B26"), // Canvas Background
		CardBg:      lipgloss.Color("#16161E"), // Deep Dark Bg
		CardBgHover: lipgloss.Color("#24283B"), // Highlight Bg
		Border:      lipgloss.Color("#292E42"), // Subtle Border
		BorderFocus: lipgloss.Color("#7AA2F7"), // Blue Focus
		Upvote:      lipgloss.Color("#FF9E64"), // Orange
		Downvote:    lipgloss.Color("#7AA2F7"), // Blue
		Positive:    lipgloss.Color("#9ECE6A"), // Green
		Negative:    lipgloss.Color("#F7768E"), // Red
		Selection:   lipgloss.Color("#7AA2F7"), // Tokyo Blue
	}
}

// DefaultTheme returns the production ReadIT dark theme.
func DefaultTheme() Theme {
	return ThemeReadIT()
}

// AvailableThemes returns all supported themes in canonical order.
func AvailableThemes() []Theme {
	return []Theme{
		ThemeReadIT(),
		ThemeCatppuccinMocha(),
		ThemeNord(),
		ThemeDracula(),
		ThemeGruvboxDark(),
		ThemeTokyoNight(),
	}
}

// Themes is an alias for AvailableThemes.
func Themes() []Theme {
	return AvailableThemes()
}

// GetTheme returns the theme matching the provided ID, falling back to DefaultTheme().
func GetTheme(id string) Theme {
	switch strings.ToLower(strings.TrimSpace(id)) {
	case "readit", "default":
		return ThemeReadIT()
	case "catppuccin", "catppuccin-mocha", "mocha":
		return ThemeCatppuccinMocha()
	case "nord":
		return ThemeNord()
	case "dracula":
		return ThemeDracula()
	case "gruvbox", "gruvbox-dark":
		return ThemeGruvboxDark()
	case "tokyonight", "tokyo-night":
		return ThemeTokyoNight()
	default:
		return DefaultTheme()
	}
}

// Global active theme for backwards compatibility.
var currentTheme = DefaultTheme()
