package tui

import (
	"context"
	"strings"
	"testing"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/jackc/pgx/v5/pgtype"
	db "github.com/sahas/readit/internal/db/sqlc"
	"github.com/sahas/readit/internal/sanitize"
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

func TestDockedViewLineCount(t *testing.T) {
	heights := []int{20, 24, 35, 50}
	for _, h := range heights {
		m := &Model{width: 80, height: h}
		header := "Header Line 1\nHeader Line 2"
		content := "Item 1\nItem 2\nItem 3"
		statusBar := "Status Bar Line"

		docked := m.renderDockedView(header, content, statusBar)
		lines := strings.Split(docked, "\n")
		if len(lines) != h {
			t.Errorf("height %d: got %d lines, want %d", h, len(lines), h)
		}

		lastLine := lines[len(lines)-1]
		if !strings.Contains(lastLine, "Status Bar Line") {
			t.Errorf("status bar not docked on last line: %q", lastLine)
		}
	}
}

func TestDockedViewWithCenteredContent(t *testing.T) {
	heights := []int{24, 40}
	for _, h := range heights {
		m := &Model{width: 80, height: h}
		header := "Header 1\nHeader 2"
		card := "Card Title\nField 1\nField 2"
		statusBar := "Status Bar Line"

		docked := m.renderDockedViewWithCenteredContent(header, card, statusBar)
		lines := strings.Split(docked, "\n")
		if len(lines) != h {
			t.Errorf("height %d: got %d lines, want %d", h, len(lines), h)
		}

		lastLine := lines[len(lines)-1]
		if !strings.Contains(lastLine, "Status Bar Line") {
			t.Errorf("status bar not docked on last line: %q", lastLine)
		}
	}
}

func TestStatusBarFormatting(t *testing.T) {
	m := &Model{
		width: 80,
		user:  &db.User{Handle: "satoshi"},
	}

	// 1. Normal status bar with shortcut pills (centered, no bottom badge)
	bar := m.renderStatusBar("/b/golang", [][2]string{
		{"u", "upvote"},
		{"d", "downvote"},
		{"r", "reply"},
	})

	if !strings.Contains(bar, "[u]") || !strings.Contains(bar, "upvote") {
		t.Errorf("status bar missing shortcut pills: %s", bar)
	}
	if strings.Contains(bar, "/b/golang") {
		t.Errorf("status bar should not contain bottom board badge: %s", bar)
	}

	// 2. Flash message overrides shortcuts
	m.flashMsg = "▲ Upvoted"
	barFlash := m.renderStatusBar("/b/golang", [][2]string{
		{"u", "upvote"},
	})
	if !strings.Contains(barFlash, "▲ Upvoted") {
		t.Errorf("status bar missing flash message: %s", barFlash)
	}
	if strings.Contains(barFlash, "upvote") {
		t.Errorf("status bar should override shortcuts when flash is active")
	}

	// 3. Narrow terminal adaptation
	m.flashMsg = ""
	m.width = 35
	narrowBar := m.renderStatusBar("/b/golang", [][2]string{
		{"u", "up"},
	})
	if lipgloss.Width(narrowBar) > 35 {
		t.Errorf("narrow status bar exceeded width: got %d, max 35", lipgloss.Width(narrowBar))
	}
}

func TestPostDetailWordWrapping(t *testing.T) {
	m := NewModel(context.Background(), nil, "test-fp", nil)
	m.width = 60
	m.height = 30
	m.updateViewportSize()

	longBody := strings.Repeat("word ", 50) // ~250 chars single line
	m.currentPost = &db.GetPostByIDRow{
		ID:           1,
		Title:        "A test post title",
		AuthorHandle: "alice",
		Body:         longBody,
		Score:        10,
	}

	longComment := strings.Repeat("comment ", 40) // ~320 chars single line
	m.comments = []db.GetCommentThreadByPostRow{
		{
			ID:           101,
			PostID:       1,
			AuthorHandle: "bob",
			Body:         longComment,
			Score:        3,
			Depth:        0,
		},
	}

	content := m.renderPostDetailContent()
	lines := strings.Split(content, "\n")
	for _, l := range lines {
		// Each line width should be constrained to viewport width (+ margin allowance)
		w := lipgloss.Width(l)
		if w > m.viewport.Width+5 {
			t.Errorf("line exceeded viewport width (%d > %d): %q", w, m.viewport.Width, l)
		}
	}
}

func TestCommentOPBadgeAndSelection(t *testing.T) {
	m := NewModel(context.Background(), nil, "test-fp", nil)
	m.width = 80
	m.height = 30
	m.updateViewportSize()

	m.currentPost = &db.GetPostByIDRow{
		ID:           1,
		Title:        "Test Title",
		AuthorHandle: "alice",
		Score:        5,
	}

	m.comments = []db.GetCommentThreadByPostRow{
		{
			ID:           10,
			PostID:       1,
			AuthorHandle: "alice", // OP
			Body:         "I am OP",
			Depth:        0,
		},
		{
			ID:           11,
			PostID:       1,
			AuthorHandle: "bob",
			Body:         "I am Bob",
			Depth:        1,
		},
	}

	// 1. Root post selected by default
	m.commentCursor = -1
	contentRoot := m.renderPostDetailContent()
	if !strings.Contains(contentRoot, "▌") {
		t.Errorf("expected root post selection indicator ▌")
	}
	if !strings.Contains(contentRoot, "[OP]") {
		t.Errorf("expected [OP] badge on alice's comment")
	}

	// 2. Select comment 0 (alice, OP)
	m.commentCursor = 0
	contentComm0 := m.renderPostDetailContent()
	if !strings.Contains(contentComm0, "▌") {
		t.Errorf("expected selection bar indicator ▌ on selected comment")
	}

	// 3. Select comment 1 (bob)
	m.commentCursor = 1
	contentComm1 := m.renderPostDetailContent()
	if !strings.Contains(contentComm1, "@bob") {
		t.Errorf("expected bob's comment")
	}
}

func TestNewPostFormNavigation(t *testing.T) {
	m := NewModel(context.Background(), nil, "test-fp", nil)
	m.currentBoard = &db.Board{ID: 1, Slug: "golang"}
	m.openNewPost()

	if m.postFormFocus != 0 {
		t.Fatalf("expected initial focus 0 (Title), got %d", m.postFormFocus)
	}

	// Press Enter on Title -> advances to Category (focus 1)
	m.updateNewPost(tea.KeyMsg{Type: tea.KeyEnter})
	if m.postFormFocus != 1 {
		t.Errorf("expected focus 1 (Category) after enter, got %d", m.postFormFocus)
	}

	// Test category cycling with space while on focus 1
	initialCat := m.newPostCategoryIdx
	m.updateNewPost(tea.KeyMsg{Type: tea.KeySpace})
	if m.newPostCategoryIdx != (initialCat+1)%len(AvailableCategories) {
		t.Errorf("expected category index incremented on space, got %d", m.newPostCategoryIdx)
	}

	// Press Enter on Category -> advances to URL (focus 2)
	m.updateNewPost(tea.KeyMsg{Type: tea.KeyEnter})
	if m.postFormFocus != 2 {
		t.Errorf("expected focus 2 (URL) after enter, got %d", m.postFormFocus)
	}

	// Press Enter on URL -> advances to Body (focus 3)
	m.updateNewPost(tea.KeyMsg{Type: tea.KeyEnter})
	if m.postFormFocus != 3 {
		t.Errorf("expected focus 3 (Body) after enter, got %d", m.postFormFocus)
	}

	// Press Tab on Body -> cycles to Title (focus 0)
	m.updateNewPost(tea.KeyMsg{Type: tea.KeyTab})
	if m.postFormFocus != 0 {
		t.Errorf("expected focus 0 (Title) after tab, got %d", m.postFormFocus)
	}

	// Press Shift+Tab on Title -> cycles back to Body (focus 3)
	m.updateNewPost(tea.KeyMsg{Type: tea.KeyShiftTab})
	if m.postFormFocus != 3 {
		t.Errorf("expected focus 3 (Body) after shift+tab, got %d", m.postFormFocus)
	}
}

func TestCommentNavigationKeys(t *testing.T) {
	m := NewModel(context.Background(), nil, "test-fp", nil)
	m.currentView = viewPostDetail
	m.currentPost = &db.GetPostByIDRow{ID: 1, AuthorHandle: "alice"}
	m.comments = []db.GetCommentThreadByPostRow{
		{ID: 10, AuthorHandle: "bob", Body: "Comment 1"},
		{ID: 11, AuthorHandle: "carol", Body: "Comment 2"},
	}
	m.commentCursor = -1

	// Navigate down with j -> comment 0
	m.updatePostDetail(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'j'}})
	if m.commentCursor != 0 {
		t.Errorf("expected commentCursor 0 after 'j', got %d", m.commentCursor)
	}

	// Navigate down with down -> comment 1
	m.updatePostDetail(tea.KeyMsg{Type: tea.KeyDown})
	if m.commentCursor != 1 {
		t.Errorf("expected commentCursor 1 after 'down', got %d", m.commentCursor)
	}

	// Navigate up with k -> comment 0
	m.updatePostDetail(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'k'}})
	if m.commentCursor != 0 {
		t.Errorf("expected commentCursor 0 after 'k', got %d", m.commentCursor)
	}

	// Navigate up with k -> root post (-1)
	m.updatePostDetail(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'k'}})
	if m.commentCursor != -1 {
		t.Errorf("expected commentCursor -1 after 'k', got %d", m.commentCursor)
	}

	// Tab cycle from -1 -> 0
	m.updatePostDetail(tea.KeyMsg{Type: tea.KeyTab})
	if m.commentCursor != 0 {
		t.Errorf("expected commentCursor 0 after tab, got %d", m.commentCursor)
	}
}

