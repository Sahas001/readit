package tui

import (
	"math"
	"strings"
	"time"

	"github.com/charmbracelet/lipgloss"
)

// animationInterval controls the frame tick for both the spinning Earth
// and the diagonal logo shine sweep.
const animationInterval = 100 * time.Millisecond

// EarthStyles contains styling for earth rendering frames.
type EarthStyles struct {
	High lipgloss.Style
	Mid  lipgloss.Style
	Low  lipgloss.Style
	Sea  lipgloss.Style
}

// LogoShineStyles contains styling for the shining logo sweep.
type LogoShineStyles struct {
	Shine0 lipgloss.Style
	Shine1 lipgloss.Style
	Shine2 lipgloss.Style
	Shine3 lipgloss.Style
	Base   lipgloss.Style
}

func newEarthStyles(_ ...Theme) EarthStyles {
	return EarthStyles{
		High: styleEarthHigh,
		Mid:  styleEarthMid,
		Low:  styleEarthLow,
		Sea:  styleEarthSea,
	}
}

func newLogoShineStyles(t ...Theme) LogoShineStyles {
	theme := DefaultTheme()
	if len(t) > 0 {
		theme = t[0]
	}

	switch theme.ID {
	case "catppuccin":
		return LogoShineStyles{
			Shine0: lipgloss.NewStyle().Foreground(lipgloss.Color("#FFFFFF")).Bold(true), // Pure crystalline highlight
			Shine1: lipgloss.NewStyle().Foreground(lipgloss.Color("#F5E0DC")).Bold(true), // Warm rosewater
			Shine2: lipgloss.NewStyle().Foreground(lipgloss.Color("#F5C2E7")),             // Radiant pink
			Shine3: lipgloss.NewStyle().Foreground(lipgloss.Color("#DDB6F2")),             // Soft lavender
			Base:   lipgloss.NewStyle().Foreground(theme.Primary),                         // Mauve base (#CBA6F7)
		}
	case "nord":
		return LogoShineStyles{
			Shine0: lipgloss.NewStyle().Foreground(lipgloss.Color("#ECEFF4")).Bold(true), // Snow storm white highlight
			Shine1: lipgloss.NewStyle().Foreground(lipgloss.Color("#E5E9F0")).Bold(true), // Diamond ice glow
			Shine2: lipgloss.NewStyle().Foreground(lipgloss.Color("#8FBCBB")),             // Aurora seafoam teal
			Shine3: lipgloss.NewStyle().Foreground(lipgloss.Color("#81A1C1")),             // Frost blue
			Base:   lipgloss.NewStyle().Foreground(theme.Primary),                         // Frost cyan base (#88C0D0)
		}
	case "dracula":
		return LogoShineStyles{
			Shine0: lipgloss.NewStyle().Foreground(lipgloss.Color("#FFFFFF")).Bold(true), // Neon white highlight
			Shine1: lipgloss.NewStyle().Foreground(lipgloss.Color("#FF79C6")).Bold(true), // Electric Dracula pink
			Shine2: lipgloss.NewStyle().Foreground(lipgloss.Color("#FF92D0")),             // Luminous neon pink
			Shine3: lipgloss.NewStyle().Foreground(lipgloss.Color("#D1A8FF")),             // Electric lilac
			Base:   lipgloss.NewStyle().Foreground(theme.Primary),                         // Dracula purple base (#BD93F9)
		}
	case "gruvbox":
		return LogoShineStyles{
			Shine0: lipgloss.NewStyle().Foreground(lipgloss.Color("#FBF1C7")).Bold(true), // Ivory highlight
			Shine1: lipgloss.NewStyle().Foreground(lipgloss.Color("#FABD2F")).Bold(true), // Gruvbox gold yellow
			Shine2: lipgloss.NewStyle().Foreground(lipgloss.Color("#F2A42B")),             // Warm brass amber
			Shine3: lipgloss.NewStyle().Foreground(lipgloss.Color("#FF8700")),             // Radiant orange
			Base:   lipgloss.NewStyle().Foreground(theme.Primary),                         // Gruvbox orange base (#FE8019)
		}
	case "tokyonight":
		return LogoShineStyles{
			Shine0: lipgloss.NewStyle().Foreground(lipgloss.Color("#FFFFFF")).Bold(true), // Pure white glare
			Shine1: lipgloss.NewStyle().Foreground(lipgloss.Color("#7DCFFF")).Bold(true), // Cyberpunk cyan
			Shine2: lipgloss.NewStyle().Foreground(lipgloss.Color("#89DDFF")),             // Ice electric blue
			Shine3: lipgloss.NewStyle().Foreground(lipgloss.Color("#7AA2F7")),             // Neon blue
			Base:   lipgloss.NewStyle().Foreground(theme.Primary),                         // Tokyo blue base (#7AA2F7)
		}
	default: // "readit"
		return LogoShineStyles{
			Shine0: styleShine0,                                   // Solar gold highlight (#FFF176)
			Shine1: styleShine1,                                   // Warm lustrous gold (#FFD54F)
			Shine2: styleShine2,                                   // Golden honey amber (#FFA726)
			Shine3: styleShine3,                                   // Radiant coral-orange (#FF7043)
			Base:   lipgloss.NewStyle().Foreground(theme.Primary), // Base Reddit orange (#FF4500)
		}
	}
}

