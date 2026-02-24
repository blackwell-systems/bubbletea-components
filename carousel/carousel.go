// Package carousel provides a peeking single-row carousel TUI component for
// the Bubble Tea framework.
//
// The active card is rendered full-width in the center. Adjacent cards peek in
// from the sides, showing only their near edge. Ghost cards are rendered at the
// extreme edges when no real card exists in that direction.
//
// Usage:
//
//	type MyDelegate struct{}
//
//	func (d MyDelegate) Render(item any, innerW int) string {
//	    m := item.(MyItem)
//	    return truncate(m.Title, innerW, "…")
//	}
//
//	func (d MyDelegate) IsMarked(item any) bool {
//	    return item.(MyItem).Done
//	}
//
//	items := []any{item1, item2, item3}
//	c := carousel.New(carousel.Config{
//	    Items:    items,
//	    Delegate: MyDelegate{},
//	    Title:    "Select an item",
//	})
//	c.SetSize(width, height)
package carousel

import (
	"fmt"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	xansi "github.com/charmbracelet/x/ansi"
)

// Layout constants control the card sizing and spacing.
const (
	minPeekW    = 8  // minimum visible columns for each side peek
	carouselGap = 2  // blank columns between the peek card and center card
	maxCenterW  = 44 // cap center card width for a realistic aspect ratio
)

// ItemDelegate handles item-specific rendering for carousel cards.
// Implement this interface to provide card interior content and mark state
// for your item type.
type ItemDelegate interface {
	// Render returns the card interior content for item. innerW is the number
	// of visible columns available for text (accounting for card padding).
	// The returned string may contain newlines but should not end with one.
	Render(item any, innerW int) string

	// IsMarked reports whether item has been marked (e.g., saved or completed).
	// Marked items receive the MarkedColor border instead of DefaultColor.
	IsMarked(item any) bool
}

// ItemSelectedMsg is emitted when the user selects the active card by pressing
// enter, down, j, or space. Handle this message in the parent model to act on
// the selection.
type ItemSelectedMsg struct {
	Index int // 0-based index of the selected item in Items
	Item  any // value of the selected item
}

// Config holds the configuration for constructing a new Model.
type Config struct {
	// Items is the initial list of items to display. May be nil.
	Items []any

	// Delegate provides item-specific rendering and mark state. Required.
	Delegate ItemDelegate

	// Title is displayed in the carousel header.
	Title string

	// ActiveColor is the border color for the focused center card.
	// Defaults to lipgloss.Color("#fb6820") (orange) if empty.
	ActiveColor lipgloss.Color

	// MarkedColor is the border color for marked (saved/completed) inactive cards.
	// Defaults to lipgloss.Color("28") (green) if empty.
	MarkedColor lipgloss.Color

	// DefaultColor is the border color for unvisited, unmarked inactive cards.
	// Defaults to lipgloss.Color("240") (gray) if empty.
	DefaultColor lipgloss.Color

	// ExtraFooter is appended to the navigation hint footer, separated by " • ".
	// Use this to surface caller-specific keyboard hints (e.g., "a Bulk edit").
	ExtraFooter string
}

// Model is the carousel state. Create it with New and pass it through Update
// and View using value semantics, matching the Bubble Tea convention.
type Model struct {
	items        []any
	delegate     ItemDelegate
	title        string
	activeColor  lipgloss.Color
	markedColor  lipgloss.Color
	defaultColor lipgloss.Color
	extraFooter  string
	cursor       int
	width        int
	height       int
}

// New creates a new Model from cfg. Missing color fields receive sensible
// defaults. Call SetSize before the first View call.
func New(cfg Config) Model {
	activeColor := cfg.ActiveColor
	if activeColor == "" {
		activeColor = lipgloss.Color("#fb6820")
	}
	markedColor := cfg.MarkedColor
	if markedColor == "" {
		markedColor = lipgloss.Color("28")
	}
	defaultColor := cfg.DefaultColor
	if defaultColor == "" {
		defaultColor = lipgloss.Color("240")
	}
	items := cfg.Items
	if items == nil {
		items = []any{}
	}
	return Model{
		items:        items,
		delegate:     cfg.Delegate,
		title:        cfg.Title,
		activeColor:  activeColor,
		markedColor:  markedColor,
		defaultColor: defaultColor,
		extraFooter:  cfg.ExtraFooter,
	}
}

