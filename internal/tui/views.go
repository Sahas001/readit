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
	header := styleLogo.Render("ReadIT") +
		styleSubtitle.Render("  •  Welcome")
	ruleWidth := 70
	if m.width > 0 {
		ruleWidth = m.width
	}
	separator := styleRule.Render(strings.Repeat("─", ruleWidth))
	topHeader := header + "\n" + separator

	cardWidth := 60
	if m.width > 0 {
		cardWidth = min(70, max(36, m.width-8))
	}
	contentWidth := max(16, cardWidth-6)
	inputWidth := max(12, contentWidth-4)

	var f strings.Builder
	f.WriteString(stylePrompt.Render("Welcome! Your SSH key is new here.") + "\n\n")
	f.WriteString(styleSubtitle.Render("Pick a username to get started (3-20 characters):") + "\n\n")
	f.WriteString(styleInputFocused.Width(inputWidth).Render(m.handleInput.View()) + "\n\n")
	f.WriteString(styleSubtitle.Render("Press enter to confirm • esc to quit"))

	if m.err != nil {
		f.WriteString("\n\n" + styleError.Render("Error: "+sanitize.SingleLine(m.err.Error())))
	}

	cardStyle := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(colorPrimary).
		Padding(1, 2)
	card := cardStyle.Render(f.String())

	statusBar := m.renderStatusBar("Welcome", [][2]string{
		{"enter", "confirm"},
		{"esc", "quit"},
	})

	return m.renderDockedViewWithCenteredContent(topHeader, card, statusBar)
}

func (m *Model) viewBoardList() string {
	safeHandle := ""
	if m.user != nil {
		safeHandle = sanitize.SingleLine(m.user.Handle)
	}
	header := styleLogo.Render("ReadIT") +
		styleSubtitle.Render(fmt.Sprintf("  logged in as @%s", safeHandle))
	ruleWidth := 70
	if m.width > 0 {
		ruleWidth = m.width
	}
	separator := styleRule.Render(strings.Repeat("─", ruleWidth))
	topHeader := header + "\n" + separator

	headerH := lipgloss.Height(topHeader)
	availH := 10
	if m.height > 0 {
		availH = max(2, m.height-headerH-1)
	}

	var content strings.Builder
	if len(m.boards) == 0 {
		content.WriteString("\n" + styleSubtitle.Render("  No boards found.") + "\n")
	} else {
		boardsPerPage := max(1, availH)
		startIdx := 0
		if m.boardCursor >= boardsPerPage {
			startIdx = m.boardCursor - boardsPerPage + 1
		}
		endIdx := min(len(m.boards), startIdx+boardsPerPage)

		for i := startIdx; i < endIdx; i++ {
			board := m.boards[i]
			safeSlug := sanitize.SingleLine(board.Slug)
			safeDesc := sanitize.SingleLine(board.Description)
			itemText := fmt.Sprintf("/b/%s — %s", safeSlug, safeDesc)

			var line string
			if i == m.boardCursor {
				line = styleSelectedItem.Render(itemText)
			} else {
				line = styleNormalItem.Render(itemText)
			}
			content.WriteString(line + "\n")
		}
	}

	statusBar := m.renderStatusBar("ReadIT", [][2]string{
		{"↑/k", "up"},
		{"↓/j", "down"},
		{"enter", "select"},
		{"q", "quit"},
	})

	return m.renderDockedView(topHeader, content.String(), statusBar)
}

