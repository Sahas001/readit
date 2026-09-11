package sanitize_test

import (
	"testing"

	"github.com/sahas/readit/internal/sanitize"
)

func TestValidateHandle(t *testing.T) {
	tests := []struct {
		handle string
		valid  bool
	}{
		{"satoshi", true},
		{"satoshi_nakamoto", true},
		{"alice-123", true},
		{"abc", true},
		{"ab", false},                        // too short
		{"this_handle_is_way_too_long_to_pass", false}, // > 20 chars
		{"-badstart", false},
		{"badend-", false},
		{"_badstart", false},
		{"badend_", false},
		{"bad handle", false},                 // contains space
		{"bad\x1b[31mcolor", false},           // contains ANSI
		{"user@domain", false},                // invalid char
		{"user!admin", false},                 // invalid char
	}

	for _, tt := range tests {
		err := sanitize.ValidateHandle(tt.handle)
		if (err == nil) != tt.valid {
			t.Errorf("ValidateHandle(%q) expected valid=%v, got err=%v", tt.handle, tt.valid, err)
		}
	}
}

func TestTextSanitizer(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "clean string",
			input:    "Hello World!",
			expected: "Hello World!",
		},
		{
			name:     "CSI color escape sequence",
			input:    "Hello \x1b[31;1mRed Bold\x1b[0m World",
			expected: "Hello Red Bold World",
		},
		{
			name:     "OSC 52 clipboard injection payload",
			input:    "Title \x1b]52;c;cGF5bG9hZA==\x07 Injected",
			expected: "Title  Injected",
		},
		{
			name:     "Terminal title manipulation OSC sequence",
			input:    "\x1b]0;Pwned Terminal\x07Legit Title",
			expected: "Legit Title",
		},
		{
			name:     "Bell and Backspace control chars",
			input:    "Danger\x07ous \x08Text",
			expected: "Dangerous Text",
		},
		{
			name:     "Preserves legitimate formatting newlines and tabs",
			input:    "Line 1\nLine 2\tTabbed",
			expected: "Line 1\nLine 2\tTabbed",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := sanitize.Text(tt.input)
			if got != tt.expected {
				t.Errorf("Text(%q)\n got:      %q\n expected: %q", tt.input, got, tt.expected)
			}
		})
	}
}

func TestSingleLine(t *testing.T) {
	input := "Multi\nLine\t\r  Title \x1b[32mClean\x1b[0m"
	expected := "Multi Line Title Clean"
	got := sanitize.SingleLine(input)
	if got != expected {
		t.Errorf("SingleLine(%q) = %q, expected %q", input, got, expected)
	}
}

func TestProfanityFilter(t *testing.T) {
	tests := []struct {
		name      string
		input     string
		isProfane bool
	}{
		{"clean discussion", "Welcome to our Go terminal forum!", false},
		{"clean technical talk", "Using postgres connection pooling and indexes", false},
		{"false positive safe word classic", "This is a classic architectural pattern", false},
		{"false positive safe word cockpit", "Pilot sitting in the cockpit", false},
		{"false positive safe word assassin", "Playing Assassin's Creed", false},
		{"false positive safe word basement", "The server rack is in the basement", false},
		{"explicit curse word", "This is total bullshit", true},
		{"f-word variation", "What the fuck is this", true},
		{"leetspeak profanity", "You are a b!tch", true},
		{"spaced out profanity", "This is s h i t", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := sanitize.ContainsProfanity(tt.input)
			if got != tt.isProfane {
				t.Errorf("ContainsProfanity(%q) = %v, expected %v", tt.input, got, tt.isProfane)
			}
		})
	}
}

func TestValidateCleanContent(t *testing.T) {
	// Clean content returns nil error
	if err := sanitize.ValidateCleanContent("title", "A clean post title"); err != nil {
		t.Errorf("expected clean content to pass, got err=%v", err)
	}

	// Profane content returns descriptive ValidationError
	err := sanitize.ValidateCleanContent("title", "This is fucking broken")
	if err == nil {
		t.Errorf("expected profane content to be rejected, got nil")
	} else if _, ok := err.(*sanitize.ValidationError); !ok {
		t.Errorf("expected *sanitize.ValidationError, got %T: %v", err, err)
	}
}

func TestCensor(t *testing.T) {
	censored := sanitize.Censor("This is shit")
	if sanitize.ContainsProfanity(censored) {
		t.Errorf("expected censored text %q to no longer trigger profanity filter", censored)
	}
}

func TestValidateHandle_Profanity(t *testing.T) {
	// Clean handles
	if err := sanitize.ValidateHandle("clean_user"); err != nil {
		t.Errorf("expected clean handle to pass, got: %v", err)
	}

	// Profane handle
	if err := sanitize.ValidateHandle("bad_asshole_99"); err == nil {
		t.Errorf("expected profane handle to be rejected, got nil")
	}
}

