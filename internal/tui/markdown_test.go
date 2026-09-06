package tui

import (
	"strings"
	"testing"

	"github.com/charmbracelet/lipgloss"
	"github.com/sahas/readit/internal/sanitize"
)

func TestRenderMarkdownEmpty(t *testing.T) {
	if got := renderMarkdown("", 80); got != "" {
		t.Errorf("expected empty string, got %q", got)
	}
	if got := renderMarkdown("   \n\t  ", 80); got != "" {
		t.Errorf("expected empty string for whitespace, got %q", got)
	}
}

func TestRenderMarkdownHeadingsAndFormatting(t *testing.T) {
	input := "# Heading One\n\nThis is **bold** text and *italic* text.\n\n- item alpha\n- item beta"
	out := renderMarkdown(input, 70)

	if out == "" {
		t.Fatalf("expected non-empty output")
	}

	clean := sanitize.Text(out)

	// Heading One should be rendered without leading "# "
	if strings.Contains(clean, "# Heading One") {
		t.Errorf("markdown header was not parsed; still contains '# Heading One': %q", clean)
	}
	if !strings.Contains(clean, "Heading One") {
		t.Errorf("missing heading text in rendered output: %q", clean)
	}

	// Bold text should be rendered without "**"
	if strings.Contains(clean, "**bold**") {
		t.Errorf("bold markdown syntax was not parsed: %q", clean)
	}
	if !strings.Contains(clean, "bold") {
		t.Errorf("missing bold word in rendered output: %q", clean)
	}

	// List items should render bullet glyphs (e.g. •)
	if !strings.Contains(clean, "item alpha") || !strings.Contains(clean, "item beta") {
		t.Errorf("missing list items in rendered output: %q", clean)
	}
}

func TestRenderMarkdownSanitizesEscapes(t *testing.T) {
	// Attacker tries to inject an OSC terminal escape sequence
	maliciousInput := "# Safe Heading\n\x1b]50;SetProfile=Evil\x07Normal text **styled**"
	out := renderMarkdown(maliciousInput, 80)

	if strings.Contains(out, "SetProfile=Evil") {
		t.Errorf("malicious OSC escape payload leaked into rendered output: %q", out)
	}
	if !strings.Contains(out, "Normal text") {
		t.Errorf("safe content missing: %q", out)
	}
}

func TestRenderMarkdownWordWrap(t *testing.T) {
	longText := "This is a very long sentence designed to verify that Glamour markdown word wrapping respects the given column boundary without exceeding the width."
	targetWidth := 40
	out := renderMarkdown(longText, targetWidth)

	lines := strings.Split(out, "\n")
	for i, line := range lines {
		w := lipgloss.Width(line)
		if w > targetWidth {
			t.Errorf("line %d width %d exceeds target %d: %q", i, w, targetWidth, line)
		}
	}
}
