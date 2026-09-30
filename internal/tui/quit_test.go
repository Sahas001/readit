package tui

import (
	"context"
	"testing"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/jackc/pgx/v5/pgtype"
	db "github.com/sahas/readit/internal/db/sqlc"
)

func TestQuitConfirmationBoardList(t *testing.T) {
	m := NewModel(context.Background(), nil, "test-fingerprint", nil)
	m.currentView = viewBoardList
	m.boards = []db.Board{
		{ID: 1, Title: "General", Slug: "general", Description: "General discussions"},
	}

	// 1. First 'q' press should NOT quit, but flash confirmation
	updatedModel, cmd := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'q'}})
	m = updatedModel.(*Model)

	if m.quitConfirm != true {
		t.Errorf("expected m.quitConfirm == true after first 'q', got %v", m.quitConfirm)
	}
	expectedFlash := "• Press 'q' again to exit"
	if m.flashMsg != expectedFlash {
		t.Errorf("expected flashMsg %q, got %q", expectedFlash, m.flashMsg)
	}
	if cmd == nil {
		t.Errorf("expected non-nil timer cmd after first 'q'")
	}

	// 2. Second 'q' press should return tea.Quit
	_, quitCmd := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'q'}})
	if quitCmd == nil {
		t.Fatalf("expected quit command on second 'q', got nil")
	}
	// Verify quitCmd produces tea.QuitMsg
	msg := quitCmd()
	if _, ok := msg.(tea.QuitMsg); !ok {
		t.Errorf("expected tea.QuitMsg, got %T (%v)", msg, msg)
	}
}

func TestQuitConfirmationPostList(t *testing.T) {
	m := NewModel(context.Background(), nil, "test-fingerprint", nil)
	m.currentView = viewPostList
	m.currentBoard = &db.Board{ID: 1, Title: "General", Slug: "general"}
	m.posts = []PostFeedItem{
		{ID: 1, Title: "Test Post", BoardID: 1},
	}

	// 1. First 'q' press
	updatedModel, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'q'}})
	m = updatedModel.(*Model)

	if !m.quitConfirm {
		t.Errorf("expected quitConfirm == true")
	}
	if m.flashMsg != "• Press 'q' again to exit" {
		t.Errorf("expected quit flash message, got %q", m.flashMsg)
	}

	// 2. Second 'q' press confirms quit
	_, quitCmd := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'q'}})
	if quitCmd == nil {
		t.Fatalf("expected quit command on second 'q', got nil")
	}
	if _, ok := quitCmd().(tea.QuitMsg); !ok {
		t.Errorf("expected tea.QuitMsg")
	}
}

func TestQuitConfirmationPostDetail(t *testing.T) {
	m := NewModel(context.Background(), nil, "test-fingerprint", nil)
	m.currentView = viewPostDetail
	m.currentBoard = &db.Board{ID: 1, Title: "General", Slug: "general"}
	m.currentPost = &db.GetPostByIDRow{
		ID:        1,
		Title:     "Test Post",
		CreatedAt: pgtype.Timestamptz{Time: time.Now(), Valid: true},
	}

	// 1. First 'q' press
	updatedModel, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'q'}})
	m = updatedModel.(*Model)

	if !m.quitConfirm {
		t.Errorf("expected quitConfirm == true")
	}
	if m.flashMsg != "• Press 'q' again to exit" {
		t.Errorf("expected quit flash message, got %q", m.flashMsg)
	}

	// 2. Second 'q' confirms quit
	_, quitCmd := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'q'}})
	if quitCmd == nil {
		t.Fatalf("expected quit command on second 'q', got nil")
	}
	if _, ok := quitCmd().(tea.QuitMsg); !ok {
		t.Errorf("expected tea.QuitMsg")
	}
}

