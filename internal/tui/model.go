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
	viewDeleteConfirm
	viewError
)

type deleteTargetType int

const (
	deleteTargetPost deleteTargetType = iota
	deleteTargetComment
)

type deleteTarget struct {
	targetType    deleteTargetType
	id            int64
	postID        int64
	authorID      int64
	titleOrBody   string
	hasDependents bool
	commentCount  int32
	returnView    viewState
}

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
	currentPost        *db.GetPostByIDRow
	comments           []db.GetCommentThreadByPostRow
	viewport           viewport.Model
	commentCursor      int
	commentLineOffsets []int

	// Micro-interactions and animations.
	flashMsg string
	animTick int // Animation tick counter for Earth rotation and logo shine.

	// Content creation: New Post.
	titleInput    textinput.Model
	urlInput      textinput.Model
	bodyInput     textarea.Model
	postFormFocus int // 0: Title, 1: URL, 2: Body

	// Content creation: New Comment.
	commentInput      textarea.Model
	replyParentID     *int64
	replyParentAuthor string

	// Deletion confirmation.
	pendingDelete *deleteTarget
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
	postID    int64
	direction int16
}

// commentVotedMsg signals that a comment vote has been counted.
type commentVotedMsg struct {
	commentID int64
	direction int16
}

// postCreatedMsg signals that a post was published.
type postCreatedMsg struct {
	post db.Post
}

// commentCreatedMsg signals that a comment was published.
type commentCreatedMsg struct {
	comment db.Comment
}

// postDeletedMsg signals that a post was deleted.
type postDeletedMsg struct {
	postID int64
	isSoft bool
}

// commentDeletedMsg signals that a comment was deleted.
type commentDeletedMsg struct {
	commentID int64
	postID    int64
	isSoft    bool
}

// postPrunedMsg is sent when a soft-deleted post has had its last comment removed and is purged.
type postPrunedMsg struct {
	postID int64
}