func TestDimensionGuards(t *testing.T) {
	m := NewModel(context.Background(), nil, "test-fp", nil)
	// Zero and negative dimensions
	m.width = 0
	m.height = 0
	m.updateViewportSize()
	m.resizeInputs()

	// Should not panic on any view
	vLoading := m.viewLoading()
	if vLoading == "" {
		t.Errorf("expected non-empty loading view")
	}

	m.user = &db.User{Handle: "test"}
	m.boards = []db.Board{{ID: 1, Slug: "test", Description: "Test board"}}
	vBoard := m.viewBoardList()
	if vBoard == "" {
		t.Errorf("expected non-empty board view")
	}

	m.currentBoard = &m.boards[0]
	m.posts = []PostFeedItem{{
		ID:           1,
		Title:        "Post 1",
		AuthorHandle: "alice",
		Category:     "general",
		CreatedAt:    pgtype.Timestamptz{Time: time.Now(), Valid: true},
	}}
	vPosts := m.viewPostList()
	if vPosts == "" {
		t.Errorf("expected non-empty post list view")
	}
}

func TestInspectCreationViews(t *testing.T) {
	m := NewModel(context.Background(), nil, "test-fp", nil)
	m.width = 80
	m.height = 24
	m.currentBoard = &db.Board{ID: 1, Slug: "golang"}
	m.openNewPost()
	m.resizeInputs()
	postView := m.viewNewPost()
	t.Logf("=== viewNewPost ===\n%s\n", postView)

	lines := strings.Split(postView, "\n")
	if len(lines) > 24 {
		t.Fatalf("expected viewNewPost to fit within 24 lines, got %d lines", len(lines))
	}
	if !strings.Contains(lines[0], "ReadIT") {
		t.Fatalf("expected header 'ReadIT' on line 0 (top not cut off), got %q", lines[0])
	}

	for i, l := range lines {
		trimmed := strings.TrimSpace(l)
		if trimmed == "─╮" || trimmed == "─╯" {
			t.Fatalf("found broken wrapped border on line %d: %q", i, l)
		}
	}

	m.openNewComment(nil, "author")
	m.resizeInputs()
	commView := m.viewNewComment()
	t.Logf("=== viewNewComment ===\n%s\n", commView)

	commLines := strings.Split(commView, "\n")
	if len(commLines) > 24 {
		t.Fatalf("expected viewNewComment to fit within 24 lines, got %d lines", len(commLines))
	}
	if !strings.Contains(commLines[0], "ReadIT") {
		t.Fatalf("expected header 'ReadIT' on line 0 for comment modal, got %q", commLines[0])
	}

	for i, l := range commLines {
		trimmed := strings.TrimSpace(l)
		if trimmed == "─╮" || trimmed == "─╯" {
			t.Fatalf("found broken wrapped border on line %d: %q", i, l)
		}
	}
}