// Update processes a key message and returns an updated Model and an optional Cmd.
//
// Keys handled by the component:
//   - left / h  — move cursor left (no-op at first card)
//   - right / l — move cursor right (no-op at last card)
//   - enter / down / j / space — emit ItemSelectedMsg for the active card
//
// The caller is responsible for handling esc, ctrl+c, and any
// application-specific keys before forwarding key messages here.
func (m Model) Update(msg tea.KeyMsg) (Model, tea.Cmd) {
	n := len(m.items)
	if n == 0 {
		return m, nil
	}

	switch msg.String() {
	case "left", "h":
		if m.cursor > 0 {
			m.cursor--
		}

	case "right", "l":
		if m.cursor < n-1 {
			m.cursor++
		}

	case "down", "j", "enter", " ":
		idx := m.cursor
		item := m.items[idx]
		return m, func() tea.Msg {
			return ItemSelectedMsg{Index: idx, Item: item}
		}
	}

	return m, nil
}

// View renders the carousel to a string ready for display.
//
// The output consists of three sections stacked vertically:
//   - Header: title, marked/total count, dot position indicator
//   - Card row: left peek (or ghost), center card, right peek (or ghost)
//   - Footer: navigation hints
//
// The returned string does not include an outer container or border; the
// caller is responsible for any wrapping (padding, border, etc.) to match
// the surrounding application style.
func (m Model) View() string {
	n := len(m.items)
	if n == 0 {
		return ""
	}

	// ── Layout math ───────────────────────────────────────────────────────────
	//
	// usable is the width available inside the caller's container. The -6 accounts
	// for the surrounding StyleBorder (1 col each side) plus outerPad (2 cols each
	// side) that the caller typically applies.
	usable := m.width - 6
	centerW := usable - 2*(minPeekW+carouselGap)
	if centerW > maxCenterW {
		centerW = maxCenterW
	}
	if centerW < 24 {
		centerW = 24
	}

	// Give leftover space to the peek slots, but cap at half the card width so
	// adjacent cards never reveal more than half of themselves.
	peekW := (usable - centerW - 2*carouselGap) / 2
	if maxPeek := (centerW + 2) / 2; peekW > maxPeek {
		peekW = maxPeek
	}
	if peekW < minPeekW {
		peekW = minPeekW
	}

	// Derive card height from width to approximate a 3:5 library card ratio.
	// Terminal cells are ~2:1 pixel aspect, so:
	//   rows = (totalCardW) × 3 / 10
	totalCardW := centerW + 2 // outer width including border columns
	cardContentH := totalCardW * 3 / 10
	if cardContentH < 8 {
		cardContentH = 8
	}

	marked := m.MarkedCount()
	cur := m.cursor

	// ── Dot position indicator ────────────────────────────────────────────────
	dots := make([]string, n)
	for i := range dots {
		if i == cur {
			dots[i] = lipgloss.NewStyle().Foreground(m.activeColor).Render("●")
		} else {
			dots[i] = lipgloss.NewStyle().Foreground(m.defaultColor).Render("○")
		}
	}
	indicator := strings.Join(dots, " ")

	// ── Header ────────────────────────────────────────────────────────────────
	headerStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("255")).Bold(true)
	helpStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("240"))

	var header strings.Builder
	header.WriteString(headerStyle.Render(m.title))
	header.WriteString("  ")
	header.WriteString(helpStyle.Render(fmt.Sprintf("%d/%d saved", marked, n)))
	header.WriteString("\n")
	header.WriteString(indicator)
	header.WriteString("\n\n")

	// ── Card row ──────────────────────────────────────────────────────────────
	centerCard := m.renderCard(cur, centerW, cardContentH, true)
	gapBlock := strings.Repeat(" ", carouselGap)

	var leftPeek, rightPeek string
	if cur > 0 {
		rendered := m.renderCard(cur-1, centerW, cardContentH, false)
		leftPeek = peekRight(rendered, peekW)
	} else {
		leftPeek = peekRight(ghostCard(centerW, cardContentH), peekW)
	}
	if cur < n-1 {
		rendered := m.renderCard(cur+1, centerW, cardContentH, false)
		rightPeek = peekLeft(rendered, peekW)
	} else {
		rightPeek = peekLeft(ghostCard(centerW, cardContentH), peekW)
	}

	row := lipgloss.JoinHorizontal(lipgloss.Top,
		leftPeek, gapBlock, centerCard, gapBlock, rightPeek,
	)

	// ── Footer ────────────────────────────────────────────────────────────────
	var navHint string
	switch cur {
	case 0:
		navHint = "→ Navigate"
	case n - 1:
		navHint = "← Navigate"
	default:
		navHint = "←→ Navigate"
	}
	footer := m.renderFooter(navHint)

	// ── Assemble ──────────────────────────────────────────────────────────────
	var b strings.Builder
	b.WriteString(header.String())
	b.WriteString(row)
	b.WriteString("\n\n")
	b.WriteString(footer)

	return b.String()
}

