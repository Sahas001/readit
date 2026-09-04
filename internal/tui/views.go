package tui

import (
	"fmt"
	"strings"
	"time"

	"github.com/charmbracelet/lipgloss"
	"github.com/sahas/readit/internal/sanitize"
)

const logo = ` ____                _ ___ _____
|  _ \ ___  __ _  __| |_ _|_   _|
| |_) / _ \/ _` + "`" + ` |/ _` + "`" + ` || |  | |
|  _ <  __/ (_| | (_| || |  | |
|_| \_\___|\__,_|\__,_|___| |_|`

func (m *Model) viewLoading() string {
	return m.centeredView(
		styleLogo.Render(logo) + "\n" +
			styleSubtitle.Render("  Connecting..."),
	)
}

func (m *Model) viewOnboarding() string {
	var b strings.Builder
	b.WriteString(styleLogo.Render(logo))
	b.WriteString("\n\n")
	b.WriteString(stylePrompt.Render("  Welcome! Your SSH key is new here."))
	b.WriteString("\n")
	b.WriteString(styleSubtitle.Render("  Pick a username to get started (3-20 characters):"))
	b.WriteString("\n\n")
	b.WriteString("  " + m.handleInput.View())
	b.WriteString("\n\n")
	b.WriteString(styleSubtitle.Render("  Press enter to confirm • esc to quit"))

	if m.err != nil {
		b.WriteString("\n\n" + styleError.Render("  Error: "+sanitize.SingleLine(m.err.Error())))
	}

	return m.centeredView(b.String())
}

func (m *Model) viewBoardList() string {
	var b strings.Builder

	// Header.
	safeHandle := sanitize.SingleLine(m.user.Handle)
	header := styleLogo.Render("ReadIT") +
		styleSubtitle.Render(fmt.Sprintf("  logged in as %s", safeHandle))
	b.WriteString(header + "\n")
	b.WriteString(strings.Repeat("─", max(0, min(m.width, 60))) + "\n\n")

	// Board list.
	for i, board := range m.boards {
		safeSlug := sanitize.SingleLine(board.Slug)
		safeDesc := sanitize.SingleLine(board.Description)
		itemText := fmt.Sprintf("/b/%s — %s", safeSlug, safeDesc)

		var line string
		if i == m.boardCursor {
			line = styleSelectedItem.Render(itemText)
		} else {
			line = styleNormalItem.Render(itemText)
		}
		b.WriteString(line + "\n")
	}

	// Status bar.
	b.WriteString("\n")
	statusBar := styleStatusBar.
		Width(max(0, min(m.width, 60))).
		Render(" ↑/k up • ↓/j down • enter select • q quit")
	b.WriteString(statusBar)

	return b.String()
}

func (m *Model) viewPostList() string {
	var b strings.Builder

	// Header.
	boardName := ""
	if m.currentBoard != nil {
		boardName = "/b/" + sanitize.SingleLine(m.currentBoard.Slug)
	}
	header := styleLogo.Render("ReadIT") +
		styleSubtitle.Render("  "+boardName)
	b.WriteString(header + "\n")
	b.WriteString(strings.Repeat("─", max(0, min(m.width, 60))) + "\n\n")

	if len(m.posts) == 0 {
		b.WriteString(styleSubtitle.Render("  No posts yet. Press n to create the first discussion!") + "\n")
	}

	for i, post := range m.posts {
		score := styleScore.Render(fmt.Sprintf("%d▲", post.Score))
		comments := styleSubtitle.Render(fmt.Sprintf("%d comments", post.CommentCount))
		timeStr := styleSubtitle.Render(relativeTime(post.CreatedAt.Time))
		author := styleSubtitle.Render(fmt.Sprintf("by %s", sanitize.SingleLine(post.AuthorHandle)))
		safeTitle := sanitize.SingleLine(post.Title)

		var title string
		if i == m.postCursor {
			title = styleSelectedItem.Render(safeTitle)
		} else {
			title = styleNormalItem.Render(safeTitle)
		}

		line := lipgloss.JoinHorizontal(lipgloss.Top, score, " ", title)
		b.WriteString(line + "\n")
		b.WriteString(fmt.Sprintf("       %s • %s • %s\n", author, timeStr, comments))
	}

	// Status bar.
	b.WriteString("\n")
	statusBar := styleStatusBar.
		Width(max(0, min(m.width, 60))).
		Render(" ↑/k up • ↓/j down • enter view • u/d vote • n new post • esc back • q quit")
	b.WriteString(statusBar)

	return b.String()
}