// animTickMsg drives the Earth rotation and logo shine animation on the landing page.
type animTickMsg struct{}

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
	bodyA.ShowLineNumbers = false
	bodyA.Prompt = ""
	bodyA.SetWidth(65)
	bodyA.SetHeight(8)
	bodyA.CharLimit = 10000

	// Comment textarea
	commA := textarea.New()
	commA.Placeholder = "Write your reply here..."
	commA.ShowLineNumbers = false
	commA.Prompt = ""
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
		viewport:      vp,
		commentCursor: -1,
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
		if msg.Width > 0 {
			m.width = msg.Width
		}
		if msg.Height > 0 {
			m.height = msg.Height
		}
		m.resizeInputs()
		m.updateViewportSize()
		if m.currentView == viewPostDetail && m.currentPost != nil {
			m.viewport.SetContent(m.renderPostDetailContent())
		}
		return m, nil

	// Global quit & flash clearing.
	case tea.KeyMsg:
		if msg.String() == "ctrl+c" {
			return m, tea.Quit
		}
		if msg.String() != "u" && msg.String() != "d" {
			m.flashMsg = ""
		}

	// Data messages.
	case userLoadedMsg:
		return m.handleUserLoaded(msg)
	case boardsLoadedMsg:
		m.boards = msg.boards
		m.currentView = viewBoardList
		m.boardCursor = 0
		m.animTick = 0
		return m, m.animTickCmd()
	case postsLoadedMsg:
		prevCursor := m.postCursor
		m.posts = msg.posts
		m.currentView = viewPostList
		if prevCursor < len(m.posts) {
			m.postCursor = prevCursor
		} else if len(m.posts) > 0 {
			m.postCursor = len(m.posts) - 1
		} else {
			m.postCursor = 0
		}
		return m, nil
	case postDetailLoadedMsg:
		samePost := (m.currentPost != nil && m.currentPost.ID == msg.post.ID)
		prevCursor := m.commentCursor
		prevYOffset := m.viewport.YOffset

		m.currentPost = msg.post
		m.comments = msg.comments
		m.currentView = viewPostDetail

		if samePost {
			m.commentCursor = prevCursor
			if m.commentCursor >= len(m.comments) {
				m.commentCursor = len(m.comments) - 1
			}
		} else {
			m.commentCursor = -1
		}

		m.updateViewportSize()
		m.viewport.SetContent(m.renderPostDetailContent())

		if samePost {
			m.viewport.SetYOffset(prevYOffset)
		} else {
			m.viewport.GotoTop()
		}
		return m, nil
	case postVotedMsg:
		if msg.direction > 0 {
			m.flashMsg = "▲ Upvoted"
		} else if msg.direction < 0 {
			m.flashMsg = "▼ Downvoted"
		} else {
			m.flashMsg = "• Vote removed"
		}
		// Reload current view data to reflect updated scores
		if m.currentView == viewPostDetail && m.currentPost != nil {
			return m, m.loadPostDetailCmd(m.currentPost.ID)
		} else if m.currentView == viewPostList && m.currentBoard != nil {
			return m, m.loadPostsCmd(m.currentBoard.ID)
		}
		return m, nil
	case commentVotedMsg:
		if msg.direction > 0 {
			m.flashMsg = "▲ Upvoted"
		} else if msg.direction < 0 {
			m.flashMsg = "▼ Downvoted"
		} else {
			m.flashMsg = "• Vote removed"
		}
		if m.currentView == viewPostDetail && m.currentPost != nil {
			return m, m.loadPostDetailCmd(m.currentPost.ID)
		}
		return m, nil
	case postCreatedMsg:
		// Return to post list and reload posts
		m.flashMsg = "✓ Post published!"
		m.currentView = viewPostList
		return m, m.loadPostsCmd(msg.post.BoardID)
	case commentCreatedMsg:
		// Return to post detail and reload discussion
		m.flashMsg = "✓ Reply posted!"
		m.currentView = viewPostDetail
		return m, m.loadPostDetailCmd(msg.comment.PostID)
	case postDeletedMsg:
		// Return to post list and reload posts
		m.currentView = viewPostList
		m.pendingDelete = nil
		if msg.isSoft {
			m.flashMsg = "✓ Post deleted (content scrubbed)"
		} else {
			m.flashMsg = "✓ Post permanently deleted"
		}
		if m.currentBoard != nil {
			return m, m.loadPostsCmd(m.currentBoard.ID)
		}
		return m, nil
	case commentDeletedMsg:
		// Return to post detail and reload comments
		m.currentView = viewPostDetail
		m.pendingDelete = nil
		if msg.isSoft {
			m.flashMsg = "✓ Comment deleted (marked [deleted])"
		} else {
			m.flashMsg = "✓ Comment permanently deleted"
		}
		if m.currentPost != nil {
			return m, m.loadPostDetailCmd(msg.postID)
		}
		return m, nil
	case postPrunedMsg:
		m.currentView = viewPostList
		m.pendingDelete = nil
		m.flashMsg = "✓ Thread closed & pruned (all comments deleted)"
		if m.currentBoard != nil {
			return m, m.loadPostsCmd(m.currentBoard.ID)
		}
		return m, nil
	case animTickMsg:
		// Only advance animation while on the landing page.
		if m.currentView == viewBoardList {
			m.animTick++
			return m, m.animTickCmd()
		}
		return m, nil
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
	case viewDeleteConfirm:
		return m.updateDeleteConfirm(msg)
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
	case viewDeleteConfirm:
		return m.viewDeleteConfirm()
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