func TestVoteRemovalAndStatePreservation(t *testing.T) {
	m := NewModel(context.Background(), nil, "test-fp", nil)
	m.width = 80
	m.height = 24

	// Test flash message when vote is cleared (direction: 0)
	m.Update(postVotedMsg{postID: 1, direction: 0})
	if m.flashMsg != "• Vote removed" {
		t.Errorf("expected flashMsg '• Vote removed', got %q", m.flashMsg)
	}

	m.Update(commentVotedMsg{commentID: 1, direction: 0})
	if m.flashMsg != "• Vote removed" {
		t.Errorf("expected flashMsg '• Vote removed', got %q", m.flashMsg)
	}

	// Test preservation of postCursor on postsLoadedMsg
	m.postCursor = 3
	m.Update(postsLoadedMsg{posts: []PostFeedItem{
		{ID: 1}, {ID: 2}, {ID: 3}, {ID: 4}, {ID: 5},
	}})
	if m.postCursor != 3 {
		t.Errorf("expected postCursor 3 preserved, got %d", m.postCursor)
	}

	// Test preservation of commentCursor and scroll on postDetailLoadedMsg for same post
	m.currentPost = &db.GetPostByIDRow{ID: 10, Title: "Test Post", Body: "Line 1\nLine 2\nLine 3\nLine 4\nLine 5\nLine 6\nLine 7\nLine 8\nLine 9\nLine 10"}
	m.commentCursor = 2
	m.viewport.Height = 3
	m.viewport.SetContent(m.renderPostDetailContent())
	m.viewport.SetYOffset(2)

	m.Update(postDetailLoadedMsg{
		post: &db.GetPostByIDRow{ID: 10, Title: "Test Post", Body: "Line 1\nLine 2\nLine 3\nLine 4\nLine 5\nLine 6\nLine 7\nLine 8\nLine 9\nLine 10"},
		comments: []db.GetCommentThreadByPostRow{
			{ID: 101}, {ID: 102}, {ID: 103},
		},
	})

	if m.commentCursor != 2 {
		t.Errorf("expected commentCursor 2 preserved, got %d", m.commentCursor)
	}
	if m.viewport.YOffset != 2 {
		t.Errorf("expected viewport YOffset 2 preserved, got %d", m.viewport.YOffset)
	}
}