func (m *Model) viewPostDetail() string {
	var b strings.Builder

	// Top breadcrumb header
	boardSlug := ""
	if m.currentBoard != nil {
		boardSlug = sanitize.SingleLine(m.currentBoard.Slug)
	}
	header := styleLogo.Render("ReadIT") +
		styleSubtitle.Render(fmt.Sprintf("  /b/%s  •  Discussion", boardSlug))
	b.WriteString(header + "\n")
	b.WriteString(strings.Repeat("─", max(0, min(m.width, 70))) + "\n")

	// Viewport containing scrollable post content & comment thread
	b.WriteString(m.viewport.View() + "\n")

	// Bottom status bar
	statusBar := styleStatusBar.
		Width(max(0, min(m.width, 70))).
		Render(" j/k scroll • u/d vote post • r reply • esc back to board • q quit")
	b.WriteString(statusBar)

	return b.String()
}

func (m *Model) renderPostDetailContent() string {
	if m.currentPost == nil {
		return "Loading post..."
	}

	var b strings.Builder
	p := m.currentPost

	// Post Header
	title := styleTitle.Copy().Foreground(colorPrimary).Render(sanitize.SingleLine(p.Title))
	score := styleScore.Render(fmt.Sprintf("%d▲", p.Score))
	author := styleAuthor.Render(sanitize.SingleLine(p.AuthorHandle))
	timeStr := styleSubtitle.Render(relativeTime(p.CreatedAt.Time))
	b.WriteString(fmt.Sprintf("%s  %s\n", score, title))
	b.WriteString(fmt.Sprintf("    Posted by %s • %s\n", author, timeStr))

	if p.Url != "" {
		urlStr := styleSubtitle.Copy().Foreground(colorAccent).Render("Link: " + sanitize.SingleLine(p.Url))
		b.WriteString("    " + urlStr + "\n")
	}

	// Post Body
	b.WriteString("\n")
	if p.Body != "" {
		body := stylePostBody.Render(sanitize.Text(p.Body))
		b.WriteString(body + "\n")
	}
	b.WriteString("\n" + strings.Repeat("─", max(0, min(m.width-6, 65))) + "\n")

	// Comments Header
	commentCount := len(m.comments)
	b.WriteString(fmt.Sprintf("  Comments (%d)   [Press r to reply]\n", commentCount))
	b.WriteString(strings.Repeat("─", max(0, min(m.width-6, 65))) + "\n\n")

	if commentCount == 0 {
		b.WriteString(styleSubtitle.Render("    No comments yet. Be the first to reply (press r)!") + "\n")
		return b.String()
	}

	// Render Threaded Comments
	for _, c := range m.comments {
		depth := int(c.Depth)
		indent := strings.Repeat("  ", depth)
		prefix := "• "
		if depth > 0 {
			prefix = "└─ "
		}

		cAuthor := styleAuthor.Render(sanitize.SingleLine(c.AuthorHandle))
		cScore := styleScore.Copy().Width(0).Render(fmt.Sprintf("%d▲", c.Score))
		cTime := styleSubtitle.Render(relativeTime(c.CreatedAt.Time))
		cBody := sanitize.Text(c.Body)

		b.WriteString(fmt.Sprintf("%s%s%s  %s  %s\n", indent, styleBranch.Render(prefix), cAuthor, cScore, cTime))

		// Indent comment body lines
		bodyLines := strings.Split(cBody, "\n")
		bodyIndent := indent + "   "
		for _, line := range bodyLines {
			b.WriteString(fmt.Sprintf("%s%s\n", bodyIndent, line))
		}
		b.WriteString("\n")
	}

	return b.String()
}