func (m *Model) viewPostList() string {
	boardSlug := ""
	boardTitle := ""
	if m.currentBoard != nil {
		boardSlug = sanitize.SingleLine(m.currentBoard.Slug)
		boardTitle = "/b/" + boardSlug
	}
	header := styleLogo.Render("ReadIT") +
		styleSubtitle.Render("  "+boardTitle)
	ruleWidth := 70
	if m.width > 0 {
		ruleWidth = m.width
	}
	separator := styleRule.Render(strings.Repeat("─", ruleWidth))
	topHeader := header + "\n" + separator

	headerH := lipgloss.Height(topHeader)
	availH := 10
	if m.height > 0 {
		availH = max(2, m.height-headerH-1)
	}

	var content strings.Builder
	if len(m.posts) == 0 {
		content.WriteString("\n" + styleSubtitle.Render("  No posts yet. Press n to create the first discussion!") + "\n")
	} else {
		// Window posts to fit in available height (each post occupies 2 lines)
		postsPerPage := max(1, availH/2)
		startIdx := 0
		if m.postCursor >= postsPerPage {
			startIdx = m.postCursor - postsPerPage + 1
		}
		endIdx := min(len(m.posts), startIdx+postsPerPage)

		for i := startIdx; i < endIdx; i++ {
			post := m.posts[i]
			score := styleScore.Render(fmt.Sprintf("%d▲", post.Score))
			comments := styleSubtitle.Render(fmt.Sprintf("%d comments", post.CommentCount))
			timeStr := styleSubtitle.Render(relativeTime(post.CreatedAt.Time))
			author := styleSubtitle.Render(fmt.Sprintf("by @%s", sanitize.SingleLine(post.AuthorHandle)))
			safeTitle := sanitize.SingleLine(post.Title)

			var title string
			if i == m.postCursor {
				title = styleSelectedItem.Render(safeTitle)
			} else {
				title = styleNormalItem.Render(safeTitle)
			}

			line := lipgloss.JoinHorizontal(lipgloss.Top, score, " ", title)
			content.WriteString(line + "\n")
			content.WriteString(fmt.Sprintf("       %s • %s • %s\n", author, timeStr, comments))
		}
	}

	statusBar := m.renderStatusBar("/b/"+boardSlug, [][2]string{
		{"↑/k", "up"},
		{"↓/j", "down"},
		{"enter", "view"},
		{"u/d", "vote"},
		{"n", "new post"},
		{"esc", "back"},
		{"q", "quit"},
	})

	return m.renderDockedView(topHeader, content.String(), statusBar)
}

func (m *Model) viewPostDetail() string {
	boardSlug := ""
	if m.currentBoard != nil {
		boardSlug = sanitize.SingleLine(m.currentBoard.Slug)
	}
	header := styleLogo.Render("ReadIT") +
		styleSubtitle.Render(fmt.Sprintf("  /b/%s  •  Discussion", boardSlug))
	ruleWidth := 70
	if m.width > 0 {
		ruleWidth = m.width
	}
	separator := styleRule.Render(strings.Repeat("─", ruleWidth))
	topHeader := header + "\n" + separator

	statusBar := m.renderStatusBar("/b/"+boardSlug, [][2]string{
		{"j/k", "navigate"},
		{"u/d", "vote"},
		{"r", "reply"},
		{"R", "reply to post"},
		{"esc", "back"},
		{"q", "quit"},
	})

	return m.renderDockedView(topHeader, m.viewport.View(), statusBar)
}

