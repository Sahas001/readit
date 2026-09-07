package tui

import (
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgtype"
	db "github.com/sahas/readit/internal/db/sqlc"
)

func TestCommentSortTree_HierarchyAndOrder(t *testing.T) {
	now := time.Now()
	t1 := now.Add(-3 * time.Hour)
	t2 := now.Add(-2 * time.Hour)
	t3 := now.Add(-1 * time.Hour)

	// Tree structure:
	// Root 1 (Score: 10, Created: t1)
	//   -> Child 1.1 (Score: 50, Created: t3)
	//   -> Child 1.2 (Score: 100, Created: t2)
	// Root 2 (Score: 50, Created: t2)
	// Root 3 (Score: 5, Created: t3)
	rawComments := []db.GetCommentThreadByPostRow{
		{
			ID:        1,
			PostID:    1,
			ParentID:  pgtype.Int8{Valid: false},
			Score:     10,
			CreatedAt: pgtype.Timestamptz{Time: t1, Valid: true},
			Body:      "Root 1",
		},
		{
			ID:        2,
			PostID:    1,
			ParentID:  pgtype.Int8{Valid: false},
			Score:     50,
			CreatedAt: pgtype.Timestamptz{Time: t2, Valid: true},
			Body:      "Root 2",
		},
		{
			ID:        3,
			PostID:    1,
			ParentID:  pgtype.Int8{Valid: false},
			Score:     5,
			CreatedAt: pgtype.Timestamptz{Time: t3, Valid: true},
			Body:      "Root 3",
		},
		{
			ID:        11,
			PostID:    1,
			ParentID:  pgtype.Int8{Int64: 1, Valid: true},
			Score:     50,
			CreatedAt: pgtype.Timestamptz{Time: t3, Valid: true},
			Body:      "Child 1.1",
		},
		{
			ID:        12,
			PostID:    1,
			ParentID:  pgtype.Int8{Int64: 1, Valid: true},
			Score:     100,
			CreatedAt: pgtype.Timestamptz{Time: t2, Valid: true},
			Body:      "Child 1.2",
		},
	}

	t.Run("SortTop", func(t *testing.T) {
		sorted := sortCommentTree(rawComments, CommentSortTop)
		if len(sorted) != len(rawComments) {
			t.Fatalf("expected %d comments, got %d", len(rawComments), len(sorted))
		}

		// Root order by score DESC: Root 2 (score 50), Root 1 (score 10), Root 3 (score 5)
		// Beneath Root 1, Child 1.2 (score 100) before Child 1.1 (score 50)
		expectedIDs := []int64{2, 1, 12, 11, 3}
		expectedDepths := []int32{0, 0, 1, 1, 0}

		for i, c := range sorted {
			if c.ID != expectedIDs[i] {
				t.Errorf("at index %d: expected ID %d, got %d", i, expectedIDs[i], c.ID)
			}
			if c.Depth != expectedDepths[i] {
				t.Errorf("at index %d: expected Depth %d, got %d", i, expectedDepths[i], c.Depth)
			}
		}
	})

	t.Run("SortNew", func(t *testing.T) {
		sorted := sortCommentTree(rawComments, CommentSortNew)
		// Root order by created DESC: Root 3 (t3), Root 2 (t2), Root 1 (t1)
		// Beneath Root 1: Child 1.1 (t3) before Child 1.2 (t2)
		expectedIDs := []int64{3, 2, 1, 11, 12}
		expectedDepths := []int32{0, 0, 0, 1, 1}

		for i, c := range sorted {
			if c.ID != expectedIDs[i] {
				t.Errorf("at index %d: expected ID %d, got %d", i, expectedIDs[i], c.ID)
			}
			if c.Depth != expectedDepths[i] {
				t.Errorf("at index %d: expected Depth %d, got %d", i, expectedDepths[i], c.Depth)
			}
		}
	})

	t.Run("SortOld", func(t *testing.T) {
		sorted := sortCommentTree(rawComments, CommentSortOld)
		// Root order by created ASC: Root 1 (t1), Root 2 (t2), Root 3 (t3)
		// Beneath Root 1: Child 1.2 (t2) before Child 1.1 (t3)
		expectedIDs := []int64{1, 12, 11, 2, 3}
		expectedDepths := []int32{0, 1, 1, 0, 0}

		for i, c := range sorted {
			if c.ID != expectedIDs[i] {
				t.Errorf("at index %d: expected ID %d, got %d", i, expectedIDs[i], c.ID)
			}
			if c.Depth != expectedDepths[i] {
				t.Errorf("at index %d: expected Depth %d, got %d", i, expectedDepths[i], c.Depth)
			}
		}
	})

	t.Run("EmptyAndSingle", func(t *testing.T) {
		if res := sortCommentTree(nil, CommentSortTop); len(res) != 0 {
			t.Errorf("expected empty result for nil input")
		}
		single := []db.GetCommentThreadByPostRow{{ID: 1, Body: "Solo"}}
		if res := sortCommentTree(single, CommentSortTop); len(res) != 1 || res[0].ID != 1 {
			t.Errorf("expected single comment unchanged")
		}
	})
}

