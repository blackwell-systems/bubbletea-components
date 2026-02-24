// Package commandpalette provides a fuzzy-search command palette overlay for
// the Bubble Tea framework.
//
// The palette renders as a self-contained bordered box: a text input at the
// top, a live-filtered list of actions below, and a keyboard hint footer.
// Actions are matched by label and optional keywords using fuzzy search.
//
// The caller is responsible for opening/closing the palette and handling
// esc and ctrl+c before forwarding key messages to Update.
//
// Typical usage:
//
//	palette := commandpalette.New(commandpalette.Config{
//	    Actions: []commandpalette.Action{
//	        {Label: "Edit Book",   Run: doEditBook},
//	        {Label: "Delete Book", Keywords: []string{"remove"}, Run: doDelete},
//	        {Label: "Sync",        Run: doSync},
//	    },
//	})
//	palette.SetSize(width, height)
//
// In Update:
//
//	case tea.KeyMsg:
//	    if msg.String() == "ctrl+p" {
//	        paletteOpen = true
//	        return m, palette.Focus()
//	    }
//	    if paletteOpen {
//	        switch msg.String() {
//	        case "esc", "ctrl+c":
//	            paletteOpen = false
//	            return m, nil
//	        }
//	        palette, cmd = palette.Update(msg)
//	        return m, cmd
//	    }
//
//	case commandpalette.ActionSelectedMsg:
//	    paletteOpen = false
//	    return m, msg.Action.Run
package commandpalette

import (
	"strings"

	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/sahilm/fuzzy"
)

const (
	defaultMaxResults  = 10
	defaultMaxWidth    = 60
	defaultPlaceholder = "Search actions…"
	promptStr          = "> "
)

// Action represents a single executable command in the palette.
type Action struct {
	// Label is the displayed name of the action and the primary search target.
	Label string

	// Keywords are additional searchable terms that are not displayed.
	// For example, an "Edit Book" action might have Keywords: []string{"modify", "update"}.
	Keywords []string

	// Run is the command executed when this action is selected.
	// It is passed directly to the parent model via ActionSelectedMsg and may be nil.
	Run tea.Cmd
}

// ActionSelectedMsg is emitted when the user confirms a selection by pressing
// enter. The parent model should close the palette and call msg.Action.Run.
type ActionSelectedMsg struct {
	Action Action
}

// Config holds the configuration for constructing a new Model.
type Config struct {
	// Actions is the initial set of actions. May be replaced later via SetActions.
	Actions []Action

	// Placeholder is shown in the text input when it is empty.
	// Defaults to "Search actions…".
	Placeholder string

	// MaxResults is the maximum number of filtered results shown at once.
	// Defaults to 10.
	MaxResults int

	// MaxWidth is the maximum column width of the palette overlay, including
	// the border. Defaults to 60.
	MaxWidth int

	// ActiveColor is the highlight color used for the prompt, selected result
	// row, and border. Defaults to lipgloss.Color("#fb6820") (orange).
	ActiveColor lipgloss.Color
}

// Model is the command palette state. Create it with New and pass it through
// Update and View using value semantics, matching the Bubble Tea convention.
type Model struct {
	actions     []Action
	filtered    []Action
	cursor      int
	input       textinput.Model
	maxResults  int
	maxWidth    int
	activeColor lipgloss.Color
	width       int
	height      int
}

// New creates a new Model from cfg. Missing fields receive sensible defaults.
// Call SetSize before the first View call.
func New(cfg Config) Model {
	if cfg.MaxResults == 0 {
		cfg.MaxResults = defaultMaxResults
	}
	if cfg.MaxWidth == 0 {
		cfg.MaxWidth = defaultMaxWidth
	}
	if cfg.ActiveColor == "" {
		cfg.ActiveColor = lipgloss.Color("#fb6820")
	}
	placeholder := cfg.Placeholder
	if placeholder == "" {
		placeholder = defaultPlaceholder
	}

	inp := textinput.New()
	inp.Placeholder = placeholder
	inp.Prompt = promptStr
	inp.PromptStyle = lipgloss.NewStyle().Foreground(cfg.ActiveColor)
	// Width is the text field width, not including the prompt.
	// Calculated as: maxWidth - border(2) - padding(2) - prompt(2).
	inp.Width = cfg.MaxWidth - 6

	actions := cfg.Actions
	if actions == nil {
		actions = []Action{}
	}

	return Model{
		actions:     actions,
		filtered:    actions, // show all when query is empty
		maxResults:  cfg.MaxResults,
		maxWidth:    cfg.MaxWidth,
		activeColor: cfg.ActiveColor,
		input:       inp,
	}
}

// Focus focuses the text input and returns textinput.Blink so the cursor
// appears immediately. Call this each time the palette is opened.
func (m *Model) Focus() tea.Cmd {
	return m.input.Focus()
}

// Reset clears the query, restores the full action list, and resets the cursor
// to 0. Call this before Focus when re-opening the palette so it starts fresh.
func (m *Model) Reset() {
	m.input.Reset()
	m.filtered = m.actions
	m.cursor = 0
}

// SetSize sets the available terminal dimensions used for width clamping.
// Call on init and on every tea.WindowSizeMsg received by the parent model.
func (m *Model) SetSize(width, height int) {
	m.width = width
	m.height = height
	// Recalculate input field width to match the clamped overlay width.
	w := min(m.maxWidth, width)
	if textW := w - 6; textW > 0 {
		m.input.Width = textW
	}
}

