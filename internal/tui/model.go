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
	viewHelp
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
	currentView    viewState
	helpReturnView viewState
	width          int
	height         int
	err            error

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
	searchIn.PromptStyle = styleFilterPrompt

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
		searchInput:  searchIn,
		viewport:      vp,
		commentCursor: -1,
		feedPage:      1,
		readPosts:     make(map[int64]bool),
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
	case viewHelp:
		return m.updateHelp(msg)
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
	case viewHelp:
		return m.viewHelp()
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

func (m *Model) updateBoardList(msg tea.Msg) (*Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch {
		case msg.String() == "?":
			m.helpReturnView = viewBoardList
			m.currentView = viewHelp
			return m, nil
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
		}
		return m, nil
	}

	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch {
		case msg.String() == "?":
			m.helpReturnView = viewPostList
			m.currentView = viewHelp
			return m, nil
		case msg.String() == "/":
			m.searchFocused = true
			m.searchInput.Focus()
			return m, textinput.Blink
		case msg.String() == "q":
			return m, tea.Quit
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
				return m, m.castPostVoteCmd(post.ID, 1)
			}
		case msg.String() == "d":
			if len(visible) > 0 && m.postCursor < len(visible) {
				post := visible[m.postCursor]
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

		commSort := m.commentSortMode
		sortedComments := sortCommentTree(comments, commSort)

		return postDetailLoadedMsg{post: &post, comments: sortedComments}
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
		case "esc":
			m.currentView = viewPostList
			if m.currentBoard != nil {
				return m, m.loadPostsCmd(m.currentBoard.ID)
			}
			return m, nil
		case "q":
			return m, tea.Quit
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

		var newDirection int16
		err := m.execTx(ctx, func(q *db.Queries) error {
			existing, err := q.GetPostVoteByUser(ctx, db.GetPostVoteByUserParams{
				UserID: m.user.ID,
				PostID: postID,
			})

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
			return nil
		})
		if err != nil {
			return errMsg{err: err}
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
			existing, err := q.GetCommentVoteByUser(ctx, db.GetCommentVoteByUserParams{
				UserID:    m.user.ID,
				CommentID: commentID,
			})

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
			return nil
		})
		if err != nil {
			return errMsg{err: err}
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
		m.urlInput.Blur()
		m.bodyInput.Blur()
	case 2:
		m.titleInput.Blur()
		m.urlInput.Focus()
		m.bodyInput.Blur()
	case 3:
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
			m.postFormFocus = (m.postFormFocus + 1) % 4
			m.syncPostFormFocus()
			return m, nil
		case msg.String() == "shift+tab" || msg.Type == tea.KeyShiftTab:
			m.postFormFocus = (m.postFormFocus - 1 + 4) % 4
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
			} else if m.postFormFocus == 2 {
				m.postFormFocus = 3
				m.syncPostFormFocus()
				return m, nil
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

		ctx, cancel := context.WithTimeout(m.ctx, 3*time.Second)
		defer cancel()

		post, err := m.queries.CreatePost(ctx, db.CreatePostParams{
			BoardID:  m.currentBoard.ID,
			AuthorID: m.user.ID,
			Title:    sanitize.SingleLine(title),
			Body:     sanitize.Text(body),
			Url:      sanitize.SingleLine(url),
			Category: category,
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
			return nil
		})
		if err != nil {
			return errMsg{err: err}
		}

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
			return m, nil
		}
	}
	return m, nil
}


