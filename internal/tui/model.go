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
	"github.com/charmbracelet/lipgloss"
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
	viewInbox
	viewProfile
	viewHelp
	viewThemePicker
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
	currentView     viewState
	helpReturnView  viewState
	errorReturnView viewState
	width           int
	height          int
	err             error

	// Key bindings.
	keys KeyMap

	// Onboarding.
	handleInput textinput.Model

	// Board list.
	boards      []db.Board
	boardCursor int

	// Post list.
	currentBoard   *db.Board
	posts          []PostFeedItem
	postCursor     int
	postSortMode   PostSortMode
	categoryFilter string // "" for all, or category name
	searchQuery    string // Active search query filter
	searchInput    textinput.Model
	searchFocused  bool

	// Keyset feed pagination & triage.
	feedPage      int                // 1-indexed current page
	feedPageStack []PostCursor       // stack of cursors for visited pages
	hasNextPage   bool               // whether a next page exists
	compactMode   bool               // 'z' toggle: false = comfortable (3-line), true = compact (1-line)
	readPosts     map[int64]bool     // session-local read triage
	hideRead      bool               // 'H' toggle to hide read posts
	postsCancel   context.CancelFunc // in-flight post fetch cancel func

	// Post detail & comments.
	currentPost        *db.GetPostByIDRow
	comments           []db.GetCommentThreadByPostRow
	viewport           viewport.Model
	commentCursor      int
	commentLineOffsets []int
	commentSortMode    CommentSortMode

	// Micro-interactions and animations.
	flashMsg string
	animTick int // Animation tick counter for Earth rotation and logo shine.

	// Content creation: New Post.
	titleInput         textinput.Model
	newPostCategoryIdx int // Index into AvailableCategories
	urlInput           textinput.Model
	bodyInput          textarea.Model
	postFormFocus      int // 0: Title, 1: Category, 2: URL, 3: Body

	// Content creation: New Comment.
	commentInput      textarea.Model
	replyParentID     *int64
	replyParentAuthor string

	// Deletion confirmation.
	pendingDelete *deleteTarget

	// Quit confirmation.
	quitConfirm   bool
	quitConfirmID int

	// Notifications.
	unreadNotificationCount int
	notifications           []db.ListNotificationsKeysetRow
	notificationCursor      int
	inboxReturnView         viewState

	// User Profile.
	profileUser       *db.User
	profileTab        int // 0: Submissions, 1: Comments
	profilePosts      []db.ListPostsByAuthorKeysetRow
	profileComments   []db.ListCommentsByAuthorKeysetRow
	profilePostCursor int
	profileCommCursor int
	profileReturnView viewState

	// Themes & Styling.
	themeID         string
	theme           Theme
	styles          Styles
	themeCursor     int
	themeReturnView viewState
}

// --- Messages ----------------------------------------------------------

// userLoadedMsg is sent after the user record is fetched/created.
type userLoadedMsg struct {
	user  *db.User
	isNew bool
}

// unreadNotificationCountMsg carries the count of unread notifications.
type unreadNotificationCountMsg struct {
	count int
}

// notificationsLoadedMsg carries the list of notifications for the inbox.
type notificationsLoadedMsg struct {
	notifications []db.ListNotificationsKeysetRow
}

// notificationMarkedReadMsg signals a single notification was marked read.
type notificationMarkedReadMsg struct {
	notificationID int64
}

// allNotificationsMarkedReadMsg signals all notifications were marked read.
type allNotificationsMarkedReadMsg struct{}

// userProfileLoadedMsg carries profile data and activity lists.
type userProfileLoadedMsg struct {
	user     *db.User
	posts    []db.ListPostsByAuthorKeysetRow
	comments []db.ListCommentsByAuthorKeysetRow
}

// boardsLoadedMsg carries the list of boards from the database.
type boardsLoadedMsg struct {
	boards []db.Board
}

// PostCursor encapsulates keyset position for O(log N) composite index seeking.
type PostCursor struct {
	ID        int64
	Score     int32
	CreatedAt time.Time
	HotScore  float64
}

func postToCursor(p PostFeedItem) PostCursor {
	return PostCursor{
		ID:        p.ID,
		Score:     p.Score,
		CreatedAt: p.CreatedAt.Time,
		HotScore:  p.HotScore,
	}
}

// postsLoadedMsg carries posts for a board along with keyset pagination metadata.
type postsLoadedMsg struct {
	posts   []PostFeedItem
	hasMore bool
	page    int
}