func (m *Model) renderPostDetailContent() string {
	if m.currentPost == nil {
		return "Loading post..."
	}

	var b strings.Builder
	p := m.currentPost

	contentWidth := 76
	if m.viewport.Width > 0 {
		contentWidth = max(20, m.viewport.Width)
	}

	// Post Header
	titleStr := sanitize.SingleLine(p.Title)
	title := styleTitle.Copy().Foreground(colorPrimary).Render(titleStr)
	score := styleScore.Render(fmt.Sprintf("%d▲", p.Score))
	author := styleAuthor.Render("@" + sanitize.SingleLine(p.AuthorHandle))
	timeStr := styleSubtitle.Render(relativeTime(p.CreatedAt.Time))

	if m.commentCursor == -1 {
		selectedIndicator := lipgloss.NewStyle().Foreground(colorAccent).Italic(true).Render("  ◄ post selected (r to reply)")
		b.WriteString(fmt.Sprintf("%s  %s%s\n", score, title, selectedIndicator))
	} else {
		b.WriteString(fmt.Sprintf("%s  %s\n", score, title))
	}
	b.WriteString(fmt.Sprintf("    Posted by %s • %s\n", author, timeStr))

	if p.Url != "" {
		urlStr := styleSubtitle.Copy().Foreground(colorAccent).Render("Link: " + sanitize.SingleLine(p.Url))
		b.WriteString("    " + urlStr + "\n")
	}

	// Post Body with wrapping
	b.WriteString("\n")
	if p.Body != "" {
		bodyWidth := max(20, contentWidth-6)
		bodyStyle := stylePostBody.Copy().Width(bodyWidth)
		b.WriteString(bodyStyle.Render(sanitize.Text(p.Body)) + "\n")
	}
	b.WriteString("\n" + styleRule.Render(strings.Repeat("─", max(10, contentWidth-4))) + "\n")

	// Comments Header
	commentCount := len(m.comments)
	b.WriteString(fmt.Sprintf("  Comments (%d)   [r: reply to selected • R: reply to post]\n", commentCount))
	b.WriteString(styleRule.Render(strings.Repeat("─", max(10, contentWidth-4))) + "\n\n")

	if commentCount == 0 {
		b.WriteString(styleSubtitle.Render("    No comments yet. Be the first to reply (press r)!") + "\n")
		return b.String()
	}

	m.commentLineOffsets = make([]int, len(m.comments))

	// Render Threaded Comments
	for i, c := range m.comments {
		// Record line offset before writing this comment
		m.commentLineOffsets[i] = strings.Count(b.String(), "\n")

		depth := int(c.Depth)
		indent := strings.Repeat("  ", depth)
		branchGlyph := "• "
		if depth > 0 {
			branchGlyph = "└─ "
		}

		isSelected := (i == m.commentCursor)
		isOP := (m.currentPost != nil && c.AuthorHandle == m.currentPost.AuthorHandle)

		var opBadge string
		if isOP {
			opBadge = " " + styleOpBadge.Render("[OP]")
		}

		var indicator string
		var branch string
		var cAuthor string
		var selPill string

		if isSelected {
			indicator = lipgloss.NewStyle().Foreground(colorPrimary).Bold(true).Render("▌ ")
			branch = styleSelectedBranch.Render(branchGlyph)
			cAuthor = lipgloss.NewStyle().Foreground(colorPrimary).Bold(true).Render("@" + sanitize.SingleLine(c.AuthorHandle))
			selPill = lipgloss.NewStyle().Foreground(colorAccent).Italic(true).Render("  ◄ selected (r to reply)")
		} else {
			indicator = "  "
			branch = styleBranch.Render(branchGlyph)
			cAuthor = styleAuthor.Render("@" + sanitize.SingleLine(c.AuthorHandle))
			selPill = ""
		}

		cScore := styleScore.Copy().Width(0).Render(fmt.Sprintf("%d▲", c.Score))
		cTime := styleSubtitle.Render(relativeTime(c.CreatedAt.Time))

		b.WriteString(fmt.Sprintf("%s%s%s%s%s  %s  %s%s\n", indicator, indent, branch, cAuthor, opBadge, cScore, cTime, selPill))

		// Indent and wrap comment body lines
		cBody := sanitize.Text(c.Body)
		indentLen := depth*2 + 6
		availCommentWidth := max(20, contentWidth-indentLen)
		wrappedBody := lipgloss.NewStyle().Width(availCommentWidth).Render(cBody)
		bodyLines := strings.Split(wrappedBody, "\n")

		var bodyPrefix string
		if isSelected {
			bodyPrefix = fmt.Sprintf("%s%s▌  ", indicator, indent)
		} else {
			bodyPrefix = fmt.Sprintf("  %s   ", indent)
		}
		for _, line := range bodyLines {
			b.WriteString(fmt.Sprintf("%s%s\n", bodyPrefix, line))
		}
		b.WriteString("\n")
	}

	return b.String()
}

