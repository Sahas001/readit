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
	var b strings.Builder
	b.WriteString(styleLogo.Render(logo) + "\n\n")
	b.WriteString(styleSubtitle.Render("Connecting to ReadIT SSH server...") + "\n")
	card := styleModalCard.Render(b.String())
	return m.centeredView(card)
}

func (m *Model) viewOnboarding() string {
	_, _, contentWidth, _ := m.shellDimensions()

	var f strings.Builder
	f.WriteString(stylePrompt.Render("Welcome to ReadIT!") + "\n\n")
	f.WriteString(styleSubtitle.Render("Your SSH public key is new here.") + "\n")
	f.WriteString(styleSubtitle.Render("Pick a unique handle to get started (3-20 characters):") + "\n\n")

	cardWidth := min(64, max(36, contentWidth-4))
	innerInputWidth := max(20, cardWidth-8)
	m.handleInput.Width = innerInputWidth

	f.WriteString(styleInputFocused.Width(innerInputWidth).Render(m.handleInput.View()) + "\n\n")
	f.WriteString(styleSubtitle.Render("Press [enter] to confirm  •  [esc] to quit"))

	if m.err != nil {
		f.WriteString("\n\n" + styleError.Render("Error: "+sanitize.SingleLine(m.err.Error())))
	}

	card := styleModalCard.Width(cardWidth).Render(f.String())
	shortcuts := [][2]string{
		{"enter", "confirm"},
		{"esc", "quit"},
	}

	return m.renderAppShell("Welcome", lipgloss.PlaceHorizontal(contentWidth, lipgloss.Center, card), shortcuts)
}

// viewBoardList renders the Reddit-inspired Terminal Landing Page.
func (m *Model) viewBoardList() string {
	_, _, contentWidth, _ := m.shellDimensions()

	var b strings.Builder

	// 1. Centered Hero Branding
	logoText := styleLogo.Render(logo)
	b.WriteString(lipgloss.PlaceHorizontal(contentWidth, lipgloss.Center, logoText) + "\n\n")

	tagline := styleTagline.Render("A Reddit-style forum in your terminal  •  SSH Edition")
	b.WriteString(lipgloss.PlaceHorizontal(contentWidth, lipgloss.Center, tagline) + "\n\n")

	// 2. Boards Directory Panel
	panelWidth := min(76, max(42, contentWidth-8))
	var boardRows strings.Builder

	headerLabel := styleTitle.Render("Community Boards")
	countLabel := styleSortPill.Render(fmt.Sprintf("%d boards", len(m.boards)))
	boardRows.WriteString(renderFormLabel(headerLabel, countLabel, panelWidth-6) + "\n")
	boardRows.WriteString(styleRule.Render(strings.Repeat("─", panelWidth-6)) + "\n\n")

	if len(m.boards) == 0 {
		boardRows.WriteString("  " + styleSubtitle.Render("No community boards found.") + "\n")
	} else {
		for i, board := range m.boards {
			safeSlug := sanitize.SingleLine(board.Slug)
			safeDesc := sanitize.SingleLine(board.Description)

			slugPart := fmt.Sprintf("/b/%-12s", safeSlug)
			descPart := safeDesc

			maxDescW := max(10, panelWidth-24)
			if lipgloss.Width(descPart) > maxDescW {
				descPart = descPart[:max(0, maxDescW-3)] + "..."
			}

			if i == m.boardCursor {
				line := lipgloss.NewStyle().
					Foreground(currentTheme.Primary).
					Bold(true).
					Render(fmt.Sprintf("▌ › %s   %s", slugPart, styleSubtitle.Render(descPart)))
				boardRows.WriteString(line + "\n")
			} else {
				line := fmt.Sprintf("    %s   %s", styleTitle.Render(slugPart), styleSubtitle.Render(descPart))
				boardRows.WriteString(line + "\n")
			}
		}
	}

	boardCard := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(currentTheme.Border).
		Padding(1, 2).
		Width(panelWidth).
		Render(boardRows.String())

	b.WriteString(lipgloss.PlaceHorizontal(contentWidth, lipgloss.Center, boardCard))

	shortcuts := [][2]string{
		{"j/k", "move"},
		{"enter", "enter board"},
		{"g/G", "top/end"},
		{"q", "quit"},
	}

	return m.renderAppShell("ReadIT", b.String(), shortcuts)
}

