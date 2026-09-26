package tui

import (
	"fmt"
	"strings"
	"time"

	"github.com/charmbracelet/lipgloss"
	"github.com/sahas/readit/internal/sanitize"
)

// viewInbox renders the notifications inbox card.
func (m *Model) viewInbox() string {
	_, _, contentWidth, contentHeight := m.shellDimensions()

	var f strings.Builder

	if len(m.notifications) == 0 {
		f.WriteString("\n")
		f.WriteString(stylePrompt.Render("No notifications yet.") + "\n\n")
		f.WriteString(styleSubtitle.Render("When other users reply to your discussions or comments,") + "\n")
		f.WriteString(styleSubtitle.Render("you'll see them right here in your inbox.") + "\n\n")
		f.WriteString(styleSubtitle.Render("Press [esc] to return."))
	} else {
		// Calculate available lines for notifications
		// Card padding takes 2 lines, header takes 2 lines
		availLines := max(3, contentHeight-6)
		itemCount := len(m.notifications)

		// Bounded scrolling window around m.notificationCursor
		startIdx := 0
		if m.notificationCursor >= availLines {
			startIdx = m.notificationCursor - availLines + 1
		}
		endIdx := min(itemCount, startIdx+availLines)

		for i := startIdx; i < endIdx; i++ {
			n := m.notifications[i]
			isSelected := (i == m.notificationCursor)

			// 1. Read / Unread Indicator
			var statusIndicator string
			if !n.IsRead {
				statusIndicator = lipgloss.NewStyle().Foreground(currentTheme.Primary).Bold(true).Render("●")
			} else {
				statusIndicator = lipgloss.NewStyle().Foreground(currentTheme.TextDim).Render("○")
			}

			// 2. Cursor indicator
			cursor := "  "
			if isSelected {
				cursor = lipgloss.NewStyle().Foreground(currentTheme.Primary).Bold(true).Render("▌ ")
			}

			// 3. Time ago
			timeStr := styleMeta.Render(timeAgo(n.CreatedAt.Time))

			// 4. Action text & Sanitized Actor Handle
			actor := sanitize.SingleLine(n.ActorHandle)
			actionText := "replied to your post"
			if n.Type == "reply_comment" {
				actionText = "replied to your comment"
			}
			headerLine := fmt.Sprintf("%s %s %s @%s %s  %s",
				cursor,
				statusIndicator,
				lipgloss.NewStyle().Foreground(currentTheme.Text).Bold(isSelected).Render(actionText),
				lipgloss.NewStyle().Foreground(currentTheme.Secondary).Bold(true).Render(actor),
				styleMeta.Render("in"),
				lipgloss.NewStyle().Foreground(currentTheme.Text).Bold(isSelected).Render(truncateRunes(sanitize.SingleLine(n.PostTitle), 35)),
			)

			// 5. Snippet line
			snippetText := truncateRunes(sanitize.SingleLine(n.CommentSnippet), 60)
			snippetLine := fmt.Sprintf("      %s %s",
				styleMeta.Render("›"),
				lipgloss.NewStyle().Foreground(currentTheme.TextMuted).Italic(true).Render(snippetText),
			)

			f.WriteString(headerLine + " " + timeStr + "\n")
			f.WriteString(snippetLine + "\n")
			if i < endIdx-1 {
				f.WriteString("\n")
			}
		}
	}

	cardWidth := min(96, max(40, contentWidth-4))
	cardStyle := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(currentTheme.Border).
		Padding(1, 2)

	card := cardStyle.Width(cardWidth).Render(f.String())

	shortcuts := [][2]string{
		{"j/k", "move"},
		{"enter", "jump to thread"},
		{"a", "mark all read"},
		{"esc", "return"},
	}

	unreadLabel := ""
	if m.unreadNotificationCount > 0 {
		unreadLabel = fmt.Sprintf(" (%d unread)", m.unreadNotificationCount)
	}

	return m.renderAppShell("Inbox"+unreadLabel, lipgloss.PlaceHorizontal(contentWidth, lipgloss.Center, card), shortcuts)
}

// truncateRunes truncates text to maxLen runes safely without slicing multi-byte characters.
func truncateRunes(s string, maxLen int) string {
	runes := []rune(s)
	if len(runes) <= maxLen {
		return s
	}
	if maxLen <= 3 {
		return string(runes[:maxLen])
	}
	return string(runes[:maxLen-3]) + "..."
}

// timeAgo formats a time.Time into a concise relative string.
func timeAgo(t time.Time) string {
	if t.IsZero() {
		return "just now"
	}
	d := time.Since(t)
	switch {
	case d < time.Minute:
		return "just now"
	case d < time.Hour:
		return fmt.Sprintf("%dm ago", int(d.Minutes()))
	case d < 24*time.Hour:
		return fmt.Sprintf("%dh ago", int(d.Hours()))
	case d < 30*24*time.Hour:
		return fmt.Sprintf("%dd ago", int(d.Hours()/24))
	default:
		return t.Format("Jan 02")
	}
}
