# Terminal UI Architecture

ReadIT uses [Charmbracelet's Bubble Tea](https://github.com/charmbracelet/bubbletea) framework, built on the Elm Architecture (Model-View-Update), coupled with [Lipgloss](https://github.com/charmbracelet/lipgloss) for layout styling and [Bubbles](https://github.com/charmbracelet/bubbles) for reusable components.

---

## The Elm Architecture in Bubble Tea

Bubble Tea structures interactive terminal applications around three core concepts:

```
         ┌────────────────────────┐
         │       tea.Model        │
         │  (Immutable app state) │
         └───────────┬────────────┘
                     │
          Updates    │ Renders
             ▼       ▼
    ┌──────────┐   ┌──────────┐
    │ Update() │   │  View()  │
    └────┬─────┘   └──────────┘
         │
         │ Returns tea.Cmd (I/O)
         ▼
    ┌──────────┐
    │  tea.Msg │ ──(dispatched back to Update)
    └──────────┘
```

1. **Model (`tea.Model`)**: State container holding current view, cursor position, database data, window dimensions, and sub-components.
2. **Update (`Update(msg tea.Msg) (tea.Model, tea.Cmd)`)**: Pure function receiving events (keypresses, window resizing, asynchronous database results) and returning updated state and next command.
3. **View (`View() string`)**: Pure renderer transforming current state into an ANSI-formatted string representing the terminal screen.
4. **Commands (`tea.Cmd`) & Messages (`tea.Msg`)**: Asynchronous side effects (like database queries) run outside the render loop and publish typed messages back to `Update`.

---

## State Machine & Navigation

The application uses an explicit state machine governed by `viewState`:

```
                 ┌───────────────┐
                 │  viewLoading  │
                 └───────┬───────┘
                         │ User lookup
            ┌────────────┴────────────┐
       New Key                   Known Key
            ▼                         ▼
   ┌─────────────────┐       ┌─────────────────┐
   │ viewOnboarding  │──────▶│  viewBoardList  │◀────┐
   │ (choose handle) │ Done  │ (list of boards)│     │ Esc
   └─────────────────┘       └────────┬────────┘     │
                                      │ Enter        │
                                      ▼              │
                             ┌─────────────────┐     │
                             │  viewPostList   │─────┘
                             │  (posts list)   │
                             └────────┬────────┘
                                      │ Enter
                                      ▼
                             ┌─────────────────┐
                             │ viewPostDetail  │
                             │(post & comments)│
                             └─────────────────┘
```

### View States Defined

* `viewLoading`: Initial state while querying PostgreSQL for the user's public key fingerprint.
* `viewOnboarding`: Displayed if the public key is not registered. Prompts for a handle.
* `viewBoardList`: Forum directory displaying available boards (`/b/general`, `/b/ask`, etc.).
* `viewPostList`: Listing of posts for the selected board with score, comments, and author.
* `viewPostDetail`: Post body and threaded recursive comments (next phase).
* `viewNewPost`: Form to submit a post using `bubbles/textarea` (next phase).
* `viewError`: Fallback rendering for recoverable and fatal application errors.

---

## Component Layout & Responsibilities

The TUI code lives in `internal/tui/`:

### `model.go`
* Declares `Model`, `viewState`, and message types (`userLoadedMsg`, `boardsLoadedMsg`, `postsLoadedMsg`, `errMsg`).
* Implements `tea.Model` methods: `Init()`, `Update()`, `View()`.
* Dispatches sub-updates cleanly per view state (`updateOnboarding`, `updateBoardList`, `updatePostList`).
* Houses async commands returning `tea.Cmd` (`lookupUserCmd`, `loadBoardsCmd`, `loadPostsCmd`, `createUserCmd`).

### `views.go`
* Pure rendering functions returning terminal string buffers:
  * `viewLoading()`
  * `viewOnboarding()`
  * `viewBoardList()`
  * `viewPostList()`
  * `viewError()`
* Utility `centeredView(content string)` utilizing `lipgloss.Place` to center dialogue views horizontally and vertically.

### `keymap.go`
* Centralized key binding declarations using `github.com/charmbracelet/bubbles/key`.
* Supplies idiomatic Vim keys (`j`/`k`, `enter`, `esc`, `q`, `u`/`d` for votes, `n` for new post).
* Exposes declarative help menus for status bars.

### `styles.go`
* Houses Lipgloss color definitions and reusable style blocks:
  * Reddit orange theme (`#FF4500`).
  * Selected item markers with thick left borders.
  * Score upvote colors (`#FF8B60`) and downvote colors (`#7193FF`).
  * Status bars and typography hierarchy.

---

## PTY & Terminal Resize Handling

Terminal sessions must gracefully adapt to arbitrary window resizing.

1. **`tea.WindowSizeMsg` Handling**:
   ```go
   case tea.WindowSizeMsg:
       m.width = msg.Width
       m.height = msg.Height
       return m, nil
   ```
2. **Alternate Screen Buffer**:
   When launching via Wish, `tea.WithAltScreen()` is passed in `MakeOptions(sess)`. This ensures that when the user quits or disconnects, their original terminal scrollback is completely preserved without forum artifacts.
3. **Dynamic Clamping**:
   Horizontal dividers and status bars dynamically clamp width:
   ```go
   strings.Repeat("─", min(m.width, 60))
   ```

---

## Best Practices for Adding New Views

When implementing a new view (e.g. `viewPostDetail`):

1. Add state constant to `viewState` enum in `model.go`.
2. Define domain message types for required asynchronous data (e.g., `commentsLoadedMsg`).
3. Add command constructors generating `tea.Cmd` that run SQL queries asynchronously.
4. Implement `update<ViewName>(msg tea.Msg) (*Model, tea.Cmd)` in `model.go`.
5. Implement `view<ViewName>() string` in `views.go`.
6. Add key bindings to `keymap.go`.
