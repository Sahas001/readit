package tui

import (
	"context"
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	db "github.com/sahas/readit/internal/db/sqlc"
)

func TestThemePalettes(t *testing.T) {
	themes := Themes()
	expectedIDs := []string{"readit", "catppuccin", "nord", "dracula", "gruvbox", "tokyonight"}

	if len(themes) != len(expectedIDs) {
		t.Fatalf("expected %d themes, got %d", len(expectedIDs), len(themes))
	}

	for i, id := range expectedIDs {
		th := themes[i]
		if th.ID != id {
			t.Errorf("theme %d: expected ID %q, got %q", i, id, th.ID)
		}
		if th.Name == "" {
			t.Errorf("theme %q: expected non-empty display name", id)
		}
		if string(th.Primary) == "" {
			t.Errorf("theme %q: expected non-empty Primary color", id)
		}
		if string(th.Background) == "" {
			t.Errorf("theme %q: expected non-empty Background color", id)
		}
		if string(th.Text) == "" {
			t.Errorf("theme %q: expected non-empty Text color", id)
		}
		if string(th.Border) == "" {
			t.Errorf("theme %q: expected non-empty Border color", id)
		}
	}
}

func TestGetThemeLookup(t *testing.T) {
	tests := []struct {
		input      string
		expectedID string
	}{
		{"readit", "readit"},
		{"READIT", "readit"},
		{"catppuccin", "catppuccin"},
		{"catppuccin-mocha", "catppuccin"},
		{"mocha", "catppuccin"},
		{"nord", "nord"},
		{"dracula", "dracula"},
		{"gruvbox", "gruvbox"},
		{"gruvbox-dark", "gruvbox"},
		{"tokyonight", "tokyonight"},
		{"tokyo-night", "tokyonight"},
		{"nonexistent", "readit"},
		{"", "readit"},
	}

	for _, tt := range tests {
		th := GetTheme(tt.input)
		if th.ID != tt.expectedID {
			t.Errorf("GetTheme(%q) = %q, expected %q", tt.input, th.ID, tt.expectedID)
		}
	}
}

func TestStylesDecoupling(t *testing.T) {
	thReadIT := ThemeReadIT()
	thDracula := ThemeDracula()
	thNord := ThemeNord()

	sReadIT := NewStyles(thReadIT)
	sDracula := NewStyles(thDracula)
	sNord := NewStyles(thNord)

	// Distinct themes must produce distinct semantic styles (e.g. Prompt/Selection/Title)
	if sReadIT.Prompt.GetForeground() == sDracula.Prompt.GetForeground() {
		t.Errorf("expected different prompt colors between ReadIT and Dracula")
	}
	if sNord.Prompt.GetForeground() == sDracula.Prompt.GetForeground() {
		t.Errorf("expected different prompt colors between Nord and Dracula")
	}

	// Logo reflects the active theme's primary color
	if sReadIT.Logo.GetForeground() == sDracula.Logo.GetForeground() {
		t.Errorf("expected different logo colors between ReadIT and Dracula")
	}

	// Earth colors remain constant across all themes
	eReadIT := newEarthStyles(thReadIT)
	eDracula := newEarthStyles(thDracula)
	if eReadIT.High.GetForeground() != eDracula.High.GetForeground() ||
		eReadIT.Sea.GetForeground() != eDracula.Sea.GetForeground() {
		t.Errorf("expected revolving Earth colors to remain constant across all themes")
	}

	// Logo shine has theme-specific complementing colors
	shReadIT := newLogoShineStyles(thReadIT)
	shDracula := newLogoShineStyles(thDracula)
	if shReadIT.Shine1.GetForeground() == shDracula.Shine1.GetForeground() {
		t.Errorf("expected different complementing shine highlights between ReadIT and Dracula")
	}

	// Model initialization attaches default theme and styles
	m := NewModel(context.Background(), nil, "test-key", nil)
	if m.themeID != "readit" {
		t.Errorf("expected default themeID 'readit', got %q", m.themeID)
	}
	if m.theme.ID != "readit" {
		t.Errorf("expected default theme.ID 'readit', got %q", m.theme.ID)
	}

	// Changing theme on Model updates styles without affecting other models
	m2 := NewModel(context.Background(), nil, "test-key-2", nil)
	m.setTheme("tokyonight")
	if m.themeID != "tokyonight" {
		t.Errorf("expected m.themeID 'tokyonight', got %q", m.themeID)
	}
	if m2.themeID != "readit" {
		t.Errorf("expected m2.themeID to remain 'readit', got %q", m2.themeID)
	}
	if m.styles.Prompt.GetForeground() == m2.styles.Prompt.GetForeground() {
		t.Errorf("expected m and m2 to have distinct styling after setTheme")
	}
}

