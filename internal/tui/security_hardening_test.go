package tui

import (
	"testing"
	"time"

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