// Cursor returns the current cursor position (0-based index into Items).
func (m Model) Cursor() int {
	return m.cursor
}

// SetCursor sets the cursor to idx. The value is clamped to the valid range
// [0, len(Items)-1]. No-op if Items is empty.
func (m *Model) SetCursor(idx int) {
	if idx < 0 {
		idx = 0
	}
	if n := len(m.items); n > 0 && idx >= n {
		idx = n - 1
	}
	m.cursor = idx
}

// SetSize sets the available terminal dimensions used for layout calculation.
// This should be called once on init and again whenever a tea.WindowSizeMsg
// is received by the parent model.
func (m *Model) SetSize(width, height int) {
	m.width = width
	m.height = height
}

// Items returns the current item slice.
func (m Model) Items() []any {
	return m.items
}

// SetItems replaces the item slice. If the current cursor would be out of
// range for the new slice, it is clamped to the last valid index.
func (m *Model) SetItems(items []any) {
	if items == nil {
		items = []any{}
	}
	m.items = items
	if n := len(items); n > 0 && m.cursor >= n {
		m.cursor = n - 1
	}
}

// MarkedCount returns the number of items for which Delegate.IsMarked is true.
func (m Model) MarkedCount() int {
	count := 0
	for _, item := range m.items {
		if m.delegate.IsMarked(item) {
			count++
		}
	}
	return count
}

// renderCard renders a single carousel card at position i.
// active=true applies the active (center) color; otherwise the border color
// reflects the item's marked/default state via the delegate.
func (m Model) renderCard(i, cardW, cardH int, active bool) string {
	item := m.items[i]
	inner := cardW - 2 // subtract card padding (1 col each side)
	content := m.delegate.Render(item, inner)

	var style lipgloss.Style
	switch {
	case active:
		style = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(m.activeColor).
			Width(cardW).Height(cardH).Padding(1, 1)
	case m.delegate.IsMarked(item):
		style = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(m.markedColor).
			Foreground(lipgloss.Color("242")).
			Width(cardW).Height(cardH).Padding(1, 1)
	default:
		style = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(m.defaultColor).
			Foreground(lipgloss.Color("242")).
			Width(cardW).Height(cardH).Padding(1, 1)
	}

	return style.Render(content)
}

// renderFooter renders the keyboard hint bar. navHint reflects the current
// cursor position (e.g., "←→ Navigate"). Extra caller-specified hints from
// Config.ExtraFooter are appended.
func (m Model) renderFooter(navHint string) string {
	dimStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("240"))
	sep := dimStyle.Render(" • ")

	parts := []string{
		dimStyle.Render(navHint),
		dimStyle.Render("Enter/↓ Select"),
		dimStyle.Render("Esc Back"),
	}
	if m.extraFooter != "" {
		parts = append(parts, dimStyle.Render(m.extraFooter))
	}

	return lipgloss.NewStyle().Padding(0, 1).Render(strings.Join(parts, sep))
}

// peekLeft clips a rendered multi-line block to its leftmost n visible columns.
// Used to show the left-edge peek of the card to the right of center.
func peekLeft(s string, n int) string {
	lines := strings.Split(s, "\n")
	for i, line := range lines {
		lines[i] = xansi.Truncate(line, n, "")
	}
	return strings.Join(lines, "\n")
}

// peekRight clips a rendered multi-line block to its rightmost n visible columns.
// Used to show the right-edge peek of the card to the left of center.
func peekRight(s string, n int) string {
	lines := strings.Split(s, "\n")
	for i, line := range lines {
		w := xansi.StringWidth(line)
		if w > n {
			lines[i] = xansi.TruncateLeft(line, w-n, "")
		}
	}
	return strings.Join(lines, "\n")
}

// ghostCard renders a blank placeholder card with a very dim border.
// It is shown at the edges of the carousel to maintain visual rhythm when
// no real adjacent card exists in that direction.
func ghostCard(cardW, cardH int) string {
	return lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(lipgloss.Color("235")).
		Width(cardW).Height(cardH).
		Render("")
}