func TestThemePickerNavigationAndSelection(t *testing.T) {
	m := NewModel(context.Background(), nil, "test-key", nil)
	m.currentView = viewBoardList

	// 1. Press 't' from Board List opens theme picker modal
	m.updateBoardList(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'t'}})
	if m.currentView != viewThemePicker {
		t.Fatalf("expected viewThemePicker after pressing 't', got %v", m.currentView)
	}
	if m.themeReturnView != viewBoardList {
		t.Errorf("expected themeReturnView viewBoardList, got %v", m.themeReturnView)
	}

	themes := Themes()
	if m.themeCursor != 0 {
		t.Errorf("expected initial themeCursor 0, got %d", m.themeCursor)
	}

	// 2. Press 'j' navigates down
	m.updateThemePicker(tea.KeyMsg{Type: tea.KeyDown})
	if m.themeCursor != 1 {
		t.Errorf("expected themeCursor 1 after 'down', got %d", m.themeCursor)
	}

	// 3. Press 'k' navigates up
	m.updateThemePicker(tea.KeyMsg{Type: tea.KeyUp})
	if m.themeCursor != 0 {
		t.Errorf("expected themeCursor 0 after 'up', got %d", m.themeCursor)
	}

	// 4. Wrap-around on 'k'
	m.updateThemePicker(tea.KeyMsg{Type: tea.KeyUp})
	if m.themeCursor != len(themes)-1 {
		t.Errorf("expected themeCursor %d after wrap-around up, got %d", len(themes)-1, m.themeCursor)
	}

	// 5. Wrap-around on 'j'
	m.updateThemePicker(tea.KeyMsg{Type: tea.KeyDown})
	if m.themeCursor != 0 {
		t.Errorf("expected themeCursor 0 after wrap-around down, got %d", m.themeCursor)
	}

	// 6. Press 'esc' cancels without changing theme
	m.themeCursor = 2 // Nord
	m.updateThemePicker(tea.KeyMsg{Type: tea.KeyEscape})
	if m.currentView != viewBoardList {
		t.Errorf("expected return to viewBoardList after esc, got %v", m.currentView)
	}
	if m.themeID != "readit" {
		t.Errorf("expected themeID to remain 'readit' after cancel, got %q", m.themeID)
	}

	// 7. Open again and press Enter on selected theme
	m.currentView = viewBoardList
	m.updateBoardList(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'t'}})
	m.themeCursor = 3 // Dracula
	selectedTheme := themes[3]
	m.updateThemePicker(tea.KeyMsg{Type: tea.KeyEnter})

	if m.currentView != viewBoardList {
		t.Errorf("expected return to viewBoardList after enter, got %v", m.currentView)
	}
	if m.themeID != selectedTheme.ID {
		t.Errorf("expected themeID %q, got %q", selectedTheme.ID, m.themeID)
	}
	if m.theme.ID != selectedTheme.ID {
		t.Errorf("expected theme.ID %q, got %q", selectedTheme.ID, m.theme.ID)
	}
	if !strings.Contains(m.flashMsg, selectedTheme.Name) {
		t.Errorf("expected flash message with theme name, got %q", m.flashMsg)
	}
}