func TestAdaptiveKeyPillsNeverTruncatesMidWord(t *testing.T) {
	shortcuts := [][2]string{
		{"j/k", "move"},
		{"enter", "view"},
		{"u/d", "vote"},
		{"n", "new post"},
		{"g/G", "top/end"},
		{"esc", "boards"},
		{"q", "quit"},
	}

	widthsToTest := []int{120, 100, 97, 85, 70, 60, 50, 40, 20}
	for _, w := range widthsToTest {
		res := formatAdaptiveKeyPills(shortcuts, w)
		wActual := lipgloss.Width(res)
		if wActual > w {
			t.Errorf("width %d: rendered width %d exceeded max %d: %q", w, wActual, w, res)
		}

		// Must never produce broken truncated words like "[q] qu"
		if strings.Contains(res, "[q] qu") && !strings.Contains(res, "[q] quit") {
			t.Errorf("width %d: [q] was truncated to 'qu': %q", w, res)
		}
		if strings.Contains(res, "[esc] boa") && !strings.Contains(res, "[esc] boards") {
			t.Errorf("width %d: [esc] was truncated to 'boa': %q", w, res)
		}
	}
}
func TestSpinningEarthAndShiningLogo(t *testing.T) {
	// 1. Verify Earth animation frames
	for tick := 0; tick < len(earthFrames)*2; tick++ {
		earth := renderEarthFrame(tick)
		if earth == "" {
			t.Fatalf("renderEarthFrame(%d) returned empty string", tick)
		}
		lines := strings.Split(earth, "\n")
		if len(lines) != 6 {
			t.Errorf("renderEarthFrame(%d) produced %d lines, want 6", tick, len(lines))
		}
	}

	// 2. Verify Shining Logo across a full shine cycle
	for tick := 0; tick < 70; tick++ {
		shined := renderShiningLogo(tick)
		if shined == "" {
			t.Fatalf("renderShiningLogo(%d) returned empty string", tick)
		}
		lines := strings.Split(shined, "\n")
		if len(lines) != 6 {
			t.Errorf("renderShiningLogo(%d) produced %d lines, want 6", tick, len(lines))
		}
	}

	// 3. Verify Hero Banner composition responsive behavior
	wideBanner := renderHeroBanner(5, 80)
	if wideBanner == "" {
		t.Errorf("renderHeroBanner on wide terminal returned empty string")
	}

	narrowBanner := renderHeroBanner(5, 50)
	if narrowBanner == "" {
		t.Errorf("renderHeroBanner on narrow terminal returned empty string")
	}
}

