package tui

import (
	"context"
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	db "github.com/sahas/readit/internal/db/sqlc"
)

func TestCensorship_NewPostProfanityRejection(t *testing.T) {
	m := NewModel(context.Background(), nil, "fp12345", nil)
	m.user = &db.User{ID: 1, Handle: "testuser"}
	m.currentBoard = &db.Board{ID: 1, Slug: "general"}
	m.openNewPost()

	// 1. Post title contains profanity
	m.titleInput.SetValue("What the fuck is this")
	m.bodyInput.SetValue("Clean post body here")

	m, _ = m.updateNewPost(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("s")}) // triggers default: m.err = nil
	// Send ctrl+s
	m, cmd := m.updateNewPost(tea.KeyMsg{Type: tea.KeyCtrlS})
	if cmd != nil {
		t.Errorf("expected cmd to be nil when title is profane, but got non-nil cmd")
	}
	if m.err == nil {
		t.Fatalf("expected m.err to be set for profane title, got nil")
	}
	if !strings.Contains(m.err.Error(), "post title contains prohibited language") {
		t.Errorf("unexpected error message: %v", m.err)
	}
	if m.currentView != viewNewPost {
		t.Errorf("expected to stay in viewNewPost, got %v", m.currentView)
	}

	// 2. Clear error on subsequent input
	m, _ = m.updateNewPost(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("a")})
	if m.err != nil {
		t.Errorf("expected m.err to be cleared on typing, got %v", m.err)
	}

	// 3. Post body contains profanity
	m.titleInput.SetValue("Valid Clean Title")
	m.bodyInput.SetValue("This is total bullshit")
	m, cmd = m.updateNewPost(tea.KeyMsg{Type: tea.KeyCtrlS})
	if cmd != nil {
		t.Errorf("expected cmd to be nil when body is profane, but got non-nil cmd")
	}
	if m.err == nil {
		t.Fatalf("expected m.err to be set for profane body, got nil")
	}
	if !strings.Contains(m.err.Error(), "post body contains prohibited language") {
		t.Errorf("unexpected error message: %v", m.err)
	}

	// 4. Defensive check in submitPostCmd
	defensiveCmd := m.submitPostCmd("Clean Title", "Fucking broken body", "")
	msg := defensiveCmd()
	errMsgMsg, ok := msg.(errMsg)
	if !ok {
		t.Fatalf("expected submitPostCmd to return errMsg for profane body, got %T", msg)
	}
	if !strings.Contains(errMsgMsg.err.Error(), "prohibited language") {
		t.Errorf("unexpected error from submitPostCmd: %v", errMsgMsg.err)
	}
}

func TestCensorship_NewCommentProfanityRejection(t *testing.T) {
	m := NewModel(context.Background(), nil, "fp12345", nil)
	m.user = &db.User{ID: 1, Handle: "testuser"}
	m.currentPost = &db.GetPostByIDRow{ID: 1, Title: "Test Post"}
	m.openNewComment(nil, "op")

	// 1. Comment contains profanity
	m.commentInput.SetValue("You are a b!tch")
	m, cmd := m.updateNewComment(tea.KeyMsg{Type: tea.KeyCtrlS})
	if cmd != nil {
		t.Errorf("expected cmd to be nil when comment is profane, but got non-nil cmd")
	}
	if m.err == nil {
		t.Fatalf("expected m.err to be set for profane comment, got nil")
	}
	if !strings.Contains(m.err.Error(), "comment contains prohibited language") {
		t.Errorf("unexpected error message: %v", m.err)
	}
	if m.currentView != viewNewComment {
		t.Errorf("expected to stay in viewNewComment, got %v", m.currentView)
	}

	// 2. Typing clears error
	m, _ = m.updateNewComment(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("a")})
	if m.err != nil {
		t.Errorf("expected m.err to be cleared on typing, got %v", m.err)
	}

	// 3. Defensive check in submitCommentCmd
	defensiveCmd := m.submitCommentCmd("Shut the fuck up")
	msg := defensiveCmd()
	errMsgMsg, ok := msg.(errMsg)
	if !ok {
		t.Fatalf("expected submitCommentCmd to return errMsg for profane comment, got %T", msg)
	}
	if !strings.Contains(errMsgMsg.err.Error(), "prohibited language") {
		t.Errorf("unexpected error from submitCommentCmd: %v", errMsgMsg.err)
	}
}
