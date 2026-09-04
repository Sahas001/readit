// Package tui implements the Bubble Tea terminal UI for ReadIT.
package tui

import (
	"context"
	"fmt"
	"log/slog"
	"strings"

	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/jackc/pgx/v5/pgxpool"

	db "github.com/sahas/readit/internal/db/sqlc"
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

// errMsg wraps an error for the Update loop.
type errMsg struct{ err error }

func (e errMsg) Error() string { return e.err.Error() }

// --- Constructor -------------------------------------------------------

// NewModel creates a new root Model for a Bubble Tea session.
func NewModel(ctx context.Context, pool *pgxpool.Pool, fingerprint string, logger *slog.Logger) *Model {
	ti := textinput.New()
	ti.Placeholder = "choose a handle (e.g. satoshi)"
	ti.CharLimit = 30
	ti.Width = 40

	return &Model{
		ctx:         ctx,
		pool:        pool,
		queries:     db.New(pool),
		logger:      logger,
		fingerprint: fingerprint,
		currentView: viewLoading,
		keys:        DefaultKeyMap(),
		handleInput: ti,
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
	case viewError:
		return m.viewError()
	default:
		return "Unknown view state"
	}
}

// --- User loading ------------------------------------------------------

func (m *Model) lookupUserCmd() tea.Cmd {
	return func() tea.Msg {
		user, err := m.queries.GetUserByPubkey(m.ctx, m.fingerprint)
		if err != nil {
			// User not found — they need to onboard.
			return userLoadedMsg{user: nil, isNew: true}
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
			if handle == "" {
				return m, nil
			}
			return m, m.createUserCmd(handle)
		case "esc":
			return m, tea.Quit
		}
	}

	var cmd tea.Cmd
	m.handleInput, cmd = m.handleInput.Update(msg)
	return m, cmd
}

func (m *Model) createUserCmd(handle string) tea.Cmd {
	return func() tea.Msg {
		user, err := m.queries.UpsertUser(m.ctx, db.UpsertUserParams{
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
		boards, err := m.queries.ListBoards(m.ctx)
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

func (m *Model) loadPostsCmd(boardID int64) tea.Cmd {
	return func() tea.Msg {
		posts, err := m.queries.ListPostsByBoardNew(m.ctx, db.ListPostsByBoardNewParams{
			BoardID: boardID,
			Limit:   25,
			Offset:  0,
		})
		if err != nil {
			return errMsg{err: fmt.Errorf("loading posts: %w", err)}
		}
		return postsLoadedMsg{posts: posts}
	}
}

// --- Post list ---------------------------------------------------------

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
		}
	}
	return m, nil
}