// viewPostList renders the rich Reddit-style discussion feed.
func (m *Model) viewPostList() string {
	_, _, contentWidth, contentHeight := m.shellDimensions()

	boardSlug := ""
	boardDesc := ""
	if m.currentBoard != nil {
		boardSlug = sanitize.SingleLine(m.currentBoard.Slug)
		boardDesc = sanitize.SingleLine(m.currentBoard.Description)
	}
	contextTitle := "/b/" + boardSlug

	var content strings.Builder

	// Board Subheader Banner
	boardBanner := fmt.Sprintf("/b/%s", boardSlug)
	if boardDesc != "" {
		boardBanner += "  ·  " + boardDesc
	}
	countStr := fmt.Sprintf("%d discussions  •  newest", len(m.posts))
	headerRow := renderFormLabel(styleTitle.Render(boardBanner), styleSortPill.Render(countStr), contentWidth)
	content.WriteString(headerRow + "\n")
	content.WriteString(styleRule.Render(strings.Repeat("─", contentWidth)) + "\n\n")

	if len(m.posts) == 0 {
		emptyMsg := fmt.Sprintf("\n  💬  No discussions yet in /b/%s\n\n  Be the first to start a conversation!\n  Press [n] to create a new discussion.\n", boardSlug)
		emptyCard := styleEmptyCard.Width(min(58, contentWidth-4)).Render(emptyMsg)
		content.WriteString(lipgloss.PlaceHorizontal(contentWidth, lipgloss.Center, emptyCard))
	} else {
		// Window posts: each post consumes ~3 lines (vote column + title + meta + spacing)
		availForPosts := max(3, contentHeight-3)
		postsPerPage := max(1, availForPosts/3)

		startIdx := 0
		if m.postCursor >= postsPerPage {
			startIdx = m.postCursor - postsPerPage + 1
		}
		endIdx := min(len(m.posts), startIdx+postsPerPage)

		for i := startIdx; i < endIdx; i++ {
			post := m.posts[i]
			isSelected := (i == m.postCursor)

			// Vote column
			var voteCol string
			if isSelected {
				voteCol = styleVoteUp.Render("▲") + "\n" +
					styleScore.Render(fmt.Sprintf("%2d", post.Score)) + "\n" +
					styleVoteDown.Render("▼")
			} else {
				voteCol = styleVoteNeutral.Render("▲") + "\n" +
					styleScore.Copy().Foreground(currentTheme.TextMuted).Render(fmt.Sprintf("%2d", post.Score)) + "\n" +
					styleVoteNeutral.Render("▼")
			}

			// Content column
			safeTitle := sanitize.SingleLine(post.Title)
			var titleView string
			if isSelected {
				titleView = stylePostTitleSelected.Render(safeTitle)
			} else {
				titleView = stylePostTitle.Render(safeTitle)
			}

			timeStr := relativeTime(post.CreatedAt.Time)
			authorStr := "@" + sanitize.SingleLine(post.AuthorHandle)
			commentsStr := fmt.Sprintf("%d comments", post.CommentCount)

			metaLine := fmt.Sprintf("%s • %s • %s",
				styleMetaAuthor.Render(authorStr),
				styleMeta.Render(timeStr),
				styleMeta.Render(commentsStr),
			)
			if post.Url != "" {
				metaLine += " • " + styleLinkBadge.Render("link")
			}

			contentBox := titleView + "\n" + metaLine
			postRow := lipgloss.JoinHorizontal(lipgloss.Top, voteCol, "   ", contentBox)

			if isSelected {
				cardWidth := contentWidth - 2
				postCard := stylePostCardSelected.Width(cardWidth).Render(postRow)
				content.WriteString(postCard + "\n\n")
			} else {
				postCard := stylePostCardNormal.Render(postRow)
				content.WriteString(postCard + "\n\n")
			}
		}
	}

	shortcuts := [][2]string{
		{"j/k", "move"},
		{"enter", "view"},
		{"u/d", "vote"},
		{"n", "new post"},
		{"g/G", "top/end"},
		{"esc", "boards"},
		{"q", "quit"},
	}

	return m.renderAppShell(contextTitle, content.String(), shortcuts)
}

