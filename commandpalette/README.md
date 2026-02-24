# Command Palette

A fuzzy-search command palette overlay for [Bubble Tea](https://github.com/charmbracelet/bubbletea). Type to filter a flat list of actions, press Enter to execute. VS Code `Ctrl+P` style — one keystroke from anywhere in the app.

## Features

- **Fuzzy search** using [`sahilm/fuzzy`](https://github.com/sahilm/fuzzy) — character-sequence matching, same as VS Code
- **Keywords** — attach hidden searchable terms to actions so "del" matches "Edit Book" if you add "delete" as a keyword
- **Live filtering** — results update on every keystroke, cursor resets to top
- **Caller-controlled open/close** — the component is just a view; you decide when to show it and handle `esc`/`ctrl+c` before forwarding keys
- **Configurable** — max width, max results, accent color
- **`Reset()`** — clear query and cursor between openings

## Installation

```bash
go get github.com/blackwell-systems/bubbletea-components/commandpalette
```

## Usage

### 1. Define your actions

```go
import "github.com/blackwell-systems/bubbletea-components/commandpalette"

actions := []commandpalette.Action{
    {
        Label: "Edit Book",
        Run:   func() tea.Msg { return EditBookMsg{} },
    },
    {
        Label:    "Delete Book",
        Keywords: []string{"remove"},
        Run:      func() tea.Msg { return DeleteBookMsg{} },
    },
    {
        Label: "Sync to GitHub",
        Keywords: []string{"upload", "push"},
        Run:   func() tea.Msg { return SyncMsg{} },
    },
    {
        Label: "Browse Library",
        Run:   func() tea.Msg { return BrowseMsg{} },
    },
}
```

`Run` is a `tea.Cmd` (`func() tea.Msg`). It is passed back to you via `ActionSelectedMsg` — you call it yourself, so you can do any pre- or post-work around execution.

`Keywords` extend the search surface without cluttering the displayed label. "del" will match "Delete Book" by label, and "remove" will also match it via keywords.

### 2. Create the model

```go
type model struct {
    palette     commandpalette.Model
    paletteOpen bool
    // ... rest of your app state
}

func newModel() model {
    p := commandpalette.New(commandpalette.Config{
        Actions:     actions,
        Placeholder: "Search actions…",
        MaxResults:  8,
        MaxWidth:    56,
        ActiveColor: lipgloss.Color("#fb6820"),
    })
    return model{palette: p}
}
```

### 3. Wire into Update

Intercept the open key and `esc`/`ctrl+c` yourself. Forward everything else to the palette while it is open.

```go
func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
    switch msg := msg.(type) {

    case tea.WindowSizeMsg:
        m.palette.SetSize(msg.Width, msg.Height)
        // ... size your other views

    case tea.KeyMsg:
        // Open the palette
        if msg.String() == "ctrl+p" && !m.paletteOpen {
            m.paletteOpen = true
            m.palette.Reset()
            return m, m.palette.Focus()
        }

        // Handle palette keys while open
        if m.paletteOpen {
            switch msg.String() {
            case "esc", "ctrl+c":
                m.paletteOpen = false
                return m, nil
            }
            var cmd tea.Cmd
            m.palette, cmd = m.palette.Update(msg)
            return m, cmd
        }

        // Normal app key handling when palette is closed...

    case commandpalette.ActionSelectedMsg:
        m.paletteOpen = false
        return m, msg.Action.Run
    }

    return m, nil
}
```

### 4. Render

The palette returns a self-contained bordered box. The simplest approach is to replace the current view entirely while the palette is open. For a true overlay, use `lipgloss.Place` to center it over your existing view.

```go
func (m model) View() string {
    if m.paletteOpen {
        // Simple: replace current view with the palette
        return m.palette.View()
    }
    return m.normalView()
}
```

**True overlay with `lipgloss.Place`:**

```go
func (m model) View() string {
    background := m.normalView()
    if !m.paletteOpen {
        return background
    }
    return lipgloss.Place(
        m.width, m.height,
        lipgloss.Center, lipgloss.Center,
        m.palette.View(),
        lipgloss.WithWhitespaceChars(" "),
    )
}
```

## Complete Example

```go
package main

import (
    "fmt"

    "github.com/blackwell-systems/bubbletea-components/commandpalette"
    tea "github.com/charmbracelet/bubbletea"
    "github.com/charmbracelet/lipgloss"
)

type executeMsg struct{ label string }

type model struct {
    palette     commandpalette.Model
    paletteOpen bool
    lastAction  string
    width       int
    height      int
}

func newModel() model {
    p := commandpalette.New(commandpalette.Config{
        Actions: []commandpalette.Action{
            {
                Label:    "Open File",
                Keywords: []string{"load", "read"},
                Run:      func() tea.Msg { return executeMsg{"Open File"} },
            },
            {
                Label: "Save",
                Run:   func() tea.Msg { return executeMsg{"Save"} },
            },
            {
                Label:    "Delete",
                Keywords: []string{"remove", "trash"},
                Run:      func() tea.Msg { return executeMsg{"Delete"} },
            },
            {
                Label: "Settings",
                Run:   func() tea.Msg { return executeMsg{"Settings"} },
            },
        },
    })
    return model{palette: p}
}

func (m model) Init() tea.Cmd { return nil }

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
    switch msg := msg.(type) {

    case tea.WindowSizeMsg:
        m.width = msg.Width
        m.height = msg.Height
        m.palette.SetSize(msg.Width, msg.Height)

    case tea.KeyMsg:
        if msg.String() == "ctrl+c" && !m.paletteOpen {
            return m, tea.Quit
        }
        if msg.String() == "ctrl+p" && !m.paletteOpen {
            m.paletteOpen = true
            m.palette.Reset()
            return m, m.palette.Focus()
        }
        if m.paletteOpen {
            switch msg.String() {
            case "esc":
                m.paletteOpen = false
                return m, nil
            case "ctrl+c":
                return m, tea.Quit
            }
            var cmd tea.Cmd
            m.palette, cmd = m.palette.Update(msg)
            return m, cmd
        }

    case commandpalette.ActionSelectedMsg:
        m.paletteOpen = false
        return m, msg.Action.Run

    case executeMsg:
        m.lastAction = msg.label
    }

    return m, nil
}

func (m model) View() string {
    hint := lipgloss.NewStyle().
        Foreground(lipgloss.Color("240")).
        Render("Press ctrl+p to open the command palette  •  ctrl+c to quit")
    last := ""
    if m.lastAction != "" {
        last = fmt.Sprintf("\n\nLast executed: %s", m.lastAction)
    }
    background := hint + last

    if m.paletteOpen {
        return m.palette.View()
    }
    return background
}

func main() {
    p := tea.NewProgram(newModel(), tea.WithAltScreen())
    if _, err := p.Run(); err != nil {
        panic(err)
    }
}
```

## API Reference

### `Action`

```go
type Action struct {
    Label    string   // displayed name; primary fuzzy match target
    Keywords []string // hidden additional search terms
    Run      tea.Cmd  // command to execute on selection; may be nil
}
```

### `ActionSelectedMsg`

```go
type ActionSelectedMsg struct {
    Action Action
}
```

Emitted on `enter`. The parent closes the palette and calls `msg.Action.Run`.

### `Config`

| Field | Type | Default | Description |
|-------|------|---------|-------------|
| `Actions` | `[]Action` | `nil` | Initial action list |
| `Placeholder` | `string` | `"Search actions…"` | Input placeholder text |
| `MaxResults` | `int` | `10` | Max visible results |
| `MaxWidth` | `int` | `60` | Max overlay width (columns) |
| `ActiveColor` | `lipgloss.Color` | `"#fb6820"` | Highlight color |

### `Model` methods

| Method | Description |
|--------|-------------|
| `New(cfg Config) Model` | Create a new palette model |
| `Focus() tea.Cmd` | Focus the input; returns `textinput.Blink` |
| `Reset()` | Clear query and cursor; call before `Focus` when reopening |
| `SetSize(w, h int)` | Set terminal dimensions; call on `tea.WindowSizeMsg` |
| `SetActions([]Action)` | Replace actions and re-filter with current query |
| `Update(tea.KeyMsg) (Model, tea.Cmd)` | Handle a key message |
| `View() string` | Render the palette overlay |
| `Query() string` | Current input value |
| `Results() []Action` | Current filtered results (capped at MaxResults) |

### Keys handled by the component

| Key | Action |
|-----|--------|
| Any printable / backspace | Update input, re-filter |
| `↑` / `k` | Move cursor up |
| `↓` / `j` | Move cursor down |
| `enter` | Emit `ActionSelectedMsg` |

`esc` and `ctrl+c` are **not** handled by the component — intercept them in the parent before forwarding.

## Design Notes

**Why does the caller handle `esc`?**
The same reason as the carousel: the component has no knowledge of what closing means for the parent (return to previous view, reset state, etc.). Keeping `esc` in the caller avoids coupling.

**Why `ActionSelectedMsg` instead of calling `Run` directly?**
The parent gets a chance to do work around execution — close the palette, log the action, update breadcrumbs, or ignore `Run` entirely and handle the action differently. This is more flexible than the component calling `Run` itself.

**Fuzzy matching against keywords**
The search string used for matching is `Label + " " + Keywords joined`. Only `Label` is displayed. This means typing "rem" can match "Delete" if its keywords include "remove", without "remove" cluttering the displayed label.

**`MaxResults` as a hard cap, not a scroll window**
Results beyond `MaxResults` are not shown or navigable. With fuzzy sorting (best match first), narrow your query to surface lower-ranked results. This keeps the component simple — no scroll offset state to manage.

## License

See project root for license information.