// Package default animation styles for fallback and backwards compatibility.
var (
	styleEarthHigh = lipgloss.NewStyle().Foreground(lipgloss.Color("#00F0C0"))
	styleEarthMid  = lipgloss.NewStyle().Foreground(lipgloss.Color("#00D1B2"))
	styleEarthLow  = lipgloss.NewStyle().Foreground(lipgloss.Color("#00A896"))
	styleEarthSea  = lipgloss.NewStyle().Foreground(lipgloss.Color("#3A4560"))

	styleShine0    = lipgloss.NewStyle().Foreground(lipgloss.Color("#FFF176")).Bold(true)
	styleShine1    = lipgloss.NewStyle().Foreground(lipgloss.Color("#FFD54F")).Bold(true)
	styleShine2    = lipgloss.NewStyle().Foreground(lipgloss.Color("#FFA726"))
	styleShine3    = lipgloss.NewStyle().Foreground(lipgloss.Color("#FF7043"))
	styleShineBase = lipgloss.NewStyle().Foreground(lipgloss.Color("#FF4500"))
)

// The project block ASCII logo lines (44 columns wide, 6 lines tall).
var logoLines = [6]string{
	"██████╗ ███████╗ █████╗ ██████╗ ██╗████████╗",
	"██╔══██╗██╔════╝██╔══██╗██╔══██╗██║╚══██╔══╝",
	"██████╔╝█████╗  ███████║██║  ██║██║   ██║   ",
	"██╔══██╗██╔══╝  ██╔══██║██║  ██║██║   ██║   ",
	"██║  ██║███████╗██║  ██║██████╔╝██║   ██║   ",
	"╚═╝  ╚═╝╚══════╝╚═╝  ╚═╝╚═════╝ ╚═╝   ╚═╝   ",
}

