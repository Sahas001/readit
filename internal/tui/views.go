package tui

import (
	"fmt"
	"strings"
	"time"

	"github.com/charmbracelet/lipgloss"
	"github.com/sahas/readit/internal/sanitize"
)

const logo = `██████╗ ███████╗ █████╗ ██████╗ ██╗████████╗
██╔══██╗██╔════╝██╔══██╗██╔══██╗██║╚══██╔══╝
██████╔╝█████╗  ███████║██║  ██║██║   ██║   
██╔══██╗██╔══╝  ██╔══██║██║  ██║██║   ██║   
██║  ██║███████╗██║  ██║██████╔╝██║   ██║   
╚═╝  ╚═╝╚══════╝╚═╝  ╚═╝╚═════╝ ╚═╝   ╚═╝`

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

	// 1. Centered Hero Branding: Spinning Earth + Shining ReadIT Logo
	heroBanner := renderHeroBanner(m.animTick, contentWidth)
	b.WriteString(lipgloss.PlaceHorizontal(contentWidth, lipgloss.Center, heroBanner) + "\n\n")

	tagline := styleTagline.Render("A simple and lightweight forum in your terminal")
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
	countStr := fmt.Sprintf("%d discussions", len(m.posts))
	sortBadge := fmt.Sprintf("[s] %s", m.postSortMode.String())
	filterBadge := "[c] all"
	if m.categoryFilter != "" {
		filterBadge = fmt.Sprintf("[c] %s", m.categoryFilter)
	}
	infoStr := fmt.Sprintf("%s  •  %s  •  %s", countStr, sortBadge, filterBadge)
	headerRow := renderFormLabel(styleTitle.Render(boardBanner), styleSortPill.Render(infoStr), contentWidth)
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
			if post.IsDeleted {
				titleView = lipgloss.NewStyle().Foreground(currentTheme.TextMuted).Italic(true).Render("[deleted by author]")
			} else if isSelected {
				titleView = stylePostTitleSelected.Render(safeTitle)
			} else {
				titleView = stylePostTitle.Render(safeTitle)
			}

			timeStr := relativeTime(post.CreatedAt.Time)
			var authorStr string
			if post.IsDeleted {
				authorStr = lipgloss.NewStyle().Foreground(currentTheme.TextMuted).Italic(true).Render("[deleted]")
			} else {
				authorStr = "@" + sanitize.SingleLine(post.AuthorHandle)
			}
			commentsStr := fmt.Sprintf("%d comments", post.CommentCount)

			categoryStr := ""
			if !post.IsDeleted && post.Category != "" {
				categoryStr = styleCategoryBadge(post.Category).Render("[" + post.Category + "]")
			}

			metaLine := fmt.Sprintf("%s • %s • %s",
				styleMetaAuthor.Render(authorStr),
				styleMeta.Render(timeStr),
				styleMeta.Render(commentsStr),
			)
			if categoryStr != "" {
				metaLine += " • " + categoryStr
			}
			if !post.IsDeleted && post.Url != "" {
				metaLine += " • " + styleLinkBadge.Render("link")
			}

			contentBox := titleView + "\n" + metaLine
			postRow := lipgloss.JoinHorizontal(lipgloss.Top, voteCol, "   ", contentBox)

			cardWidth := max(10, contentWidth-1)
			var postCard string
			if isSelected {
				postCard = stylePostCardSelected.Width(cardWidth).Render(postRow)
			} else {
				postCard = stylePostCardNormal.Width(cardWidth).Render(postRow)
			}
			content.WriteString(postCard + "\n\n")
		}
	}

	shortcuts := [][2]string{
		{"j/k", "move"},
		{"enter", "view"},
		{"s", "sort"},
		{"c", "flair"},
		{"u/d", "vote"},
		{"n", "new post"},
		{"x", "delete"},
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
		{"s", "sort"},
		{"u/d", "vote"},
		{"x", "delete"},
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
	var title string
	if p.IsDeleted {
		title = styleTitle.Copy().Foreground(currentTheme.TextMuted).Italic(true).Render("[deleted]")
	} else {
		title = styleTitle.Copy().Foreground(currentTheme.Primary).Render(titleStr)
	}

	var author string
	if p.IsDeleted {
		author = lipgloss.NewStyle().Foreground(currentTheme.TextMuted).Italic(true).Render("[deleted]")
	} else {
		author = styleAuthor.Render("@" + sanitize.SingleLine(p.AuthorHandle))
	}

	timeStr := styleSubtitle.Render(relativeTime(p.CreatedAt.Time))
	meta := fmt.Sprintf("Posted by %s • %s", author, timeStr)
	if !p.IsDeleted && p.Category != "" {
		meta += " • " + styleCategoryBadge(p.Category).Render("["+p.Category+"]")
	}
	if p.IsDeleted {
		meta += " • " + lipgloss.NewStyle().Foreground(currentTheme.Upvote).Italic(true).Render("(deleted)")
	}

	if m.commentCursor == -1 {
		selectedIndicator := lipgloss.NewStyle().Foreground(currentTheme.Accent).Italic(true).Render("  ◄ post selected (r to reply)")
		title = title + selectedIndicator
	}

	contentBox := title + "\n" + meta
	if !p.IsDeleted && p.Url != "" {
		urlStr := styleSubtitle.Copy().Foreground(currentTheme.Accent).Render("Link: " + sanitize.SingleLine(p.Url))
		contentBox += "\n" + urlStr
	}

	headerRow := lipgloss.JoinHorizontal(lipgloss.Top, voteCol, "   ", contentBox)
	b.WriteString(headerRow + "\n\n")

	// 2. Post Body with Markdown Rendering
	if !p.IsDeleted && strings.TrimSpace(p.Body) != "" {
		bodyWidth := max(20, contentWidth-4)
		renderedBody := renderMarkdown(p.Body, bodyWidth)
		if renderedBody != "" {
			b.WriteString(lipgloss.NewStyle().PaddingLeft(2).Render(renderedBody) + "\n\n")
		}
	} else if p.IsDeleted {
		bodyWidth := max(20, contentWidth-4)
		bodyStyle := lipgloss.NewStyle().Foreground(currentTheme.TextMuted).Italic(true).PaddingLeft(2).Width(bodyWidth)
		b.WriteString(bodyStyle.Render("[This post has been deleted by author]") + "\n\n")
	}
	b.WriteString(styleRule.Render(strings.Repeat("─", max(10, contentWidth-4))) + "\n")

	// 3. Comments Header
	commentCount := len(m.comments)
	commentHeader := fmt.Sprintf("  Comments (%d)   •   [s] sort: %s", commentCount, m.commentSortMode.String())
	if contentWidth >= 65 {
		commentHeader += "   •   [r: reply • R: reply to post]"
	}
	b.WriteString(commentHeader + "\n")
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
		isOP := (m.currentPost != nil && !c.IsDeleted && !m.currentPost.IsDeleted && c.AuthorHandle == m.currentPost.AuthorHandle)

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
			if c.IsDeleted {
				cAuthor = lipgloss.NewStyle().Foreground(currentTheme.TextMuted).Italic(true).Render("[deleted]")
			} else {
				cAuthor = lipgloss.NewStyle().Foreground(currentTheme.Primary).Bold(true).Render("@" + sanitize.SingleLine(c.AuthorHandle))
			}
			selPill = lipgloss.NewStyle().Foreground(currentTheme.Accent).Italic(true).Render("  ◄ selected (r to reply)")
		} else {
			indicator = "  "
			branch = styleBranch.Render(branchGlyph)
			if c.IsDeleted {
				cAuthor = lipgloss.NewStyle().Foreground(currentTheme.TextMuted).Italic(true).Render("[deleted]")
			} else {
				cAuthor = styleAuthor.Render("@" + sanitize.SingleLine(c.AuthorHandle))
			}
			selPill = ""
		}

		cScore := styleScore.Copy().Width(0).Render(fmt.Sprintf("%d▲", c.Score))
		cTime := styleSubtitle.Render(relativeTime(c.CreatedAt.Time))

		b.WriteString(fmt.Sprintf("%s%s%s%s%s  %s  %s%s\n", indicator, indent, branch, cAuthor, opBadge, cScore, cTime, selPill))

		// Indent and wrap comment body lines with markdown rendering
		indentLen := depth*2 + 6
		availCommentWidth := max(20, contentWidth-indentLen)
		var wrappedBody string
		if c.IsDeleted {
			wrappedBody = lipgloss.NewStyle().Foreground(currentTheme.TextMuted).Italic(true).Width(availCommentWidth).Render("[deleted]")
		} else {
			wrappedBody = renderMarkdown(c.Body, availCommentWidth)
		}
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

	// Flair / Category Selector (Focus 1)
	catLabel := styleSubtitle.Render("Flair / Category (h/l or ←/→ to cycle):")
	var catPills strings.Builder
	for idx, cat := range AvailableCategories {
		isSelectedCat := (idx == m.newPostCategoryIdx)
		if isSelectedCat {
			if m.postFormFocus == 1 {
				catPills.WriteString(lipgloss.NewStyle().Foreground(lipgloss.Color("#FFFFFF")).Background(currentTheme.Primary).Bold(true).Render(" " + cat + " ") + " ")
			} else {
				catPills.WriteString(styleCategoryBadge(cat).Bold(true).Underline(true).Render("[" + cat + "]") + " ")
			}
		} else {
			catPills.WriteString(lipgloss.NewStyle().Foreground(currentTheme.TextDim).Render("[" + cat + "]") + " ")
		}
	}
	f.WriteString(catLabel + "\n")
	catBoxStyle := styleInputBlurred
	if m.postFormFocus == 1 {
		catBoxStyle = styleInputFocused
	}
	f.WriteString(catBoxStyle.Width(inputWidth).Render(strings.TrimSpace(catPills.String())) + "\n\n")

	// URL Input + Counter (Focus 2)
	urlStyle := styleInputBlurred
	if m.postFormFocus == 2 {
		urlStyle = styleInputFocused
	}
	urlLen := len([]rune(m.urlInput.Value()))
	urlLimit := m.urlInput.CharLimit
	urlLabel := styleSubtitle.Render("Link URL (optional):")
	urlCount := styleCharCount.Render(fmt.Sprintf("%d/%d", urlLen, urlLimit))
	f.WriteString(renderFormLabel(urlLabel, urlCount, contentWidth) + "\n")
	f.WriteString(urlStyle.Width(inputWidth).Render(m.urlInput.View()) + "\n\n")

	// Body Input + Counter (Focus 3)
	bodyStyle := styleInputBlurred
	if m.postFormFocus == 3 {
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
		{"h/l", "flair"},
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
func (m *Model) renderStatusBar(_ string, shortcuts [][2]string) string {
	var centerStr string
	if m.flashMsg != "" {
		centerStr = styleStatusFlash.Render(m.flashMsg)
	} else {
		centerStr = formatKeyPills(shortcuts)
	}

	w := m.width
	if w <= 0 {
		return centerStr
	}

	if w < lipgloss.Width(centerStr) {
		centerStr = formatAdaptiveKeyPills(shortcuts, w)
	}

	return lipgloss.PlaceHorizontal(w, lipgloss.Center, centerStr)
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

func (m *Model) viewDeleteConfirm() string {
	_, _, contentWidth, _ := m.shellDimensions()
	if m.pendingDelete == nil {
		return m.centeredView("No pending item to delete.")
	}

	target := m.pendingDelete
	cardWidth := min(64, max(38, contentWidth-4))
	innerContentWidth := max(24, cardWidth-6)

	var b strings.Builder

	titleStyle := lipgloss.NewStyle().
		Bold(true).
		Foreground(currentTheme.Negative)

	itemStyle := lipgloss.NewStyle().
		Foreground(currentTheme.Text).
		Bold(true).
		Width(innerContentWidth)

	descStyle := lipgloss.NewStyle().
		Foreground(currentTheme.TextMuted).
		Width(innerContentWidth)

	warnStyle := lipgloss.NewStyle().
		Foreground(currentTheme.Upvote).
		Width(innerContentWidth)

	safeItemSnippet := sanitize.SingleLine(target.titleOrBody)
	if len(safeItemSnippet) > innerContentWidth*2 {
		safeItemSnippet = safeItemSnippet[:innerContentWidth*2-3] + "..."
	}

	if target.targetType == deleteTargetPost {
		if target.hasDependents {
			b.WriteString(titleStyle.Render("⚠️  Delete Discussion?") + "\n\n")
			b.WriteString(itemStyle.Render(fmt.Sprintf("%q", safeItemSnippet)) + "\n\n")
			b.WriteString(warnStyle.Render(fmt.Sprintf("This post has %d active comments. The title, body, and author will be scrubbed to [deleted] to preserve thread continuity.", target.commentCount)) + "\n\n")
		} else {
			b.WriteString(titleStyle.Render("🗑  Permanently Delete Discussion?") + "\n\n")
			b.WriteString(itemStyle.Render(fmt.Sprintf("%q", safeItemSnippet)) + "\n\n")
			b.WriteString(descStyle.Render("This post has no comments. It will be completely removed from the database.") + "\n\n")
		}
	} else {
		if target.hasDependents {
			b.WriteString(titleStyle.Render("⚠️  Delete Comment?") + "\n\n")
			b.WriteString(itemStyle.Render(fmt.Sprintf("%q", safeItemSnippet)) + "\n\n")
			b.WriteString(warnStyle.Render("This comment has active replies. Its text and author will be replaced with [deleted] to preserve the conversation thread.") + "\n\n")
		} else {
			b.WriteString(titleStyle.Render("🗑  Permanently Delete Comment?") + "\n\n")
			b.WriteString(itemStyle.Render(fmt.Sprintf("%q", safeItemSnippet)) + "\n\n")
			b.WriteString(descStyle.Render("This comment has no replies. It will be completely removed from the database.") + "\n\n")
		}
	}

	b.WriteString(styleRule.Render(strings.Repeat("─", innerContentWidth)) + "\n\n")

	btnConfirm := lipgloss.NewStyle().
		Bold(true).
		Foreground(lipgloss.Color("#FFFFFF")).
		Background(currentTheme.Negative).
		Padding(0, 2).
		Render("[y] Confirm Delete")

	btnCancel := lipgloss.NewStyle().
		Foreground(currentTheme.TextMuted).
		Padding(0, 2).
		Render("[n / esc] Cancel")

	btnRow := lipgloss.JoinHorizontal(lipgloss.Center, btnConfirm, "  ", btnCancel)
	b.WriteString(lipgloss.PlaceHorizontal(innerContentWidth, lipgloss.Center, btnRow))

	card := styleModalCard.Width(cardWidth).Render(b.String())

	shortcuts := [][2]string{
		{"y", "confirm delete"},
		{"n/esc", "cancel"},
	}

	return m.renderAppShell("Confirm Deletion", lipgloss.PlaceHorizontal(contentWidth, lipgloss.Center, card), shortcuts)
}
