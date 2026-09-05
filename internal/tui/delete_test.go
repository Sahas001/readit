package tui

import (
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	db "github.com/sahas/readit/internal/db/sqlc"
)

func TestDeleteKeyTriggersConfirmationForAuthorOnly(t *testing.T) {
	m := &Model{
		width:       100,
		height:      30,
		currentView: viewPostList,
		user:        &db.User{ID: 42, Handle: "alice"},
		posts: []db.ListPostsByBoardNewRow{
			{
				ID:           1,
				AuthorID:     42, // authored by alice
				AuthorHandle: "alice",
				Title:        "Alice's Post",
				CommentCount: 0,
			},
			{
				ID:           2,
				AuthorID:     99, // authored by someone else
				AuthorHandle: "bob",
				Title:        "Bob's Post",
				CommentCount: 3,
			},
		},
	}

	// 1. Press x on post 0 (authored by alice) -> triggers delete confirmation modal
	m.postCursor = 0
	m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'x'}})
	if m.currentView != viewDeleteConfirm {
		t.Fatalf("expected viewDeleteConfirm, got %v", m.currentView)
	}
	if m.pendingDelete == nil || m.pendingDelete.id != 1 || m.pendingDelete.hasDependents != false {
		t.Fatalf("unexpected pendingDelete state: %+v", m.pendingDelete)
	}

	// Cancel out
	m.Update(tea.KeyMsg{Type: tea.KeyEsc})
	if m.currentView != viewPostList {
		t.Fatalf("expected return to viewPostList after cancel, got %v", m.currentView)
	}
	if m.pendingDelete != nil {
		t.Fatalf("expected pendingDelete to be nil after cancel")
	}

	// 2. Press x on post 1 (authored by bob) -> blocked with flash message
	m.postCursor = 1
	m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'x'}})
	if m.currentView != viewPostList {
		t.Fatalf("expected view to remain viewPostList, got %v", m.currentView)
	}
	if !strings.Contains(m.flashMsg, "only delete your own") {
		t.Fatalf("expected error flashMsg, got %q", m.flashMsg)
	}
}

func TestDeleteCommentKeyInPostDetail(t *testing.T) {
	m := &Model{
		width:         100,
		height:        30,
		currentView:   viewPostDetail,
		user:          &db.User{ID: 42, Handle: "alice"},
		commentCursor: 0,
		currentPost: &db.GetPostByIDRow{
			ID:           1,
			AuthorID:     99,
			AuthorHandle: "bob",
			Title:        "Bob's Post",
		},
		comments: []db.GetCommentThreadByPostRow{
			{
				ID:           101,
				PostID:       1,
				AuthorID:     42, // Alice's comment
				AuthorHandle: "alice",
				Body:         "Alice says hello",
				Depth:        0,
			},
			{
				ID:           102,
				PostID:       1,
				AuthorID:     88, // Charlie's reply
				AuthorHandle: "charlie",
				Body:         "Charlie replies",
				Depth:        1,
			},
		},
	}

	// Press x on Alice's comment (index 0)
	m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'x'}})
	if m.currentView != viewDeleteConfirm {
		t.Fatalf("expected viewDeleteConfirm, got %v", m.currentView)
	}
	if m.pendingDelete == nil || m.pendingDelete.id != 101 {
		t.Fatalf("unexpected pendingDelete: %+v", m.pendingDelete)
	}

	// Cancel deletion
	m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'n'}})
	if m.currentView != viewPostDetail {
		t.Fatalf("expected viewPostDetail after cancel, got %v", m.currentView)
	}

	// Try deleting Charlie's comment (index 1)
	m.commentCursor = 1
	m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'x'}})
	if m.currentView != viewPostDetail {
		t.Fatalf("expected view to remain viewPostDetail, got %v", m.currentView)
	}
	if !strings.Contains(m.flashMsg, "only delete your own") {
		t.Fatalf("expected unauthorized error flashMsg, got %q", m.flashMsg)
	}
}

func TestDeleteConfirmModalRendering(t *testing.T) {
	m := &Model{
		width:  80,
		height: 24,
	}

	// 1. Post with comments (soft delete explanation)
	m.pendingDelete = &deleteTarget{
		targetType:    deleteTargetPost,
		id:            1,
		titleOrBody:   "Important Community Discussion",
		hasDependents: true,
		commentCount:  12,
		returnView:    viewPostList,
	}
	viewSoftPost := m.viewDeleteConfirm()
	if !strings.Contains(viewSoftPost, "Delete Discussion?") {
		t.Errorf("expected warning title")
	}
	if !strings.Contains(viewSoftPost, "12 active comments") {
		t.Errorf("expected comment count in modal warning")
	}
	if !strings.Contains(viewSoftPost, "[y] Confirm Delete") {
		t.Errorf("expected confirm button")
	}

	// 2. Post without comments (hard delete explanation)
	m.pendingDelete = &deleteTarget{
		targetType:    deleteTargetPost,
		id:            2,
		titleOrBody:   "Empty Post",
		hasDependents: false,
		commentCount:  0,
		returnView:    viewPostList,
	}
	viewHardPost := m.viewDeleteConfirm()
	if !strings.Contains(viewHardPost, "Permanently Delete") {
		t.Errorf("expected permanent delete title")
	}
	if !strings.Contains(viewHardPost, "completely removed") {
		t.Errorf("expected hard delete explanation")
	}

	// 3. Comment with replies (soft delete explanation)
	m.pendingDelete = &deleteTarget{
		targetType:    deleteTargetComment,
		id:            10,
		titleOrBody:   "Comment with children",
		hasDependents: true,
		returnView:    viewPostDetail,
	}
	viewSoftComment := m.viewDeleteConfirm()
	if !strings.Contains(viewSoftComment, "Delete Comment?") {
		t.Errorf("expected comment delete title")
	}
	if !strings.Contains(viewSoftComment, "active replies") {
		t.Errorf("expected active replies warning")
	}
}