func TestPostSortModeString(t *testing.T) {
	cases := []struct {
		mode     PostSortMode
		expected string
	}{
		{PostSortHot, "hot"},
		{PostSortNew, "new"},
		{PostSortTop, "top"},
		{PostSortMode(99), "hot"},
	}

	for _, tc := range cases {
		if tc.mode.String() != tc.expected {
			t.Errorf("mode %d: expected %q, got %q", tc.mode, tc.expected, tc.mode.String())
		}
	}
}

func TestCommentSortModeString(t *testing.T) {
	cases := []struct {
		mode     CommentSortMode
		expected string
	}{
		{CommentSortTop, "top"},
		{CommentSortNew, "new"},
		{CommentSortOld, "old"},
		{CommentSortMode(99), "top"},
	}

	for _, tc := range cases {
		if tc.mode.String() != tc.expected {
			t.Errorf("mode %d: expected %q, got %q", tc.mode, tc.expected, tc.mode.String())
		}
	}
}

func TestCycleCategoryFilter(t *testing.T) {
	m := &Model{}

	// Initial is empty ("all")
	if m.categoryFilter != "" {
		t.Fatalf("expected initial filter to be empty, got %q", m.categoryFilter)
	}

	// 1st cycle -> AvailableCategories[0] ("general")
	m.cycleCategoryFilter()
	if m.categoryFilter != AvailableCategories[0] {
		t.Errorf("expected %q, got %q", AvailableCategories[0], m.categoryFilter)
	}

	// Cycle through remaining categories
	for i := 1; i < len(AvailableCategories); i++ {
		m.cycleCategoryFilter()
		if m.categoryFilter != AvailableCategories[i] {
			t.Errorf("expected %q, got %q", AvailableCategories[i], m.categoryFilter)
		}
	}

	// Next cycle wraps back to "" (all categories)
	m.cycleCategoryFilter()
	if m.categoryFilter != "" {
		t.Errorf("expected empty string after complete cycle, got %q", m.categoryFilter)
	}
}

func TestStyleCategoryBadge(t *testing.T) {
	for _, cat := range AvailableCategories {
		badgeStyle := styleCategoryBadge(cat)
		rendered := badgeStyle.Render("[" + cat + "]")
		if rendered == "" {
			t.Errorf("expected non-empty rendered badge for category %q", cat)
		}
	}

	// Unknown category defaults to muted
	unknown := styleCategoryBadge("unknown").Render("[unknown]")
	if unknown == "" {
		t.Errorf("expected non-empty rendered badge for unknown category")
	}
}

func TestRowToPostConverters(t *testing.T) {
	now := pgtype.Timestamptz{Time: time.Now(), Valid: true}

	hotRow := db.ListPostsByBoardHotRow{
		ID:           1,
		BoardID:      2,
		AuthorID:     3,
		Title:        "Hot Post",
		Url:          "https://example.com",
		Score:        42,
		CommentCount: 5,
		CreatedAt:    now,
		IsDeleted:    false,
		Category:     "showcase",
		HotScore:     123.45,
		AuthorHandle: "alice",
	}
	p1 := hotRowToPost(hotRow)
	if p1.ID != 1 || p1.Category != "showcase" || p1.AuthorHandle != "alice" || p1.Score != 42 || p1.HotScore != 123.45 {
		t.Errorf("unexpected conversion from hotRow: %+v", p1)
	}

	newRow := db.ListPostsByBoardNewRow{
		ID:           10,
		BoardID:      2,
		AuthorID:     4,
		Title:        "New Post",
		Url:          "",
		Score:        1,
		CommentCount: 0,
		CreatedAt:    now,
		IsDeleted:    false,
		Category:     "question",
		HotScore:     67.89,
		AuthorHandle: "bob",
	}
	p2 := newRowToPost(newRow)
	if p2.ID != 10 || p2.Category != "question" || p2.AuthorHandle != "bob" || p2.HotScore != 67.89 {
		t.Errorf("unexpected conversion from newRow: %+v", p2)
	}

	topRow := db.ListPostsByBoardTopRow{
		ID:           20,
		BoardID:      2,
		AuthorID:     5,
		Title:        "Top Post",
		Url:          "https://golang.org",
		Score:        100,
		CommentCount: 15,
		CreatedAt:    now,
		IsDeleted:    false,
		Category:     "guide",
		HotScore:     999.99,
		AuthorHandle: "carol",
	}
	p3 := topRowToPost(topRow)
	if p3.ID != 20 || p3.Category != "guide" || p3.AuthorHandle != "carol" || p3.Score != 100 || p3.HotScore != 999.99 {
		t.Errorf("unexpected conversion from topRow: %+v", p3)
	}
}