// 30 frames of spinning Earth derived directly from https://ascii.co.uk/art/earth
var earthFrames = [30][6]string{
	// Frame 0
	{
		"     ▒▓▒▓▓█     ",
		"  █▓░░░▓██████  ",
		" ███▓░▒████████ ",
		" ██▓░░░░▒████▒█ ",
		"  ▓░░░░░████░▒  ",
		"     ▓░░░░░     ",
	},
	// Frame 1
	{
		"     ██▓▓▓▓     ",
		"  ██▒░░░░▒████  ",
		" ███▓░░░▒██████ ",
		" ███▓░░░░▒█████ ",
		"  ██▓░░░░░███▒  ",
		"     ▒░░░░░     ",
	},
	// Frame 2
	{
		"     ███▓▓▓     ",
		"  ███▒░░░░▒███  ",
		" ████▓░░░░▒████ ",
		" ████▓░░░░░▒███ ",
		"  ███▓░░░░░██▒  ",
		"     ▒░░░░░     ",
	},
	// Frame 3
	{
		"     ████▓▓     ",
		"  ████▒░░░░▒██  ",
		" █████▓░░░░░▒██ ",
		" █████▓░░░░░░▒█ ",
		"  ████▓░░░░░█▒  ",
		"     ▒░░░░░     ",
	},
	// Frame 4
	{
		"     █████▓     ",
		"  █████▒░░░░▒█  ",
		" ██████▓░░░░░▒█ ",
		" ██████▓░░░░░░▒ ",
		"  █████▓░░░░░▒  ",
		"     ▒░░░░░     ",
	},
	// Frame 5
	{
		"     ▓█████     ",
		"  ░█████▒░░░░▒  ",
		" ▒██████▓░░░░░▒ ",
		" ▓██████▓░░░░░░ ",
		"  ██████▓░░░░░  ",
		"     ▒░░░░░     ",
	},
	// Frame 6
	{
		"     ▓▓████     ",
		"  ░░█████▒░░░░  ",
		" ░▒██████▓░░░░░ ",
		" ▒▓██████▓░░░░░ ",
		"  ███████▓░░░░  ",
		"     ▒░░░░░     ",
	},
	// Frame 7
	{
		"     ▓▓▓███     ",
		"  ░░░█████▒░░░  ",
		" ░░▒██████▓░░░░ ",
		" ░▒▓██████▓░░░░ ",
		"  ▒███████▓░░░  ",
		"     ▒░░░░░     ",
	},
	// Frame 8
	{
		"     ▓▓▓▓██     ",
		"  ░░░░█████▒░░  ",
		" ░░░▒██████▓░░░ ",
		" ░░▒▓██████▓░░░ ",
		"  ░▒███████▓░░  ",
		"     ▒░░░░░     ",
	},
	// Frame 9
	{
		"     ▓▓▓▓▓█     ",
		"  ░░░░░█████▒░  ",
		" ░░░░▒██████▓░░ ",
		" ░░░▒▓██████▓░░ ",
		"  ░░▒███████▓░  ",
		"     ▒░░░░░     ",
	},
	// Frame 10
	{
		"     █▓▓▓▓▓     ",
		"  ░░░░░░█████▒  ",
		" ░░░░░▒██████▓░ ",
		" ░░░░▒▓██████▓░ ",
		"  ░░░▒███████▓  ",
		"     ▒░░░░░     ",
	},
	// Frame 11
	{
		"     ██▓▓▓▓     ",
		"  ▒░░░░░░█████  ",
		" ░░░░░░▒██████▓ ",
		" ░░░░░▒▓██████▓ ",
		"  ░░░░▒███████  ",
		"     ▒░░░░░     ",
	},
	// Frame 12
	{
		"     ███▓▓▓     ",
		"  ▓▒░░░░░░████  ",
		" ▒░░░░░░▒██████ ",
		" ░░░░░░▒▓██████ ",
		"  ░░░░░▒██████  ",
		"     ▒░░░░░     ",
	},
	// Frame 13
	{
		"     ████▓▓     ",
		"  █▓▒░░░░░░███  ",
		" ▓▒░░░░░░▒█████ ",
		" ▒░░░░░░▒▓█████ ",
		"  ░░░░░░▒█████  ",
		"     ▒░░░░░     ",
	},
	// Frame 14
	{
		"     █████▓     ",
		"  ██▓▒░░░░░░██  ",
		" █▓▒░░░░░░▒████ ",
		" ▓▒░░░░░░▒▓████ ",
		"  ▒░░░░░░▒████  ",
		"     ▒░░░░░     ",
	},
	// Frame 15
	{
		"     ██████     ",
		"  ███▓▒░░░░░░█  ",
		" ██▓▒░░░░░░▒███ ",
		" █▓▒░░░░░░▒▓███ ",
		"  ▓▒░░░░░░▒███  ",
		"     ▒░░░░░     ",
	},
	// Frame 16
	{
		"     ▓█████     ",
		"  ████▓▒░░░░░░  ",
		" ███▓▒░░░░░░▒██ ",
		" ██▓▒░░░░░░▒▓██ ",
		"  █▓▒░░░░░░▒██  ",
		"     ▒░░░░░     ",
	},
	// Frame 17
	{
		"     ▓▓████     ",
		"  ░████▓▒░░░░░  ",
		" ████▓▒░░░░░░▒█ ",
		" ███▓▒░░░░░░▒▓█ ",
		"  ██▓▒░░░░░░▒█  ",
		"     ▒░░░░░     ",
	},
	// Frame 18
	{
		"     ▒▓▓███     ",
		"  ░░████▓▒░░░░  ",
		" █▒███▓▒░░░░░░▒ ",
		" ████▓▒░░░░░░▒▓ ",
		"  ███▓▒░░░░░░▒  ",
		"     ▒░░░░░     ",
	},
	// Frame 19
	{
		"     ░▒▓▓██     ",
		"  █░░████▓▒░░░  ",
		" ██▒███▓▒░░░░░░ ",
		" █████▓▒░░░░░░▒ ",
		"  ████▓▒░░░░░░  ",
		"     ▒░░░░░     ",
	},
	// Frame 20
	{
		"     ░░▒▓▓█     ",
		"  ██░░████▓▒░░  ",
		" ███▒███▓▒░░░░░ ",
		" █▓███▓▒░░░░░░▒ ",
		"  ████▓▒░░░░░░  ",
		"     ▒░░░░░     ",
	},
	// Frame 21
	{
		"     ░░░▒▓▓     ",
		"  ███░░████▓▒░  ",
		" ████▒███▓▒░░░░ ",
		" ▓▒███▓▒░░░░░░▒ ",
		"  █▓██▓▒░░░░░░  ",
		"     ▒░░░░░     ",
	},
	// Frame 22
	{
		"     █░░░▒▓     ",
		"  ████░░████▓▒  ",
		" █▓███▒███▓▒░░░ ",
		" ▒░▒██▓▒░░░░░░▒ ",
		"  ▒▓██▓▒░░░░░░  ",
		"     ▒░░░░░     ",
	},
	// Frame 23
	{
		"     ██░░░▒     ",
		"  █████░░████▓  ",
		" ▒▓████▒███▓▒░░ ",
		" ░░░▒██▓▒░░░░░░ ",
		"  ░▒▓██▓▒░░░░░  ",
		"     ▒░░░░░     ",
	},
	// Frame 24
	{
		"     ███░░░     ",
		"  ██████░░████  ",
		" ░▒▓████▒███▓▒░ ",
		" ░░░░▒██▓▒░░░░░ ",
		"  ░░▒▓██▓▒░░░░  ",
		"     ▒░░░░░     ",
	},
	// Frame 25
	{
		"     ▓███░░     ",
		"  ░██████░░███  ",
		" ░░▒▓████▒███▓▒ ",
		" ░░░░░▒██▓▒░░░░ ",
		"  ░░░▒▓██▓▒░░░  ",
		"     ▒░░░░░     ",
	},
	// Frame 26
	{
		"     ▓▓███░     ",
		"  ░░██████░░██  ",
		" █░░▒▓████▒███▓ ",
		" ░░░░░░▒██▓▒░░░ ",
		"  ░░░░▒▓██▓▒░░  ",
		"     ▒░░░░░     ",
	},
	// Frame 27
	{
		"     ▒▓▓███     ",
		"  █░░░█████░░█  ",
		" ██░░▒▓████▒███ ",
		" █░░░░░░▒██▓▒░░ ",
		"  ░░░░░▒▓██▓▒░  ",
		"     ▒░░░░░     ",
	},
	// Frame 28
	{
		"     ░▒▓███     ",
		"  █░░█████████  ",
		" █▓░░██████████ ",
		" ▓░░░░████▓░░▓█ ",
		"  ░░░░███▓░░░▓  ",
		"     ▓░░░░░     ",
	},
	// Frame 29
	{
		"     ▒▓▒▓██     ",
		"  █░░░▓███████  ",
		" ██▓░▒█████████ ",
		" █▓░░░░▓████▒▒█ ",
		"  ▒░░░░████░░▒  ",
		"     ▓░░░░░     ",
	},
}