func (m *Model) viewNewPost() string {
	boardSlug := ""
	if m.currentBoard != nil {
		boardSlug = sanitize.SingleLine(m.currentBoard.Slug)
	}

	header := styleLogo.Render("ReadIT") +
		styleSubtitle.Render(fmt.Sprintf("  /b/%s  •  Create Discussion", boardSlug))
	ruleWidth := 70
	if m.width > 0 {
		ruleWidth = m.width
	}
	separator := styleRule.Render(strings.Repeat("─", ruleWidth))
	topHeader := header + "\n" + separator

	_, contentWidth, inputWidth := m.formDimensions()

	var f strings.Builder
	f.WriteString(stylePrompt.Render(fmt.Sprintf("Create Discussion in /b/%s", boardSlug)) + "\n\n")

	// Title Input + Counter
	titleStyle := styleInputBlurred
	if m.postFormFocus == 0 {
		titleStyle = styleInputFocused
	}
	titleLen := len([]rune(m.titleInput.Value()))
	titleLimit := m.titleInput.CharLimit
	titleLabel := styleSubtitle.Render("Title:")
	titleCount := styleCharCount.Render(fmt.Sprintf("%d/%d", titleLen, titleLimit))
	f.WriteString(renderFormLabel(titleLabel, titleCount, contentWidth) + "\n")
	f.WriteString(titleStyle.Width(inputWidth).Render(m.titleInput.View()) + "\n\n")

	// URL Input + Counter
	urlStyle := styleInputBlurred
	if m.postFormFocus == 1 {
		urlStyle = styleInputFocused
	}
	urlLen := len([]rune(m.urlInput.Value()))
	urlLimit := m.urlInput.CharLimit
	urlLabel := styleSubtitle.Render("Link URL (optional):")
	urlCount := styleCharCount.Render(fmt.Sprintf("%d/%d", urlLen, urlLimit))
	f.WriteString(renderFormLabel(urlLabel, urlCount, contentWidth) + "\n")
	f.WriteString(urlStyle.Width(inputWidth).Render(m.urlInput.View()) + "\n\n")

	// Body Input + Counter
	bodyStyle := styleInputBlurred
	if m.postFormFocus == 2 {
		bodyStyle = styleInputFocused
	}
	bodyLen := len([]rune(m.bodyInput.Value()))
	bodyLimit := m.bodyInput.CharLimit
	bodyLabel := styleSubtitle.Render("Body (markdown supported):")
	bodyCount := styleCharCount.Render(fmt.Sprintf("%d/%d", bodyLen, bodyLimit))
	f.WriteString(renderFormLabel(bodyLabel, bodyCount, contentWidth) + "\n")
	f.WriteString(bodyStyle.Width(inputWidth).Render(m.bodyInput.View()))

	if m.err != nil {
		f.WriteString("\n\n" + styleError.Render("Error: "+sanitize.SingleLine(m.err.Error())))
	}

	cardStyle := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(colorPrimary).
		Padding(1, 2)
	card := cardStyle.Render(f.String())

	statusBar := m.renderStatusBar("/b/"+boardSlug, [][2]string{
		{"enter/tab", "next"},
		{"shift+tab", "prev"},
		{"ctrl+s", "publish"},
		{"esc", "cancel"},
	})

	return m.renderDockedViewWithCenteredContent(topHeader, card, statusBar)
}

