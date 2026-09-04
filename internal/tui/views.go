package tui

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"
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
	b.WriteString(styleSubtitle.Render("  Pick a username to get started:"))
	b.WriteString("\n\n")
	b.WriteString("  " + m.handleInput.View())
	b.WriteString("\n\n")
	b.WriteString(styleSubtitle.Render("  Press enter to confirm • esc to quit"))

	if m.err != nil {
		b.WriteString("\n" + styleError.Render("  Error: "+m.err.Error()))
	}

	return m.centeredView(b.String())
}

func (m *Model) viewBoardList() string {
	var b strings.Builder

	// Header.
	header := styleLogo.Render("ReadIT") +
		styleSubtitle.Render(fmt.Sprintf("  logged in as %s", m.user.Handle))
	b.WriteString(header + "\n")
	b.WriteString(strings.Repeat("─", min(m.width, 60)) + "\n\n")

	// Board list.
	for i, board := range m.boards {
		var line string
		if i == m.boardCursor {
			line = styleSelectedItem.Render(
				fmt.Sprintf("/b/%s — %s", board.Slug, board.Description),
			)
		} else {
			line = styleNormalItem.Render(
				fmt.Sprintf("/b/%s — %s", board.Slug, board.Description),
			)
		}
		b.WriteString(line + "\n")
	}

	// Status bar.
	b.WriteString("\n")
	statusBar := styleStatusBar.
		Width(min(m.width, 60)).
		Render(" ↑/k up • ↓/j down • enter select • q quit")
	b.WriteString(statusBar)

	return b.String()
}

func (m *Model) viewPostList() string {
	var b strings.Builder

	// Header.
	boardName := ""
	if m.currentBoard != nil {
		boardName = "/b/" + m.currentBoard.Slug
	}
	header := styleLogo.Render("ReadIT") +
		styleSubtitle.Render("  " + boardName)
	b.WriteString(header + "\n")
	b.WriteString(strings.Repeat("─", min(m.width, 60)) + "\n\n")

	if len(m.posts) == 0 {
		b.WriteString(styleSubtitle.Render("  No posts yet. Press n to create one.") + "\n")
	}

	for i, post := range m.posts {
		score := styleScore.Render(fmt.Sprintf("%d▲", post.Score))
		comments := styleSubtitle.Render(fmt.Sprintf("%d comments", post.CommentCount))
		author := styleSubtitle.Render(fmt.Sprintf("by %s", post.AuthorHandle))

		var title string
		if i == m.postCursor {
			title = styleSelectedItem.Render(post.Title)
		} else {
			title = styleNormalItem.Render(post.Title)
		}

		line := lipgloss.JoinHorizontal(lipgloss.Top, score, " ", title)
		b.WriteString(line + "\n")
		b.WriteString(fmt.Sprintf("       %s  •  %s\n", author, comments))
	}

	// Status bar.
	b.WriteString("\n")
	statusBar := styleStatusBar.
		Width(min(m.width, 60)).
		Render(" ↑/k up • ↓/j down • n new post • esc back • q quit")
	b.WriteString(statusBar)

	return b.String()
}

func (m *Model) viewError() string {
	errText := "unknown error"
	if m.err != nil {
		errText = m.err.Error()
	}
	return m.centeredView(
		styleError.Render("Error: "+errText) + "\n\n" +
			styleSubtitle.Render("Press q to quit"),
	)
}

// centeredView centers content vertically and horizontally.
func (m *Model) centeredView(content string) string {
	if m.width == 0 || m.height == 0 {
		return content
	}
	return lipgloss.Place(
		m.width, m.height,
		lipgloss.Center, lipgloss.Center,
		content,
	)
}