// renderEarthFrame renders the styled spinning Earth at the specified tick.
func renderEarthFrame(tick int, t ...Theme) string {
	theme := DefaultTheme()
	if len(t) > 0 {
		theme = t[0]
	}
	es := newEarthStyles(theme)

	frameIdx := tick % len(earthFrames)
	frame := earthFrames[frameIdx]

	var b strings.Builder
	for r, row := range frame {
		for _, ch := range row {
			switch ch {
			case '█':
				b.WriteString(es.High.Render("█"))
			case '▓':
				b.WriteString(es.Mid.Render("▓"))
			case '▒':
				b.WriteString(es.Low.Render("▒"))
			case '░':
				b.WriteString(es.Sea.Render("░"))
			default:
				b.WriteRune(' ')
			}
		}
		if r < len(frame)-1 {
			b.WriteByte('\n')
		}
	}
	return b.String()
}

// renderShiningLogo renders the ReadIT ASCII logo with an animated specular
// highlight beam sweeping diagonally from TOP-LEFT to BOTTOM-RIGHT.
func renderShiningLogo(tick int, t ...Theme) string {
	theme := DefaultTheme()
	if len(t) > 0 {
		theme = t[0]
	}
	ls := newLogoShineStyles(theme)

	const cycleLength = 68
	sweepPos := float64(tick % cycleLength)

	var b strings.Builder
	for r, line := range logoLines {
		runes := []rune(line)
		for col, ch := range runes {
			if ch == ' ' {
				b.WriteRune(' ')
				continue
			}

			waveCoord := float64(col) + float64(r)*2.4
			dist := math.Abs(waveCoord - sweepPos)

			chStr := string(ch)
			switch {
			case dist < 1.0:
				b.WriteString(ls.Shine0.Render(chStr))
			case dist < 2.2:
				b.WriteString(ls.Shine1.Render(chStr))
			case dist < 3.5:
				b.WriteString(ls.Shine2.Render(chStr))
			case dist < 4.8:
				b.WriteString(ls.Shine3.Render(chStr))
			default:
				b.WriteString(ls.Base.Render(chStr))
			}
		}
		if r < len(logoLines)-1 {
			b.WriteByte('\n')
		}
	}
	return b.String()
}

// renderHeroBanner composites the circular spinning Earth animation and the
// diagonally shining ReadIT logo side-by-side when screen width permits.
func renderHeroBanner(tick int, contentWidth int, t ...Theme) string {
	theme := DefaultTheme()
	if len(t) > 0 {
		theme = t[0]
	}
	shiningLogo := renderShiningLogo(tick, theme)
	if contentWidth >= 68 {
		earth := renderEarthFrame(tick, theme)
		return lipgloss.JoinHorizontal(lipgloss.Center, earth, "   ", shiningLogo)
	}
	return shiningLogo
}

// renderHeroBanner on Model renders the hero banner using the Model's active theme.
func (m *Model) renderHeroBanner(contentWidth int) string {
	m.ensureStyles()
	return renderHeroBanner(m.animTick, contentWidth, m.theme)
}
