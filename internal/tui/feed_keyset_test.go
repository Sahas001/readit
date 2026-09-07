package tui

import (
	"context"
	"strings"
	"testing"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/jackc/pgx/v5/pgtype"
	db "github.com/sahas/readit/internal/db/sqlc"
)

func makeTestPosts(count int) []PostFeedItem {
	posts := make([]PostFeedItem, count)
	now := time.Now()
	for i := 0; i < count; i++ {
		posts[i] = PostFeedItem{
			ID:           int64(i + 1),
			BoardID:      1,
			AuthorID:     1,
			Title:        "Post Title " + string(rune('A'+i%26)),
			Score:        int32(100 - i),
			CommentCount: int32(i * 2),
			CreatedAt:    pgtype.Timestamptz{Time: now.Add(-time.Duration(i) * time.Hour), Valid: true},
			Category:     "general",
			HotScore:     float64(1000 - i*10),
			AuthorHandle: "alice",
		}
	}
	return posts
}

func TestFeedKeysetPagination_Navigation(t *testing.T) {
	m := NewModel(context.Background(), nil, "fp", nil)
	m.currentBoard = &db.Board{ID: 1, Slug: "general"}
	m.currentView = viewPostList
	m.posts = makeTestPosts(10)
	m.hasNextPage = true
	m.feedPage = 1

	// 1. Navigate to Next Page via ']'
	m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{']'}})
	if m.feedPage != 2 {
		t.Fatalf("expected feedPage to advance to 2, got %d", m.feedPage)
	}
	if len(m.feedPageStack) != 1 {
		t.Fatalf("expected 1 cursor in feedPageStack, got %d", len(m.feedPageStack))
	}
	if m.feedPageStack[0].ID != 10 {
		t.Errorf("expected cursor ID 10 (last post), got %d", m.feedPageStack[0].ID)
	}
	if m.postCursor != 0 {
		t.Errorf("expected postCursor reset to 0, got %d", m.postCursor)
	}

	// 2. Navigate to Previous Page via '['
	m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'['}})
	if m.feedPage != 1 {
		t.Fatalf("expected feedPage to decrement to 1, got %d", m.feedPage)
	}

	// 3. At Page 1, '[' should not decrement below 1
	m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'['}})
	if m.feedPage != 1 {
		t.Errorf("expected feedPage to remain 1, got %d", m.feedPage)
	}

	// 4. Changing sort mode should reset pagination to Page 1
	m.feedPage = 3
	m.feedPageStack = []PostCursor{{ID: 10}, {ID: 20}}
	m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'s'}})
	if m.feedPage != 1 {
		t.Errorf("expected sort mode change to reset feedPage to 1, got %d", m.feedPage)
	}
	if len(m.feedPageStack) != 0 {
		t.Errorf("expected sort mode change to clear feedPageStack, got %d", len(m.feedPageStack))
	}

	// 5. Changing category filter should reset pagination to Page 1
	m.feedPage = 2
	m.feedPageStack = []PostCursor{{ID: 10}}
	m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'c'}})
	if m.feedPage != 1 {
		t.Errorf("expected flair filter change to reset feedPage to 1, got %d", m.feedPage)
	}
}

func TestFeedDualDensity_CompactMode(t *testing.T) {
	m := NewModel(context.Background(), nil, "fp", nil)
	m.width = 80
	m.height = 24
	m.currentBoard = &db.Board{ID: 1, Slug: "general"}
	m.currentView = viewPostList
	m.posts = makeTestPosts(20)

	// Default mode is comfortable (3 lines per card)
	if m.compactMode {
		t.Errorf("expected initial compactMode to be false")
	}
	comfortView := m.viewPostList()
	if !strings.Contains(comfortView, "[compact: z]") {
		t.Errorf("expected '[compact: z]' badge in header, got view:\n%s", comfortView)
	}

	// Toggle to compact mode with 'z'
	m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'z'}})
	if !m.compactMode {
		t.Fatalf("expected compactMode to be true after 'z'")
	}
	compactView := m.viewPostList()
	if !strings.Contains(compactView, "[comfort: z]") {
		t.Errorf("expected '[comfort: z]' badge in header, got view:\n%s", compactView)
	}

	// Verify compact view fits significantly more posts on screen than comfortable view
	comfortPostCount := strings.Count(comfortView, "Post Title")
	compactPostCount := strings.Count(compactView, "Post Title")
	if compactPostCount <= comfortPostCount {
		t.Errorf("expected compact view to show more posts (%d) than comfortable (%d)", compactPostCount, comfortPostCount)
	}
}

