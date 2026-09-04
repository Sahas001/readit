// Package tui implements the Bubble Tea terminal UI for ReadIT.
package tui

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/textarea"
	"github.com/charmbracelet/bubbles/textinput"
	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"

	db "github.com/sahas/readit/internal/db/sqlc"
	"github.com/sahas/readit/internal/sanitize"
)

// View state identifiers.
type viewState int

const (
	viewLoading viewState = iota
	viewOnboarding
	viewBoardList
	viewPostList
	viewPostDetail
	viewNewPost
	viewNewComment
	viewError
)

// Model is the root Bubble Tea model for the application.
type Model struct {
	// Dependencies (injected, not owned).
	ctx     context.Context
	pool    *pgxpool.Pool
	queries *db.Queries
	logger  *slog.Logger

	// Session state.
	fingerprint string
	user        *db.User

	// UI state.
	currentView viewState
	width       int
	height      int
	err         error

	// Key bindings.
	keys KeyMap

	// Onboarding.
	handleInput textinput.Model

	// Board list.
	boards      []db.Board
	boardCursor int

	// Post list.
	currentBoard *db.Board
	posts        []db.ListPostsByBoardNewRow
	postCursor   int

	// Post detail & comments.
	currentPost *db.GetPostByIDRow
	comments    []db.GetCommentThreadByPostRow
	viewport    viewport.Model

	// Content creation: New Post.
	titleInput     textinput.Model
	urlInput       textinput.Model
	bodyInput      textarea.Model
	postFormFocus  int // 0: Title, 1: URL, 2: Body

	// Content creation: New Comment.
	commentInput       textarea.Model
	replyParentID      *int64
	replyParentAuthor  string
}

// --- Messages ----------------------------------------------------------

// userLoadedMsg is sent after the user record is fetched/created.
type userLoadedMsg struct {
	user  *db.User
	isNew bool
}

// boardsLoadedMsg carries the list of boards from the database.
type boardsLoadedMsg struct {
	boards []db.Board
}

// postsLoadedMsg carries posts for a board.
type postsLoadedMsg struct {
	posts []db.ListPostsByBoardNewRow
}

// postDetailLoadedMsg carries the post details and its threaded comments.
type postDetailLoadedMsg struct {
	post     *db.GetPostByIDRow
	comments []db.GetCommentThreadByPostRow
}

// postVotedMsg signals that a vote has been counted and score recalculated.
type postVotedMsg struct {
	postID int64
}

// postCreatedMsg signals that a post was published.
type postCreatedMsg struct {
	post db.Post
}

// commentCreatedMsg signals that a comment was published.
type commentCreatedMsg struct {
	comment db.Comment
}

// errMsg wraps an error for the Update loop.
type errMsg struct{ err error }

func (e errMsg) Error() string { return e.err.Error() }

// --- Constructor -------------------------------------------------------

// NewModel creates a new root Model for a Bubble Tea session.
func NewModel(ctx context.Context, pool *pgxpool.Pool, fingerprint string, logger *slog.Logger) *Model {
	ti := textinput.New()
	ti.Placeholder = "choose a handle (e.g. satoshi)"
	ti.CharLimit = 20
	ti.Width = 35

	// New Post Title input
	titleIn := textinput.New()
	titleIn.Placeholder = "Post title..."
	titleIn.CharLimit = 150
	titleIn.Width = 60

	// New Post URL input
	urlIn := textinput.New()
	urlIn.Placeholder = "https://... (optional)"
	urlIn.CharLimit = 250
	urlIn.Width = 60

	// New Post Body textarea
	bodyA := textarea.New()
	bodyA.Placeholder = "Write your post body here (supports markdown)..."
	bodyA.SetWidth(65)
	bodyA.SetHeight(8)
	bodyA.CharLimit = 10000

	// Comment textarea
	commA := textarea.New()
	commA.Placeholder = "Write your reply here..."
	commA.SetWidth(65)
	commA.SetHeight(6)
	commA.CharLimit = 5000

	vp := viewport.New(80, 20)

	return &Model{
		ctx:          ctx,
		pool:         pool,
		queries:      db.New(pool),
		logger:       logger,
		fingerprint:  fingerprint,
		currentView:  viewLoading,
		keys:         DefaultKeyMap(),
		handleInput:  ti,
		titleInput:   titleIn,
		urlInput:     urlIn,
		bodyInput:    bodyA,
		commentInput: commA,
		viewport:     vp,
	}
}