// postDetailLoadedMsg carries the post details and its threaded comments.
type postDetailLoadedMsg struct {
	post            *db.GetPostByIDRow
	comments        []db.GetCommentThreadByPostRow
	targetCommentID *int64
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

// quitConfirmTimeoutMsg signals the expiration of the two-step quit confirmation prompt.
type quitConfirmTimeoutMsg struct {
	id int
}

// errMsg wraps an error for the Update loop.
type errMsg struct{ err error }

func (e errMsg) Error() string { return e.err.Error() }

// voteErrorMsg represents a non-fatal voting rejection (e.g. self-voting or deleted content)
type voteErrorMsg struct{ err error }

func (e voteErrorMsg) Error() string { return e.err.Error() }

// --- Constructor -------------------------------------------------------

// NewModel creates a new root Model for a Bubble Tea session.
func NewModel(ctx context.Context, pool *pgxpool.Pool, fingerprint string, logger *slog.Logger) *Model {
	ti := textinput.New()
	ti.Placeholder = "choose a handle (e.g. ram, shyam, etc)"
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
	bodyA.FocusedStyle.CursorLine = lipgloss.NewStyle()
	bodyA.BlurredStyle.CursorLine = lipgloss.NewStyle()
	bodyA.FocusedStyle.Base = lipgloss.NewStyle()
	bodyA.BlurredStyle.Base = lipgloss.NewStyle()
	bodyA.SetWidth(65)
	bodyA.SetHeight(8)
	bodyA.CharLimit = 10000

	// Comment textarea
	commA := textarea.New()
	commA.Placeholder = "Write your reply here..."
	commA.ShowLineNumbers = false
	commA.Prompt = ""
	commA.FocusedStyle.CursorLine = lipgloss.NewStyle()
	commA.BlurredStyle.CursorLine = lipgloss.NewStyle()
	commA.FocusedStyle.Base = lipgloss.NewStyle()
	commA.BlurredStyle.Base = lipgloss.NewStyle()
	commA.SetWidth(65)
	commA.SetHeight(6)
	commA.CharLimit = 5000

	// Post search input
	searchIn := textinput.New()
	searchIn.Placeholder = "type to filter discussions..."
	searchIn.CharLimit = 64
	searchIn.Width = 36
	searchIn.Prompt = "/ filter: "

	vp := viewport.New(80, 20)

	th := DefaultTheme()
	sty := NewStyles(th)
	searchIn.PromptStyle = sty.FilterPrompt

	m := &Model{
		ctx:           ctx,
		pool:          pool,
		queries:       db.New(pool),
		logger:        logger,
		fingerprint:   fingerprint,
		currentView:   viewLoading,
		keys:          DefaultKeyMap(),
		themeID:       th.ID,
		theme:         th,
		styles:        sty,
		handleInput:   ti,
		titleInput:    titleIn,
		urlInput:      urlIn,
		bodyInput:     bodyA,
		commentInput:  commA,
		searchInput:   searchIn,
		viewport:      vp,
		commentCursor: -1,
		feedPage:      1,
		readPosts:     make(map[int64]bool),
	}
	m.syncCursorStyles()
	return m
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

	case quitConfirmTimeoutMsg:
		if msg.id == m.quitConfirmID && m.quitConfirm {
			m.quitConfirm = false
			if m.flashMsg == "• Press 'q' again to exit" {
				m.flashMsg = ""
			}
		}
		return m, nil

	// Global quit & flash clearing.
	case tea.KeyMsg:
		if msg.String() == "ctrl+c" {
			return m, tea.Quit
		}
		if m.quitConfirm && (msg.String() == "y" || msg.String() == "Y") {
			return m, tea.Quit
		}
		if m.quitConfirm && (msg.String() == "esc" || msg.String() == "n" || msg.String() == "N") {
			m.quitConfirm = false
			if m.flashMsg == "• Press 'q' again to exit" {
				m.flashMsg = ""
			}
			return m, nil
		}
		if msg.String() != "u" && msg.String() != "d" && msg.String() != "q" {
			m.flashMsg = ""
		}
		if msg.String() != "q" {
			m.quitConfirm = false
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
		m.hasNextPage = msg.hasMore
		m.feedPage = msg.page
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

		if msg.targetCommentID != nil {
			foundIdx := -1
			for i, c := range m.comments {
				if c.ID == *msg.targetCommentID {
					foundIdx = i
					break
				}
			}
			if foundIdx != -1 {
				m.commentCursor = foundIdx
			} else {
				m.commentCursor = -1
				m.flashMsg = "• Comment is unavailable or deleted"
			}
		} else if samePost {
			m.commentCursor = prevCursor
			if m.commentCursor >= len(m.comments) {
				m.commentCursor = len(m.comments) - 1
			}
		} else {
			m.commentCursor = -1
		}

		m.updateViewportSize()
		m.viewport.SetContent(m.renderPostDetailContent())

		if samePost && msg.targetCommentID == nil {
			m.viewport.SetYOffset(prevYOffset)
		} else {
			m.viewport.GotoTop()
		}
		return m, nil
	case unreadNotificationCountMsg:
		m.unreadNotificationCount = msg.count
		return m, nil
	case notificationsLoadedMsg:
		m.notifications = msg.notifications
		m.currentView = viewInbox
		if m.notificationCursor >= len(m.notifications) {
			m.notificationCursor = 0
		}
		return m, nil
	case notificationMarkedReadMsg:
		for i, n := range m.notifications {
			if n.ID == msg.notificationID {
				m.notifications[i].IsRead = true
				break
			}
		}
		if m.unreadNotificationCount > 0 {
			m.unreadNotificationCount--
		}
		return m, nil
	case allNotificationsMarkedReadMsg:
		for i := range m.notifications {
			m.notifications[i].IsRead = true
		}
		m.unreadNotificationCount = 0
		return m, nil
	case userProfileLoadedMsg:
		m.profileUser = msg.user
		m.profilePosts = msg.posts
		m.profileComments = msg.comments
		m.currentView = viewProfile
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
		return m, tea.Batch(m.loadPostsCmd(msg.post.BoardID), m.checkUnreadNotificationsCmd())
	case commentCreatedMsg:
		// Return to post detail and reload discussion
		m.flashMsg = "✓ Reply posted!"
		m.currentView = viewPostDetail
		return m, tea.Batch(m.loadPostDetailCmd(msg.comment.PostID), m.checkUnreadNotificationsCmd())
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
	case voteErrorMsg:
		m.flashMsg = "• " + msg.err.Error()
		return m, nil
	case errMsg:
		if m.currentView == viewNewPost || m.currentView == viewNewComment {
			m.err = msg.err
			return m, nil
		}
		if m.currentView != viewError {
			m.errorReturnView = m.currentView
		}
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
	case viewInbox:
		return m.updateInbox(msg)
	case viewProfile:
		return m.updateProfile(msg)
	case viewHelp:
		return m.updateHelp(msg)
	case viewThemePicker:
		return m.updateThemePicker(msg)
	case viewError:
		return m.updateError(msg)
	}

	return m, nil
}

// View renders the current view.
func (m *Model) View() string {
	m.ensureStyles()
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
	case viewInbox:
		return m.viewInbox()
	case viewProfile:
		return m.viewProfile()
	case viewHelp:
		return m.viewHelp()
	case viewThemePicker:
		return m.viewThemePicker()
	case viewError:
		return m.viewError()
	default:
		return "Unknown view state"
	}
}

// execTx executes fn within an explicit database transaction when pool is available.
func (m *Model) execTx(ctx context.Context, fn func(q *db.Queries) error) error {
	if m.pool == nil {
		return fn(m.queries)
	}
	tx, err := m.pool.BeginTx(ctx, pgx.TxOptions{})
	if err != nil {
		return fmt.Errorf("beginning transaction: %w", err)
	}
	defer tx.Rollback(ctx)

	q := m.queries.WithTx(tx)
	if err := fn(q); err != nil {
		return err
	}
	return tx.Commit(ctx)
}

// --- User loading ------------------------------------------------------

func (m *Model) lookupUserCmd() tea.Cmd {
	return func() tea.Msg {
		if strings.TrimSpace(m.fingerprint) == "" {
			return errMsg{err: fmt.Errorf("public key authentication required")}
		}
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
		return m, m.handleInput.Focus()
	}
	m.user = msg.user
	if m.user != nil && m.user.Theme != "" {
		m.setTheme(m.user.Theme)
	}
	m.currentView = viewBoardList
	return m, tea.Batch(m.loadBoardsCmd(), m.checkUnreadNotificationsCmd())
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
		if strings.TrimSpace(m.fingerprint) == "" {
			return errMsg{err: fmt.Errorf("public key authentication required")}
		}
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

// handleQuitKey handles exiting with 'q' using two-step confirmation.
// The first press flashes a confirmation message; a second press confirms exit.
func (m *Model) handleQuitKey() (*Model, tea.Cmd) {
	if m.quitConfirm {
		return m, tea.Quit
	}
	m.quitConfirm = true
	m.quitConfirmID++
	m.flashMsg = "• Press 'q' again to exit"
	return m, m.quitConfirmTimeoutCmd(m.quitConfirmID)
}

// quitConfirmTimeoutCmd returns a command that sends a quitConfirmTimeoutMsg after 3 seconds.
func (m *Model) quitConfirmTimeoutCmd(id int) tea.Cmd {
	return tea.Tick(3*time.Second, func(time.Time) tea.Msg {
		return quitConfirmTimeoutMsg{id: id}
	})
}

func (m *Model) updateBoardList(msg tea.Msg) (*Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch {
		case msg.String() == "?":
			m.helpReturnView = viewBoardList
			m.currentView = viewHelp
			return m, nil
		case msg.String() == "t":
			m.openThemePicker(viewBoardList)
			return m, nil
		case msg.String() == "i":
			return m, m.openInboxCmd(viewBoardList)
		case msg.String() == "p":
			if m.user != nil {
				return m, m.openProfileCmd(m.user.Handle, viewBoardList)
			}
			return m, nil
		case msg.String() == "q":
			return m.handleQuitKey()
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
				m.postCursor = 0
				m.feedPage = 1
				m.feedPageStack = nil
				m.searchQuery = ""
				m.searchInput.SetValue("")
				m.searchFocused = false
				return m, m.loadPostsCmd(board.ID)
			}
		}
	}
	return m, nil
}

// --- Post list ---------------------------------------------------------