// animTickCmd returns a command that sends an animTickMsg after the animation interval.
func (m *Model) animTickCmd() tea.Cmd {
	return tea.Tick(animationInterval, func(time.Time) tea.Msg {
		return animTickMsg{}
	})
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
		case msg.String() == "g" || msg.String() == "home":
			m.boardCursor = 0
		case msg.String() == "G" || msg.String() == "end":
			if len(m.boards) > 0 {
				m.boardCursor = len(m.boards) - 1
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

		// Proactively prune any empty soft-deleted posts
		_ = m.queries.PruneAllEmptyDeletedPosts(ctx)

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
			return m, m.animTickCmd()
		case msg.String() == "k" || msg.String() == "up":
			if m.postCursor > 0 {
				m.postCursor--
			}
		case msg.String() == "j" || msg.String() == "down":
			if m.postCursor < len(m.posts)-1 {
				m.postCursor++
			}
		case msg.String() == "g" || msg.String() == "home":
			m.postCursor = 0
		case msg.String() == "G" || msg.String() == "end":
			if len(m.posts) > 0 {
				m.postCursor = len(m.posts) - 1
			}
		case msg.String() == "enter":
			if len(m.posts) > 0 {
				post := m.posts[m.postCursor]
				return m, m.loadPostDetailCmd(post.ID)
			}
		case msg.String() == "n":
			return m, m.openNewPost()
		case msg.String() == "x":
			if len(m.posts) == 0 {
				return m, nil
			}
			post := m.posts[m.postCursor]
			if post.IsDeleted {
				m.flashMsg = "• Post is already deleted"
				return m, nil
			}
			if m.user == nil || post.AuthorID != m.user.ID {
				m.flashMsg = "✗ You can only delete your own posts"
				return m, nil
			}
			m.pendingDelete = &deleteTarget{
				targetType:    deleteTargetPost,
				id:            post.ID,
				postID:        post.ID,
				authorID:      post.AuthorID,
				titleOrBody:   post.Title,
				hasDependents: post.CommentCount > 0,
				commentCount:  post.CommentCount,
				returnView:    viewPostList,
			}
			m.currentView = viewDeleteConfirm
			return m, nil
		case msg.String() == "u":
			if len(m.posts) > 0 {
				post := m.posts[m.postCursor]
				if post.IsDeleted {
					m.flashMsg = "• Voting is disabled on deleted content"
					return m, nil
				}
				return m, m.castPostVoteCmd(post.ID, 1)
			}
		case msg.String() == "d":
			if len(m.posts) > 0 {
				post := m.posts[m.postCursor]
				if post.IsDeleted {
					m.flashMsg = "• Voting is disabled on deleted content"
					return m, nil
				}
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
			if errors.Is(err, pgx.ErrNoRows) {
				return postPrunedMsg{postID: postID}
			}
			return errMsg{err: fmt.Errorf("loading post: %w", err)}
		}

		// If this post is soft-deleted and already has 0 comments, prune immediately
		if post.IsDeleted && post.CommentCount == 0 {
			_ = m.queries.PruneDeletedPostIfEmpty(ctx, postID)
			return postPrunedMsg{postID: postID}
		}

		comments, err := m.queries.GetCommentThreadByPost(ctx, db.GetCommentThreadByPostParams{
			PostID: postID,
			Limit:  200,
		})
		if err != nil {
			return errMsg{err: fmt.Errorf("loading comments: %w", err)}
		}

		// If post is deleted and all remaining comments are tombstones, prune the entire thread
		if post.IsDeleted {
			allTombstones := true
			for _, c := range comments {
				if !c.IsDeleted {
					allTombstones = false
					break
				}
			}
			if allTombstones {
				_ = m.queries.PruneTombstoneComments(ctx, postID)
				_ = m.queries.RecalculatePostCommentCount(ctx, postID)
				_ = m.queries.PruneDeletedPostIfEmpty(ctx, postID)
				return postPrunedMsg{postID: postID}
			}
		}

		return postDetailLoadedMsg{post: &post, comments: comments}
	}
}

func (m *Model) updateViewportSize() {
	_, _, contentWidth, contentHeight := m.shellDimensions()
	m.viewport.Width = max(20, contentWidth)
	m.viewport.Height = max(4, contentHeight)
}