func TestPostListSelectionNeverShiftsHorizontally(t *testing.T) {
	m := &Model{
		width:        100,
		height:       30,
		currentView:  viewPostList,
		currentBoard: &db.Board{Slug: "ask", Description: "Ask anything"},
		posts: []PostFeedItem{
			{
				ID:           1,
				Title:        "Short Title",
				AuthorHandle: "alice",
				Score:        5,
				CommentCount: 2,
				Category:     "general",
			},
			{
				ID:           2,
				Title:        "A Much Longer Discussion Title Here",
				AuthorHandle: "bob",
				Score:        10,
				CommentCount: 8,
				Category:     "general",
			},
		},
	}

	// 1. Render with post 0 selected
	m.postCursor = 0
	view0 := m.viewPostList()

	// 2. Render with post 1 selected
	m.postCursor = 1
	view1 := m.viewPostList()

	// Helper to find the visible column index of a substring within lines
	findCol := func(view, text string) int {
		for _, line := range strings.Split(view, "\n") {
			clean := sanitize.Text(line)
			if idx := strings.Index(clean, text); idx != -1 {
				return lipgloss.Width(clean[:idx])
			}
		}
		return -1
	}

	// Both posts should have their titles start at the exact same visible column, whether selected or unselected
	colShortWhenSelected := findCol(view0, "Short Title")
	colShortWhenUnselected := findCol(view1, "Short Title")
	colLongWhenUnselected := findCol(view0, "A Much Longer Discussion")
	colLongWhenSelected := findCol(view1, "A Much Longer Discussion")

	if colShortWhenSelected != colShortWhenUnselected {
		t.Errorf("horizontal position shifted for 'Short Title': selected at %d, unselected at %d",
			colShortWhenSelected, colShortWhenUnselected)
	}
	if colLongWhenSelected != colLongWhenUnselected {
		t.Errorf("horizontal position shifted for 'A Much Longer Discussion': selected at %d, unselected at %d",
			colLongWhenSelected, colLongWhenUnselected)
	}
	if colShortWhenSelected != colLongWhenSelected {
		t.Errorf("titles do not align horizontally: short at %d, long at %d",
			colShortWhenSelected, colLongWhenSelected)
	}

	// Ensure no CardBgHover background color exists in the view
	if strings.Contains(view0, "\x1b[48;2;34;34;52m") { // #222234 in 24-bit RGB
		t.Errorf("view contains unwanted slate-blue background color escape sequence")
	}
}

func TestRenderThreeColumnHeader(t *testing.T) {
	left := "Ask anything"
	center := "[ / search posts... ]"
	right := "5 discussions • [s] hot • [c] all"

	// Wide screen: should fit within 120
	wide := renderThreeColumnHeader(left, center, right, 120)
	if lipgloss.Width(wide) > 120 {
		t.Errorf("expected wide header width <= 120, got %d", lipgloss.Width(wide))
	}
	if !strings.Contains(wide, left) || !strings.Contains(wide, center) || !strings.Contains(wide, right) {
		t.Errorf("wide header missing components: %q", wide)
	}

	// Medium screen: 80
	med := renderThreeColumnHeader(left, center, right, 80)
	if lipgloss.Width(med) > 80 {
		t.Errorf("expected medium header width <= 80, got %d", lipgloss.Width(med))
	}
}