func (m *Model) loadPostsCmd(boardID int64) tea.Cmd {
	// Cancel any active in-flight post fetch to immediately free the database connection
	if m.postsCancel != nil {
		m.postsCancel()
		m.postsCancel = nil
	}

	baseCtx := m.ctx
	if baseCtx == nil {
		baseCtx = context.Background()
	}
	fetchCtx, cancel := context.WithTimeout(baseCtx, 3*time.Second)
	m.postsCancel = cancel

	sortMode := m.postSortMode
	catFilter := m.categoryFilter
	searchQuery := m.searchQuery
	page := m.feedPage
	if page < 1 {
		page = 1
	}

	var cursor *PostCursor
	if page > 1 && len(m.feedPageStack) >= page-1 {
		c := m.feedPageStack[page-2]
		cursor = &c
	}

	return func() tea.Msg {
		defer cancel()

		var feedItems []PostFeedItem
		var hasMore bool

		switch sortMode {
		case PostSortHot:
			var curHot pgtype.Float8
			var curID pgtype.Int8
			if cursor != nil && cursor.ID > 0 {
				curHot = pgtype.Float8{Float64: cursor.HotScore, Valid: true}
				curID = pgtype.Int8{Int64: cursor.ID, Valid: true}
			}
			rows, err := m.queries.ListPostsByBoardHot(fetchCtx, db.ListPostsByBoardHotParams{
				BoardID:        boardID,
				Limit:          51,
				Category:       catFilter,
				SearchQuery:    searchQuery,
				CursorHotScore: curHot,
				CursorID:       curID,
			})
			if err != nil {
				if errors.Is(err, context.Canceled) {
					return nil
				}
				return errMsg{err: fmt.Errorf("loading hot posts: %w", err)}
			}
			if len(rows) > 50 {
				hasMore = true
				rows = rows[:50]
			}
			feedItems = make([]PostFeedItem, len(rows))
			for i, r := range rows {
				feedItems[i] = hotRowToPost(r)
			}
		case PostSortTop:
			var curScore pgtype.Int4
			var curCreated pgtype.Timestamptz
			var curID pgtype.Int8
			if cursor != nil && cursor.ID > 0 {
				curScore = pgtype.Int4{Int32: cursor.Score, Valid: true}
				curCreated = pgtype.Timestamptz{Time: cursor.CreatedAt, Valid: true}
				curID = pgtype.Int8{Int64: cursor.ID, Valid: true}
			}
			rows, err := m.queries.ListPostsByBoardTop(fetchCtx, db.ListPostsByBoardTopParams{
				BoardID:         boardID,
				Limit:           51,
				Category:        catFilter,
				SearchQuery:     searchQuery,
				CursorScore:     curScore,
				CursorCreatedAt: curCreated,
				CursorID:        curID,
			})
			if err != nil {
				if errors.Is(err, context.Canceled) {
					return nil
				}
				return errMsg{err: fmt.Errorf("loading top posts: %w", err)}
			}
			if len(rows) > 50 {
				hasMore = true
				rows = rows[:50]
			}
			feedItems = make([]PostFeedItem, len(rows))
			for i, r := range rows {
				feedItems[i] = topRowToPost(r)
			}
		default: // PostSortNew
			var curCreated pgtype.Timestamptz
			var curID pgtype.Int8
			if cursor != nil && cursor.ID > 0 {
				curCreated = pgtype.Timestamptz{Time: cursor.CreatedAt, Valid: true}
				curID = pgtype.Int8{Int64: cursor.ID, Valid: true}
			}
			rows, err := m.queries.ListPostsByBoardNew(fetchCtx, db.ListPostsByBoardNewParams{
				BoardID:         boardID,
				Limit:           51,
				Category:        catFilter,
				SearchQuery:     searchQuery,
				CursorCreatedAt: curCreated,
				CursorID:        curID,
			})
			if err != nil {
				if errors.Is(err, context.Canceled) {
					return nil
				}
				return errMsg{err: fmt.Errorf("loading new posts: %w", err)}
			}
			if len(rows) > 50 {
				hasMore = true
				rows = rows[:50]
			}
			feedItems = make([]PostFeedItem, len(rows))
			for i, r := range rows {
				feedItems[i] = newRowToPost(r)
			}
		}

		return postsLoadedMsg{posts: feedItems, hasMore: hasMore, page: page}
	}
}

func (m *Model) cycleCategoryFilter() {
	if m.categoryFilter == "" {
		m.categoryFilter = AvailableCategories[0]
		return
	}
	for i, cat := range AvailableCategories {
		if m.categoryFilter == cat {
			if i+1 < len(AvailableCategories) {
				m.categoryFilter = AvailableCategories[i+1]
			} else {
				m.categoryFilter = "" // wrap back to all
			}
			return
		}
	}
	m.categoryFilter = ""
}

// visiblePosts returns the active list of posts honoring session read triage.
func (m *Model) visiblePosts() []PostFeedItem {
	if !m.hideRead {
		return m.posts
	}
	filtered := make([]PostFeedItem, 0, len(m.posts))
	for _, p := range m.posts {
		if !m.readPosts[p.ID] {
			filtered = append(filtered, p)
		}
	}
	return filtered
}