func TestDeletedPostAndCommentGuards(t *testing.T) {
	m := &Model{
		width:         80,
		height:        24,
		currentView:   viewPostDetail,
		user:          &db.User{ID: 1, Handle: "user"},
		commentCursor: -1, // Root post selected
		currentPost: &db.GetPostByIDRow{
			ID:           5,
			IsDeleted:    true,
			AuthorHandle: "[deleted]",
			Title:        "[deleted]",
		},
		comments: []db.GetCommentThreadByPostRow{
			{
				ID:           50,
				IsDeleted:    true,
				AuthorHandle: "[deleted]",
				Body:         "[deleted]",
			},
		},
	}

	// 1. Voting on deleted post is blocked
	m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'u'}})
	if !strings.Contains(m.flashMsg, "Voting is disabled on deleted content") {
		t.Errorf("expected voting disabled flash, got %q", m.flashMsg)
	}

	// 2. Replying to deleted post is blocked
	m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'r'}})
	if !strings.Contains(m.flashMsg, "Post is deleted. Discussion is locked.") {
		t.Errorf("expected locked discussion flash, got %q", m.flashMsg)
	}

	// 3. Voting on deleted comment is blocked
	m.commentCursor = 0
	m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'d'}})
	if !strings.Contains(m.flashMsg, "Voting is disabled on deleted content") {
		t.Errorf("expected voting disabled flash on comment, got %q", m.flashMsg)
	}

	// 4. Replying to deleted comment is blocked
	m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'r'}})
	if !strings.Contains(m.flashMsg, "Cannot reply to a deleted comment") {
		t.Errorf("expected reply blocked on deleted comment, got %q", m.flashMsg)
	}
}

func TestSoftDeletedPostWithCommentsRenderedInFeed(t *testing.T) {
	m := &Model{
		width:        80,
		height:       24,
		currentView:  viewPostList,
		currentBoard: &db.Board{Slug: "general"},
		posts: []db.ListPostsByBoardNewRow{
			{
				ID:           1,
				Title:        "Active Post",
				AuthorHandle: "alice",
				CommentCount: 5,
				IsDeleted:    false,
			},
			{
				ID:           2,
				Title:        "[deleted]",
				AuthorHandle: "[deleted]",
				CommentCount: 3,
				IsDeleted:    true,
			},
		},
	}

	rendered := m.viewPostList()
	if !strings.Contains(rendered, "Active Post") {
		t.Errorf("expected active post to be rendered")
	}
	if !strings.Contains(rendered, "[deleted by author]") {
		t.Errorf("expected soft-deleted post to be labeled [deleted by author]")
	}
	if !strings.Contains(rendered, "3 comments") {
		t.Errorf("expected comment count for soft-deleted post")
	}

	// Voting on deleted post in feed should be blocked
	m.postCursor = 1
	m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'u'}})
	if !strings.Contains(m.flashMsg, "Voting is disabled on deleted content") {
		t.Errorf("expected voting blocked on deleted post in feed")
	}

	// Deleting already deleted post in feed should be blocked
	m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'x'}})
	if !strings.Contains(m.flashMsg, "Post is already deleted") {
		t.Errorf("expected already deleted message")
	}
}

func TestPostPrunedWhenLastCommentRemoved(t *testing.T) {
	m := &Model{
		width:        80,
		height:       24,
		currentView:  viewPostDetail,
		currentBoard: &db.Board{ID: 1, Slug: "general"},
	}

	// When postPrunedMsg arrives
	m.Update(postPrunedMsg{postID: 42})
	if m.currentView != viewPostList {
		t.Errorf("expected view to switch to viewPostList, got %v", m.currentView)
	}
	if !strings.Contains(m.flashMsg, "Thread closed & pruned") {
		t.Errorf("expected thread pruned flash message, got %q", m.flashMsg)
	}
}

func TestAllTombstonesThreadAutoPruning(t *testing.T) {
	m := &Model{
		width:        80,
		height:       24,
		currentView:  viewPostDetail,
		currentBoard: &db.Board{ID: 1, Slug: "general"},
	}

	// Simulating receiving postPrunedMsg when thread has only tombstones
	_, cmd := m.Update(postPrunedMsg{postID: 99})
	if m.currentView != viewPostList {
		t.Fatalf("expected viewPostList, got %v", m.currentView)
	}
	if !strings.Contains(m.flashMsg, "Thread closed & pruned") {
		t.Fatalf("expected flash message, got %q", m.flashMsg)
	}
	if cmd == nil {
		t.Fatalf("expected command to reload posts")
	}
}