func (m *Model) formDimensions() (cardWidth, contentWidth, inputWidth int) {
	_, _, maxContentW, _ := m.shellDimensions()
	cardWidth = min(74, max(36, maxContentW-4))
	// Card has Border(1 left + 1 right = 2) and Padding(1, 2 = 4 horizontal).
	// Content width inside card is cardWidth - 6.
	contentWidth = max(16, cardWidth-6)

	// Input boxes have Border(1 left + 1 right = 2) and Padding(0, 1 = 2 horizontal).
	// Inner input component width must be contentWidth - 4 so the box outer width equals contentWidth.
	inputWidth = max(12, contentWidth-4)
	return cardWidth, contentWidth, inputWidth
}

func (m *Model) resizeInputs() {
	_, _, inputWidth := m.formDimensions()

	m.titleInput.Width = inputWidth
	m.urlInput.Width = inputWidth
	m.bodyInput.SetWidth(inputWidth)
	m.commentInput.SetWidth(inputWidth)

	bodyHeight := 8
	if m.height > 0 {
		bodyHeight = min(12, max(4, m.height-18))
	}
	m.bodyInput.SetHeight(bodyHeight)

	commHeight := 6
	if m.height > 0 {
		commHeight = min(10, max(4, m.height-16))
	}
	m.commentInput.SetHeight(commHeight)
}

func (m *Model) ensureCommentVisible(idx int) {
	if idx < 0 || idx >= len(m.commentLineOffsets) {
		return
	}
	targetLine := m.commentLineOffsets[idx]
	if targetLine < m.viewport.YOffset {
		m.viewport.SetYOffset(max(0, targetLine-1))
	} else if targetLine >= m.viewport.YOffset+m.viewport.Height-2 {
		m.viewport.SetYOffset(max(0, targetLine-m.viewport.Height+4))
	}
}