func (m *Model) updatePostList(msg tea.Msg) (*Model, tea.Cmd) {
	visible := m.visiblePosts()

	if m.searchFocused {
		switch msg := msg.(type) {
		case tea.KeyMsg:
			switch msg.String() {
			case "enter":
				m.searchQuery = strings.TrimSpace(m.searchInput.Value())
				m.searchFocused = false
				m.searchInput.Blur()
				m.postCursor = 0
				m.feedPage = 1
				m.feedPageStack = nil
				if m.searchQuery != "" {
					m.flashMsg = fmt.Sprintf("• Searching for %q", m.searchQuery)
				} else {
					m.flashMsg = "• Search cleared"
				}
				if m.currentBoard != nil {
					return m, m.loadPostsCmd(m.currentBoard.ID)
				}
				return m, nil
			case "esc":
				m.searchFocused = false
				m.searchInput.Blur()
				if m.searchQuery != "" {
					m.searchQuery = ""
					m.searchInput.SetValue("")
					m.postCursor = 0
					m.feedPage = 1
					m.feedPageStack = nil
					m.flashMsg = "• Search cleared"
					if m.currentBoard != nil {
						return m, m.loadPostsCmd(m.currentBoard.ID)
					}
				}
				return m, nil
			case "down", "tab":
				m.searchFocused = false
				m.searchInput.Blur()
				return m, nil
			default:
				var cmd tea.Cmd
				m.searchInput, cmd = m.searchInput.Update(msg)
				return m, cmd
			}
		default:
			var cmd tea.Cmd
			m.searchInput, cmd = m.searchInput.Update(msg)
			return m, cmd
		}
	}

	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch {
		case msg.String() == "?":
			m.helpReturnView = viewPostList
			m.currentView = viewHelp
			return m, nil
		case msg.String() == "t":
			m.openThemePicker(viewPostList)
			return m, nil
		case msg.String() == "i":
			return m, m.openInboxCmd(viewPostList)
		case msg.String() == "p":
			if m.user != nil {
				return m, m.openProfileCmd(m.user.Handle, viewPostList)
			}
			return m, nil
		case msg.String() == "/":
			m.searchFocused = true
			return m, m.searchInput.Focus()
		case msg.String() == "q":
			return m.handleQuitKey()
		case msg.String() == "esc":
			if m.searchQuery != "" {
				m.searchQuery = ""
				m.searchInput.SetValue("")
				m.postCursor = 0
				m.feedPage = 1
				m.feedPageStack = nil
				m.flashMsg = "• Search cleared"
				if m.currentBoard != nil {
					return m, m.loadPostsCmd(m.currentBoard.ID)
				}
				return m, nil
			}
			m.currentView = viewBoardList
			return m, m.animTickCmd()
		case msg.String() == "k" || msg.String() == "up":
			if m.postCursor > 0 {
				m.postCursor--
			}
		case msg.String() == "j" || msg.String() == "down":
			if m.postCursor < len(visible)-1 {
				m.postCursor++
			}
		case msg.String() == "g" || msg.String() == "home":
			m.postCursor = 0
		case msg.String() == "G" || msg.String() == "end":
			if len(visible) > 0 {
				m.postCursor = len(visible) - 1
			}
		case msg.String() == "s":
			m.postSortMode = (m.postSortMode + 1) % 3
			m.flashMsg = "• Posts sorted by " + m.postSortMode.String()
			m.postCursor = 0
			m.feedPage = 1
			m.feedPageStack = nil
			if m.currentBoard != nil {
				return m, m.loadPostsCmd(m.currentBoard.ID)
			}
			return m, nil
		case msg.String() == "c":
			m.cycleCategoryFilter()
			if m.categoryFilter == "" {
				m.flashMsg = "• Flair: all categories"
			} else {
				m.flashMsg = "• Flair: " + m.categoryFilter
			}
			m.postCursor = 0
			m.feedPage = 1
			m.feedPageStack = nil
			if m.currentBoard != nil {
				return m, m.loadPostsCmd(m.currentBoard.ID)
			}
			return m, nil
		case msg.String() == "z":
			m.compactMode = !m.compactMode
			if m.compactMode {
				m.flashMsg = "• Compact view enabled"
			} else {
				m.flashMsg = "• Comfortable view enabled"
			}
			return m, nil
		case msg.String() == "]" || msg.String() == "pgdown" || msg.String() == "ctrl+f":
			if !m.hasNextPage || len(m.posts) == 0 {
				m.flashMsg = "• At last page"
				return m, nil
			}
			lastPost := m.posts[len(m.posts)-1]
			nextCursor := postToCursor(lastPost)
			if len(m.feedPageStack) >= m.feedPage {
				m.feedPageStack[m.feedPage-1] = nextCursor
			} else {
				m.feedPageStack = append(m.feedPageStack, nextCursor)
			}
			m.feedPage++
			m.postCursor = 0
			m.flashMsg = fmt.Sprintf("• Page %d", m.feedPage)
			if m.currentBoard != nil {
				return m, m.loadPostsCmd(m.currentBoard.ID)
			}
			return m, nil
		case msg.String() == "[" || msg.String() == "pgup" || msg.String() == "ctrl+b":
			if m.feedPage <= 1 {
				m.flashMsg = "• Already on first page"
				return m, nil
			}
			m.feedPage--
			m.postCursor = 0
			m.flashMsg = fmt.Sprintf("• Page %d", m.feedPage)
			if m.currentBoard != nil {
				return m, m.loadPostsCmd(m.currentBoard.ID)
			}
			return m, nil
		case msg.String() == "ctrl+d":
			jump := 5
			if m.compactMode {
				jump = 12
			}
			if len(visible) > 0 {
				m.postCursor = min(len(visible)-1, m.postCursor+jump)
			}
			return m, nil
		case msg.String() == "ctrl+u":
			jump := 5
			if m.compactMode {
				jump = 12
			}
			m.postCursor = max(0, m.postCursor-jump)
			return m, nil
		case msg.String() == "m":
			if len(visible) > 0 && m.postCursor < len(visible) {
				post := visible[m.postCursor]
				m.readPosts[post.ID] = !m.readPosts[post.ID]
				if m.readPosts[post.ID] {
					m.flashMsg = "• Marked discussion as read"
				} else {
					m.flashMsg = "• Marked discussion as unread"
				}
			}
			return m, nil
		case msg.String() == "H":
			m.hideRead = !m.hideRead
			if m.hideRead {
				m.flashMsg = "• Hiding read discussions"
			} else {
				m.flashMsg = "• Showing all discussions"
			}
			m.postCursor = 0
			return m, nil
		case msg.String() == "enter":
			if len(visible) > 0 && m.postCursor < len(visible) {
				post := visible[m.postCursor]
				m.readPosts[post.ID] = true
				return m, m.loadPostDetailCmd(post.ID)
			}
		case msg.String() == "n":
			return m, m.openNewPost()
		case msg.String() == "x":
			if len(visible) == 0 || m.postCursor >= len(visible) {
				return m, nil
			}
			post := visible[m.postCursor]
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
			if len(visible) > 0 && m.postCursor < len(visible) {
				post := visible[m.postCursor]
				if post.IsDeleted {
					m.flashMsg = "• Voting is disabled on deleted content"
					return m, nil
				}
				if m.user != nil && post.AuthorID == m.user.ID {
					m.flashMsg = "• You cannot vote on your own post"
					return m, nil
				}
				return m, m.castPostVoteCmd(post.ID, 1)
			}
		case msg.String() == "d":
			if len(visible) > 0 && m.postCursor < len(visible) {
				post := visible[m.postCursor]
				if post.IsDeleted {
					m.flashMsg = "• Voting is disabled on deleted content"
					return m, nil
				}
				if m.user != nil && post.AuthorID == m.user.ID {
					m.flashMsg = "• You cannot vote on your own post"
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
	return m.loadPostDetailAndFocusCommentCmd(postID, nil)
}

func (m *Model) loadPostDetailAndFocusCommentCmd(postID int64, targetCommentID *int64) tea.Cmd {
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
				_ = m.execTx(ctx, func(q *db.Queries) error {
					if err := q.PruneTombstoneComments(ctx, postID); err != nil {
						return err
					}
					if err := q.RecalculatePostCommentCount(ctx, postID); err != nil {
						return err
					}
					return q.PruneDeletedPostIfEmpty(ctx, postID)
				})
				return postPrunedMsg{postID: postID}
			}
		}

		commSort := m.commentSortMode
		sortedComments := sortCommentTree(comments, commSort)

		return postDetailLoadedMsg{post: &post, comments: sortedComments, targetCommentID: targetCommentID}
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

	searchW := 36
	if m.width > 0 {
		searchW = min(50, max(24, m.width/2))
	}
	m.searchInput.Width = searchW

	_, _, _, contentHeight := m.shellDimensions()

	bodyHeight := 3
	if contentHeight > 16 {
		bodyHeight = min(10, max(3, contentHeight-16))
	}
	m.bodyInput.SetHeight(bodyHeight)

	commHeight := 6
	if contentHeight > 8 {
		commHeight = min(8, max(3, contentHeight-8))
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
		case "?":
			m.helpReturnView = viewPostDetail
			m.currentView = viewHelp
			return m, nil
		case "t":
			m.openThemePicker(viewPostDetail)
			return m, nil
		case "i":
			return m, m.openInboxCmd(viewPostDetail)
		case "p":
			if m.user != nil {
				return m, m.openProfileCmd(m.user.Handle, viewPostDetail)
			}
			return m, nil
		case "P":
			var authorHandle string
			if m.commentCursor >= 0 && m.commentCursor < len(m.comments) {
				authorHandle = m.comments[m.commentCursor].AuthorHandle
			} else if m.currentPost != nil {
				authorHandle = m.currentPost.AuthorHandle
			}
			if authorHandle != "" && authorHandle != "[deleted]" {
				return m, m.openProfileCmd(authorHandle, viewPostDetail)
			}
			m.flashMsg = "• Author profile is unavailable"
			return m, nil
		case "esc":
			m.currentView = viewPostList
			if m.currentBoard != nil {
				return m, m.loadPostsCmd(m.currentBoard.ID)
			}
			return m, nil
		case "q":
			return m.handleQuitKey()
		case "s":
			m.commentSortMode = (m.commentSortMode + 1) % 3
			m.comments = sortCommentTree(m.comments, m.commentSortMode)
			m.commentCursor = -1
			m.flashMsg = "• Comments sorted by " + m.commentSortMode.String()
			m.viewport.SetContent(m.renderPostDetailContent())
			m.viewport.GotoTop()
			return m, nil
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
				comm := m.comments[m.commentCursor]
				if comm.IsDeleted {
					m.flashMsg = "• Voting is disabled on deleted content"
					return m, nil
				}
				if m.user != nil && comm.AuthorID == m.user.ID {
					m.flashMsg = "• You cannot vote on your own comment"
					return m, nil
				}
				return m, m.castCommentVoteCmd(comm.ID, 1)
			}
			if m.currentPost != nil {
				if m.currentPost.IsDeleted {
					m.flashMsg = "• Voting is disabled on deleted content"
					return m, nil
				}
				if m.user != nil && m.currentPost.AuthorID == m.user.ID {
					m.flashMsg = "• You cannot vote on your own post"
					return m, nil
				}
				return m, m.castPostVoteCmd(m.currentPost.ID, 1)
			}
		case "d":
			if m.commentCursor >= 0 && m.commentCursor < len(m.comments) {
				comm := m.comments[m.commentCursor]
				if comm.IsDeleted {
					m.flashMsg = "• Voting is disabled on deleted content"
					return m, nil
				}
				if m.user != nil && comm.AuthorID == m.user.ID {
					m.flashMsg = "• You cannot vote on your own comment"
					return m, nil
				}
				return m, m.castCommentVoteCmd(comm.ID, -1)
			}
			if m.currentPost != nil {
				if m.currentPost.IsDeleted {
					m.flashMsg = "• Voting is disabled on deleted content"
					return m, nil
				}
				if m.user != nil && m.currentPost.AuthorID == m.user.ID {
					m.flashMsg = "• You cannot vote on your own post"
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

		var newDirection int16
		err := m.execTx(ctx, func(q *db.Queries) error {
			post, err := q.GetPostByID(ctx, postID)
			if err != nil {
				return fmt.Errorf("fetching post for vote: %w", err)
			}
			if post.IsDeleted {
				return fmt.Errorf("voting is disabled on deleted content")
			}
			if post.AuthorID == m.user.ID {
				return fmt.Errorf("you cannot vote on your own post")
			}

			existing, err := q.GetPostVoteByUser(ctx, db.GetPostVoteByUserParams{
				UserID: m.user.ID,
				PostID: postID,
			})
			var oldDir int16
			if err == nil {
				oldDir = existing.Direction
			}

			if err == nil && existing.Direction == direction {
				// User pressed the same direction again -> toggle off (delete vote)
				if err := q.DeletePostVote(ctx, db.DeletePostVoteParams{
					UserID: m.user.ID,
					PostID: postID,
				}); err != nil {
					return fmt.Errorf("deleting post vote: %w", err)
				}
				newDirection = 0
			} else {
				// New vote or flipping from upvote to downvote (or vice versa)
				if err := q.UpsertPostVote(ctx, db.UpsertPostVoteParams{
					UserID:    m.user.ID,
					PostID:    postID,
					Direction: direction,
				}); err != nil {
					return fmt.Errorf("voting on post: %w", err)
				}
				newDirection = direction
			}

			// Recalculate denormalized score atomically in same transaction
			if err := q.RecalculatePostScore(ctx, postID); err != nil {
				return fmt.Errorf("recalculating score: %w", err)
			}

			// Adjust post author's karma atomically
			delta := int32(newDirection - oldDir)
			if delta != 0 {
				if err := q.AdjustUserPostKarma(ctx, db.AdjustUserPostKarmaParams{
					ID:        post.AuthorID,
					PostKarma: delta,
				}); err != nil {
					return fmt.Errorf("adjusting post karma: %w", err)
				}
			}
			return nil
		})
		if err != nil {
			return voteErrorMsg{err: err}
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

		var newDirection int16
		err := m.execTx(ctx, func(q *db.Queries) error {
			comment, err := q.GetCommentByID(ctx, commentID)
			if err != nil {
				return fmt.Errorf("fetching comment for vote: %w", err)
			}
			if comment.IsDeleted {
				return fmt.Errorf("voting is disabled on deleted content")
			}
			if comment.AuthorID == m.user.ID {
				return fmt.Errorf("you cannot vote on your own comment")
			}

			existing, err := q.GetCommentVoteByUser(ctx, db.GetCommentVoteByUserParams{
				UserID:    m.user.ID,
				CommentID: commentID,
			})
			var oldDir int16
			if err == nil {
				oldDir = existing.Direction
			}

			if err == nil && existing.Direction == direction {
				// User pressed the same direction again -> toggle off (delete vote)
				if err := q.DeleteCommentVote(ctx, db.DeleteCommentVoteParams{
					UserID:    m.user.ID,
					CommentID: commentID,
				}); err != nil {
					return fmt.Errorf("deleting comment vote: %w", err)
				}
				newDirection = 0
			} else {
				// New vote or flipping from upvote to downvote (or vice versa)
				if err := q.UpsertCommentVote(ctx, db.UpsertCommentVoteParams{
					UserID:    m.user.ID,
					CommentID: commentID,
					Direction: direction,
				}); err != nil {
					return fmt.Errorf("voting on comment: %w", err)
				}
				newDirection = direction
			}

			if err := q.RecalculateCommentScore(ctx, commentID); err != nil {
				return fmt.Errorf("recalculating comment score: %w", err)
			}

			delta := int32(newDirection - oldDir)
			if delta != 0 {
				if err := q.AdjustUserCommentKarma(ctx, db.AdjustUserCommentKarmaParams{
					ID:           comment.AuthorID,
					CommentKarma: delta,
				}); err != nil {
					return fmt.Errorf("adjusting comment karma: %w", err)
				}
			}
			return nil
		})
		if err != nil {
			return voteErrorMsg{err: err}
		}

		return commentVotedMsg{commentID: commentID, direction: newDirection}
	}
}

// --- Content Creation: Post --------------------------------------------

func (m *Model) openNewPost() tea.Cmd {
	m.currentView = viewNewPost
	m.postFormFocus = 0
	m.newPostCategoryIdx = 0
	m.titleInput.Reset()
	m.urlInput.Reset()
	m.bodyInput.Reset()
	m.resizeInputs()
	cmd := m.syncPostFormFocus()
	m.err = nil
	return cmd
}

func (m *Model) syncPostFormFocus() tea.Cmd {
	switch m.postFormFocus {
	case 0:
		m.urlInput.Blur()
		m.bodyInput.Blur()
		return m.titleInput.Focus()
	case 1:
		m.titleInput.Blur()
		m.urlInput.Blur()
		m.bodyInput.Blur()
		return nil
	case 2:
		m.titleInput.Blur()
		m.bodyInput.Blur()
		return m.urlInput.Focus()
	case 3:
		m.titleInput.Blur()
		m.urlInput.Blur()
		return m.bodyInput.Focus()
	}
	return nil
}

func (m *Model) updateNewPost(msg tea.Msg) (*Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch {
		case msg.String() == "esc":
			m.currentView = viewPostList
			return m, nil
		case msg.String() == "tab":
			m.postFormFocus = (m.postFormFocus + 1) % 4
			return m, m.syncPostFormFocus()
		case msg.String() == "shift+tab" || msg.Type == tea.KeyShiftTab:
			m.postFormFocus = (m.postFormFocus - 1 + 4) % 4
			return m, m.syncPostFormFocus()
		case msg.String() == "enter":
			if m.postFormFocus == 0 {
				m.postFormFocus = 1
				return m, m.syncPostFormFocus()
			} else if m.postFormFocus == 1 {
				m.postFormFocus = 2
				return m, m.syncPostFormFocus()
			} else if m.postFormFocus == 2 {
				m.postFormFocus = 3
				return m, m.syncPostFormFocus()
			}
			// When postFormFocus == 3 (body textarea), enter inserts a newline
		case msg.String() == "ctrl+s":
			title := strings.TrimSpace(m.titleInput.Value())
			if title == "" {
				m.err = fmt.Errorf("title cannot be empty")
				return m, nil
			}
			url := strings.TrimSpace(m.urlInput.Value())
			body := strings.TrimSpace(m.bodyInput.Value())

			if err := sanitize.ValidateCleanContent("post title", title); err != nil {
				m.err = err
				return m, nil
			}
			if err := sanitize.ValidateCleanContent("post body", body); err != nil {
				m.err = err
				return m, nil
			}
			if url != "" {
				if err := sanitize.ValidateCleanContent("post URL", url); err != nil {
					m.err = err
					return m, nil
				}
			}

			return m, m.submitPostCmd(title, body, url)
		default:
			m.err = nil
		}

		if m.postFormFocus == 1 {
			switch msg.String() {
			case "h", "left":
				m.newPostCategoryIdx = (m.newPostCategoryIdx - 1 + len(AvailableCategories)) % len(AvailableCategories)
				return m, nil
			case "l", "right", " ":
				m.newPostCategoryIdx = (m.newPostCategoryIdx + 1) % len(AvailableCategories)
				return m, nil
			}
		}
	}

	var cmd tea.Cmd
	switch m.postFormFocus {
	case 0:
		m.titleInput, cmd = m.titleInput.Update(msg)
	case 2:
		m.urlInput, cmd = m.urlInput.Update(msg)
	case 3:
		m.bodyInput, cmd = m.bodyInput.Update(msg)
	}
	return m, cmd
}

func (m *Model) submitPostCmd(title, body, url string) tea.Cmd {
	category := "general"
	if m.newPostCategoryIdx >= 0 && m.newPostCategoryIdx < len(AvailableCategories) {
		category = AvailableCategories[m.newPostCategoryIdx]
	}

	return func() tea.Msg {
		if m.user == nil || m.currentBoard == nil {
			return errMsg{err: fmt.Errorf("missing user or board context")}
		}
		if err := sanitize.ValidateCleanContent("post title", title); err != nil {
			return errMsg{err: err}
		}
		if err := sanitize.ValidateCleanContent("post body", body); err != nil {
			return errMsg{err: err}
		}
		if url != "" {
			if err := sanitize.ValidateCleanContent("post URL", url); err != nil {
				return errMsg{err: err}
			}
		}

		// Cooldown check (prevent board flooding / post spam)
		if m.user.LastPostAt.Valid && time.Since(m.user.LastPostAt.Time) < 30*time.Second {
			remaining := 30*time.Second - time.Since(m.user.LastPostAt.Time)
			return errMsg{err: fmt.Errorf("please wait %s before posting again", remaining.Round(time.Second))}
		}

		ctx, cancel := context.WithTimeout(m.ctx, 3*time.Second)
		defer cancel()

		var post db.Post
		err := m.execTx(ctx, func(q *db.Queries) error {
			rows, err := q.TryUpdateUserLastPostAt(ctx, m.user.ID)
			if err != nil {
				return fmt.Errorf("updating post rate limit: %w", err)
			}
			if rows == 0 {
				return fmt.Errorf("please wait before posting again (cooldown: 30s)")
			}

			var errCreate error
			post, errCreate = q.CreatePost(ctx, db.CreatePostParams{
				BoardID:  m.currentBoard.ID,
				AuthorID: m.user.ID,
				Title:    sanitize.SingleLine(title),
				Body:     sanitize.Text(body),
				Url:      sanitize.SingleLine(url),
				Category: category,
			})
			if errCreate != nil {
				return fmt.Errorf("creating post: %w", errCreate)
			}
			return nil
		})
		if err != nil {
			return errMsg{err: err}
		}

		m.user.LastPostAt = pgtype.Timestamptz{Time: time.Now(), Valid: true}
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
	cmd := m.commentInput.Focus()
	m.err = nil
	return cmd
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
			if err := sanitize.ValidateCleanContent("comment", body); err != nil {
				m.err = err
				return m, nil
			}
			return m, m.submitCommentCmd(body)
		default:
			m.err = nil
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
		if err := sanitize.ValidateCleanContent("comment", body); err != nil {
			return errMsg{err: err}
		}

		// Cooldown check (prevent reply flooding / notification spam)
		if m.user.LastCommentAt.Valid && time.Since(m.user.LastCommentAt.Time) < 3*time.Second {
			remaining := 3*time.Second - time.Since(m.user.LastCommentAt.Time)
			return errMsg{err: fmt.Errorf("please wait %s before commenting again", remaining.Round(time.Second))}
		}

		ctx, cancel := context.WithTimeout(m.ctx, 3*time.Second)
		defer cancel()

		var parentID pgtype.Int8
		if m.replyParentID != nil {
			parentID = pgtype.Int8{Int64: *m.replyParentID, Valid: true}
		}

		var comment db.Comment
		err := m.execTx(ctx, func(q *db.Queries) error {
			var err error
			comment, err = q.CreateComment(ctx, db.CreateCommentParams{
				PostID:   m.currentPost.ID,
				ParentID: parentID,
				AuthorID: m.user.ID,
				Body:     sanitize.Text(body),
			})
			if err != nil {
				return fmt.Errorf("creating comment: %w", err)
			}

			if err := q.IncrementPostCommentCount(ctx, m.currentPost.ID); err != nil {
				return fmt.Errorf("incrementing comment count: %w", err)
			}

			rows, err := q.TryUpdateUserLastCommentAt(ctx, m.user.ID)
			if err != nil {
				return fmt.Errorf("updating user last comment time: %w", err)
			}
			if rows == 0 {
				return fmt.Errorf("please wait before commenting again (cooldown: 3s)")
			}

			// Determine recipient for notification
			var recipientID int64
			var notifType string
			if m.replyParentID != nil {
				parentComm, err := q.GetCommentByID(ctx, *m.replyParentID)
				if err != nil {
					return fmt.Errorf("fetching parent comment for notification: %w", err)
				}
				recipientID = parentComm.AuthorID
				notifType = "reply_comment"
			} else {
				recipientID = m.currentPost.AuthorID
				notifType = "reply_post"
			}

			// Do not notify self
			if recipientID != m.user.ID {
				if err := q.CreateNotification(ctx, db.CreateNotificationParams{
					UserID:    recipientID,
					ActorID:   m.user.ID,
					PostID:    m.currentPost.ID,
					CommentID: pgtype.Int8{Int64: comment.ID, Valid: true},
					Type:      notifType,
				}); err != nil {
					return fmt.Errorf("creating reply notification: %w", err)
				}
			}

			return nil
		})
		if err != nil {
			return errMsg{err: err}
		}

		m.user.LastCommentAt = pgtype.Timestamptz{Time: time.Now(), Valid: true}
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

		var isSoft bool
		if target.targetType == deleteTargetPost {
			err := m.execTx(ctx, func(q *db.Queries) error {
				rows, err := q.HardDeletePostIfEmpty(ctx, db.HardDeletePostIfEmptyParams{
					ID:       target.id,
					AuthorID: m.user.ID,
				})
				if err != nil {
					return fmt.Errorf("hard deleting post: %w", err)
				}

				if rows > 0 {
					isSoft = false
					return nil
				}

				isSoft = true
				if err := q.SoftDeletePost(ctx, db.SoftDeletePostParams{
					ID:       target.id,
					AuthorID: m.user.ID,
				}); err != nil {
					return fmt.Errorf("soft deleting post: %w", err)
				}
				_ = q.PruneTombstoneComments(ctx, target.id)
				_ = q.RecalculatePostCommentCount(ctx, target.id)
				_ = q.PruneDeletedPostIfEmpty(ctx, target.id)
				return nil
			})
			if err != nil {
				return errMsg{err: err}
			}
			return postDeletedMsg{postID: target.id, isSoft: isSoft}
		}

		// Comment deletion
		err := m.execTx(ctx, func(q *db.Queries) error {
			rows, err := q.HardDeleteCommentIfNoChildren(ctx, db.HardDeleteCommentIfNoChildrenParams{
				ID:       target.id,
				AuthorID: m.user.ID,
			})
			if err != nil {
				return fmt.Errorf("hard deleting comment: %w", err)
			}

			if rows > 0 {
				isSoft = false
			} else {
				isSoft = true
				if err := q.SoftDeleteComment(ctx, db.SoftDeleteCommentParams{
					ID:       target.id,
					AuthorID: m.user.ID,
				}); err != nil {
					return fmt.Errorf("soft deleting comment: %w", err)
				}
			}

			_ = q.PruneTombstoneComments(ctx, target.postID)
			_ = q.RecalculatePostCommentCount(ctx, target.postID)
			_ = q.PruneDeletedPostIfEmpty(ctx, target.postID)
			return nil
		})
		if err != nil {
			return errMsg{err: err}
		}

		return commentDeletedMsg{commentID: target.id, postID: target.postID, isSoft: isSoft}
	}
}

// --- Keyboard Help Modal -----------------------------------------------

func (m *Model) updateHelp(msg tea.Msg) (*Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "?", "esc", "enter", "q":
			targetView := m.helpReturnView
			if targetView == viewLoading || targetView == viewHelp {
				targetView = viewBoardList
			}
			m.currentView = targetView
			if targetView == viewBoardList {
				return m, m.animTickCmd()
			}
			return m, nil
		}
	}
	return m, nil
}