func (m *Model) viewNewPost() string {
	var b strings.Builder

	boardSlug := ""
	if m.currentBoard != nil {
		boardSlug = sanitize.SingleLine(m.currentBoard.Slug)
	}

	b.WriteString(styleLogo.Render("ReadIT") + "\n\n")
	b.WriteString(stylePrompt.Render(fmt.Sprintf("  Create a Discussion in /b/%s", boardSlug)) + "\n\n")

	// Title Input
	titleStyle := styleInputBlurred
	if m.postFormFocus == 0 {
		titleStyle = styleInputFocused
	}
	b.WriteString(styleSubtitle.Render("  Title:") + "\n")
	b.WriteString("  " + titleStyle.Render(m.titleInput.View()) + "\n\n")

	// URL Input
	urlStyle := styleInputBlurred
	if m.postFormFocus == 1 {
		urlStyle = styleInputFocused
	}
	b.WriteString(styleSubtitle.Render("  Link URL (optional):") + "\n")
	b.WriteString("  " + urlStyle.Render(m.urlInput.View()) + "\n\n")

	// Body Input
	bodyStyle := styleInputBlurred
	if m.postFormFocus == 2 {
		bodyStyle = styleInputFocused
	}
	b.WriteString(styleSubtitle.Render("  Body:") + "\n")
	b.WriteString("  " + bodyStyle.Render(m.bodyInput.View()) + "\n\n")

	if m.err != nil {
		b.WriteString("  " + styleError.Render("Error: "+sanitize.SingleLine(m.err.Error())) + "\n\n")
	}

	b.WriteString(styleStatusBar.
		Width(max(0, min(m.width, 70))).
		Render(" tab: next field • ctrl+s: publish • esc: cancel"))

	return m.centeredView(b.String())
}

func (m *Model) viewNewComment() string {
	var b strings.Builder

	target := "post"
	if m.replyParentAuthor != "" {
		target = fmt.Sprintf("@%s", sanitize.SingleLine(m.replyParentAuthor))
	}

	b.WriteString(styleLogo.Render("ReadIT") + "\n\n")
	b.WriteString(stylePrompt.Render(fmt.Sprintf("  Replying to %s", target)) + "\n\n")

	commentStyle := styleInputFocused
	b.WriteString("  " + commentStyle.Render(m.commentInput.View()) + "\n\n")

	if m.err != nil {
		b.WriteString("  " + styleError.Render("Error: "+sanitize.SingleLine(m.err.Error())) + "\n\n")
	}

	b.WriteString(styleStatusBar.
		Width(max(0, min(m.width, 70))).
		Render(" ctrl+s: submit reply • esc: cancel"))

	return m.centeredView(b.String())
}

func (m *Model) viewError() string {
	errText := "unknown error"
	if m.err != nil {
		errText = sanitize.SingleLine(m.err.Error())
	}
	return m.centeredView(
		styleError.Render("Error: "+errText) + "\n\n" +
			styleSubtitle.Render("Press q to quit"),
	)
}

// centeredView centers content vertically and horizontally.
func (m *Model) centeredView(content string) string {
	if m.width <= 0 || m.height <= 0 {
		return content
	}
	return lipgloss.Place(
		m.width, m.height,
		lipgloss.Center, lipgloss.Center,
		content,
	)
}

// relativeTime formats timestamps into human-readable relative duration.
func relativeTime(t time.Time) string {
	if t.IsZero() {
		return ""
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
		return t.Format("Jan 02, 2006")
	}
}
