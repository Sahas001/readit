package tui

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"
	"github.com/sahas/readit/internal/sanitize"
)

// viewProfile renders the user profile and activity card.
func (m *Model) viewProfile() string {
	m.ensureStyles()
	_, _, contentWidth, contentHeight := m.shellDimensions()

	var f strings.Builder

	if m.profileUser == nil {
		f.WriteString("\n" + m.styles.Subtitle.Render("User not found.") + "\n\nPress [esc] to return.")
		card := lipgloss.NewStyle().Border(lipgloss.RoundedBorder()).BorderForeground(m.theme.Border).Padding(1, 2).Render(f.String())
		return m.renderAppShell("Profile", lipgloss.PlaceHorizontal(contentWidth, lipgloss.Center, card), [][2]string{{"esc", "return"}})
	}

	u := m.profileUser
	handle := sanitize.SingleLine(u.Handle)
	postKarma := u.PostKarma
	commKarma := u.CommentKarma
	totalKarma := postKarma + commKarma

	// 1. Profile Header Box: Handle, Karma Badges, Join Date
	titleLine := lipgloss.NewStyle().Foreground(m.theme.Primary).Bold(true).Render("@"+handle) +
		"  " + m.styles.Meta.Render("•") + "  " +
		lipgloss.NewStyle().Foreground(m.theme.Upvote).Bold(true).Render(fmt.Sprintf("▲ %d karma", totalKarma)) +
		"  " + m.styles.Meta.Render(fmt.Sprintf("(%d post · %d comment)", postKarma, commKarma))

	joinDate := u.CreatedAt.Time.Format("Jan 02, 2006")
	metaLine := m.styles.Meta.Render("Member since " + joinDate)

	f.WriteString(titleLine + "\n")
	f.WriteString(metaLine + "\n")

	if u.Bio != "" {
		bioLine := lipgloss.NewStyle().Foreground(m.theme.Text).Italic(true).Render(truncateRunes(sanitize.SingleLine(u.Bio), 80))
		f.WriteString(bioLine + "\n")
	}

	f.WriteString(m.styles.Rule.Render(strings.Repeat("─", min(contentWidth-8, 70))) + "\n\n")

	// 2. Tab Bar: [ Submissions (N) ]   [ Comments (M) ]
	subCount := len(m.profilePosts)
	commCount := len(m.profileComments)

	var subTabStyle, commTabStyle lipgloss.Style
	if m.profileTab == 0 {
		subTabStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("#FFFFFF")).Background(m.theme.Primary).Bold(true).Padding(0, 1)
		commTabStyle = lipgloss.NewStyle().Foreground(m.theme.TextDim).Padding(0, 1)
	} else {
		subTabStyle = lipgloss.NewStyle().Foreground(m.theme.TextDim).Padding(0, 1)
		commTabStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("#FFFFFF")).Background(m.theme.Primary).Bold(true).Padding(0, 1)
	}

	tabBar := fmt.Sprintf("%s   %s",
		subTabStyle.Render(fmt.Sprintf("Submissions (%d)", subCount)),
		commTabStyle.Render(fmt.Sprintf("Comments (%d)", commCount)),
	)
	f.WriteString(tabBar + "\n\n")

	// 3. Tab Content
	availLines := max(3, contentHeight-11)

	if m.profileTab == 0 {
		// Submissions Tab
		if subCount == 0 {
			f.WriteString(m.styles.Subtitle.Render("No submissions yet.") + "\n")
		} else {
			startIdx := 0
			if m.profilePostCursor >= availLines {
				startIdx = m.profilePostCursor - availLines + 1
			}
			endIdx := min(subCount, startIdx+availLines)

			for i := startIdx; i < endIdx; i++ {
				p := m.profilePosts[i]
				isSelected := (i == m.profilePostCursor)

				cursor := "  "
				if isSelected {
					cursor = lipgloss.NewStyle().Foreground(m.theme.Primary).Bold(true).Render("▌ ")
				}

				scoreStr := fmt.Sprintf("▲ %-3d", p.Score)
				scoreRendered := lipgloss.NewStyle().Foreground(m.theme.Upvote).Render(scoreStr)
				if p.Score < 0 {
					scoreRendered = lipgloss.NewStyle().Foreground(m.theme.Downvote).Render(fmt.Sprintf("▼ %-3d", p.Score))
				}

				title := truncateRunes(sanitize.SingleLine(p.Title), 42)
				titleRendered := lipgloss.NewStyle().Foreground(m.theme.Text).Bold(isSelected).Render(title)

				meta := m.styles.Meta.Render(fmt.Sprintf("/b/%s · %d comments · %s", p.BoardSlug, p.CommentCount, timeAgo(p.CreatedAt.Time)))

				f.WriteString(fmt.Sprintf("%s%s %s  %s\n", cursor, scoreRendered, titleRendered, meta))
			}
		}
	} else {
		// Comments Tab
		if commCount == 0 {
			f.WriteString(m.styles.Subtitle.Render("No comments yet.") + "\n")
		} else {
			startIdx := 0
			if m.profileCommCursor >= availLines {
				startIdx = m.profileCommCursor - availLines + 1
			}
			endIdx := min(commCount, startIdx+availLines)

			for i := startIdx; i < endIdx; i++ {
				c := m.profileComments[i]
				isSelected := (i == m.profileCommCursor)

				cursor := "  "
				if isSelected {
					cursor = lipgloss.NewStyle().Foreground(m.theme.Primary).Bold(true).Render("▌ ")
				}

				scoreStr := fmt.Sprintf("▲ %-3d", c.Score)
				scoreRendered := lipgloss.NewStyle().Foreground(m.theme.Upvote).Render(scoreStr)
				if c.Score < 0 {
					scoreRendered = lipgloss.NewStyle().Foreground(m.theme.Downvote).Render(fmt.Sprintf("▼ %-3d", c.Score))
				}

				snippet := truncateRunes(sanitize.SingleLine(c.Body), 38)
				snippetRendered := lipgloss.NewStyle().Foreground(m.theme.Text).Bold(isSelected).Render(snippet)

				postContext := m.styles.Meta.Render(fmt.Sprintf("in %s (/b/%s)", truncateRunes(sanitize.SingleLine(c.PostTitle), 22), c.BoardSlug))

				f.WriteString(fmt.Sprintf("%s%s %s  %s\n", cursor, scoreRendered, snippetRendered, postContext))
			}
		}
	}

	cardWidth := min(96, max(44, contentWidth-4))
	cardStyle := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(m.theme.Border).
		Padding(1, 2)

	card := cardStyle.Width(cardWidth).Render(f.String())

	shortcuts := [][2]string{
		{"tab/h/l", "switch tab"},
		{"j/k", "move"},
		{"enter", "view discussion"},
		{"t", "theme"},
		{"esc", "return"},
	}

	return m.renderAppShell("@"+handle+" Profile", lipgloss.PlaceHorizontal(contentWidth, lipgloss.Center, card), shortcuts)
}