// --- Error View --------------------------------------------------------

func (m *Model) updateError(msg tea.Msg) (*Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "esc", "enter", "space":
			targetView := m.errorReturnView
			if targetView == viewLoading || targetView == viewError || targetView == 0 {
				if m.currentBoard != nil {
					targetView = viewPostList
				} else {
					targetView = viewBoardList
				}
			}
			m.currentView = targetView
			m.err = nil
			if targetView == viewBoardList {
				return m, m.animTickCmd()
			}
			return m, nil
		case "q":
			return m, tea.Quit
		}
	}
	return m, nil
}

// --- Theme Picker Modal ------------------------------------------------

func (m *Model) ensureStyles() {
	if m.theme.ID == "" {
		m.theme = DefaultTheme()
		m.themeID = m.theme.ID
		m.styles = NewStyles(m.theme)
		m.searchInput.PromptStyle = m.styles.FilterPrompt
		m.syncCursorStyles()
	}
}

func (m *Model) syncCursorStyles() {
	curStyle := lipgloss.NewStyle().Foreground(m.theme.Primary)
	m.handleInput.Cursor.Style = curStyle
	m.titleInput.Cursor.Style = curStyle
	m.urlInput.Cursor.Style = curStyle
	m.bodyInput.Cursor.Style = curStyle
	m.commentInput.Cursor.Style = curStyle
	m.searchInput.Cursor.Style = curStyle
}

