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

// Original glowing Earth palette (shaded block style):
// Luminous cyan/teal continents against deep slate-blue oceans.
var (
	styleEarthHigh = lipgloss.NewStyle().Foreground(lipgloss.Color("#00F0C0"))
	styleEarthMid  = lipgloss.NewStyle().Foreground(lipgloss.Color("#00D1B2"))
	styleEarthLow  = lipgloss.NewStyle().Foreground(lipgloss.Color("#00A896"))
	styleEarthSea  = lipgloss.NewStyle().Foreground(lipgloss.Color("#3A4560"))
)

// Warm solar-golden shine palette that naturally complements Reddit orange (#FF4500)
// providing an authentic, metallic/lacquer gleam across the block logo.
var (
	styleShine0    = lipgloss.NewStyle().Foreground(lipgloss.Color("#FFF176")).Bold(true) // Peak solar gold highlight
	styleShine1    = lipgloss.NewStyle().Foreground(lipgloss.Color("#FFD54F")).Bold(true) // Warm lustrous gold
	styleShine2    = lipgloss.NewStyle().Foreground(lipgloss.Color("#FFA726"))             // Golden honey amber
	styleShine3    = lipgloss.NewStyle().Foreground(lipgloss.Color("#FF7043"))             // Radiant coral-orange
	styleShineBase = lipgloss.NewStyle().Foreground(lipgloss.Color("#FF4500"))             // Base Reddit brand orange
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
// Engineered with a circular profile and equatorial bulge (16 columns x 6 rows)
// and an authentic 23.4-degree axial tilt rotation.
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
		" ▓███░░░███████ ",
		" ▒███░░░░░████▓ ",
		"  ▓▓░░░░░████▒  ",
		"     ▓░░░▓░     ",
	},
	// Frame 2
	{
		"     ████▓█     ",
		"  ▒██░░░░░░███  ",
		" ▒████░░░▓█████ ",
		" ░▒███▒░░▒░████ ",
		"  ▒█░░░░░░▓███  ",
		"     ▓░░░▒▓     ",
	},
	// Frame 3
	{
		"     ████▓█     ",
		"  ▓███░░░░░▒██  ",
		" ▒░████░░░░████ ",
		" ░░▓████░░░████ ",
		"  ░██░░░░░░▒██  ",
		"     ▓░░░░▓     ",
	},
	// Frame 4
	{
		"     █████▓     ",
		"  ▒░███░░░░░██  ",
		" ▒░░▓██▓░░░░███ ",
		" ▒░░▒████▒░▓███ ",
		"  ░░██▓▓░░░░▒█  ",
		"     ▓░░░░░     ",
	},
	// Frame 5
	{
		"     ▓█████     ",
		"  ▒░▒███░░▒░▓▓  ",
		" ▓░░░▒██▓░░░░██ ",
		" ░░░░▓████▓░▓██ ",
		"  ░░░████░░░░█  ",
		"     █░░░░░     ",
	},
	// Frame 6
	{
		"     ░█████     ",
		"  ▒░░▓████▓░░▓  ",
		" ▓░░░░▒██▓░░░░█ ",
		" ░░░░░▓████░░▓█ ",
		"  ░░░░█████░░▓  ",
		"     █▓░░░░     ",
	},
	// Frame 7
	{
		"     ░░████     ",
		"  ▓▒░░▓████▓░░  ",
		" █░░░░░▒██▓░░░▓ ",
		" ░░░░░░░████░░█ ",
		"  ░░░░░░████▒▒  ",
		"     ▓█▓▒░░     ",
	},
	// Frame 8
	{
		"     ░░▓███     ",
		"  ▒░░░░▓████▓░  ",
		" ▓░░░░░░▓█▓▓░░▒ ",
		" ▒░░░░░░░▓██▓░█ ",
		"  ░░░░░░░████▒  ",
		"     ▓░███░     ",
	},
	// Frame 9
	{
		"     ░░▓███     ",
		"  ▓░░░░░█████▓  ",
		" █░░░░░░░▓██▓░░ ",
		" ▒░░░░░░░░▓██▓▒ ",
		"  ░░░░░░░░████  ",
		"     ▓░░▓█▓     ",
	},
	// Frame 10
	{
		"     ░░▓███     ",
		"  ▒░░▒░░░████▓  ",
		" ▓░░░░░░░░███▒░ ",
		" ▓░░░░░░░░░▒██▒ ",
		"  ░░░░░░░░░███  ",
		"     ▓░░░██     ",
	},
	// Frame 11
	{
		"     ░▒▒███     ",
		"  ▓░░░░░░░████  ",
		" █░░░░░░░░░███░ ",
		" █░░░░░░░░░░██▓ ",
		"  ▒░░░░░░░░░██  ",
		"     ▓░░░▒█     ",
	},
	// Frame 12
	{
		"     ▓▓▓▓██     ",
		"  ▓░░░░░░░░███  ",
		" █▒░░░░░░░░▓██▓ ",
		" █▓░░░░░░░░░░██ ",
		"  ▒▓░░░░░░░░░█  ",
		"     ▓░░░░▒     ",
	},
	// Frame 13
	{
		"     ████▓█     ",
		"  █░░░░░░░░▒██  ",
		" ██▒░░░░░░░░▒██ ",
		" ██░░░░░░░░░░░█ ",
		"  ░▒▓░░░░░░░░▓  ",
		"     ▓░░░░▒     ",
	},
	// Frame 14
	{
		"     ██▓█▓█     ",
		"  █▓░░░░░░░░██  ",
		" ███░░░░░░▒░░██ ",
		" ██▓▒░░░░░░░░░█ ",
		"  ░░▒▒░░░░░░░▓  ",
		"     ▒░░░░░     ",
	},
	// Frame 15
	{
		"     ██████     ",
		"  ██░░░░░░░░▓█  ",
		" ███▓░░░░░░░░▓█ ",
		" ▒███░░▒░░░░░░▒ ",
		"  ░▓░▓▓░░░░░░░  ",
		"     ▒░░░░░     ",
	},
	// Frame 16
	{
		"     ██████     ",
		"  ███░░░░░░░▓▓  ",
		" ▒██▓▒░░░░░░░░▓ ",
		" ░███▓░░░░░░░░▒ ",
		"  ░░█░░░░░░░░░  ",
		"     ▓░░░░░     ",
	},
	// Frame 17
	{
		"     ██████     ",
		"  ████▒░░░░░▓▓  ",
		" ▓▓██▓▒░░░░░░░▒ ",
		" ░░████▓░░░░░░▒ ",
		"  ░░▓█▓▒░▒░░░▒  ",
		"     ▓▒▒░░░     ",
	},
	// Frame 18
	{
		"     ██████     ",
		"  ▒████▒█▓░░▓░  ",
		" ▓░▒██▓░░░░░░░░ ",
		" ░░░█████░░░░░▒ ",
		"  ░░▓▓██░░░░░▒  ",
		"     ▓░▒▓░░     ",
	},
	// Frame 19
	{
		"     ██████     ",
		"  █▓███████░▒▒  ",
		" ▒░░▓███░░░░░░░ ",
		" ░░░░▓██▓▓▒░░░▒ ",
		"  ░░░▓███░▒░░▓  ",
		"     ▓░░▒▒░     ",
	},
	// Frame 20
	{
		"     ██████     ",
		"  █▓▓███████▓░  ",
		" █░░░▒█▓▓▓░░░░░ ",
		" ▒░░░░▓▒▓▓▓░░░▒ ",
		"  ░░░░████▒░░▓  ",
		"     ▓░░▓▒▓     ",
	},
	// Frame 21
	{
		"     ██████     ",
		"  ██▓████████░  ",
		" █░░░▓░██░▓░░░░ ",
		" ▓░░░░░▓██▓▒░░▒ ",
		"  ░░░░░████▓▒▒  ",
		"     ▓░░░█▒     ",
	},
	// Frame 22
	{
		"     ██████     ",
		"  ███████████▓  ",
		" ██░░░▓░▓██▓░░░ ",
		" █▓░░░░░███▓░░▒ ",
		"  ░░░░░░▒████▓  ",
		"     ▓░░░██     ",
	},
	// Frame 23
	{
		"     ██████     ",
		"  ████████████  ",
		" ███▓░░▓█▒███▓░ ",
		" ▓██░░░░░███▒░▓ ",
		"  ░░░░░░░▒▓███  ",
		"     ▓░░░▓█     ",
	},
	// Frame 24
	{
		"     ██████     ",
		"  ████████████  ",
		" ▒████░░▒█████▒ ",
		" ▒███░░░░░▓█▓▓▓ ",
		"  ░░░░░░░░░▓██  ",
		"     ▓░░░▒█     ",
	},
	// Frame 25
	{
		"     ██████     ",
		"  ████████████  ",
		" ▒░████▓▓░█████ ",
		" ░▒███▓░░░░▒█▓▒ ",
		"  ░░░░░░░░░▒▓█  ",
		"     ▓░░░░▒     ",
	},
	// Frame 26
	{
		"     ▒█████     ",
		"  ▒███████████  ",
		" █░░█████▓▓████ ",
		" ░░▓███▓░░░▓░██ ",
		"  ░░▓░░░░░░░▒█  ",
		"     ▓░░░░░     ",
	},
	// Frame 27
	{
		"     ░▒████     ",
		"  ▒░██████████  ",
		" █░░░██████████ ",
		" ▒░░░████▒░░▓▒█ ",
		"  ░░░▓▒▒▓░░░░█  ",
		"     ▓░░░░░     ",
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
func renderEarthFrame(tick int) string {
	frameIdx := tick % len(earthFrames)
	frame := earthFrames[frameIdx]

	var b strings.Builder
	for r, row := range frame {
		for _, ch := range row {
			switch ch {
			case '█':
				b.WriteString(styleEarthHigh.Render("█"))
			case '▓':
				b.WriteString(styleEarthMid.Render("▓"))
			case '▒':
				b.WriteString(styleEarthLow.Render("▒"))
			case '░':
				b.WriteString(styleEarthSea.Render("░"))
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
func renderShiningLogo(tick int) string {
	// A full shine sweep takes 68 ticks (approx 6.8 seconds).
	// Ticks 0..58: light beam sweeps diagonally across the logo.
	// Ticks 59..67: brief resting pause in pure brand orange before next sweep.
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

			// Diagonal wave projection from top-left (col 0, row 0) to bottom-right (col 43, row 5).
			// Row is scaled by 2.4 to match standard terminal character aspect-ratio.
			waveCoord := float64(col) + float64(r)*2.4
			dist := math.Abs(waveCoord - sweepPos)

			chStr := string(ch)
			switch {
			case dist < 1.0:
				b.WriteString(styleShine0.Render(chStr))
			case dist < 2.2:
				b.WriteString(styleShine1.Render(chStr))
			case dist < 3.5:
				b.WriteString(styleShine2.Render(chStr))
			case dist < 4.8:
				b.WriteString(styleShine3.Render(chStr))
			default:
				b.WriteString(styleShineBase.Render(chStr))
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
func renderHeroBanner(tick int, contentWidth int) string {
	shiningLogo := renderShiningLogo(tick)
	if contentWidth >= 68 {
		earth := renderEarthFrame(tick)
		return lipgloss.JoinHorizontal(lipgloss.Center, earth, "   ", shiningLogo)
	}
	return shiningLogo
}
