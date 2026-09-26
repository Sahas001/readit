package tui

import (
	"strings"
	"testing"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/jackc/pgx/v5/pgtype"
	db "github.com/sahas/readit/internal/db/sqlc"
)

func TestHeaderNotificationBadge(t *testing.T) {
	m := &Model{
		width: 80,
		user:  &db.User{Handle: "alice"},
	}

	// 1. Zero unread notifications
	m.unreadNotificationCount = 0
	hdrZero := m.renderHeader("General", 80)
	if !strings.Contains(hdrZero, "@alice") {
		t.Errorf("header should show handle: %s", hdrZero)
	}
	if strings.Contains(hdrZero, "[") && strings.Contains(hdrZero, "]") {
		t.Errorf("header should not show unread badge when count is 0: %s", hdrZero)
	}

	// 2. 5 unread notifications
	m.unreadNotificationCount = 5
	hdrUnread := m.renderHeader("General", 80)
	if !strings.Contains(hdrUnread, "@alice") || !strings.Contains(hdrUnread, "[5]") {
		t.Errorf("header should show unread badge '@alice [5]': %s", hdrUnread)
	}
}

func TestViewInbox_EmptyAndPopulated(t *testing.T) {
	m := &Model{
		width:       80,
		height:      24,
		currentView: viewInbox,
		user:        &db.User{Handle: "alice"},
	}

	// 1. Empty state
	m.notifications = nil
	m.notificationCursor = 0
	outEmpty := m.viewInbox()
	if !strings.Contains(outEmpty, "No notifications yet.") {
		t.Errorf("expected empty inbox prompt: %s", outEmpty)
	}

	// 2. Populated state with unread and read items
	now := time.Now()
	m.notifications = []db.ListNotificationsKeysetRow{
		{
			ID:             1,
			UserID:         10,
			ActorID:        20,
			PostID:         100,
			CommentID:      pgtype.Int8{Int64: 201, Valid: true},
			Type:           "reply_post",
			IsRead:         false,
			CreatedAt:      pgtype.Timestamptz{Time: now.Add(-5 * time.Minute), Valid: true},
			ActorHandle:    "bob",
			PostTitle:      "Why Go is great",
			CommentSnippet: "I totally agree with your points!",
		},
		{
			ID:             2,
			UserID:         10,
			ActorID:        30,
			PostID:         100,
			CommentID:      pgtype.Int8{Int64: 202, Valid: true},
			Type:           "reply_comment",
			IsRead:         true,
			CreatedAt:      pgtype.Timestamptz{Time: now.Add(-2 * time.Hour), Valid: true},
			ActorHandle:    "charlie",
			PostTitle:      "Why Go is great",
			CommentSnippet: "Check out this alternative approach.",
		},
	}
	m.unreadNotificationCount = 1
	m.notificationCursor = 0

	out := m.viewInbox()

	// Verify unread badge in header
	if !strings.Contains(out, "Inbox (1 unread)") {
		t.Errorf("expected inbox header with unread count: %s", out)
	}

	// Verify cursor indicator on first item
	if !strings.Contains(out, "▌") {
		t.Errorf("expected selection cursor indicator: %s", out)
	}

	// Verify actor and action text
	if !strings.Contains(out, "@bob") || !strings.Contains(out, "replied to your post") {
		t.Errorf("expected post reply notification text for bob: %s", out)
	}
	if !strings.Contains(out, "I totally agree with your points!") {
		t.Errorf("expected snippet text: %s", out)
	}

	// Verify read notification for charlie
	if !strings.Contains(out, "@charlie") || !strings.Contains(out, "replied to your comment") {
		t.Errorf("expected comment reply notification text for charlie: %s", out)
	}
}

func TestUpdateInbox_NavigationAndActions(t *testing.T) {
	m := &Model{
		currentView:     viewInbox,
		inboxReturnView: viewPostList,
		notifications: []db.ListNotificationsKeysetRow{
			{ID: 1, PostID: 100, IsRead: false},
			{ID: 2, PostID: 101, IsRead: false},
			{ID: 3, PostID: 102, IsRead: true},
		},
		notificationCursor: 0,
	}

	// 1. Move down with j
	m, _ = m.updateInbox(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'j'}})
	if m.notificationCursor != 1 {
		t.Errorf("expected cursor 1 after j, got %d", m.notificationCursor)
	}

	// 2. Move to bottom with G
	m, _ = m.updateInbox(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'G'}})
	if m.notificationCursor != 2 {
		t.Errorf("expected cursor 2 after G, got %d", m.notificationCursor)
	}

	// 3. Move up with k
	m, _ = m.updateInbox(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'k'}})
	if m.notificationCursor != 1 {
		t.Errorf("expected cursor 1 after k, got %d", m.notificationCursor)
	}

	// 4. Move to top with g
	m, _ = m.updateInbox(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'g'}})
	if m.notificationCursor != 0 {
		t.Errorf("expected cursor 0 after g, got %d", m.notificationCursor)
	}

	// 5. Mark all read with a
	m, cmd := m.updateInbox(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'a'}})
	if cmd == nil {
		t.Errorf("expected markAllNotificationsReadCmd on 'a'")
	}
	if !strings.Contains(m.flashMsg, "marked as read") {
		t.Errorf("expected flash message on 'a': %s", m.flashMsg)
	}

	// 6. Return with esc
	m, _ = m.updateInbox(tea.KeyMsg{Type: tea.KeyEscape})
	if m.currentView != viewPostList {
		t.Errorf("expected return to viewPostList on esc, got %d", m.currentView)
	}
}