func TestThemePickerFromAllViews(t *testing.T) {
	viewsToTest := []struct {
		view       viewState
		name       string
		triggerKey string
	}{
		{viewBoardList, "boardList", "t"},
		{viewPostList, "postList", "t"},
		{viewPostDetail, "postDetail", "t"},
		{viewInbox, "inbox", "t"},
		{viewProfile, "profile", "t"},
	}

	for _, vt := range viewsToTest {
		t.Run(vt.name, func(t *testing.T) {
			m := NewModel(context.Background(), nil, "test-key", nil)
			m.currentView = vt.view

			switch vt.view {
			case viewBoardList:
				m.updateBoardList(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'t'}})
			case viewPostList:
				m.updatePostList(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'t'}})
			case viewPostDetail:
				m.updatePostDetail(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'t'}})
			case viewInbox:
				m.updateInbox(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'t'}})
			case viewProfile:
				m.updateProfile(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'t'}})
			}

			if m.currentView != viewThemePicker {
				t.Fatalf("expected viewThemePicker from %s, got %v", vt.name, m.currentView)
			}
			if m.themeReturnView != vt.view {
				t.Fatalf("expected themeReturnView %v, got %v", vt.view, m.themeReturnView)
			}

			// Cancel with esc
			m.updateThemePicker(tea.KeyMsg{Type: tea.KeyEscape})
			if m.currentView != vt.view {
				t.Fatalf("expected return to view %v, got %v", vt.view, m.currentView)
			}
		})
	}
}

func TestThemePickerRendering(t *testing.T) {
	m := NewModel(context.Background(), nil, "test-key", nil)
	m.width = 80
	m.height = 30
	m.currentView = viewThemePicker
	m.initThemePicker()

	viewOutput := m.viewThemePicker()

	if !strings.Contains(viewOutput, "Select Color Theme") {
		t.Errorf("expected modal title in viewThemePicker, got:\n%s", viewOutput)
	}
	if !strings.Contains(viewOutput, "ACTIVE") {
		t.Errorf("expected ACTIVE badge in viewThemePicker, got:\n%s", viewOutput)
	}

	for _, th := range Themes() {
		if !strings.Contains(viewOutput, th.Name) {
			t.Errorf("expected theme name %q in viewThemePicker, got:\n%s", th.Name, viewOutput)
		}
	}
}

func TestUserLoadedWithThemePreference(t *testing.T) {
	m := NewModel(context.Background(), nil, "test-key", nil)
	user := &db.User{
		ID:     42,
		Handle: "themeuser",
		Theme:  "gruvbox",
	}

	m.handleUserLoaded(userLoadedMsg{user: user, isNew: false})
	if m.themeID != "gruvbox" {
		t.Errorf("expected user theme preference 'gruvbox' applied, got %q", m.themeID)
	}
	if m.theme.ID != "gruvbox" {
		t.Errorf("expected m.theme.ID 'gruvbox', got %q", m.theme.ID)
	}
}

func TestThemeAwareMarkdownAndAnimation(t *testing.T) {
	mdContent := "# Header 1\n**bold text** and [link](https://example.com)\n```go\nfmt.Println(\"hi\")\n```"

	for _, th := range Themes() {
		rendered := renderMarkdown(mdContent, 60, th)
		if rendered == "" {
			t.Errorf("expected non-empty rendered markdown for theme %q", th.ID)
		}

		hero := renderHeroBanner(5, 80, th)
		if hero == "" {
			t.Errorf("expected non-empty hero banner for theme %q", th.ID)
		}

		badge := styleCategoryBadgeWithTheme("showcase", th).Render("[showcase]")
		if badge == "" {
			t.Errorf("expected non-empty category badge for theme %q", th.ID)
		}
	}
}

func TestThemeChangeInPostDetailUpdatesViewport(t *testing.T) {
	m := NewModel(context.Background(), nil, "test-key", nil)
	m.width = 80
	m.height = 30
	m.currentView = viewPostDetail
	m.currentPost = &db.GetPostByIDRow{
		ID:           10,
		Title:        "Theme Switch Post",
		AuthorHandle: "alice",
		Body:         "Hello world, this is a discussion post body.",
		Score:        15,
	}
	m.comments = []db.GetCommentThreadByPostRow{
		{
			ID:           101,
			PostID:       10,
			AuthorHandle: "bob",
			Body:         "Great post!",
			Score:        5,
			Depth:        0,
		},
	}
	m.updateViewportSize()
	m.viewport.SetContent(m.renderPostDetailContent())

	contentReadIT := m.viewport.View()

	// Switch theme to Tokyo Night
	m.setTheme("tokyonight")
	contentTokyo := m.viewport.View()

	if contentTokyo == contentReadIT {
		t.Errorf("expected viewport content to update after setTheme('tokyonight')")
	}

	// Verify Tokyo Night primary color is in the new viewport content
	tokyoTheme := GetTheme("tokyonight")
	if !strings.Contains(contentTokyo, tokyoTheme.Name) && !strings.Contains(contentTokyo, "Theme Switch Post") {
		t.Errorf("expected post title in updated viewport content")
	}
}