// --- tea.Model interface -----------------------------------------------

// Init kicks off the initial user lookup command.
func (m *Model) Init() tea.Cmd {
	return m.lookupUserCmd()
}

// Update processes messages and returns the updated model and next command.
func (m *Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {

	// Terminal resize.
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		m.viewport.Width = max(20, msg.Width-4)
		m.viewport.Height = max(5, msg.Height-6)
		if m.currentView == viewPostDetail && m.currentPost != nil {
			m.viewport.SetContent(m.renderPostDetailContent())
		}
		return m, nil

	// Global quit.
	case tea.KeyMsg:
		if msg.String() == "ctrl+c" {
			return m, tea.Quit
		}

	// Data messages.
	case userLoadedMsg:
		return m.handleUserLoaded(msg)
	case boardsLoadedMsg:
		m.boards = msg.boards
		m.currentView = viewBoardList
		m.boardCursor = 0
		return m, nil
	case postsLoadedMsg:
		m.posts = msg.posts
		m.currentView = viewPostList
		m.postCursor = 0
		return m, nil
	case postDetailLoadedMsg:
		m.currentPost = msg.post
		m.comments = msg.comments
		m.currentView = viewPostDetail
		m.viewport.SetContent(m.renderPostDetailContent())
		m.viewport.GotoTop()
		return m, nil
	case postVotedMsg:
		// Reload current view data to reflect updated scores
		if m.currentView == viewPostDetail && m.currentPost != nil {
			return m, m.loadPostDetailCmd(m.currentPost.ID)
		} else if m.currentView == viewPostList && m.currentBoard != nil {
			return m, m.loadPostsCmd(m.currentBoard.ID)
		}
		return m, nil
	case postCreatedMsg:
		// Return to post list and reload posts
		m.currentView = viewPostList
		return m, m.loadPostsCmd(msg.post.BoardID)
	case commentCreatedMsg:
		// Return to post detail and reload discussion
		m.currentView = viewPostDetail
		return m, m.loadPostDetailCmd(msg.comment.PostID)
	case errMsg:
		m.err = msg.err
		m.currentView = viewError
		return m, nil
	}

	// Delegate to view-specific handlers.
	switch m.currentView {
	case viewOnboarding:
		return m.updateOnboarding(msg)
	case viewBoardList:
		return m.updateBoardList(msg)
	case viewPostList:
		return m.updatePostList(msg)
	case viewPostDetail:
		return m.updatePostDetail(msg)
	case viewNewPost:
		return m.updateNewPost(msg)
	case viewNewComment:
		return m.updateNewComment(msg)
	}

	return m, nil
}

// View renders the current view.
func (m *Model) View() string {
	switch m.currentView {
	case viewLoading:
		return m.viewLoading()
	case viewOnboarding:
		return m.viewOnboarding()
	case viewBoardList:
		return m.viewBoardList()
	case viewPostList:
		return m.viewPostList()
	case viewPostDetail:
		return m.viewPostDetail()
	case viewNewPost:
		return m.viewNewPost()
	case viewNewComment:
		return m.viewNewComment()
	case viewError:
		return m.viewError()
	default:
		return "Unknown view state"
	}
}

// --- User loading ------------------------------------------------------