func (m *Model) viewNewComment() string {
	target := "post"
	if m.replyParentAuthor != "" {
		target = fmt.Sprintf("@%s", sanitize.SingleLine(m.replyParentAuthor))
	}

	header := styleLogo.Render("ReadIT") +
		styleSubtitle.Render(fmt.Sprintf("  Replying to %s", target))
	ruleWidth := 70
	if m.width > 0 {
		ruleWidth = m.width
	}
	separator := styleRule.Render(strings.Repeat("─", ruleWidth))
	topHeader := header + "\n" + separator

	_, contentWidth, inputWidth := m.formDimensions()

	var f strings.Builder
	f.WriteString(stylePrompt.Render(fmt.Sprintf("Write Reply to %s", target)) + "\n\n")

	commLen := len([]rune(m.commentInput.Value()))
	commLimit := m.commentInput.CharLimit
	commLabel := styleSubtitle.Render("Reply:")
	commCount := styleCharCount.Render(fmt.Sprintf("%d/%d", commLen, commLimit))
	f.WriteString(renderFormLabel(commLabel, commCount, contentWidth) + "\n")
	f.WriteString(styleInputFocused.Width(inputWidth).Render(m.commentInput.View()))

	if m.err != nil {
		f.WriteString("\n\n" + styleError.Render("Error: "+sanitize.SingleLine(m.err.Error())))
	}

	cardStyle := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(colorPrimary).
		Padding(1, 2)
	card := cardStyle.Render(f.String())

	statusBar := m.renderStatusBar("reply", [][2]string{
		{"ctrl+s", "submit reply"},
		{"esc", "cancel"},
	})

	return m.renderDockedViewWithCenteredContent(topHeader, card, statusBar)
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

func (m *Model) renderDockedView(header, content, statusBar string) string {
	if m.width <= 0 || m.height <= 0 {
		return header + "\n" + content + "\n" + statusBar
	}

	header = strings.TrimRight(header, "\n")
	content = strings.TrimRight(content, "\n")
	statusBar = strings.TrimRight(statusBar, "\n")

	headerHeight := lipgloss.Height(header)
	statusHeight := lipgloss.Height(statusBar)
	availHeight := max(0, m.height-headerHeight-statusHeight)

	contentHeight := 0
	if content != "" {
		contentHeight = lipgloss.Height(content)
	}

	if contentHeight < availHeight {
		padding := strings.Repeat("\n", availHeight-contentHeight)
		content = content + padding
	}

	return header + "\n" + content + "\n" + statusBar
}

func (m *Model) renderDockedViewWithCenteredContent(header, card, statusBar string) string {
	if m.width <= 0 || m.height <= 0 {
		return header + "\n" + card + "\n" + statusBar
	}

	header = strings.TrimRight(header, "\n")
	statusBar = strings.TrimRight(statusBar, "\n")

	headerHeight := lipgloss.Height(header)
	statusHeight := lipgloss.Height(statusBar)
	availHeight := max(0, m.height-headerHeight-statusHeight)

	cardHeight := lipgloss.Height(card)
	var content string
	if cardHeight < availHeight {
		padTop := (availHeight - cardHeight) / 2
		padBottom := availHeight - cardHeight - padTop
		content = strings.Repeat("\n", padTop) + card + strings.Repeat("\n", padBottom)
	} else {
		content = card
	}

	horizCentered := lipgloss.PlaceHorizontal(m.width, lipgloss.Center, content)
	return header + "\n" + horizCentered + "\n" + statusBar
}

func (m *Model) renderStatusBar(left string, shortcuts [][2]string) string {
	if left == "" {
		left = "ReadIT"
	}
	leftBadge := styleStatusBadge.Render(left)

	var centerStr string
	if m.flashMsg != "" {
		centerStr = styleStatusFlash.Render(m.flashMsg)
	} else {
		centerStr = formatKeyPills(shortcuts)
	}

	var userStr string
	if m.user != nil {
		userStr = lipgloss.NewStyle().Foreground(colorAccent).Bold(true).Padding(0, 1).Render("@" + sanitize.SingleLine(m.user.Handle))
	}

	w := m.width
	if w <= 0 {
		if userStr != "" {
			return leftBadge + "  " + centerStr + "  " + userStr
		}
		return leftBadge + "  " + centerStr
	}

	lW := lipgloss.Width(leftBadge)
	rW := lipgloss.Width(userStr)
	cW := lipgloss.Width(centerStr)

	// Adaptive sizing for compact terminals
	if w < lW+cW+rW+4 {
		userStr = ""
		rW = 0
		if w < lW+cW+4 {
			maxCW := max(0, w-lW-2)
			centerStr = lipgloss.NewStyle().MaxWidth(maxCW).Render(centerStr)
			cW = lipgloss.Width(centerStr)
		}
	}

	totalSpaces := max(0, w-lW-cW-rW)
	spaceLeft := totalSpaces / 2
	spaceRight := totalSpaces - spaceLeft

	barContent := leftBadge + strings.Repeat(" ", spaceLeft) + centerStr + strings.Repeat(" ", spaceRight) + userStr
	return styleStatusBar.Width(w).Render(barContent)
}

func formatKeyPills(pairs [][2]string) string {
	var parts []string
	for _, p := range pairs {
		k := styleStatusKey.Render("[" + p[0] + "]")
		d := styleStatusDesc.Render(p[1])
		parts = append(parts, k+" "+d)
	}
	return strings.Join(parts, "  ")
}

func renderFormLabel(label, count string, targetWidth int) string {
	lW := lipgloss.Width(label)
	cW := lipgloss.Width(count)
	spaces := max(1, targetWidth-lW-cW)
	return label + strings.Repeat(" ", spaces) + count
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