func TestViewProfile_LayoutAndTabs(t *testing.T) {
	now := time.Now()
	m := &Model{
		width:       80,
		height:      24,
		currentView: viewProfile,
		profileUser: &db.User{
			ID:           1,
			Handle:       "satoshis_ghost",
			Bio:          "Building decentralized terminal tools.",
			PostKarma:    120,
			CommentKarma: 45,
			CreatedAt:    pgtype.Timestamptz{Time: now.Add(-30 * 24 * time.Hour), Valid: true},
		},
		profileTab: 0,
		profilePosts: []db.ListPostsByAuthorKeysetRow{
			{
				ID:           1,
				Title:        "Bitcoin in terminal",
				BoardSlug:    "crypto",
				Score:        25,
				CommentCount: 14,
				CreatedAt:    pgtype.Timestamptz{Time: now.Add(-2 * time.Hour), Valid: true},
			},
		},
		profileComments: []db.ListCommentsByAuthorKeysetRow{
			{
				ID:        10,
				PostID:    1,
				PostTitle: "Bitcoin in terminal",
				BoardSlug: "crypto",
				Body:      "Code is open source on GitHub.",
				Score:     12,
				CreatedAt: pgtype.Timestamptz{Time: now.Add(-1 * time.Hour), Valid: true},
			},
		},
	}

	// 1. Submissions Tab
	m.profileTab = 0
	outSub := m.viewProfile()
	if !strings.Contains(outSub, "@satoshis_ghost") {
		t.Errorf("missing handle in profile: %s", outSub)
	}
	if !strings.Contains(outSub, "165 karma") || !strings.Contains(outSub, "120 post · 45 comment") {
		t.Errorf("missing or incorrect karma breakdown in profile: %s", outSub)
	}
	if !strings.Contains(outSub, "Building decentralized terminal tools.") {
		t.Errorf("missing bio in profile: %s", outSub)
	}
	if !strings.Contains(outSub, "Bitcoin in terminal") {
		t.Errorf("missing submission title: %s", outSub)
	}

	// 2. Comments Tab
	m.profileTab = 1
	outComm := m.viewProfile()
	if !strings.Contains(outComm, "Code is open source on GitHub.") {
		t.Errorf("missing comment snippet in comments tab: %s", outComm)
	}
	if !strings.Contains(outComm, "/b/crypto") {
		t.Errorf("missing board slug context: %s", outComm)
	}

	// 3. User Not Found
	m.profileUser = nil
	outNotFound := m.viewProfile()
	if !strings.Contains(outNotFound, "User not found") {
		t.Errorf("expected User not found message: %s", outNotFound)
	}
}

func TestUpdateProfile_TabSwitchingAndNavigation(t *testing.T) {
	m := &Model{
		currentView:       viewProfile,
		profileReturnView: viewBoardList,
		profileTab:        0,
		profilePosts: []db.ListPostsByAuthorKeysetRow{
			{ID: 1, Title: "Post 1"},
			{ID: 2, Title: "Post 2"},
		},
		profileComments: []db.ListCommentsByAuthorKeysetRow{
			{ID: 10, PostID: 1, Body: "Comment 1"},
			{ID: 20, PostID: 2, Body: "Comment 2"},
		},
	}

	// 1. Tab switches to Comments
	m, _ = m.updateProfile(tea.KeyMsg{Type: tea.KeyTab})
	if m.profileTab != 1 {
		t.Errorf("expected tab 1 after tab key, got %d", m.profileTab)
	}

	// 2. Move comment cursor
	m, _ = m.updateProfile(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'j'}})
	if m.profileCommCursor != 1 {
		t.Errorf("expected comment cursor 1, got %d", m.profileCommCursor)
	}

	// 3. Shift+Tab switches back to Submissions
	m, _ = m.updateProfile(tea.KeyMsg{Type: tea.KeyShiftTab})
	if m.profileTab != 0 {
		t.Errorf("expected tab 0 after shift+tab, got %d", m.profileTab)
	}

	// 4. Move post cursor
	m, _ = m.updateProfile(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'j'}})
	if m.profilePostCursor != 1 {
		t.Errorf("expected post cursor 1, got %d", m.profilePostCursor)
	}

	// 5. Esc returns to returnView
	m, _ = m.updateProfile(tea.KeyMsg{Type: tea.KeyEscape})
	if m.currentView != viewBoardList {
		t.Errorf("expected currentView viewBoardList on esc, got %d", m.currentView)
	}
}