func (m *Model) lookupUserCmd() tea.Cmd {
	return func() tea.Msg {
		ctx, cancel := context.WithTimeout(m.ctx, 3*time.Second)
		defer cancel()

		user, err := m.queries.GetUserByPubkey(ctx, m.fingerprint)
		if err != nil {
			if errors.Is(err, pgx.ErrNoRows) {
				return userLoadedMsg{user: nil, isNew: true}
			}
			return errMsg{err: fmt.Errorf("looking up user: %w", err)}
		}
		return userLoadedMsg{user: &user, isNew: false}
	}
}

func (m *Model) handleUserLoaded(msg userLoadedMsg) (*Model, tea.Cmd) {
	if msg.isNew {
		m.currentView = viewOnboarding
		m.handleInput.Focus()
		return m, textinput.Blink
	}
	m.user = msg.user
	m.currentView = viewBoardList
	return m, m.loadBoardsCmd()
}

// --- Onboarding --------------------------------------------------------

func (m *Model) updateOnboarding(msg tea.Msg) (*Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "enter":
			handle := strings.TrimSpace(m.handleInput.Value())
			if err := sanitize.ValidateHandle(handle); err != nil {
				m.err = err
				return m, nil
			}
			m.err = nil
			return m, m.createUserCmd(handle)
		case "esc":
			return m, tea.Quit
		default:
			m.err = nil
		}
	}

	var cmd tea.Cmd
	m.handleInput, cmd = m.handleInput.Update(msg)
	return m, cmd
}

func (m *Model) createUserCmd(handle string) tea.Cmd {
	return func() tea.Msg {
		ctx, cancel := context.WithTimeout(m.ctx, 3*time.Second)
		defer cancel()

		user, err := m.queries.UpsertUser(ctx, db.UpsertUserParams{
			PubkeySha256: m.fingerprint,
			Handle:       handle,
		})
		if err != nil {
			return errMsg{err: fmt.Errorf("creating user: %w", err)}
		}
		return userLoadedMsg{user: &user, isNew: false}
	}
}

// --- Board list --------------------------------------------------------

func (m *Model) loadBoardsCmd() tea.Cmd {
	return func() tea.Msg {
		ctx, cancel := context.WithTimeout(m.ctx, 3*time.Second)
		defer cancel()

		boards, err := m.queries.ListBoards(ctx)
		if err != nil {
			return errMsg{err: fmt.Errorf("loading boards: %w", err)}
		}
		return boardsLoadedMsg{boards: boards}
	}
}

func (m *Model) updateBoardList(msg tea.Msg) (*Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch {
		case msg.String() == "q":
			return m, tea.Quit
		case msg.String() == "k" || msg.String() == "up":
			if m.boardCursor > 0 {
				m.boardCursor--
			}
		case msg.String() == "j" || msg.String() == "down":
			if m.boardCursor < len(m.boards)-1 {
				m.boardCursor++
			}
		case msg.String() == "enter":
			if len(m.boards) > 0 {
				board := m.boards[m.boardCursor]
				m.currentBoard = &board
				return m, m.loadPostsCmd(board.ID)
			}
		}
	}
	return m, nil
}

// --- Post list ---------------------------------------------------------

func (m *Model) loadPostsCmd(boardID int64) tea.Cmd {
	return func() tea.Msg {
		ctx, cancel := context.WithTimeout(m.ctx, 3*time.Second)
		defer cancel()

		posts, err := m.queries.ListPostsByBoardNew(ctx, db.ListPostsByBoardNewParams{
			BoardID: boardID,
			Limit:   50,
			Offset:  0,
		})
		if err != nil {
			return errMsg{err: fmt.Errorf("loading posts: %w", err)}
		}
		return postsLoadedMsg{posts: posts}
	}
}