func TestFeedSessionTriage_ReadAndHide(t *testing.T) {
	m := NewModel(context.Background(), nil, "fp", nil)
	m.width = 80
	m.height = 24
	m.currentBoard = &db.Board{ID: 1, Slug: "general"}
	m.currentView = viewPostList
	m.posts = makeTestPosts(5)
	m.postCursor = 0

	// 1. Toggle mark-as-read with 'm'
	postID := m.posts[0].ID
	if m.readPosts[postID] {
		t.Errorf("expected initial post to be unread")
	}
	m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'m'}})
	if !m.readPosts[postID] {
		t.Fatalf("expected post %d to be marked read after 'm'", postID)
	}

	// 2. Comfortable view shows checkmark for read post
	view := m.viewPostList()
	if !strings.Contains(view, "✓ @alice") {
		t.Errorf("expected read checkmark '✓ @alice' in metadata, got view:\n%s", view)
	}

	// 3. Toggle hide-read with 'H'
	m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'H'}})
	if !m.hideRead {
		t.Fatalf("expected hideRead to be true after 'H'")
	}

	// Post 0 should now be hidden from display
	viewHidden := m.viewPostList()
	if strings.Contains(viewHidden, m.posts[0].Title) {
		t.Errorf("expected read post %q to be hidden from view", m.posts[0].Title)
	}
	if !strings.Contains(viewHidden, "[hide-read: H]") {
		t.Errorf("expected '[hide-read: H]' badge in header")
	}

	// 4. Entering discussion thread with 'enter' marks it as read
	m.postCursor = 0 // points to post 1 in filtered list
	activePost := m.posts[1]
	m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	if !m.readPosts[activePost.ID] {
		t.Errorf("expected post %d to be marked read upon opening", activePost.ID)
	}
}

func TestFeedVimMotions_HalfPageJump(t *testing.T) {
	m := NewModel(context.Background(), nil, "fp", nil)
	m.currentBoard = &db.Board{ID: 1, Slug: "general"}
	m.currentView = viewPostList
	m.posts = makeTestPosts(20)
	m.postCursor = 0

	// Comfortable mode ctrl+d jump is 5
	m.Update(tea.KeyMsg{Type: tea.KeyCtrlD})
	if m.postCursor != 5 {
		t.Errorf("expected postCursor 5 after ctrl+d, got %d", m.postCursor)
	}

	m.Update(tea.KeyMsg{Type: tea.KeyCtrlU})
	if m.postCursor != 0 {
		t.Errorf("expected postCursor 0 after ctrl+u, got %d", m.postCursor)
	}

	// Compact mode ctrl+d jump is 12
	m.compactMode = true
	m.Update(tea.KeyMsg{Type: tea.KeyCtrlD})
	if m.postCursor != 12 {
		t.Errorf("expected postCursor 12 after ctrl+d in compact mode, got %d", m.postCursor)
	}
}

