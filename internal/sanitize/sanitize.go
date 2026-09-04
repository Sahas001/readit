// Package sanitize provides security sanitizers to prevent terminal escape injections,
// prompt spoofing, and control sequence attacks over SSH PTY sessions.
package sanitize

import (
	"regexp"
	"strings"
	"unicode"
)

// ValidHandleRegex enforces alphanumeric handles with underscores and hyphens (3 to 20 chars).
var ValidHandleRegex = regexp.MustCompile(`^[a-zA-Z0-9_-]{3,20}$`)

// ansiEscapeRegex matches ANSI/VT100 escape codes:
// 1. OSC sequences: ESC ] ... (ST | BEL)
// 2. CSI sequences: ESC [ ... [@-~]
// 3. DCS / APC / PM sequences: ESC P ... ST
// 4. 2-character Fe escape sequences: ESC followed by single char
var ansiEscapeRegex = regexp.MustCompile(`\x1b(?:\][^\x07\x1b]*(?:\x07|\x1b\\)|\[[0-?]*[ -/]*[@-~]|P[^\x1b]*\x1b\\|[@-Z\\-_])`)

// ValidateHandle checks if a handle adheres to security rules:
// - Length between 3 and 20 characters
// - Strictly alphanumeric, underscores, or hyphens
// - Cannot start or end with a hyphen or underscore
func ValidateHandle(handle string) error {
	trimmed := strings.TrimSpace(handle)
	if len(trimmed) < 3 || len(trimmed) > 20 {
		return &ValidationError{Msg: "handle must be between 3 and 20 characters"}
	}
	if !ValidHandleRegex.MatchString(trimmed) {
		return &ValidationError{Msg: "handle may only contain letters, numbers, underscores, and hyphens"}
	}
	if trimmed[0] == '-' || trimmed[0] == '_' || trimmed[len(trimmed)-1] == '-' || trimmed[len(trimmed)-1] == '_' {
		return &ValidationError{Msg: "handle cannot start or end with an underscore or hyphen"}
	}
	return nil
}

// Text strips ANSI escape codes and unprintable C0/C1 control characters from user text.
// Allowed whitespace: standard space, tab (\t), newline (\n), and carriage return (\r).
// All other control characters (e.g. BEL \x07, ESC \x1b, BS \x08) are stripped.
func Text(s string) string {
	// First pass: remove full multi-byte ANSI/OSC/CSI escape sequences
	cleaned := ansiEscapeRegex.ReplaceAllString(s, "")

	// Second pass: remove any lingering C0/C1 non-printable control characters
	var b strings.Builder
	b.Grow(len(cleaned))
	for _, r := range cleaned {
		if r == '\n' || r == '\r' || r == '\t' {
			b.WriteRune(r)
		} else if !unicode.IsControl(r) {
			b.WriteRune(r)
		}
	}
	return b.String()
}

// SingleLine sanitizes text and normalizes newlines/tabs into standard spaces.
// Perfect for post titles, usernames, and board slugs that must remain on one line.
func SingleLine(s string) string {
	cleaned := Text(s)
	// Replace all newlines, carriage returns, and tabs with single spaces
	fields := strings.Fields(cleaned)
	return strings.Join(fields, " ")
}

// ValidationError represents an invalid input error.
type ValidationError struct {
	Msg string
}

func (e *ValidationError) Error() string {
	return e.Msg
}
