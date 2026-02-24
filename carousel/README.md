# Carousel

A peeking single-row card carousel for [Bubble Tea](https://github.com/charmbracelet/bubbletea). The active card occupies the center at full width; adjacent cards peek in from both sides to give the user a sense of what comes next.

![Carousel demo](carousel_demo.gif)

## Features

- **Peeking layout**: Adjacent cards peek from both sides so users always see context
- **Ghost cards**: Placeholder cards render at the edges when no real neighbor exists, maintaining visual rhythm
- **Delegate-driven rendering**: Implement one interface to control card content and mark state for any item type
- **Marked state**: Cards change border color when `IsMarked` returns true (useful for saved, completed, or visited items)
- **Dot position indicator**: A row of dots below the title tracks position across all items
- **Customizable colors**: Override active, marked, and default border colors at construction time
- **Caller-controlled keys**: `esc`, `ctrl+c`, and app-specific keys stay in the parent — only navigation and selection keys are handled internally
- **Extra footer hints**: Append caller-specific keyboard hints to the built-in navigation footer

## Installation

```bash
go get github.com/blackwell-systems/bubbletea-components/carousel
```

## Usage

### 1. Implement the ItemDelegate Interface

`ItemDelegate` controls what appears inside each card and whether a card is considered marked. The carousel calls `Render` for every visible card and `IsMarked` to choose the card's border color.

```go
type BookDelegate struct{}

func (d BookDelegate) Render(item any, innerW int) string {
    b := item.(Book)
    // innerW is the number of columns available inside the card border.
    // Truncate all text to fit; the carousel does not wrap content for you.
    title := truncate(b.Title, innerW)
    author := truncate("by "+b.Author, innerW)
    return title + "\n" + author
}

func (d BookDelegate) IsMarked(item any) bool {
    return item.(Book).Saved
}

func truncate(s string, w int) string {
    if len(s) <= w {
        return s
    }
    if w <= 1 {
        return "…"
    }
    return s[:w-1] + "…"
}
```

`innerW` is computed as `cardW - 2` inside the component, where `cardW` is the calculated center card width. Text that exceeds `innerW` will overflow the card border visually, so delegates must truncate.

### 2. Create the Model

```go
items := []any{book1, book2, book3}

c := carousel.New(carousel.Config{
    Items:       items,
    Delegate:    BookDelegate{},
    Title:       "My Reading List",
    ExtraFooter: "a  Add to shelf",
})
c.SetSize(width, height)
```

### 3. Wire Into a Parent Model

The parent model is responsible for handling `esc`, `ctrl+c`, and any application-specific keys before forwarding key messages to the carousel. `Update` takes a `tea.KeyMsg` directly, not a `tea.Msg`.

```go
package main

import (
    "fmt"

    "github.com/blackwell-systems/bubbletea-components/carousel"
    tea "github.com/charmbracelet/bubbletea"
)

type Book struct {
    Title  string
    Author string
    Saved  bool
}

type BookDelegate struct{}

func (d BookDelegate) Render(item any, innerW int) string {
    b := item.(Book)
    title := truncate(b.Title, innerW)
    author := truncate("by "+b.Author, innerW)
    return title + "\n" + author
}

func (d BookDelegate) IsMarked(item any) bool {
    return item.(Book).Saved
}

type model struct {
    carousel carousel.Model
    chosen   *Book
    quitting bool
}

func (m model) Init() tea.Cmd {
    return nil
}

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
    switch msg := msg.(type) {

    case tea.WindowSizeMsg:
        m.carousel.SetSize(msg.Width, msg.Height)
        return m, nil

    case carousel.ItemSelectedMsg:
        b := msg.Item.(Book)
        m.chosen = &b
        m.quitting = true
        return m, tea.Quit

    case tea.KeyMsg:
        // Handle exit keys before forwarding to the carousel.
        switch msg.String() {
        case "ctrl+c", "esc":
            m.quitting = true
            return m, tea.Quit
        }
        // Forward all other key messages to the carousel.
        var cmd tea.Cmd
        m.carousel, cmd = m.carousel.Update(msg)
        return m, cmd
    }

    return m, nil
}

func (m model) View() string {
    if m.quitting {
        if m.chosen != nil {
            return fmt.Sprintf("Selected: %s\n", m.chosen.Title)
        }
        return "Cancelled.\n"
    }
    // View() returns inner content with no outer border.
    // Wrap it with padding, a border, or any other container here.
    return m.carousel.View()
}

func main() {
    items := []any{
        Book{Title: "The Go Programming Language", Author: "Donovan & Kernighan"},
        Book{Title: "Clean Code", Author: "Robert C. Martin", Saved: true},
        Book{Title: "Designing Data-Intensive Applications", Author: "Martin Kleppmann"},
    }

    c := carousel.New(carousel.Config{
        Items:    items,
        Delegate: BookDelegate{},
        Title:    "Book Picker",
    })

    // SetSize must be called before the first render. Use a reasonable
    // default here; the WindowSizeMsg handler will correct it at startup.
    c.SetSize(120, 40)

    p := tea.NewProgram(model{carousel: c}, tea.WithAltScreen())
    if _, err := p.Run(); err != nil {
        panic(err)
    }
}

func truncate(s string, w int) string {
    if len(s) <= w {
        return s
    }
    if w <= 1 {
        return "…"
    }
    return s[:w-1] + "…"
}
```

## API Reference

### `Config`

```go
type Config struct {
    Items        []any          // Initial item slice. May be nil.
    Delegate     ItemDelegate   // Required. Provides card content and mark state.
    Title        string         // Displayed in the carousel header.
    ActiveColor  lipgloss.Color // Center card border. Default: #fb6820 (orange).
    MarkedColor  lipgloss.Color // Marked inactive card border. Default: "28" (green).
    DefaultColor lipgloss.Color // Unvisited inactive card border. Default: "240" (gray).
    ExtraFooter  string         // Appended to footer hints, separated by " • ".
}
```

### `ItemDelegate`

```go
type ItemDelegate interface {
    // Render returns the card body for item. innerW is the available column
    // count inside the card border. Text must be truncated to fit.
    Render(item any, innerW int) string

    // IsMarked reports whether item should receive the MarkedColor border.
    IsMarked(item any) bool
}
```

### `ItemSelectedMsg`

Emitted when the user presses `enter`, `down`, `j`, or `space` on the active card. Handle this in the parent model.

```go
type ItemSelectedMsg struct {
    Index int // 0-based index in Items
    Item  any // value of the selected item
}
```

### `New`

```go
func New(cfg Config) Model
```

Creates a new Model. Missing color fields receive sensible defaults. Call `SetSize` before the first `View` call.

### `Model` Methods

| Method | Signature | Description |
|---|---|---|
| `Update` | `(msg tea.KeyMsg) (Model, tea.Cmd)` | Process a key message. Returns updated model and optional command. |
| `View` | `() string` | Render the carousel. Returns inner content with no outer container. |
| `SetSize` | `(width, height int)` | Set terminal dimensions. Call on init and on every `tea.WindowSizeMsg`. |
| `SetItems` | `(items []any)` | Replace the item slice. Clamps cursor to new bounds. |
| `SetCursor` | `(idx int)` | Move cursor to idx. Clamped to `[0, len(Items)-1]`. |
| `Items` | `() []any` | Return the current item slice. |
| `Cursor` | `() int` | Return the current cursor position (0-based). |
| `MarkedCount` | `() int` | Return the count of items for which `Delegate.IsMarked` is true. |

### Key Bindings

| Key | Action |
|---|---|
| `left` / `h` | Move cursor left (no-op at first card) |
| `right` / `l` | Move cursor right (no-op at last card) |
| `enter` / `down` / `j` / `space` | Emit `ItemSelectedMsg` for the active card |

The component does **not** handle `esc`, `ctrl+c`, or any application-specific keys. The parent model handles those before forwarding to `Update`.

## Design Notes

### Why the caller handles esc

The carousel is a component embedded inside a larger application, not a standalone program. How to respond to `esc` — navigate back, open a confirm dialog, quit entirely — depends entirely on the parent's state machine. Handling `esc` inside the component would force that decision on every consumer. Instead, the parent intercepts `esc` (and `ctrl+c`) before the `tea.KeyMsg` reaches `carousel.Update`, keeping the component's contract simple.

### What ExtraFooter is for

The built-in footer always shows navigation and selection hints. `ExtraFooter` lets the parent surface its own keyboard hints in the same style without subclassing or re-rendering the footer. For example, passing `"a  Bulk edit"` appends that hint to the footer row alongside the built-in hints.

### View() has no outer container

`View()` returns the header, card row, and footer — no surrounding border or padding. This lets callers apply whatever outer container (lipgloss border, padding, full-screen layout) fits the surrounding application without fighting the component's own styling.

## License

See project root for license information.
