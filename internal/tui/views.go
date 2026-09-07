package tui

import (
	"fmt"
	"strings"
	"time"

	"github.com/charmbracelet/lipgloss"
	db "github.com/sahas/readit/internal/db/sqlc"
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
		{"enter", "open"},
		{"?", "help"},
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
	// 1. Left: Board description (shifted left, /b/slug removed)
	leftTitle := boardDesc
	if leftTitle == "" {
		leftTitle = "/b/" + boardSlug
	}
	leftView := styleTitle.Render(leftTitle)

	// 2. Right: Counts + Page + Sort + Flair + Density
	countLabel := "discussions"
	if m.searchQuery != "" {
		countLabel = "matches"
	}
	countStr := fmt.Sprintf("%d %s", len(m.posts), countLabel)
	pageStr := fmt.Sprintf("pg %d", m.feedPage)
	if m.hasNextPage {
		pageStr += " ▶"
	}
	sortBadge := strings.Title(m.postSortMode.String())
	filterBadge := "All"
	if m.categoryFilter != "" {
		filterBadge = m.categoryFilter
	}
	densityBadge := "[compact: z]"
	if m.compactMode {
		densityBadge = "[comfort: z]"
	}
	hideBadge := ""
	if m.hideRead {
		hideBadge = "[hide-read: H]  •  "
	}
	infoStr := fmt.Sprintf("%s%s  •  %s  •  %s  •  %s  •  %s", hideBadge, countStr, pageStr, sortBadge, filterBadge, densityBadge)
	rightView := styleSortPill.Render(infoStr)

	headerRow := renderTwoColumnHeader(leftView, rightView, contentWidth)
	content.WriteString(headerRow + "\n")
	content.WriteString(styleRule.Render(strings.Repeat("─", contentWidth)) + "\n")

	// Dedicated Search Filter Bar (Active when focused or filtered)
	if m.searchFocused {
		hint := styleFilterHint.Render("enter search  •  esc cancel")
		avail := max(10, contentWidth-lipgloss.Width(hint)-2)
		m.searchInput.Width = avail
		filterInput := m.searchInput.View()
		spaces := max(1, contentWidth-lipgloss.Width(filterInput)-lipgloss.Width(hint))
		content.WriteString(filterInput + strings.Repeat(" ", spaces) + hint + "\n")
		content.WriteString(styleRule.Render(strings.Repeat("─", contentWidth)) + "\n")
	} else if m.searchQuery != "" {
		queryPart := styleFilterPrompt.Render("filter: ") + styleFilterQuery.Render(fmt.Sprintf("%q", m.searchQuery))
		matchesPart := styleFilterHint.Render(fmt.Sprintf("(%d matches)  •  esc to clear", len(m.posts)))
		spaces := max(1, contentWidth-lipgloss.Width(queryPart)-lipgloss.Width(matchesPart))
		content.WriteString(queryPart + strings.Repeat(" ", spaces) + matchesPart + "\n")
		content.WriteString(styleRule.Render(strings.Repeat("─", contentWidth)) + "\n")
	} else {
		content.WriteString("\n")
	}

	displayPosts := m.visiblePosts()

	if len(displayPosts) == 0 {
		var emptyMsg string
		if m.hideRead && len(m.posts) > 0 {
			emptyMsg = "\n  ✓  All discussions on this page are marked as read!\n\n  Press [H] to unhide or [ ] ] to navigate to the next page.\n"
		} else if m.searchQuery != "" {
			emptyMsg = fmt.Sprintf("\n  •  No discussions found for %q\n\n  Try searching for a different keyword\n  or press [esc] to clear the search filter.\n", sanitize.SingleLine(m.searchQuery))
		} else {
			emptyMsg = fmt.Sprintf("\n  •  No discussions yet in /b/%s\n\n  Be the first to start a conversation!\n  Press [n] to create a new discussion.\n", boardSlug)
		}
		emptyCard := styleEmptyCard.Width(min(58, contentWidth-4)).Render(emptyMsg)
		content.WriteString(lipgloss.PlaceHorizontal(contentWidth, lipgloss.Center, emptyCard))
	} else {
		availForPosts := max(3, contentHeight-3)
		if m.searchFocused || m.searchQuery != "" {
			availForPosts = max(3, contentHeight-5)
		}

		if m.compactMode {
			// Compact 1-line table view
			postsPerPage := max(1, availForPosts)
			startIdx := 0
			if m.postCursor >= postsPerPage {
				startIdx = m.postCursor - postsPerPage + 1
			}
			endIdx := min(len(displayPosts), startIdx+postsPerPage)

			for i := startIdx; i < endIdx; i++ {
				post := displayPosts[i]
				isSelected := (i == m.postCursor)
				isRead := m.readPosts[post.ID]

				sel := "  "
				if isSelected {
					sel = styleSelectedBranch.Render("▌ ")
				}

				scoreStr := fmt.Sprintf("%d", post.Score)
				var votePill string
				if post.Score > 0 {
					if isSelected {
						votePill = styleVoteUp.Render("▲") + styleScore.Copy().Width(0).Render(scoreStr)
					} else {
						votePill = styleVoteUp.Render("▲") + styleScore.Copy().Width(0).Foreground(currentTheme.TextMuted).Render(scoreStr)
					}
				} else if post.Score < 0 {
					votePill = styleVoteDown.Render("▼") + styleScore.Copy().Width(0).Foreground(currentTheme.Downvote).Render(scoreStr)
				} else {
					votePill = styleVoteNeutral.Render("▲") + styleScore.Copy().Width(0).Foreground(currentTheme.TextMuted).Render(scoreStr)
				}
				voteFormatted := lipgloss.NewStyle().Width(6).Align(lipgloss.Right).Render(votePill)

				categoryStr := ""
				if !post.IsDeleted && post.Category != "" {
					categoryStr = styleCategoryBadge(post.Category).Render("["+post.Category+"]") + " "
				}

				safeTitle := sanitize.SingleLine(post.Title)
				var titleView string
				if post.IsDeleted {
					titleView = lipgloss.NewStyle().Foreground(currentTheme.TextMuted).Italic(true).Render("[deleted by author]")
				} else if isSelected {
					titleView = stylePostTitleSelected.Render(safeTitle)
				} else if isRead {
					titleView = lipgloss.NewStyle().Foreground(currentTheme.TextMuted).Render(safeTitle)
				} else {
					titleView = stylePostTitle.Render(safeTitle)
				}

				authorStr := "@" + sanitize.SingleLine(post.AuthorHandle)
				if post.IsDeleted {
					authorStr = "[deleted]"
				}
				readMark := ""
				if isRead {
					readMark = "✓ "
				}
				metaLine := styleMeta.Render(fmt.Sprintf("%s%s • %s • %dc", readMark, authorStr, relativeTime(post.CreatedAt.Time), post.CommentCount))

				leftPart := sel + voteFormatted + " " + categoryStr
				rightPart := "  " + metaLine
				availTitle := max(10, contentWidth-lipgloss.Width(leftPart)-lipgloss.Width(rightPart))
				titleFormatted := lipgloss.NewStyle().MaxWidth(availTitle).Render(titleView)
				spaces := max(1, contentWidth-lipgloss.Width(leftPart)-lipgloss.Width(titleFormatted)-lipgloss.Width(rightPart))

				content.WriteString(leftPart + titleFormatted + strings.Repeat(" ", spaces) + metaLine + "\n")
			}
		} else {
			// Comfortable 3-line card view
			postsPerPage := max(1, availForPosts/3)
			startIdx := 0
			if m.postCursor >= postsPerPage {
				startIdx = m.postCursor - postsPerPage + 1
			}
			endIdx := min(len(displayPosts), startIdx+postsPerPage)

			for i := startIdx; i < endIdx; i++ {
				post := displayPosts[i]
				isSelected := (i == m.postCursor)
				isRead := m.readPosts[post.ID]

				scoreStr := fmt.Sprintf("%d", post.Score)
				var votePill string
				if post.Score > 0 {
					if isSelected {
						votePill = styleVoteUp.Render("▲") + " " + styleScore.Copy().Width(0).Render(scoreStr)
					} else {
						votePill = styleVoteUp.Render("▲") + " " + styleScore.Copy().Width(0).Foreground(currentTheme.TextMuted).Render(scoreStr)
					}
				} else if post.Score < 0 {
					if isSelected {
						votePill = styleVoteDown.Render("▼") + " " + styleScore.Copy().Width(0).Foreground(currentTheme.Downvote).Render(scoreStr)
					} else {
						votePill = styleVoteDown.Render("▼") + " " + styleScore.Copy().Width(0).Foreground(currentTheme.TextMuted).Render(scoreStr)
					}
				} else {
					if isSelected {
						votePill = styleVoteNeutral.Render("▲") + " " + styleScore.Copy().Width(0).Render(scoreStr)
					} else {
						votePill = styleVoteNeutral.Render("▲") + " " + styleScore.Copy().Width(0).Foreground(currentTheme.TextMuted).Render(scoreStr)
					}
				}

				safeTitle := sanitize.SingleLine(post.Title)
				var titleView string
				if post.IsDeleted {
					titleView = lipgloss.NewStyle().Foreground(currentTheme.TextMuted).Italic(true).Render("[deleted by author]")
				} else if isSelected {
					titleView = stylePostTitleSelected.Render(safeTitle)
				} else if isRead {
					titleView = lipgloss.NewStyle().Foreground(currentTheme.TextMuted).Render(safeTitle)
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
				if isRead {
					authorStr = "✓ " + authorStr
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
					metaLine = categoryStr + "  " + metaLine
				}

				voteWidth := 6
				voteFormatted := lipgloss.NewStyle().Width(voteWidth).Align(lipgloss.Right).Render(votePill)
				spacer := strings.Repeat(" ", voteWidth)
				line1 := voteFormatted + "  " + titleView
				line2 := spacer + "  " + metaLine
				postBox := line1 + "\n" + line2

				cardWidth := max(10, contentWidth-1)
				var postCard string
				if isSelected {
					postCard = stylePostCardSelected.Width(cardWidth).Render(postBox)
				} else {
					postCard = stylePostCardNormal.Width(cardWidth).Render(postBox)
				}
				content.WriteString(postCard + "\n\n")
			}
		}
	}

	var shortcuts [][2]string
	if m.searchFocused {
		shortcuts = [][2]string{
			{"enter", "search"},
			{"esc", "cancel"},
		}
	} else if m.searchQuery != "" {
		shortcuts = [][2]string{
			{"j/k", "move"},
			{"enter", "open"},
			{"n", "new"},
			{"esc", "clear"},
			{"?", "help"},
			{"q", "quit"},
		}
	} else {
		shortcuts = [][2]string{
			{"j/k", "move"},
			{"enter", "open"},
			{"n", "new"},
			{"/", "filter"},
			{"?", "help"},
			{"q", "quit"},
		}
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
		{"j/k", "move"},
		{"r", "reply"},
		{"u/d", "vote"},
		{"esc", "back"},
		{"?", "help"},
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
	voteCol := styleVoteUp.Copy().Width(5).Align(lipgloss.Center).Render("▲") + "\n" +
		styleScore.Render(scoreStr) + "\n" +
		styleVoteDown.Copy().Width(5).Align(lipgloss.Center).Render("▼")

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

	cursorIndicator := "  "
	if m.commentCursor == -1 {
		cursorIndicator = lipgloss.NewStyle().Foreground(currentTheme.Primary).Bold(true).Render("▌ ")
	}

	contentBox := title + "\n" + meta
	if !p.IsDeleted && p.Url != "" {
		urlStr := styleSubtitle.Copy().Foreground(currentTheme.Accent).Render("Link: " + sanitize.SingleLine(p.Url))
		contentBox += "\n" + urlStr
	}

	headerRow := lipgloss.JoinHorizontal(lipgloss.Top, cursorIndicator, voteCol, "  ", contentBox)
	b.WriteString(headerRow + "\n\n")

	// 2. Post Body with Markdown Rendering
	if !p.IsDeleted && strings.TrimSpace(p.Body) != "" {
		bodyWidth := max(20, contentWidth-4)
		renderedBody := renderMarkdown(p.Body, bodyWidth)
		if renderedBody != "" {
			b.WriteString(lipgloss.NewStyle().PaddingLeft(4).Render(renderedBody) + "\n\n")
		}
	} else if p.IsDeleted {
		bodyWidth := max(20, contentWidth-4)
		bodyStyle := lipgloss.NewStyle().Foreground(currentTheme.TextMuted).Italic(true).PaddingLeft(4).Width(bodyWidth)
		b.WriteString(bodyStyle.Render("[This post has been deleted by author]") + "\n\n")
	}
	b.WriteString(styleRule.Render(strings.Repeat("─", max(10, contentWidth-4))) + "\n")

	// 3. Comments Header
	commentCount := len(m.comments)
	commentHeader := fmt.Sprintf("  Comments (%d)      •      sort: %s", commentCount, m.commentSortMode.String())
	b.WriteString(commentHeader + "\n")
	b.WriteString(styleRule.Render(strings.Repeat("─", max(10, contentWidth-4))) + "\n\n")

	if commentCount == 0 {
		b.WriteString(styleSubtitle.Render("    •  No comments yet. Press [r] to reply!") + "\n")
		return b.String()
	}

	m.commentLineOffsets = make([]int, len(m.comments))

	// 4. Threaded Comments Tree with True Hierarchy Trunks
	for i, c := range m.comments {
		m.commentLineOffsets[i] = strings.Count(b.String(), "\n")

		isSelected := (i == m.commentCursor)
		isOP := (m.currentPost != nil && !c.IsDeleted && !m.currentPost.IsDeleted && c.AuthorHandle == m.currentPost.AuthorHandle)

		var opBadge string
		if isOP {
			opBadge = " " + styleOpBadge.Render("[OP]")
		}

		branchStr := renderTreePrefix(m.comments, i)
		var indicator string
		var branch string
		var cAuthor string

		if isSelected {
			indicator = lipgloss.NewStyle().Foreground(currentTheme.Primary).Bold(true).Render("▌ ")
			branch = styleSelectedBranch.Render(branchStr)
			if c.IsDeleted {
				cAuthor = lipgloss.NewStyle().Foreground(currentTheme.TextMuted).Italic(true).Render("[deleted]")
			} else {
				cAuthor = lipgloss.NewStyle().Foreground(currentTheme.Primary).Bold(true).Render("@" + sanitize.SingleLine(c.AuthorHandle))
			}
		} else {
			indicator = "  "
			branch = styleBranch.Render(branchStr)
			if c.IsDeleted {
				cAuthor = lipgloss.NewStyle().Foreground(currentTheme.TextMuted).Italic(true).Render("[deleted]")
			} else {
				cAuthor = styleAuthor.Render("@" + sanitize.SingleLine(c.AuthorHandle))
			}
		}

		cScore := styleScore.Copy().Width(0).Render(fmt.Sprintf("%d▲", c.Score))
		cTime := styleSubtitle.Render(relativeTime(c.CreatedAt.Time))

		b.WriteString(fmt.Sprintf("%s%s%s%s  %s  %s\n", indicator, branch, cAuthor, opBadge, cScore, cTime))

		// Indent and wrap comment body lines with markdown rendering
		branchWidth := lipgloss.Width(branchStr)
		indentLen := branchWidth + 2
		availCommentWidth := max(20, contentWidth-indentLen)
		var wrappedBody string
		if c.IsDeleted {
			wrappedBody = lipgloss.NewStyle().Foreground(currentTheme.TextMuted).Italic(true).Width(availCommentWidth).Render("[deleted]")
		} else {
			wrappedBody = renderMarkdown(c.Body, availCommentWidth)
		}
		bodyLines := strings.Split(wrappedBody, "\n")

		bodyPrefix := fmt.Sprintf("%s%s", indicator, strings.Repeat(" ", branchWidth))
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

	cardWidth, contentWidth, inputWidth := m.formDimensions()
	_, _, shellContentWidth, _ := m.shellDimensions()

	var f strings.Builder

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
	catLabel := styleSubtitle.Render("Flair:")
	if m.postFormFocus == 1 {
		catLabel = lipgloss.NewStyle().Foreground(currentTheme.Primary).Bold(true).Render("Flair (h/l):")
	}
	var catPills strings.Builder
	for idx, cat := range AvailableCategories {
		isSelectedCat := (idx == m.newPostCategoryIdx)
		if isSelectedCat {
			if m.postFormFocus == 1 {
				catPills.WriteString(lipgloss.NewStyle().Foreground(lipgloss.Color("#FFFFFF")).Background(currentTheme.Primary).Bold(true).Render(" "+cat+" ") + " ")
			} else {
				catPills.WriteString(styleCategoryBadge(cat).Bold(true).Underline(true).Render("["+cat+"]") + " ")
			}
		} else {
			catPills.WriteString(lipgloss.NewStyle().Foreground(currentTheme.TextDim).Render("["+cat+"]") + " ")
		}
	}
	pillsText := strings.TrimSpace(catPills.String())
	if lipgloss.Width(catLabel)+1+lipgloss.Width(pillsText) <= contentWidth {
		f.WriteString(catLabel + " " + pillsText + "\n\n")
	} else {
		f.WriteString(catLabel + "\n" + pillsText + "\n\n")
	}

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
		Padding(0, 2)
	card := cardStyle.Width(cardWidth).Render(f.String())

	shortcuts := [][2]string{
		{"enter/tab", "next"},
		{"shift+tab", "prev"},
		{"h/l", "flair"},
		{"ctrl+s", "publish"},
		{"esc", "cancel"},
	}

	return m.renderAppShell(contextTitle, lipgloss.PlaceHorizontal(shellContentWidth, lipgloss.Center, card), shortcuts)
}

func (m *Model) viewNewComment() string {
	target := "post"
	if m.replyParentAuthor != "" {
		target = fmt.Sprintf("@%s", sanitize.SingleLine(m.replyParentAuthor))
	}
	contextTitle := "Reply to " + target

	cardWidth, contentWidth, inputWidth := m.formDimensions()
	_, _, shellContentWidth, _ := m.shellDimensions()

	var f strings.Builder

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
		Padding(0, 2)
	card := cardStyle.Width(cardWidth).Render(f.String())

	shortcuts := [][2]string{
		{"ctrl+s", "submit reply"},
		{"esc", "cancel"},
	}

	return m.renderAppShell(contextTitle, lipgloss.PlaceHorizontal(shellContentWidth, lipgloss.Center, card), shortcuts)
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

func renderTwoColumnHeader(left, right string, targetWidth int) string {
	lW := lipgloss.Width(left)
	rW := lipgloss.Width(right)
	if lW+rW+2 <= targetWidth {
		gap := targetWidth - lW - rW
		return left + strings.Repeat(" ", gap) + right
	}
	availForLeft := max(4, targetWidth-rW-2)
	leftTrunc := lipgloss.NewStyle().MaxWidth(availForLeft).Render(left)
	lW = lipgloss.Width(leftTrunc)
	gap := max(1, targetWidth-lW-rW)
	return leftTrunc + strings.Repeat(" ", gap) + right
}

// renderTreePrefix builds proper tree branch continuation lines with ancestor trunks
func renderTreePrefix(comments []db.GetCommentThreadByPostRow, idx int) string {
	c := comments[idx]
	depth := int(c.Depth)
	if depth <= 0 {
		return "• "
	}

	ancestorHasNext := make([]bool, depth)
	for d := 1; d < depth; d++ {
		hasNext := false
		for j := idx + 1; j < len(comments); j++ {
			if int(comments[j].Depth) < d {
				break
			}
			if int(comments[j].Depth) == d {
				hasNext = true
				break
			}
		}
		ancestorHasNext[d] = hasNext
	}

	isLast := true
	for j := idx + 1; j < len(comments); j++ {
		if int(comments[j].Depth) < depth {
			break
		}
		if int(comments[j].Depth) == depth {
			isLast = false
			break
		}
	}

	var b strings.Builder
	for d := 1; d < depth; d++ {
		if ancestorHasNext[d] {
			b.WriteString("│  ")
		} else {
			b.WriteString("   ")
		}
	}
	if isLast {
		b.WriteString("└─ ")
	} else {
		b.WriteString("├─ ")
	}
	return b.String()
}

func renderThreeColumnHeader(left, center, right string, targetWidth int) string {
	lW := lipgloss.Width(left)
	cW := lipgloss.Width(center)
	rW := lipgloss.Width(right)

	if lW+cW+rW+2 <= targetWidth {
		centerStart := (targetWidth - cW) / 2
		centerEnd := centerStart + cW
		if centerStart > lW && centerEnd < targetWidth-rW {
			gap1 := centerStart - lW
			gap2 := targetWidth - rW - centerEnd
			return left + strings.Repeat(" ", gap1) + center + strings.Repeat(" ", gap2) + right
		}
		totalSpaces := targetWidth - lW - cW - rW
		gap1 := totalSpaces / 2
		gap2 := totalSpaces - gap1
		return left + strings.Repeat(" ", gap1) + center + strings.Repeat(" ", gap2) + right
	}

	availForLeft := targetWidth - cW - rW - 2
	if availForLeft >= 6 {
		leftTrunc := lipgloss.NewStyle().MaxWidth(availForLeft).Render(left)
		lW = lipgloss.Width(leftTrunc)
		gap1 := 1
		gap2 := max(1, targetWidth-lW-cW-rW-gap1)
		return leftTrunc + strings.Repeat(" ", gap1) + center + strings.Repeat(" ", gap2) + right
	}

	if cW > 0 && targetWidth-cW-rW >= 1 {
		return center + strings.Repeat(" ", max(1, targetWidth-cW-rW)) + right
	}
	return renderFormLabel(left, right, targetWidth)
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
			b.WriteString(titleStyle.Render("!  Delete Discussion?") + "\n\n")
			b.WriteString(itemStyle.Render(fmt.Sprintf("%q", safeItemSnippet)) + "\n\n")
			b.WriteString(warnStyle.Render(fmt.Sprintf("This post has %d active comments. The title, body, and author will be scrubbed to [deleted] to preserve thread continuity.", target.commentCount)) + "\n\n")
		} else {
			b.WriteString(titleStyle.Render("✕  Permanently Delete Discussion?") + "\n\n")
			b.WriteString(itemStyle.Render(fmt.Sprintf("%q", safeItemSnippet)) + "\n\n")
			b.WriteString(descStyle.Render("This post has no comments. It will be completely removed from the database.") + "\n\n")
		}
	} else {
		if target.hasDependents {
			b.WriteString(titleStyle.Render("!  Delete Comment?") + "\n\n")
			b.WriteString(itemStyle.Render(fmt.Sprintf("%q", safeItemSnippet)) + "\n\n")
			b.WriteString(warnStyle.Render("This comment has active replies. Its text and author will be replaced with [deleted] to preserve the conversation thread.") + "\n\n")
		} else {
			b.WriteString(titleStyle.Render("✕  Permanently Delete Comment?") + "\n\n")
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
		Render("[esc] Cancel")

	btnRow := lipgloss.JoinHorizontal(lipgloss.Center, btnConfirm, "  ", btnCancel)
	b.WriteString(lipgloss.PlaceHorizontal(innerContentWidth, lipgloss.Center, btnRow))

	card := styleModalCard.Width(cardWidth).Render(b.String())

	shortcuts := [][2]string{
		{"y", "confirm delete"},
		{"n/esc", "cancel"},
	}

	return m.renderAppShell("Confirm Deletion", lipgloss.PlaceHorizontal(contentWidth, lipgloss.Center, card), shortcuts)
}

func formatHelpItem(key, desc string, maxW int) string {
	k := styleStatusKey.Render(fmt.Sprintf("%-10s", key))
	d := styleStatusDesc.Render(desc)
	line := k + " " + d
	if maxW > 0 && lipgloss.Width(line) > maxW {
		avail := max(5, maxW-lipgloss.Width(k)-1)
		line = k + " " + lipgloss.NewStyle().MaxWidth(avail).Render(d)
	}
	return line
}

// viewHelp renders the full keyboard shortcut cheatsheet modal.
func (m *Model) viewHelp() string {
	_, _, contentWidth, _ := m.shellDimensions()

	cardWidth := min(76, max(48, contentWidth-2))
	innerWidth := cardWidth - 6 // accounting for card border + padding

	var content string
	if innerWidth >= 58 {
		// 2-column layout
		colW := (innerWidth - 2) / 2

		// Left Column: Navigation & Triage
		var left strings.Builder
		left.WriteString(stylePrompt.Render("NAVIGATION & FEED") + "\n")
		left.WriteString(formatHelpItem("j / k", "Move cursor up / down", colW) + "\n")
		left.WriteString(formatHelpItem("enter", "Open post / board", colW) + "\n")
		left.WriteString(formatHelpItem("[ / ]", "Prev / next page", colW) + "\n")
		left.WriteString(formatHelpItem("ctrl+d/u", "Jump half page", colW) + "\n")
		left.WriteString(formatHelpItem("g / G", "Jump to top / bottom", colW) + "\n")
		left.WriteString(formatHelpItem("esc", "Back to board list", colW) + "\n\n")

		left.WriteString(stylePrompt.Render("TRIAGE & FILTER") + "\n")
		left.WriteString(formatHelpItem("/", "Search / filter posts", colW) + "\n")
		left.WriteString(formatHelpItem("s", "Cycle sort (Hot/New/Top)", colW) + "\n")
		left.WriteString(formatHelpItem("c", "Cycle category flair", colW) + "\n")
		left.WriteString(formatHelpItem("z", "Comfortable / compact", colW) + "\n")
		left.WriteString(formatHelpItem("m", "Mark read / unread", colW) + "\n")
		left.WriteString(formatHelpItem("H", "Toggle hide read", colW))

		// Right Column: Discussions & General
		var right strings.Builder
		right.WriteString(stylePrompt.Render("DISCUSSIONS & COMMENTS") + "\n")
		right.WriteString(formatHelpItem("r", "Reply to item", colW) + "\n")
		right.WriteString(formatHelpItem("R", "Reply to root post", colW) + "\n")
		right.WriteString(formatHelpItem("u / d", "Upvote / downvote", colW) + "\n")
		right.WriteString(formatHelpItem("s", "Sort comments (Top/New/Old)", colW) + "\n")
		right.WriteString(formatHelpItem("tab", "Next comment", colW) + "\n")
		right.WriteString(formatHelpItem("shift+tab", "Previous comment", colW) + "\n")
		right.WriteString(formatHelpItem("x", "Delete own post/comment", colW) + "\n\n")

		right.WriteString(stylePrompt.Render("COMPOSERS & GLOBAL") + "\n")
		right.WriteString(formatHelpItem("n", "New discussion post", colW) + "\n")
		right.WriteString(formatHelpItem("ctrl+s", "Publish post / reply", colW) + "\n")
		right.WriteString(formatHelpItem("h / l", "Select flair in composer", colW) + "\n")
		right.WriteString(formatHelpItem("?", "Close this cheatsheet", colW) + "\n")
		right.WriteString(formatHelpItem("q", "Quit ReadIT", colW))

		leftBlock := lipgloss.NewStyle().Width(colW).Render(left.String())
		rightBlock := lipgloss.NewStyle().Width(colW).Render(right.String())
		content = lipgloss.JoinHorizontal(lipgloss.Top, leftBlock, "  ", rightBlock)
	} else {
		// Single-column layout for narrow terminals
		var s strings.Builder
		s.WriteString(stylePrompt.Render("NAVIGATION & FEED") + "\n")
		s.WriteString(formatHelpItem("j / k", "Move up / down", innerWidth) + "\n")
		s.WriteString(formatHelpItem("enter", "Open post / board", innerWidth) + "\n")
		s.WriteString(formatHelpItem("[ / ]", "Prev / next page", innerWidth) + "\n")
		s.WriteString(formatHelpItem("/", "Search posts", innerWidth) + "\n")
		s.WriteString(formatHelpItem("s", "Sort feed / comments", innerWidth) + "\n")
		s.WriteString(formatHelpItem("c", "Filter flair", innerWidth) + "\n")
		s.WriteString(formatHelpItem("r", "Reply to item", innerWidth) + "\n")
		s.WriteString(formatHelpItem("u / d", "Upvote / downvote", innerWidth) + "\n")
		s.WriteString(formatHelpItem("n", "New post", innerWidth) + "\n")
		s.WriteString(formatHelpItem("?", "Close cheatsheet", innerWidth) + "\n")
		s.WriteString(formatHelpItem("q", "Quit ReadIT", innerWidth))
		content = s.String()
	}

	var cardBody strings.Builder
	title := styleTitle.Render("Keyboard Cheatsheet")
	sub := styleSubtitle.Render("All shortcuts across ReadIT (press [?] or [esc] to return)")
	cardBody.WriteString(title + "\n" + sub + "\n\n")
	cardBody.WriteString(content)

	card := styleModalCard.Width(cardWidth).Render(cardBody.String())
	shortcuts := [][2]string{
		{"?", "close"},
		{"esc", "back"},
		{"q", "quit"},
	}
	return m.renderAppShell("Cheatsheet", lipgloss.PlaceHorizontal(contentWidth, lipgloss.Center, card), shortcuts)
}