func (m *Model) updatePostDetail(msg tea.Msg) (*Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "esc":
			m.currentView = viewPostList
			if m.currentBoard != nil {
				return m, m.loadPostsCmd(m.currentBoard.ID)
			}
			return m, nil
		case "q":
			return m, tea.Quit
		case "j", "down":
			if len(m.comments) > 0 && m.commentCursor < len(m.comments)-1 {
				m.commentCursor++
				m.viewport.SetContent(m.renderPostDetailContent())
				m.ensureCommentVisible(m.commentCursor)
				return m, nil
			}
		case "k", "up":
			if m.commentCursor > -1 {
				m.commentCursor--
				m.viewport.SetContent(m.renderPostDetailContent())
				if m.commentCursor == -1 {
					m.viewport.GotoTop()
				} else {
					m.ensureCommentVisible(m.commentCursor)
				}
				return m, nil
			}
		case "g", "home":
			m.commentCursor = -1
			m.viewport.SetContent(m.renderPostDetailContent())
			m.viewport.GotoTop()
			return m, nil
		case "G", "end":
			if len(m.comments) > 0 {
				m.commentCursor = len(m.comments) - 1
				m.viewport.SetContent(m.renderPostDetailContent())
				m.viewport.GotoBottom()
				return m, nil
			}
		case "tab":
			if len(m.comments) > 0 {
				if m.commentCursor >= len(m.comments)-1 {
					m.commentCursor = -1
					m.viewport.SetContent(m.renderPostDetailContent())
					m.viewport.GotoTop()
				} else {
					m.commentCursor++
					m.viewport.SetContent(m.renderPostDetailContent())
					m.ensureCommentVisible(m.commentCursor)
				}
				return m, nil
			}
		case "shift+tab":
			if len(m.comments) > 0 {
				if m.commentCursor <= -1 {
					m.commentCursor = len(m.comments) - 1
				} else {
					m.commentCursor--
				}
				m.viewport.SetContent(m.renderPostDetailContent())
				if m.commentCursor == -1 {
					m.viewport.GotoTop()
				} else {
					m.ensureCommentVisible(m.commentCursor)
				}
				return m, nil
			}
		case "r":
			if m.commentCursor >= 0 && m.commentCursor < len(m.comments) {
				comment := m.comments[m.commentCursor]
				if comment.IsDeleted {
					m.flashMsg = "• Cannot reply to a deleted comment"
					return m, nil
				}
				return m, m.openNewComment(&comment.ID, comment.AuthorHandle)
			}
			if m.currentPost != nil {
				if m.currentPost.IsDeleted {
					m.flashMsg = "• Post is deleted. Discussion is locked."
					return m, nil
				}
				return m, m.openNewComment(nil, m.currentPost.AuthorHandle)
			}
		case "R":
			if m.currentPost != nil {
				if m.currentPost.IsDeleted {
					m.flashMsg = "• Post is deleted. Discussion is locked."
					return m, nil
				}
				return m, m.openNewComment(nil, m.currentPost.AuthorHandle)
			}
		case "u":
			if m.commentCursor >= 0 && m.commentCursor < len(m.comments) {
				if m.comments[m.commentCursor].IsDeleted {
					m.flashMsg = "• Voting is disabled on deleted content"
					return m, nil
				}
				return m, m.castCommentVoteCmd(m.comments[m.commentCursor].ID, 1)
			}
			if m.currentPost != nil {
				if m.currentPost.IsDeleted {
					m.flashMsg = "• Voting is disabled on deleted content"
					return m, nil
				}
				return m, m.castPostVoteCmd(m.currentPost.ID, 1)
			}
		case "d":
			if m.commentCursor >= 0 && m.commentCursor < len(m.comments) {
				if m.comments[m.commentCursor].IsDeleted {
					m.flashMsg = "• Voting is disabled on deleted content"
					return m, nil
				}
				return m, m.castCommentVoteCmd(m.comments[m.commentCursor].ID, -1)
			}
			if m.currentPost != nil {
				if m.currentPost.IsDeleted {
					m.flashMsg = "• Voting is disabled on deleted content"
					return m, nil
				}
				return m, m.castPostVoteCmd(m.currentPost.ID, -1)
			}
		case "x":
			if m.commentCursor == -1 {
				// Post is selected
				if m.currentPost == nil {
					return m, nil
				}
				if m.currentPost.IsDeleted {
					m.flashMsg = "• Post is already deleted"
					return m, nil
				}
				if m.user == nil || m.currentPost.AuthorID != m.user.ID {
					m.flashMsg = "✗ You can only delete your own posts"
					return m, nil
				}
				hasComments := len(m.comments) > 0 || m.currentPost.CommentCount > 0
				m.pendingDelete = &deleteTarget{
					targetType:    deleteTargetPost,
					id:            m.currentPost.ID,
					postID:        m.currentPost.ID,
					authorID:      m.currentPost.AuthorID,
					titleOrBody:   m.currentPost.Title,
					hasDependents: hasComments,
					commentCount:  m.currentPost.CommentCount,
					returnView:    viewPostList,
				}
				m.currentView = viewDeleteConfirm
				return m, nil
			} else if m.commentCursor >= 0 && m.commentCursor < len(m.comments) {
				// Comment is selected
				c := m.comments[m.commentCursor]
				if c.IsDeleted {
					m.flashMsg = "• Comment is already deleted"
					return m, nil
				}
				if m.user == nil || c.AuthorID != m.user.ID {
					m.flashMsg = "✗ You can only delete your own comments"
					return m, nil
				}
				hasReplies := false
				for _, other := range m.comments {
					if other.ParentID.Valid && other.ParentID.Int64 == c.ID {
						hasReplies = true
						break
					}
				}
				m.pendingDelete = &deleteTarget{
					targetType:    deleteTargetComment,
					id:            c.ID,
					postID:        m.currentPost.ID,
					authorID:      c.AuthorID,
					titleOrBody:   c.Body,
					hasDependents: hasReplies,
					returnView:    viewPostDetail,
				}
				m.currentView = viewDeleteConfirm
				return m, nil
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

		existing, err := m.queries.GetPostVoteByUser(ctx, db.GetPostVoteByUserParams{
			UserID: m.user.ID,
			PostID: postID,
		})

		var newDirection int16
		if err == nil && existing.Direction == direction {
			// User pressed the same direction again -> toggle off (delete vote)
			if err := m.queries.DeletePostVote(ctx, db.DeletePostVoteParams{
				UserID: m.user.ID,
				PostID: postID,
			}); err != nil {
				return errMsg{err: fmt.Errorf("deleting post vote: %w", err)}
			}
			newDirection = 0
		} else {
			// New vote or flipping from upvote to downvote (or vice versa)
			if err := m.queries.UpsertPostVote(ctx, db.UpsertPostVoteParams{
				UserID:    m.user.ID,
				PostID:    postID,
				Direction: direction,
			}); err != nil {
				return errMsg{err: fmt.Errorf("voting on post: %w", err)}
			}
			newDirection = direction
		}

		// Recalculate denormalized score
		if err := m.queries.RecalculatePostScore(ctx, postID); err != nil {
			return errMsg{err: fmt.Errorf("recalculating score: %w", err)}
		}

		return postVotedMsg{postID: postID, direction: newDirection}
	}
}

