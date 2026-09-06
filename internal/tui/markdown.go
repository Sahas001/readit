package tui

import (
	"strings"

	"github.com/charmbracelet/glamour"
	"github.com/charmbracelet/glamour/styles"
	"github.com/charmbracelet/lipgloss"
	"github.com/sahas/readit/internal/sanitize"
)

// renderMarkdown renders user markdown text into ANSI-styled terminal output
// using Glamour with a transparent background and ReadIT dark palette styling.
// It ensures strict sanitization of raw terminal escapes before rendering,
// preserves newlines, and gracefully falls back to plain wrapped text on error.
func renderMarkdown(content string, width int) string {
	clean := sanitize.Text(content)
	if strings.TrimSpace(clean) == "" {
		return ""
	}
	if width <= 0 {
		width = 80
	}

	cfg := styles.DarkStyleConfig
	zero := uint(0)
	cfg.Document.Margin = &zero
	cfg.Document.StylePrimitive.BackgroundColor = nil
	cfg.CodeBlock.Margin = &zero

	r, err := glamour.NewTermRenderer(
		glamour.WithStyles(cfg),
		glamour.WithWordWrap(width),
		glamour.WithEmoji(),
		glamour.WithPreservedNewLines(),
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
