package tui

import (
	"sort"
	"strings"

	"github.com/charmbracelet/lipgloss"
	"github.com/jackc/pgx/v5/pgtype"
	db "github.com/sahas/readit/internal/db/sqlc"
)

// PostSortMode defines the sorting order for post feeds.
type PostSortMode int

const (
	PostSortHot PostSortMode = iota
	PostSortNew
	PostSortTop
)

func (s PostSortMode) String() string {
	switch s {
	case PostSortHot:
		return "hot"
	case PostSortNew:
		return "new"
	case PostSortTop:
		return "top"
	default:
		return "hot"
	}
}

// CommentSortMode defines the sorting order for comments in discussion threads.
type CommentSortMode int

const (
	CommentSortTop CommentSortMode = iota
	CommentSortNew
	CommentSortOld
)

func (s CommentSortMode) String() string {
	switch s {
	case CommentSortTop:
		return "top"
	case CommentSortNew:
		return "new"
	case CommentSortOld:
		return "old"
	default:
		return "top"
	}
}

// AvailableCategories lists the standard discussion categories / flairs.
var AvailableCategories = []string{
	"general",
	"discussion",
	"question",
	"showcase",
	"guide",
	"news",
}

// PostFeedItem represents a post row in the board feed, unifying results across sort modes.
type PostFeedItem struct {
	ID           int64
	BoardID      int64
	AuthorID     int64
	Title        string
	Url          string
	Score        int32
	CommentCount int32
	CreatedAt    pgtype.Timestamptz
	IsDeleted    bool
	Category     string
	HotScore     float64
	AuthorHandle string
}

func hotRowToPost(r db.ListPostsByBoardHotRow) PostFeedItem {
	return PostFeedItem{
		ID:           r.ID,
		BoardID:      r.BoardID,
		AuthorID:     r.AuthorID,
		Title:        r.Title,
		Url:          r.Url,
		Score:        r.Score,
		CommentCount: r.CommentCount,
		CreatedAt:    r.CreatedAt,
		IsDeleted:    r.IsDeleted,
		Category:     r.Category,
		HotScore:     r.HotScore,
		AuthorHandle: r.AuthorHandle,
	}
}

func newRowToPost(r db.ListPostsByBoardNewRow) PostFeedItem {
	return PostFeedItem{
		ID:           r.ID,
		BoardID:      r.BoardID,
		AuthorID:     r.AuthorID,
		Title:        r.Title,
		Url:          r.Url,
		Score:        r.Score,
		CommentCount: r.CommentCount,
		CreatedAt:    r.CreatedAt,
		IsDeleted:    r.IsDeleted,
		Category:     r.Category,
		HotScore:     r.HotScore,
		AuthorHandle: r.AuthorHandle,
	}
}

func topRowToPost(r db.ListPostsByBoardTopRow) PostFeedItem {
	return PostFeedItem{
		ID:           r.ID,
		BoardID:      r.BoardID,
		AuthorID:     r.AuthorID,
		Title:        r.Title,
		Url:          r.Url,
		Score:        r.Score,
		CommentCount: r.CommentCount,
		CreatedAt:    r.CreatedAt,
		IsDeleted:    r.IsDeleted,
		Category:     r.Category,
		HotScore:     r.HotScore,
		AuthorHandle: r.AuthorHandle,
	}
}

// styleCategoryBadge returns a styled pill badge for a category flair.
func styleCategoryBadge(category string) lipgloss.Style {
	cat := strings.ToLower(strings.TrimSpace(category))
	switch cat {
	case "discussion":
		return lipgloss.NewStyle().
			Foreground(currentTheme.Secondary).
			Bold(true)
	case "question":
		return lipgloss.NewStyle().
			Foreground(currentTheme.Accent).
			Bold(true)
	case "showcase":
		return lipgloss.NewStyle().
			Foreground(lipgloss.Color("#00E676")).
			Bold(true)
	case "guide":
		return lipgloss.NewStyle().
			Foreground(lipgloss.Color("#BB86FC")).
			Bold(true)
	case "news":
		return lipgloss.NewStyle().
			Foreground(currentTheme.Primary).
			Bold(true)
	default: // general
		return lipgloss.NewStyle().
			Foreground(currentTheme.TextMuted)
	}
}

// sortCommentTree sorts comments within their respective tree hierarchy levels.
// It guarantees that child comments remain grouped beneath their parent while
// ordering siblings according to Top (highest score), New (most recent), or Old (chronological).
func sortCommentTree(comments []db.GetCommentThreadByPostRow, mode CommentSortMode) []db.GetCommentThreadByPostRow {
	if len(comments) <= 1 {
		return comments
	}

	// 1. Group by parent ID
	childrenMap := make(map[int64][]db.GetCommentThreadByPostRow)
	var rootComments []db.GetCommentThreadByPostRow

	for _, c := range comments {
		if c.ParentID.Valid && c.ParentID.Int64 != 0 {
			childrenMap[c.ParentID.Int64] = append(childrenMap[c.ParentID.Int64], c)
		} else {
			rootComments = append(rootComments, c)
		}
	}

	// 2. Sorting comparator for siblings in the same branch
	sortGroup := func(group []db.GetCommentThreadByPostRow) {
		sort.SliceStable(group, func(i, j int) bool {
			switch mode {
			case CommentSortTop:
				if group[i].Score != group[j].Score {
					return group[i].Score > group[j].Score
				}
				return group[i].CreatedAt.Time.Before(group[j].CreatedAt.Time)
			case CommentSortNew:
				return group[i].CreatedAt.Time.After(group[j].CreatedAt.Time)
			case CommentSortOld:
				return group[i].CreatedAt.Time.Before(group[j].CreatedAt.Time)
			default:
				return group[i].Score > group[j].Score
			}
		})
	}

	sortGroup(rootComments)
	for k := range childrenMap {
		sortGroup(childrenMap[k])
	}

	// 3. Reconstruct depth-first flattened tree
	result := make([]db.GetCommentThreadByPostRow, 0, len(comments))
	var traverse func(parentID int64, depth int32)
	traverse = func(parentID int64, depth int32) {
		var list []db.GetCommentThreadByPostRow
		if parentID == 0 {
			list = rootComments
		} else {
			list = childrenMap[parentID]
		}

		for _, item := range list {
			item.Depth = depth
			result = append(result, item)
			traverse(item.ID, depth+1)
		}
	}

	traverse(0, 0)
	return result
}