func (m *Model) castCommentVoteCmd(commentID int64, direction int16) tea.Cmd {
	return func() tea.Msg {
		if m.user == nil {
			return nil
		}
		ctx, cancel := context.WithTimeout(m.ctx, 3*time.Second)
		defer cancel()

		existing, err := m.queries.GetCommentVoteByUser(ctx, db.GetCommentVoteByUserParams{
			UserID:    m.user.ID,
			CommentID: commentID,
		})

		var newDirection int16
		if err == nil && existing.Direction == direction {
			// User pressed the same direction again -> toggle off (delete vote)
			if err := m.queries.DeleteCommentVote(ctx, db.DeleteCommentVoteParams{
				UserID:    m.user.ID,
				CommentID: commentID,
			}); err != nil {
				return errMsg{err: fmt.Errorf("deleting comment vote: %w", err)}
			}
			newDirection = 0
		} else {
			// New vote or flipping from upvote to downvote (or vice versa)
			if err := m.queries.UpsertCommentVote(ctx, db.UpsertCommentVoteParams{
				UserID:    m.user.ID,
				CommentID: commentID,
				Direction: direction,
			}); err != nil {
				return errMsg{err: fmt.Errorf("voting on comment: %w", err)}
			}
			newDirection = direction
		}

		if err := m.queries.RecalculateCommentScore(ctx, commentID); err != nil {
			return errMsg{err: fmt.Errorf("recalculating comment score: %w", err)}
		}

		return commentVotedMsg{commentID: commentID, direction: newDirection}
	}
}

// --- Content Creation: Post --------------------------------------------

func (m *Model) openNewPost() tea.Cmd {
	m.currentView = viewNewPost
	m.postFormFocus = 0
	m.titleInput.Reset()
	m.urlInput.Reset()
	m.bodyInput.Reset()
	m.resizeInputs()
	m.syncPostFormFocus()
	m.err = nil
	return textinput.Blink
}

func (m *Model) syncPostFormFocus() {
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
}