func TestKeyboardHelpModal_NavigationAndRendering(t *testing.T) {
	m := NewModel(context.Background(), nil, "fp", nil)
	m.width = 100
	m.height = 30
	m.resizeInputs()

	// 1. From Board List, press '?' opens viewHelp
	m.currentView = viewBoardList
	m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'?'}})
	if m.currentView != viewHelp {
		t.Fatalf("expected currentView viewHelp, got %v", m.currentView)
	}
	if m.helpReturnView != viewBoardList {
		t.Fatalf("expected helpReturnView viewBoardList, got %v", m.helpReturnView)
	}

	// Dismiss with '?' returns to viewBoardList
	m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'?'}})
	if m.currentView != viewBoardList {
		t.Fatalf("expected return to viewBoardList, got %v", m.currentView)
	}

	// 2. From Post List, press '?' opens viewHelp
	m.currentView = viewPostList
	m.currentBoard = &db.Board{ID: 1, Slug: "general"}
	m.posts = makeTestPosts(5)
	m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'?'}})
	if m.currentView != viewHelp {
		t.Fatalf("expected currentView viewHelp from post list, got %v", m.currentView)
	}
	if m.helpReturnView != viewPostList {
		t.Fatalf("expected helpReturnView viewPostList, got %v", m.helpReturnView)
	}

	// Dismiss with 'esc' returns to viewPostList
	m.Update(tea.KeyMsg{Type: tea.KeyEscape})
	if m.currentView != viewPostList {
		t.Fatalf("expected return to viewPostList after esc, got %v", m.currentView)
	}

	// 3. From Post Detail, press '?' opens viewHelp
	m.currentView = viewPostDetail
	m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'?'}})
	if m.currentView != viewHelp {
		t.Fatalf("expected currentView viewHelp from post detail, got %v", m.currentView)
	}
	if m.helpReturnView != viewPostDetail {
		t.Fatalf("expected helpReturnView viewPostDetail, got %v", m.helpReturnView)
	}

	// Dismiss with 'q' returns to viewPostDetail
	m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'q'}})
	if m.currentView != viewPostDetail {
		t.Fatalf("expected return to viewPostDetail after q, got %v", m.currentView)
	}

	// 4. Verify search focused in Post List intercepts '?' as input instead of opening help
	m.currentView = viewPostList
	m.searchFocused = false
	m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'/'}})
	if !m.searchFocused {
		t.Fatalf("expected searchFocused true after '/'")
	}
	m.searchInput.SetValue("")
	m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'?'}})
	if m.currentView != viewPostList {
		t.Fatalf("expected viewPostList when search is focused, got %v", m.currentView)
	}
	if m.searchInput.Value() != "?" {
		t.Fatalf("expected '?' typed into searchInput, got %q", m.searchInput.Value())
	}

	// 5. Test rendering viewHelp (wide layout)
	m.currentView = viewHelp
	renderedWide := m.viewHelp()
	if !strings.Contains(renderedWide, "Keyboard Cheatsheet") {
		t.Errorf("expected title in viewHelp, got:\n%s", renderedWide)
	}
	if !strings.Contains(renderedWide, "NAVIGATION & FEED") {
		t.Errorf("expected NAVIGATION & FEED section in viewHelp, got:\n%s", renderedWide)
	}
	if !strings.Contains(renderedWide, "TRIAGE & FILTER") {
		t.Errorf("expected TRIAGE & FILTER section in viewHelp, got:\n%s", renderedWide)
	}
	if !strings.Contains(renderedWide, "DISCUSSIONS & COMMENTS") {
		t.Errorf("expected DISCUSSIONS & COMMENTS section in viewHelp, got:\n%s", renderedWide)
	}
	if !strings.Contains(renderedWide, "COMPOSERS & GLOBAL") {
		t.Errorf("expected COMPOSERS & GLOBAL section in viewHelp, got:\n%s", renderedWide)
	}

	// 6. Test rendering viewHelp (narrow layout)
	m.width = 50
	m.height = 24
	renderedNarrow := m.viewHelp()
	if !strings.Contains(renderedNarrow, "Keyboard Cheatsheet") {
		t.Errorf("expected title in narrow viewHelp, got:\n%s", renderedNarrow)
	}
}