func (m *Model) setTheme(id string) {
	m.theme = GetTheme(id)
	m.themeID = m.theme.ID
	m.styles = NewStyles(m.theme)
	m.searchInput.PromptStyle = m.styles.FilterPrompt
	m.syncCursorStyles()
	if m.currentPost != nil {
		m.viewport.SetContent(m.renderPostDetailContent())
	}
}

func (m *Model) openThemePicker(returnView viewState) {
	m.themeReturnView = returnView
	m.currentView = viewThemePicker
	m.initThemePicker()
}

func (m *Model) initThemePicker() {
	m.ensureStyles()
	themes := Themes()
	m.themeCursor = 0
	for i, t := range themes {
		if t.ID == m.themeID {
			m.themeCursor = i
			break
		}
	}
}

func (m *Model) saveThemeCmd(themeID string) tea.Cmd {
	return func() tea.Msg {
		if m.user == nil || m.queries == nil {
			return nil
		}
		baseCtx := m.ctx
		if baseCtx == nil {
			baseCtx = context.Background()
		}
		ctx, cancel := context.WithTimeout(baseCtx, 3*time.Second)
		defer cancel()

		_, err := m.queries.UpdateUserTheme(ctx, db.UpdateUserThemeParams{
			ID:    m.user.ID,
			Theme: themeID,
		})
		if err != nil {
			return errMsg{err: fmt.Errorf("saving theme: %w", err)}
		}
		return nil
	}
}

