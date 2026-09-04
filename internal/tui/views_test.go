package tui

import (
	"testing"
	"time"
)

func TestRelativeTime(t *testing.T) {
	now := time.Now()

	tests := []struct {
		timestamp time.Time
		expected  string
	}{
		{now.Add(-10 * time.Second), "just now"},
		{now.Add(-5 * time.Minute), "5m ago"},
		{now.Add(-2 * time.Hour), "2h ago"},
		{now.Add(-3 * 24 * time.Hour), "3d ago"},
		{time.Time{}, ""},
	}

	for _, tt := range tests {
		got := relativeTime(tt.timestamp)
		if got != tt.expected {
			t.Errorf("relativeTime(%v) = %q, expected %q", tt.timestamp, got, tt.expected)
		}
	}
}