func (m *Model) updateNewPost(msg tea.Msg) (*Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch {
		case msg.String() == "esc":
			m.currentView = viewPostList
			return m, nil
		case msg.String() == "tab":
			m.postFormFocus = (m.postFormFocus + 1) % 3
			m.syncPostFormFocus()
			return m, nil
		case msg.String() == "shift+tab" || msg.Type == tea.KeyShiftTab:
			m.postFormFocus = (m.postFormFocus - 1 + 3) % 3
			m.syncPostFormFocus()
			return m, nil
		case msg.String() == "enter":
			if m.postFormFocus == 0 {
				m.postFormFocus = 1
				m.syncPostFormFocus()
				return m, nil
			} else if m.postFormFocus == 1 {
				m.postFormFocus = 2
				m.syncPostFormFocus()
				return m, nil
			}
			// When postFormFocus == 2 (body textarea), enter inserts a newline
		case msg.String() == "ctrl+s":
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
	m.resizeInputs()
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

// --- Deletion Confirmation ---------------------------------------------

func (m *Model) updateDeleteConfirm(msg tea.Msg) (*Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "y", "Y", "enter":
			target := m.pendingDelete
			m.pendingDelete = nil
			return m, m.executeDeleteCmd(target)
		case "n", "N", "esc", "q":
			returnView := viewPostList
			if m.pendingDelete != nil {
				returnView = m.pendingDelete.returnView
			}
			m.pendingDelete = nil
			m.currentView = returnView
			m.flashMsg = "• Deletion cancelled"
			return m, nil
		}
	}
	return m, nil
}

func (m *Model) executeDeleteCmd(target *deleteTarget) tea.Cmd {
	return func() tea.Msg {
		if target == nil || m.user == nil {
			return nil
		}
		ctx, cancel := context.WithTimeout(m.ctx, 3*time.Second)
		defer cancel()

		if target.targetType == deleteTargetPost {
			hasComments, err := m.queries.HasPostComments(ctx, target.id)
			if err != nil {
				return errMsg{err: fmt.Errorf("checking post comments: %w", err)}
			}
			if hasComments {
				if err := m.queries.SoftDeletePost(ctx, db.SoftDeletePostParams{
					ID:       target.id,
					AuthorID: m.user.ID,
				}); err != nil {
					return errMsg{err: fmt.Errorf("soft deleting post: %w", err)}
				}
				_ = m.queries.PruneTombstoneComments(ctx, target.id)
				_ = m.queries.RecalculatePostCommentCount(ctx, target.id)
				_ = m.queries.PruneDeletedPostIfEmpty(ctx, target.id)
				return postDeletedMsg{postID: target.id, isSoft: true}
			}

			if err := m.queries.HardDeletePost(ctx, db.HardDeletePostParams{
				ID:       target.id,
				AuthorID: m.user.ID,
			}); err != nil {
				return errMsg{err: fmt.Errorf("hard deleting post: %w", err)}
			}
			return postDeletedMsg{postID: target.id, isSoft: false}
		}

		// Comment deletion
		hasChildren, err := m.queries.HasCommentChildren(ctx, pgtype.Int8{Int64: target.id, Valid: true})
		if err != nil {
			return errMsg{err: fmt.Errorf("checking comment replies: %w", err)}
		}
		isSoft := false
		if hasChildren {
			if err := m.queries.SoftDeleteComment(ctx, db.SoftDeleteCommentParams{
				ID:       target.id,
				AuthorID: m.user.ID,
			}); err != nil {
				return errMsg{err: fmt.Errorf("soft deleting comment: %w", err)}
			}
			isSoft = true
		} else {
			if err := m.queries.HardDeleteComment(ctx, db.HardDeleteCommentParams{
				ID:       target.id,
				AuthorID: m.user.ID,
			}); err != nil {
				return errMsg{err: fmt.Errorf("hard deleting comment: %w", err)}
			}
		}

		// 1. Prune dead tombstones (any soft-deleted parent comment that now has no active descendants)
		_ = m.queries.PruneTombstoneComments(ctx, target.postID)

		// 2. Ensure post comment count accurately reflects remaining active comments
		_ = m.queries.RecalculatePostCommentCount(ctx, target.postID)

		// 3. Prune the post if it was soft-deleted and now has 0 comments remaining!
		_ = m.queries.PruneDeletedPostIfEmpty(ctx, target.postID)

		return commentDeletedMsg{commentID: target.id, postID: target.postID, isSoft: isSoft}
	}
}