func (m *Model) updateThemePicker(msg tea.Msg) (*Model, tea.Cmd) {
	themes := Themes()
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "esc", "q":
			targetView := m.themeReturnView
			if targetView == viewLoading || targetView == viewThemePicker || targetView == viewHelp {
				targetView = viewBoardList
			}
			m.currentView = targetView
			if targetView == viewBoardList {
				return m, m.animTickCmd()
			}
			return m, nil
		case "j", "down":
			if len(themes) > 0 {
				m.themeCursor = (m.themeCursor + 1) % len(themes)
			}
			return m, nil
		case "k", "up":
			if len(themes) > 0 {
				m.themeCursor = (m.themeCursor - 1 + len(themes)) % len(themes)
			}
			return m, nil
		case "g", "home":
			m.themeCursor = 0
			return m, nil
		case "G", "end":
			if len(themes) > 0 {
				m.themeCursor = len(themes) - 1
			}
			return m, nil
		case "enter":
			if len(themes) > 0 && m.themeCursor >= 0 && m.themeCursor < len(themes) {
				selected := themes[m.themeCursor]
				m.setTheme(selected.ID)
				targetView := m.themeReturnView
				if targetView == viewLoading || targetView == viewThemePicker || targetView == viewHelp {
					targetView = viewBoardList
				}
				m.currentView = targetView
				m.flashMsg = fmt.Sprintf("✓ Theme set to %s", selected.Name)
				var cmds []tea.Cmd
				if targetView == viewBoardList {
					cmds = append(cmds, m.animTickCmd())
				}
				if m.user != nil {
					m.user.Theme = selected.ID
					cmds = append(cmds, m.saveThemeCmd(selected.ID))
				}
				if len(cmds) > 0 {
					return m, tea.Batch(cmds...)
				}
				return m, nil
			}
			return m, nil
		}
	}
	return m, nil
}

func (m *Model) formatKeyPills(pairs [][2]string) string {
	m.ensureStyles()
	var parts []string
	for _, p := range pairs {
		k := m.styles.StatusKey.Render("[" + p[0] + "]")
		d := m.styles.StatusDesc.Render(p[1])
		parts = append(parts, k+" "+d)
	}
	return strings.Join(parts, "  ")
}

func (m *Model) formatAdaptiveKeyPills(shortcuts [][2]string, maxAllowedWidth int) string {
	if len(shortcuts) == 0 || maxAllowedWidth <= 0 {
		return ""
	}

	full := m.formatKeyPills(shortcuts)
	if lipgloss.Width(full) <= maxAllowedWidth {
		return full
	}

	filteredA := make([][2]string, 0, len(shortcuts))
	for _, s := range shortcuts {
		if s[0] != "g/G" {
			filteredA = append(filteredA, s)
		}
	}
	fStrA := m.formatKeyPills(filteredA)
	if lipgloss.Width(fStrA) <= maxAllowedWidth {
		return fStrA
	}

	filteredB := make([][2]string, 0, len(filteredA))
	for _, s := range filteredA {
		if s[0] != "u/d" {
			filteredB = append(filteredB, s)
		}
	}
	fStrB := m.formatKeyPills(filteredB)
	if lipgloss.Width(fStrB) <= maxAllowedWidth {
		return fStrB
	}

	curr := filteredB
	for len(curr) > 2 {
		dropIdx := len(curr) - 2
		curr = append(curr[:dropIdx], curr[dropIdx+1:]...)
		cStr := m.formatKeyPills(curr)
		if lipgloss.Width(cStr) <= maxAllowedWidth {
			return cStr
		}
	}

	if len(shortcuts) > 0 {
		last := m.formatKeyPills([][2]string{shortcuts[len(shortcuts)-1]})
		if lipgloss.Width(last) <= maxAllowedWidth {
			return last
		}
	}

	return ""
}

func (m *Model) formatHelpItem(key, desc string, maxW int) string {
	m.ensureStyles()
	k := m.styles.StatusKey.Render(fmt.Sprintf("%-10s", key))
	d := m.styles.StatusDesc.Render(desc)
	line := k + " " + d
	if maxW > 0 && lipgloss.Width(line) > maxW {
		avail := max(5, maxW-lipgloss.Width(k)-1)
		line = k + " " + lipgloss.NewStyle().MaxWidth(avail).Render(d)
	}
	return line
}

// --- Inbox -------------------------------------------------------------