func (m *Model) updatePostList(msg tea.Msg) (*Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch {
		case msg.String() == "q":
			return m, tea.Quit
		case msg.String() == "esc":
			m.currentView = viewBoardList
			return m, nil
		case msg.String() == "k" || msg.String() == "up":
			if m.postCursor > 0 {
				m.postCursor--
			}
		case msg.String() == "j" || msg.String() == "down":
			if m.postCursor < len(m.posts)-1 {
				m.postCursor++
			}
		case msg.String() == "enter":
			if len(m.posts) > 0 {
				post := m.posts[m.postCursor]
				return m, m.loadPostDetailCmd(post.ID)
			}
		case msg.String() == "n":
			return m, m.openNewPost()
		case msg.String() == "u":
			if len(m.posts) > 0 {
				post := m.posts[m.postCursor]
				return m, m.castPostVoteCmd(post.ID, 1)
			}
		case msg.String() == "d":
			if len(m.posts) > 0 {
				post := m.posts[m.postCursor]
				return m, m.castPostVoteCmd(post.ID, -1)
			}
		}
	}
	return m, nil
}

// --- Post detail -------------------------------------------------------

func (m *Model) loadPostDetailCmd(postID int64) tea.Cmd {
	return func() tea.Msg {
		ctx, cancel := context.WithTimeout(m.ctx, 3*time.Second)
		defer cancel()

		post, err := m.queries.GetPostByID(ctx, postID)
		if err != nil {
			return errMsg{err: fmt.Errorf("loading post: %w", err)}
		}

		comments, err := m.queries.GetCommentThreadByPost(ctx, db.GetCommentThreadByPostParams{
			PostID: postID,
			Limit:  200,
		})
		if err != nil {
			return errMsg{err: fmt.Errorf("loading comments: %w", err)}
		}

		return postDetailLoadedMsg{post: &post, comments: comments}
	}
}

func (m *Model) updatePostDetail(msg tea.Msg) (*Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "esc":
			m.currentView = viewPostList
			return m, nil
		case "q":
			return m, tea.Quit
		case "r":
			if m.currentPost != nil {
				return m, m.openNewComment(nil, m.currentPost.AuthorHandle)
			}
		case "u":
			if m.currentPost != nil {
				return m, m.castPostVoteCmd(m.currentPost.ID, 1)
			}
		case "d":
			if m.currentPost != nil {
				return m, m.castPostVoteCmd(m.currentPost.ID, -1)
			}
		}
	}

	var cmd tea.Cmd
	m.viewport, cmd = m.viewport.Update(msg)
	return m, cmd
}

// --- Voting ------------------------------------------------------------

func (m *Model) castPostVoteCmd(postID int64, direction int16) tea.Cmd {
	return func() tea.Msg {
		if m.user == nil {
			return nil
		}
		ctx, cancel := context.WithTimeout(m.ctx, 3*time.Second)
		defer cancel()

		err := m.queries.UpsertPostVote(ctx, db.UpsertPostVoteParams{
			UserID:    m.user.ID,
			PostID:    postID,
			Direction: direction,
		})
		if err != nil {
			return errMsg{err: fmt.Errorf("voting on post: %w", err)}
		}

		// Recalculate denormalized score
		if err := m.queries.RecalculatePostScore(ctx, postID); err != nil {
			return errMsg{err: fmt.Errorf("recalculating score: %w", err)}
		}

		return postVotedMsg{postID: postID}
	}
}

// --- Content Creation: Post --------------------------------------------

func (m *Model) openNewPost() tea.Cmd {
	m.currentView = viewNewPost
	m.postFormFocus = 0
	m.titleInput.Reset()
	m.urlInput.Reset()
	m.bodyInput.Reset()
	m.titleInput.Focus()
	m.urlInput.Blur()
	m.bodyInput.Blur()
	m.err = nil
	return textinput.Blink
}

