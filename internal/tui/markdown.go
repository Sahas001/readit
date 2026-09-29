package tui

import (
	"strings"

	"github.com/charmbracelet/glamour"
	"github.com/charmbracelet/glamour/ansi"
	"github.com/charmbracelet/glamour/styles"
	"github.com/charmbracelet/lipgloss"
	"github.com/muesli/termenv"
	"github.com/sahas/readit/internal/sanitize"
)

// markdownStyleForTheme builds a Glamour ANSI style configuration adapted to the given Theme.
func markdownStyleForTheme(t Theme) ansi.StyleConfig {
	var cfg ansi.StyleConfig
	switch t.ID {
	case "dracula":
		cfg = styles.DraculaStyleConfig
	case "tokyonight":
		cfg = styles.TokyoNightStyleConfig
	default:
		cfg = styles.DarkStyleConfig
	}

	primaryHex := string(t.Primary)
	secondaryHex := string(t.Secondary)
	accentHex := string(t.Accent)
	textHex := string(t.Text)

	cfg.H1.StylePrimitive.Color = &primaryHex
	cfg.H2.StylePrimitive.Color = &secondaryHex
	cfg.H3.StylePrimitive.Color = &accentHex
	cfg.Link.Color = &accentHex
	cfg.LinkText.Color = &primaryHex
	cfg.Document.StylePrimitive.Color = &textHex

	return cfg
}

// renderMarkdown renders user markdown text into ANSI-styled terminal output
// using Glamour with a transparent background and theme-matched styling.
// It ensures strict sanitization of raw terminal escapes before rendering,
// preserves newlines, and gracefully falls back to plain wrapped text on error.
func renderMarkdown(content string, width int, t ...Theme) string {
	clean := sanitize.Text(content)
	if strings.TrimSpace(clean) == "" {
		return ""
	}
	if width <= 0 {
		width = 80
	}

	theme := DefaultTheme()
	if len(t) > 0 {
		theme = t[0]
	}

	cfg := markdownStyleForTheme(theme)
	zero := uint(0)
	cfg.Document.Margin = &zero
	cfg.Document.StylePrimitive.BackgroundColor = nil
	cfg.CodeBlock.Margin = &zero

	r, err := glamour.NewTermRenderer(
		glamour.WithStyles(cfg),
		glamour.WithWordWrap(width),
		glamour.WithEmoji(),
		glamour.WithPreservedNewLines(),
		glamour.WithColorProfile(termenv.TrueColor),
	)
	if err != nil {
		return lipgloss.NewStyle().Width(width).Render(clean)
	}

	rendered, err := r.Render(clean)
	if err != nil {
		return lipgloss.NewStyle().Width(width).Render(clean)
	}

	return strings.Trim(rendered, "\n")
}

// renderMarkdown on Model renders markdown matching m.theme.
func (m *Model) renderMarkdown(content string, width int) string {
	m.ensureStyles()
	return renderMarkdown(content, width, m.theme)
}