// SetActions replaces the full action list and re-applies the current filter.
// The cursor is clamped to the new result count.
func (m *Model) SetActions(actions []Action) {
	if actions == nil {
		actions = []Action{}
	}
	m.actions = actions
	m.applyFilter(m.input.Value())
	limit := min(len(m.filtered), m.maxResults)
	if m.cursor >= limit {
		m.cursor = max(0, limit-1)
	}
}

// Update processes a key message and returns an updated Model and optional Cmd.
//
// Keys handled by the component:
//   - Any printable character / backspace — update input, re-filter results
//   - up / k — move cursor up through visible results
//   - down / j — move cursor down through visible results
//   - enter — emit ActionSelectedMsg for the highlighted action
//
// The caller is responsible for handling esc and ctrl+c before forwarding
// key messages here.
func (m Model) Update(msg tea.KeyMsg) (Model, tea.Cmd) {
	visibleCount := min(len(m.filtered), m.maxResults)

	switch msg.String() {
	case "up", "k":
		if m.cursor > 0 {
			m.cursor--
		}
		return m, nil

	case "down", "j":
		if m.cursor < visibleCount-1 {
			m.cursor++
		}
		return m, nil

	case "enter":
		if visibleCount == 0 {
			return m, nil
		}
		action := m.filtered[m.cursor]
		return m, func() tea.Msg {
			return ActionSelectedMsg{Action: action}
		}
	}

	// Forward all other keys (printable characters, backspace, etc.) to the
	// text input and re-filter if the query changed.
	prevQuery := m.input.Value()
	var cmd tea.Cmd
	m.input, cmd = m.input.Update(msg)

	if q := m.input.Value(); q != prevQuery {
		m.applyFilter(q)
		m.cursor = 0
	}

	return m, cmd
}

// View renders the command palette as a self-contained bordered overlay.
//
// The returned string does not assume a position on screen; the caller
// decides where to display it (typically replacing the current view or
// using lipgloss.Place to center it over existing content).
func (m Model) View() string {
	w := m.maxWidth
	if m.width > 0 && m.width < w {
		w = m.width
	}
	if w < 20 {
		w = 20
	}
	// innerW is the content width: outer width minus border (2) minus padding (2).
	innerW := w - 4

	dimStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("240"))
	activeStyle := lipgloss.NewStyle().Foreground(m.activeColor).Bold(true)
	labelStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("255"))
	divider := dimStyle.Render(strings.Repeat("─", innerW))

	var b strings.Builder

	// ── Input ─────────────────────────────────────────────────────────────────
	b.WriteString(m.input.View())
	b.WriteString("\n")
	b.WriteString(divider)
	b.WriteString("\n")

	// ── Filtered results ──────────────────────────────────────────────────────
	results := m.filtered
	if len(results) > m.maxResults {
		results = results[:m.maxResults]
	}

	if len(results) == 0 {
		b.WriteString(dimStyle.Render("  no matching actions"))
		b.WriteString("\n")
	} else {
		for i, action := range results {
			label := truncate(action.Label, innerW-2) // -2 for the "▶ " or "  " prefix
			if i == m.cursor {
				b.WriteString(activeStyle.Render("▶ " + label))
			} else {
				b.WriteString(labelStyle.Render("  " + label))
			}
			b.WriteString("\n")
		}
	}

	// ── Footer ────────────────────────────────────────────────────────────────
	b.WriteString(divider)
	b.WriteString("\n")
	footer := strings.Join([]string{
		dimStyle.Render("↑↓ Navigate"),
		dimStyle.Render("Enter Execute"),
		dimStyle.Render("Esc Dismiss"),
	}, dimStyle.Render("  •  "))
	b.WriteString(footer)

	return lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(m.activeColor).
		Padding(0, 1).
		Width(w).
		Render(b.String())
}

// Query returns the current text input value.
func (m Model) Query() string {
	return m.input.Value()
}

// Results returns the current filtered action slice (capped at MaxResults).
func (m Model) Results() []Action {
	if len(m.filtered) > m.maxResults {
		return m.filtered[:m.maxResults]
	}
	return m.filtered
}

// applyFilter updates m.filtered using fuzzy matching against each action's
// label and keywords. Results are sorted by fuzzy score (best match first).
// When the query is empty or all whitespace, all actions are shown unfiltered.
func (m *Model) applyFilter(query string) {
	if strings.TrimSpace(query) == "" {
		m.filtered = m.actions
		return
	}

	// Build a search string per action: "Label keyword1 keyword2 …"
	// Fuzzy matches against this string; only Label is displayed.
	sources := make([]string, len(m.actions))
	for i, a := range m.actions {
		sources[i] = a.Label
		if len(a.Keywords) > 0 {
			sources[i] += " " + strings.Join(a.Keywords, " ")
		}
	}

	matches := fuzzy.Find(query, sources)
	m.filtered = make([]Action, len(matches))
	for i, match := range matches {
		m.filtered[i] = m.actions[match.Index]
	}
}

// truncate clips s to at most n visible rune positions, appending "…" if
// truncated. Action labels are plain text so rune count is sufficient.
func truncate(s string, n int) string {
	runes := []rune(s)
	if len(runes) <= n {
		return s
	}
	if n <= 1 {
		return "…"
	}
	return string(runes[:n-1]) + "…"
}
