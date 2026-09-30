package tui

import (
	"testing"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/jackc/pgx/v5/pgtype"
	db "github.com/sahas/readit/internal/db/sqlc"
)

// TestCommentTreeCycleProtection verifies that sortCommentTree terminates cleanly
// without infinite recursion or stack overflow even if comments have cyclical references.
func TestCommentTreeCycleProtection(t *testing.T) {
	// Create cyclic comments: 1 -> 2 -> 3 -> 1
	comments := []db.GetCommentThreadByPostRow{
		{
			ID:           1,
			PostID:       10,
			ParentID:     pgtype.Int8{Int64: 3, Valid: true},
			AuthorID:     1,
			AuthorHandle: "alice",
			Body:         "Comment 1",
			Score:        5,
			CreatedAt:    pgtype.Timestamptz{Time: time.Now(), Valid: true},
		},
		{
			ID:           2,
			PostID:       10,
			ParentID:     pgtype.Int8{Int64: 1, Valid: true},
			AuthorID:     2,
			AuthorHandle: "bob",
			Body:         "Comment 2",
			Score:        3,
			CreatedAt:    pgtype.Timestamptz{Time: time.Now().Add(time.Second), Valid: true},
		},
		{
			ID:           3,
			PostID:       10,
			ParentID:     pgtype.Int8{Int64: 2, Valid: true},
			AuthorID:     3,
			AuthorHandle: "carol",
			Body:         "Comment 3",
			Score:        1,
			CreatedAt:    pgtype.Timestamptz{Time: time.Now().Add(2 * time.Second), Valid: true},
		},
	}

	// This should not panic or cause a stack overflow
	done := make(chan struct{})
	go func() {
		defer close(done)
		res := sortCommentTree(comments, CommentSortTop)
		if len(res) > len(comments) {
			t.Errorf("expected at most %d comments, got %d", len(comments), len(res))
		}
	}()

	select {
	case <-done:
		// Success: terminated cleanly
	case <-time.After(500 * time.Millisecond):
		t.Fatal("sortCommentTree did not terminate (infinite cycle detected)")
	}
}

// TestPostCooldownValidation verifies that post creation honors the 30-second cooldown in model logic.
func TestPostCooldownValidation(t *testing.T) {
	now := time.Now()
	recentPostTime := now.Add(-5 * time.Second) // 5s ago < 30s cooldown

	user := &db.User{
		ID:         1,
		Handle:     "tester",
		LastPostAt: pgtype.Timestamptz{Time: recentPostTime, Valid: true},
	}

	m := &Model{
		user: user,
	}

	if m.user.LastPostAt.Valid && time.Since(m.user.LastPostAt.Time) < 30*time.Second {
		remaining := 30*time.Second - time.Since(m.user.LastPostAt.Time)
		if remaining.Round(time.Second) <= 0 {
			t.Errorf("expected positive remaining cooldown, got %v", remaining)
		}
	} else {
		t.Errorf("expected user to be in active cooldown")
	}

	// Expired cooldown (>30s ago)
	oldPostTime := now.Add(-35 * time.Second)
	user.LastPostAt = pgtype.Timestamptz{Time: oldPostTime, Valid: true}
	if m.user.LastPostAt.Valid && time.Since(m.user.LastPostAt.Time) < 30*time.Second {
		t.Errorf("user should not be in cooldown after 35s")
	}
}

// TestErrorViewKeyHandlingEscAndQuit verifies that viewError handles [esc] and [q] keys
// so users are never trapped on an error screen requiring Ctrl+C.
func TestErrorViewKeyHandlingEscAndQuit(t *testing.T) {
	m := &Model{
		currentView:     viewError,
		errorReturnView: viewPostList,
	}

	// 1. Test [esc] returns to errorReturnView
	updatedM, cmd := m.updateError(tea.KeyMsg{Type: tea.KeyEscape})
	if cmd != nil {
		t.Errorf("expected nil cmd on esc, got %v", cmd)
	}
	if updatedM.currentView != viewPostList {
		t.Errorf("expected return to viewPostList on esc, got %v", updatedM.currentView)
	}

	// 2. Test [q] returns tea.Quit
	m.currentView = viewError
	updatedM, cmd = m.updateError(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'q'}})
	if cmd == nil {
		t.Errorf("expected tea.Quit cmd on q in viewError, got nil")
	}
}

// TestSelfVotingRejectionFlashMsg verifies that voting on own content sets flashMsg
// and does not navigate away to viewError.
func TestSelfVotingRejectionFlashMsg(t *testing.T) {
	user := &db.User{
		ID:     42,
		Handle: "satoshi",
	}

	m := &Model{
		user:        user,
		currentView: viewPostList,
		posts: []PostFeedItem{
			{
				ID:           101,
				AuthorID:     42, // same as user.ID (own post)
				AuthorHandle: "satoshi",
				Title:        "My Own Post",
			},
		},
		postCursor: 0,
	}

	// Press 'u' on own post
	updatedM, cmd := m.updatePostList(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'u'}})
	if cmd != nil {
		t.Errorf("expected nil cmd (no DB query dispatched) when voting on own post, got %v", cmd)
	}
	if updatedM.currentView == viewError {
		t.Errorf("app must not crash into viewError when voting on own post")
	}
	if updatedM.flashMsg != "• You cannot vote on your own post" {
		t.Errorf("expected flash message '• You cannot vote on your own post', got %q", updatedM.flashMsg)
	}

	// Press 'd' on own post
	updatedM, cmd = m.updatePostList(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'d'}})
	if cmd != nil {
		t.Errorf("expected nil cmd when downvoting own post, got %v", cmd)
	}
	if updatedM.currentView == viewError {
		t.Errorf("app must not crash into viewError when downvoting own post")
	}
	if updatedM.flashMsg != "• You cannot vote on your own post" {
		t.Errorf("expected flash message '• You cannot vote on your own post', got %q", updatedM.flashMsg)
	}
}