func (m *Model) updateNewPost(msg tea.Msg) (*Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "esc":
			m.currentView = viewPostList
			return m, nil
		case "tab":
			m.postFormFocus = (m.postFormFocus + 1) % 3
			switch m.postFormFocus {
			case 0:
				m.titleInput.Focus()
				m.urlInput.Blur()
				m.bodyInput.Blur()
			case 1:
				m.titleInput.Blur()
				m.urlInput.Focus()
				m.bodyInput.Blur()
			case 2:
				m.titleInput.Blur()
				m.urlInput.Blur()
				m.bodyInput.Focus()
			}
			return m, nil
		case "ctrl+s":
			title := strings.TrimSpace(m.titleInput.Value())
			if title == "" {
				m.err = fmt.Errorf("title cannot be empty")
				return m, nil
			}
			url := strings.TrimSpace(m.urlInput.Value())
			body := strings.TrimSpace(m.bodyInput.Value())
			return m, m.submitPostCmd(title, body, url)
		}
	}

	var cmd tea.Cmd
	switch m.postFormFocus {
	case 0:
		m.titleInput, cmd = m.titleInput.Update(msg)
	case 1:
		m.urlInput, cmd = m.urlInput.Update(msg)
	case 2:
		m.bodyInput, cmd = m.bodyInput.Update(msg)
	}
	return m, cmd
}

func (m *Model) submitPostCmd(title, body, url string) tea.Cmd {
	return func() tea.Msg {
		if m.user == nil || m.currentBoard == nil {
			return errMsg{err: fmt.Errorf("missing user or board context")}
		}
		ctx, cancel := context.WithTimeout(m.ctx, 3*time.Second)
		defer cancel()

		post, err := m.queries.CreatePost(ctx, db.CreatePostParams{
			BoardID:  m.currentBoard.ID,
			AuthorID: m.user.ID,
			Title:    sanitize.SingleLine(title),
			Body:     sanitize.Text(body),
			Url:      sanitize.SingleLine(url),
		})
		if err != nil {
			return errMsg{err: fmt.Errorf("creating post: %w", err)}
		}
		return postCreatedMsg{post: post}
	}
}

// --- Content Creation: Comment -----------------------------------------

func (m *Model) openNewComment(parentID *int64, parentAuthor string) tea.Cmd {
	m.currentView = viewNewComment
	m.replyParentID = parentID
	m.replyParentAuthor = parentAuthor
	m.commentInput.Reset()
	m.commentInput.Focus()
	m.err = nil
	return textarea.Blink
}

func (m *Model) updateNewComment(msg tea.Msg) (*Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "esc":
			m.currentView = viewPostDetail
			return m, nil
		case "ctrl+s":
			body := strings.TrimSpace(m.commentInput.Value())
			if body == "" {
				m.err = fmt.Errorf("comment cannot be empty")
				return m, nil
			}
			return m, m.submitCommentCmd(body)
		}
	}

	var cmd tea.Cmd
	m.commentInput, cmd = m.commentInput.Update(msg)
	return m, cmd
}

func (m *Model) submitCommentCmd(body string) tea.Cmd {
	return func() tea.Msg {
		if m.user == nil || m.currentPost == nil {
			return errMsg{err: fmt.Errorf("missing user or post context")}
		}
		ctx, cancel := context.WithTimeout(m.ctx, 3*time.Second)
		defer cancel()

		var parentID pgtype.Int8
		if m.replyParentID != nil {
			parentID = pgtype.Int8{Int64: *m.replyParentID, Valid: true}
		}

		comment, err := m.queries.CreateComment(ctx, db.CreateCommentParams{
			PostID:   m.currentPost.ID,
			ParentID: parentID,
			AuthorID: m.user.ID,
			Body:     sanitize.Text(body),
		})
		if err != nil {
			return errMsg{err: fmt.Errorf("creating comment: %w", err)}
		}

		// Increment denormalized comment count
		_ = m.queries.IncrementPostCommentCount(ctx, m.currentPost.ID)

		return commentCreatedMsg{comment: comment}
	}
}