func TestQuitConfirmationCancelledByEscape(t *testing.T) {
	m := NewModel(context.Background(), nil, "test-fingerprint", nil)
	m.currentView = viewPostList
	m.currentBoard = &db.Board{ID: 1, Title: "General", Slug: "general"}

	// 1. Press 'q'
	updatedModel, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'q'}})
	m = updatedModel.(*Model)
	if !m.quitConfirm {
		t.Fatalf("expected quitConfirm == true")
	}

	// 2. Press 'esc' to cancel
	updatedModel, cmd := m.Update(tea.KeyMsg{Type: tea.KeyEscape})
	m = updatedModel.(*Model)

	if m.quitConfirm {
		t.Errorf("expected quitConfirm == false after esc")
	}
	if m.flashMsg != "" {
		t.Errorf("expected flashMsg to be cleared after esc, got %q", m.flashMsg)
	}
	if m.currentView != viewPostList {
		t.Errorf("expected to stay in viewPostList after canceling quit with esc, got %v", m.currentView)
	}
	if cmd != nil {
		t.Errorf("expected nil cmd after canceling quit with esc")
	}
}

func TestQuitConfirmationConfirmedByY(t *testing.T) {
	m := NewModel(context.Background(), nil, "test-fingerprint", nil)
	m.currentView = viewBoardList
	m.boards = []db.Board{{ID: 1, Slug: "general", Title: "General"}}

	// 1. Press 'q'
	updatedModel, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'q'}})
	m = updatedModel.(*Model)
	if !m.quitConfirm {
		t.Fatalf("expected quitConfirm == true")
	}

	// 2. Press 'y' to confirm
	_, quitCmd := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'y'}})
	if quitCmd == nil {
		t.Fatalf("expected quitCmd on 'y', got nil")
	}
	if _, ok := quitCmd().(tea.QuitMsg); !ok {
		t.Errorf("expected tea.QuitMsg")
	}
}

func TestQuitConfirmationCancelledByNavigation(t *testing.T) {
	m := NewModel(context.Background(), nil, "test-fingerprint", nil)
	m.currentView = viewBoardList
	m.boards = []db.Board{
		{ID: 1, Slug: "general", Title: "General"},
		{ID: 2, Slug: "programming", Title: "Programming"},
	}

	// 1. Press 'q'
	updatedModel, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'q'}})
	m = updatedModel.(*Model)
	if !m.quitConfirm {
		t.Fatalf("expected quitConfirm == true")
	}

	// 2. Press 'j' (move down)
	updatedModel, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'j'}})
	m = updatedModel.(*Model)

	if m.quitConfirm {
		t.Errorf("expected quitConfirm to be reset after navigation 'j'")
	}
	if m.flashMsg != "" {
		t.Errorf("expected flashMsg to be cleared after navigation 'j'")
	}
	if m.boardCursor != 1 {
		t.Errorf("expected boardCursor to move to 1, got %d", m.boardCursor)
	}
}

func TestQuitConfirmationTimeout(t *testing.T) {
	m := NewModel(context.Background(), nil, "test-fingerprint", nil)
	m.currentView = viewBoardList

	// 1. Press 'q'
	updatedModel, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'q'}})
	m = updatedModel.(*Model)
	currentID := m.quitConfirmID

	// 2. Send mismatched timeout message (should not clear)
	updatedModel, _ = m.Update(quitConfirmTimeoutMsg{id: currentID - 1})
	m = updatedModel.(*Model)
	if !m.quitConfirm {
		t.Errorf("expected quitConfirm to remain true for mismatched timeout ID")
	}

	// 3. Send matching timeout message (should clear)
	updatedModel, _ = m.Update(quitConfirmTimeoutMsg{id: currentID})
	m = updatedModel.(*Model)
	if m.quitConfirm {
		t.Errorf("expected quitConfirm to be false after matching timeout")
	}
	if m.flashMsg != "" {
		t.Errorf("expected flashMsg to be cleared after matching timeout, got %q", m.flashMsg)
	}
}

func TestSearchFocusDoesNotTriggerQuit(t *testing.T) {
	m := NewModel(context.Background(), nil, "test-fingerprint", nil)
	m.currentView = viewPostList
	m.currentBoard = &db.Board{ID: 1, Slug: "general", Title: "General"}
	m.searchFocused = true
	m.searchInput.Focus()

	// Pressing 'q' while search is focused should type 'q' into search input
	updatedModel, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'q'}})
	m = updatedModel.(*Model)

	if m.quitConfirm {
		t.Errorf("search focus should not trigger quitConfirm")
	}
	if m.searchInput.Value() != "q" {
		t.Errorf("expected searchInput to contain 'q', got %q", m.searchInput.Value())
	}
}