func TestThemePickerAnimationResumeOnExit(t *testing.T) {
	m := NewModel(context.Background(), nil, "test-key", nil)
	m.currentView = viewBoardList
	m.openThemePicker(viewBoardList)

	if m.currentView != viewThemePicker {
		t.Fatalf("expected viewThemePicker, got %v", m.currentView)
	}

	// 1. Esc should return animTickCmd
	_, cmdEsc := m.updateThemePicker(tea.KeyMsg{Type: tea.KeyEscape})
	if m.currentView != viewBoardList {
		t.Errorf("expected return to viewBoardList after esc, got %v", m.currentView)
	}
	if cmdEsc == nil {
		t.Errorf("expected non-nil cmd (animTickCmd) on exit to viewBoardList via esc")
	}

	// 2. Enter should also return animTickCmd
	m.openThemePicker(viewBoardList)
	_, cmdEnter := m.updateThemePicker(tea.KeyMsg{Type: tea.KeyEnter})
	if m.currentView != viewBoardList {
		t.Errorf("expected return to viewBoardList after enter, got %v", m.currentView)
	}
	if cmdEnter == nil {
		t.Errorf("expected non-nil cmd (batch with animTickCmd) on exit to viewBoardList via enter")
	}

	// 3. Help modal exit should also resume animTickCmd
	m.currentView = viewBoardList
	m.updateBoardList(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'?'}})
	if m.currentView != viewHelp {
		t.Fatalf("expected viewHelp, got %v", m.currentView)
	}
	_, cmdHelpExit := m.updateHelp(tea.KeyMsg{Type: tea.KeyEscape})
	if m.currentView != viewBoardList {
		t.Errorf("expected return to viewBoardList after help exit, got %v", m.currentView)
	}
	if cmdHelpExit == nil {
		t.Errorf("expected non-nil cmd (animTickCmd) on exit from help to viewBoardList")
	}
}

func TestThemePickerTopBottomNavigation(t *testing.T) {
	m := NewModel(context.Background(), nil, "test-key", nil)
	m.openThemePicker(viewBoardList)

	themes := Themes()

	// Press 'G' to jump to bottom
	m.updateThemePicker(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'G'}})
	if m.themeCursor != len(themes)-1 {
		t.Errorf("expected cursor at bottom (%d), got %d", len(themes)-1, m.themeCursor)
	}

	// Press 'g' to jump to top
	m.updateThemePicker(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'g'}})
	if m.themeCursor != 0 {
		t.Errorf("expected cursor at top (0), got %d", m.themeCursor)
	}
}

func TestSaveThemeCmdGuards(t *testing.T) {
	// 1. Model with nil user: cmd returns nil Msg without panicking
	m := &Model{}
	cmd := m.saveThemeCmd("nord")
	if msg := cmd(); msg != nil {
		t.Errorf("expected nil msg for nil user, got %v", msg)
	}

	// 2. Model with user but nil queries: cmd returns nil Msg without panicking
	mWithUser := &Model{
		user: &db.User{ID: 1},
	}
	cmdWithUser := mWithUser.saveThemeCmd("nord")
	if msg := cmdWithUser(); msg != nil {
		t.Errorf("expected nil msg for nil queries, got %v", msg)
	}
}

func TestThemePickerNarrowTerminalResponsiveness(t *testing.T) {
	m := NewModel(context.Background(), nil, "test-key", nil)
	m.width = 40
	m.height = 24
	m.openThemePicker(viewBoardList)

	output := m.viewThemePicker()
	if output == "" {
		t.Fatalf("expected non-empty output for narrow terminal")
	}

	lines := strings.Split(output, "\n")
	for i, l := range lines {
		w := lipgloss.Width(l)
		if w > 40 {
			t.Errorf("line %d exceeded terminal width 40: %d columns (line: %q)", i, w, l)
		}
	}
}