func (m *Model) updateInbox(msg tea.Msg) (*Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "?":
			m.helpReturnView = viewInbox
			m.currentView = viewHelp
			return m, nil
		case "t":
			m.openThemePicker(viewInbox)
			return m, nil
		case "esc", "q":
			targetView := m.inboxReturnView
			if targetView == viewLoading || targetView == viewInbox {
				targetView = viewBoardList
			}
			m.currentView = targetView
			if targetView == viewBoardList {
				return m, tea.Batch(m.animTickCmd(), m.checkUnreadNotificationsCmd())
			}
			return m, m.checkUnreadNotificationsCmd()
		case "j", "down":
			if len(m.notifications) > 0 && m.notificationCursor < len(m.notifications)-1 {
				m.notificationCursor++
			}
			return m, nil
		case "k", "up":
			if m.notificationCursor > 0 {
				m.notificationCursor--
			}
			return m, nil
		case "g", "home":
			m.notificationCursor = 0
			return m, nil
		case "G", "end":
			if len(m.notifications) > 0 {
				m.notificationCursor = len(m.notifications) - 1
			}
			return m, nil
		case "a":
			if len(m.notifications) > 0 {
				m.flashMsg = "✓ All notifications marked as read"
				return m, m.markAllNotificationsReadCmd()
			}
			return m, nil
		case "enter":
			if len(m.notifications) > 0 && m.notificationCursor < len(m.notifications) {
				n := m.notifications[m.notificationCursor]
				var targetCommID *int64
				if n.CommentID.Valid {
					id := n.CommentID.Int64
					targetCommID = &id
				}
				cmds := []tea.Cmd{
					m.loadPostDetailAndFocusCommentCmd(n.PostID, targetCommID),
				}
				if !n.IsRead {
					cmds = append(cmds, m.markNotificationReadCmd(n.ID))
				}
				return m, tea.Batch(cmds...)
			}
			return m, nil
		}
	}
	return m, nil
}

// --- Profile -----------------------------------------------------------

func (m *Model) updateProfile(msg tea.Msg) (*Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "?":
			m.helpReturnView = viewProfile
			m.currentView = viewHelp
			return m, nil
		case "t":
			m.openThemePicker(viewProfile)
			return m, nil
		case "esc", "q":
			targetView := m.profileReturnView
			if targetView == viewLoading || targetView == viewProfile {
				targetView = viewBoardList
			}
			m.currentView = targetView
			if targetView == viewBoardList {
				return m, m.animTickCmd()
			}
			return m, nil
		case "tab", "l", "right":
			m.profileTab = (m.profileTab + 1) % 2
			return m, nil
		case "shift+tab", "h", "left":
			m.profileTab = (m.profileTab - 1 + 2) % 2
			return m, nil
		case "j", "down":
			if m.profileTab == 0 {
				if len(m.profilePosts) > 0 && m.profilePostCursor < len(m.profilePosts)-1 {
					m.profilePostCursor++
				}
			} else {
				if len(m.profileComments) > 0 && m.profileCommCursor < len(m.profileComments)-1 {
					m.profileCommCursor++
				}
			}
			return m, nil
		case "k", "up":
			if m.profileTab == 0 {
				if m.profilePostCursor > 0 {
					m.profilePostCursor--
				}
			} else {
				if m.profileCommCursor > 0 {
					m.profileCommCursor--
				}
			}
			return m, nil
		case "g", "home":
			if m.profileTab == 0 {
				m.profilePostCursor = 0
			} else {
				m.profileCommCursor = 0
			}
			return m, nil
		case "G", "end":
			if m.profileTab == 0 {
				if len(m.profilePosts) > 0 {
					m.profilePostCursor = len(m.profilePosts) - 1
				}
			} else {
				if len(m.profileComments) > 0 {
					m.profileCommCursor = len(m.profileComments) - 1
				}
			}
			return m, nil
		case "enter":
			if m.profileTab == 0 {
				if len(m.profilePosts) > 0 && m.profilePostCursor < len(m.profilePosts) {
					p := m.profilePosts[m.profilePostCursor]
					return m, m.loadPostDetailCmd(p.ID)
				}
			} else {
				if len(m.profileComments) > 0 && m.profileCommCursor < len(m.profileComments) {
					c := m.profileComments[m.profileCommCursor]
					commID := c.ID
					return m, m.loadPostDetailAndFocusCommentCmd(c.PostID, &commID)
				}
			}
			return m, nil
		}
	}
	return m, nil
}

// --- Notifications & Inbox Commands ------------------------------------

func (m *Model) checkUnreadNotificationsCmd() tea.Cmd {
	return func() tea.Msg {
		if m.user == nil {
			return nil
		}
		ctx, cancel := context.WithTimeout(m.ctx, 3*time.Second)
		defer cancel()

		count, err := m.queries.GetUnreadNotificationCount(ctx, m.user.ID)
		if err != nil {
			return nil
		}
		return unreadNotificationCountMsg{count: int(count)}
	}
}

func (m *Model) openInboxCmd(returnView viewState) tea.Cmd {
	m.inboxReturnView = returnView
	m.notificationCursor = 0
	return func() tea.Msg {
		if m.user == nil {
			return nil
		}
		ctx, cancel := context.WithTimeout(m.ctx, 3*time.Second)
		defer cancel()

		notifs, err := m.queries.ListNotificationsKeyset(ctx, db.ListNotificationsKeysetParams{
			UserID: m.user.ID,
			Limit:  50,
		})
		if err != nil {
			return errMsg{err: fmt.Errorf("loading inbox: %w", err)}
		}
		return notificationsLoadedMsg{notifications: notifs}
	}
}

func (m *Model) markNotificationReadCmd(notificationID int64) tea.Cmd {
	return func() tea.Msg {
		if m.user == nil {
			return nil
		}
		ctx, cancel := context.WithTimeout(m.ctx, 3*time.Second)
		defer cancel()

		_ = m.queries.MarkNotificationAsReadByID(ctx, db.MarkNotificationAsReadByIDParams{
			ID:     notificationID,
			UserID: m.user.ID,
		})
		return notificationMarkedReadMsg{notificationID: notificationID}
	}
}

func (m *Model) markAllNotificationsReadCmd() tea.Cmd {
	return func() tea.Msg {
		if m.user == nil {
			return nil
		}
		ctx, cancel := context.WithTimeout(m.ctx, 3*time.Second)
		defer cancel()

		_ = m.queries.MarkNotificationsRead(ctx, m.user.ID)
		return allNotificationsMarkedReadMsg{}
	}
}

// --- User Profile Commands ---------------------------------------------

func (m *Model) openProfileCmd(handle string, returnView viewState) tea.Cmd {
	m.profileReturnView = returnView
	m.profileTab = 0
	m.profilePostCursor = 0
	m.profileCommCursor = 0
	return func() tea.Msg {
		ctx, cancel := context.WithTimeout(m.ctx, 3*time.Second)
		defer cancel()

		user, err := m.queries.GetUserProfileByHandle(ctx, handle)
		if err != nil {
			if errors.Is(err, pgx.ErrNoRows) {
				return userProfileLoadedMsg{user: nil, posts: nil, comments: nil}
			}
			return errMsg{err: fmt.Errorf("loading user profile: %w", err)}
		}

		posts, err := m.queries.ListPostsByAuthorKeyset(ctx, db.ListPostsByAuthorKeysetParams{
			AuthorID: user.ID,
			Limit:    30,
		})
		if err != nil {
			return errMsg{err: fmt.Errorf("loading user posts: %w", err)}
		}

		comments, err := m.queries.ListCommentsByAuthorKeyset(ctx, db.ListCommentsByAuthorKeysetParams{
			AuthorID: user.ID,
			Limit:    30,
		})
		if err != nil {
			return errMsg{err: fmt.Errorf("loading user comments: %w", err)}
		}

		return userProfileLoadedMsg{
			user:     &user,
			posts:    posts,
			comments: comments,
		}
	}
}
