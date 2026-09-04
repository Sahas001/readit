package tui

import "github.com/charmbracelet/bubbles/key"

// KeyMap defines the key bindings for the application.
type KeyMap struct {
	Up       key.Binding
	Down     key.Binding
	Enter    key.Binding
	Back     key.Binding
	Quit     key.Binding
	Tab      key.Binding
	Upvote   key.Binding
	Downvote key.Binding
	NewPost  key.Binding
	Reply     key.Binding
	ReplyRoot key.Binding
	Submit    key.Binding
	Help      key.Binding
}

// DefaultKeyMap returns the default key bindings.
func DefaultKeyMap() KeyMap {
	return KeyMap{
		Up: key.NewBinding(
			key.WithKeys("k", "up"),
			key.WithHelp("↑/k", "up"),
		),
		Down: key.NewBinding(
			key.WithKeys("j", "down"),
			key.WithHelp("↓/j", "down"),
		),
		Enter: key.NewBinding(
			key.WithKeys("enter"),
			key.WithHelp("enter", "select"),
		),
		Back: key.NewBinding(
			key.WithKeys("esc", "backspace"),
			key.WithHelp("esc", "back"),
		),
		Quit: key.NewBinding(
			key.WithKeys("q", "ctrl+c"),
			key.WithHelp("q", "quit"),
		),
		Tab: key.NewBinding(
			key.WithKeys("tab"),
			key.WithHelp("tab", "switch sort/field"),
		),
		Upvote: key.NewBinding(
			key.WithKeys("u"),
			key.WithHelp("u", "upvote"),
		),
		Downvote: key.NewBinding(
			key.WithKeys("d"),
			key.WithHelp("d", "downvote"),
		),
		NewPost: key.NewBinding(
			key.WithKeys("n"),
			key.WithHelp("n", "new post"),
		),
		Reply: key.NewBinding(
			key.WithKeys("r"),
			key.WithHelp("r", "reply"),
		),
		ReplyRoot: key.NewBinding(
			key.WithKeys("R"),
			key.WithHelp("R", "reply to post"),
		),
		Submit: key.NewBinding(
			key.WithKeys("ctrl+s"),
			key.WithHelp("ctrl+s", "submit"),
		),
		Help: key.NewBinding(
			key.WithKeys("?"),
			key.WithHelp("?", "help"),
		),
	}
}