func TestPostListSearchBarAndDescriptionHeader(t *testing.T) {
	m := NewModel(context.Background(), nil, "test-fp", nil)
	m.width = 100
	m.height = 30
	m.currentView = viewPostList
	m.currentBoard = &db.Board{Slug: "ask", Description: "Ask the community anything"}
	m.posts = []PostFeedItem{
		{
			ID:           1,
			Title:        "How to use goroutines?",
			AuthorHandle: "alice",
			Category:     "question",
		},
	}
	m.resizeInputs()

	view := m.viewPostList()

	// 1. Verify /b/ask is NOT on the left subheader banner
	// Subheader is around line 2 or 3 (below shell header and separator)
	lines := strings.Split(view, "\n")
	var subheader string
	for _, l := range lines {
		if strings.Contains(l, "Ask the community anything") {
			subheader = l
			break
		}
	}
	if subheader == "" {
		t.Fatalf("could not find subheader line with description in view:\n%s", view)
	}

	// Verify "Ask the community anything" is shifted to the left and not prefixed with "/b/ask"
	cleanSub := sanitize.Text(subheader)
	if strings.Contains(cleanSub, "/b/ask  ·") || strings.Contains(cleanSub, "/b/ask ·") {
		t.Errorf("expected /b/ask to be removed from subheader left side, got: %q", cleanSub)
	}

	// 2. Verify dedicated search bar appears when focused
	m.searchFocused = true
	viewFocused := m.viewPostList()
	if !strings.Contains(viewFocused, "/ filter:") {
		t.Errorf("expected dedicated search filter row when focused, got:\n%s", viewFocused)
	}

	// 3. Test active search filter display (zero emojis)
	m.searchFocused = false
	m.searchQuery = "goroutines"
	searchView := m.viewPostList()
	if !strings.Contains(searchView, `filter: "goroutines"`) {
		t.Errorf("expected active query in search view, got:\n%s", searchView)
	}
	if !strings.Contains(searchView, "1 matches") {
		t.Errorf("expected '1 matches' in search view, got:\n%s", searchView)
	}
}

func TestPostListSearchNavigationAndClear(t *testing.T) {
	m := NewModel(context.Background(), nil, "test-fp", nil)
	m.width = 100
	m.height = 30
	m.currentBoard = &db.Board{ID: 1, Slug: "golang", Description: "Go Language"}
	m.currentView = viewPostList

	// 1. Pressing '/' should focus searchInput
	m.updatePostList(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'/'}})
	if !m.searchFocused {
		t.Fatalf("expected searchFocused to be true after pressing '/'")
	}

	// 2. Typing into search input
	m.updatePostList(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'c', 'h', 'a', 'n'}})
	if m.searchInput.Value() != "chan" {
		t.Errorf("expected searchInput value 'chan', got %q", m.searchInput.Value())
	}

	// 3. Pressing Enter commits search query
	m.updatePostList(tea.KeyMsg{Type: tea.KeyEnter})
	if m.searchFocused {
		t.Errorf("expected searchFocused to be false after pressing Enter")
	}
	if m.searchQuery != "chan" {
		t.Errorf("expected searchQuery 'chan', got %q", m.searchQuery)
	}

	// 4. Pressing Esc clears the search query
	m.updatePostList(tea.KeyMsg{Type: tea.KeyEsc})
	if m.searchQuery != "" {
		t.Errorf("expected searchQuery to be cleared after Esc, got %q", m.searchQuery)
	}
	if m.currentView != viewPostList {
		t.Errorf("expected to stay in viewPostList after first Esc (clearing search), got %v", m.currentView)
	}

	// 5. Pressing Esc again returns to viewBoardList
	m.updatePostList(tea.KeyMsg{Type: tea.KeyEsc})
	if m.currentView != viewBoardList {
		t.Errorf("expected viewBoardList after second Esc, got %v", m.currentView)
	}
}