func (m *Model) viewPostDetail() string {
	boardSlug := ""
	if m.currentBoard != nil {
		boardSlug = sanitize.SingleLine(m.currentBoard.Slug)
	}
	contextTitle := "/b/" + boardSlug + " · Discussion"

	shortcuts := [][2]string{
		{"j/k", "navigate"},
		{"r", "reply"},
		{"R", "reply root"},
		{"u/d", "vote"},
		{"g/G", "top/end"},
		{"esc", "back"},
		{"q", "quit"},
	}

	return m.renderAppShell(contextTitle, m.viewport.View(), shortcuts)
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

	// 1. Post Header with Vote Column & Content
	scoreStr := fmt.Sprintf("%2d", p.Score)
	voteCol := styleVoteUp.Render("▲") + "\n" +
		styleScore.Render(scoreStr) + "\n" +
		styleVoteDown.Render("▼")

	titleStr := sanitize.SingleLine(p.Title)
	title := styleTitle.Copy().Foreground(currentTheme.Primary).Render(titleStr)
	author := styleAuthor.Render("@" + sanitize.SingleLine(p.AuthorHandle))
	timeStr := styleSubtitle.Render(relativeTime(p.CreatedAt.Time))
	meta := fmt.Sprintf("Posted by %s • %s", author, timeStr)

	if m.commentCursor == -1 {
		selectedIndicator := lipgloss.NewStyle().Foreground(currentTheme.Accent).Italic(true).Render("  ◄ post selected (r to reply)")
		title = title + selectedIndicator
	}

	contentBox := title + "\n" + meta
	if p.Url != "" {
		urlStr := styleSubtitle.Copy().Foreground(currentTheme.Accent).Render("Link: " + sanitize.SingleLine(p.Url))
		contentBox += "\n" + urlStr
	}

	headerRow := lipgloss.JoinHorizontal(lipgloss.Top, voteCol, "   ", contentBox)
	b.WriteString(headerRow + "\n\n")

	// 2. Post Body with wrapping
	if p.Body != "" {
		bodyWidth := max(20, contentWidth-6)
		bodyStyle := stylePostBody.Copy().Width(bodyWidth)
		b.WriteString(bodyStyle.Render(sanitize.Text(p.Body)) + "\n")
	}
	b.WriteString("\n" + styleRule.Render(strings.Repeat("─", max(10, contentWidth-4))) + "\n")

	// 3. Comments Header
	commentCount := len(m.comments)
	b.WriteString(fmt.Sprintf("  Comments (%d)   •   [r: reply to selected • R: reply to post]\n", commentCount))
	b.WriteString(styleRule.Render(strings.Repeat("─", max(10, contentWidth-4))) + "\n\n")

	if commentCount == 0 {
		b.WriteString(styleSubtitle.Render("    💬  No comments yet. Be the first to reply (press r)!") + "\n")
		return b.String()
	}

	m.commentLineOffsets = make([]int, len(m.comments))

	// 4. Threaded Comments Tree
	for i, c := range m.comments {
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
			indicator = lipgloss.NewStyle().Foreground(currentTheme.Primary).Bold(true).Render("▌ ")
			branch = styleSelectedBranch.Render(branchGlyph)
			cAuthor = lipgloss.NewStyle().Foreground(currentTheme.Primary).Bold(true).Render("@" + sanitize.SingleLine(c.AuthorHandle))
			selPill = lipgloss.NewStyle().Foreground(currentTheme.Accent).Italic(true).Render("  ◄ selected (r to reply)")
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
			bodyPrefix = fmt.Sprintf("%s%s   ", indicator, indent)
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
	contextTitle := "/b/" + boardSlug + " · New Discussion"

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
		BorderForeground(currentTheme.Border).
		Padding(1, 2).
		Background(currentTheme.CardBg)
	card := cardStyle.Render(f.String())

	shortcuts := [][2]string{
		{"enter/tab", "next"},
		{"shift+tab", "prev"},
		{"ctrl+s", "publish"},
		{"esc", "cancel"},
	}

	return m.renderAppShell(contextTitle, lipgloss.PlaceHorizontal(contentWidth, lipgloss.Center, card), shortcuts)
}

func (m *Model) viewNewComment() string {
	target := "post"
	if m.replyParentAuthor != "" {
		target = fmt.Sprintf("@%s", sanitize.SingleLine(m.replyParentAuthor))
	}
	contextTitle := "Reply to " + target

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
		BorderForeground(currentTheme.Border).
		Padding(1, 2).
		Background(currentTheme.CardBg)
	card := cardStyle.Render(f.String())

	shortcuts := [][2]string{
		{"ctrl+s", "submit reply"},
		{"esc", "cancel"},
	}

	return m.renderAppShell(contextTitle, lipgloss.PlaceHorizontal(contentWidth, lipgloss.Center, card), shortcuts)
}

func (m *Model) viewError() string {
	errText := "unknown error"
	if m.err != nil {
		errText = sanitize.SingleLine(m.err.Error())
	}
	var b strings.Builder
	b.WriteString(styleError.Render("⚠ Application Error") + "\n\n")
	b.WriteString(styleSubtitle.Render(errText) + "\n\n")
	b.WriteString(styleSubtitle.Render("Press [esc] or [q] to return."))
	card := styleErrorCard.Render(b.String())
	return m.centeredView(card)
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

// renderDockedView preserved for helper compatibility and line testing.
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

// renderDockedViewWithCenteredContent preserved for helper compatibility and line testing.
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

// renderStatusBar preserved for status bar helper tests.
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
		userStr = lipgloss.NewStyle().Foreground(currentTheme.Accent).Bold(true).Padding(0, 1).Render("@" + sanitize.SingleLine(m.user.Handle))
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

	if w < lW+cW+rW+4 {
		userStr = ""
		rW = 0
		if w < lW+cW+4 {
			maxCW := max(0, w-lW-2)
			centerStr = formatAdaptiveKeyPills(shortcuts, maxCW)
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
