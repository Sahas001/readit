// Package db contains generated code from sqlc. DO NOT EDIT (except this stub).
// Run `make sqlc` to regenerate from SQL queries.
package db

import (
	"context"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
)

// DBTX is the interface that both pgxpool.Pool and pgx.Tx satisfy.
type DBTX interface {
	Exec(context.Context, string, ...interface{}) (pgconn.CommandTag, error)
	Query(context.Context, string, ...interface{}) (pgx.Rows, error)
	QueryRow(context.Context, string, ...interface{}) pgx.Row
}

// Queries provides all database query methods.
type Queries struct {
	db DBTX
}

// New creates a new Queries instance.
func New(db DBTX) *Queries {
	return &Queries{db: db}
}

// WithTx returns a copy of Queries that uses the provided transaction.
func (q *Queries) WithTx(tx pgx.Tx) *Queries {
	return &Queries{db: tx}
}

// --- Models ----------------------------------------------------------------

// User represents a row in the users table.
type User struct {
	ID           int64     `json:"id"`
	PubkeySha256 string    `json:"pubkey_sha256"`
	Handle       string    `json:"handle"`
	Bio          string    `json:"bio"`
	CreatedAt    time.Time `json:"created_at"`
	UpdatedAt    time.Time `json:"updated_at"`
}

// Board represents a row in the boards table.
type Board struct {
	ID          int64     `json:"id"`
	Slug        string    `json:"slug"`
	Title       string    `json:"title"`
	Description string    `json:"description"`
	CreatedAt   time.Time `json:"created_at"`
}

// Post represents a row in the posts table.
type Post struct {
	ID           int64     `json:"id"`
	BoardID      int64     `json:"board_id"`
	AuthorID     int64     `json:"author_id"`
	Title        string    `json:"title"`
	Body         string    `json:"body"`
	URL          string    `json:"url"`
	Score        int32     `json:"score"`
	CommentCount int32     `json:"comment_count"`
	CreatedAt    time.Time `json:"created_at"`
	UpdatedAt    time.Time `json:"updated_at"`
}

// Comment represents a row in the comments table.
type Comment struct {
	ID        int64     `json:"id"`
	PostID    int64     `json:"post_id"`
	ParentID  *int64    `json:"parent_id"`
	AuthorID  int64     `json:"author_id"`
	Body      string    `json:"body"`
	Score     int32     `json:"score"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

// --- Query result types ----------------------------------------------------

// ListPostsByBoardNewRow is the result type for ListPostsByBoardNew.
type ListPostsByBoardNewRow struct {
	ID           int64     `json:"id"`
	BoardID      int64     `json:"board_id"`
	AuthorID     int64     `json:"author_id"`
	Title        string    `json:"title"`
	URL          string    `json:"url"`
	Score        int32     `json:"score"`
	CommentCount int32     `json:"comment_count"`
	CreatedAt    time.Time `json:"created_at"`
	AuthorHandle string    `json:"author_handle"`
}

// --- Query params ----------------------------------------------------------

// UpsertUserParams holds parameters for UpsertUser.
type UpsertUserParams struct {
	PubkeySha256 string `json:"pubkey_sha256"`
	Handle       string `json:"handle"`
}

// ListPostsByBoardNewParams holds parameters for ListPostsByBoardNew.
type ListPostsByBoardNewParams struct {
	BoardID int64 `json:"board_id"`
	Limit   int32 `json:"limit"`
	Offset  int32 `json:"offset"`
}
