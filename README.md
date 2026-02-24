# Bubble Tea Components

Production-ready reusable components for [Bubble Tea](https://github.com/charmbracelet/bubbletea) applications.

## Components

### Base Picker
Foundation for building picker components. Handles standard key bindings, window resizing, border rendering, and selection logic. Reduces boilerplate by 60-70%.

[Documentation →](picker/README.md)

### Multi-Select
Generic multi-selection wrapper that works with any `list.Item`. Adds checkbox UI with persistent state across view changes.

[Documentation →](multiselect/README.md)

### Miller Columns
Hierarchical navigation layout inspired by macOS Finder. Display multiple directory levels side-by-side for visual context.

[Documentation →](millercolumns/README.md)

## Installation

```bash
go get github.com/blackwell-systems/bubbletea-components
```

## Usage

```go
import (
    "github.com/blackwell-systems/bubbletea-components/picker"
    "github.com/blackwell-systems/bubbletea-components/multiselect"
    "github.com/blackwell-systems/bubbletea-components/millercolumns"
)
```

## Dependencies

- `github.com/charmbracelet/bubbles` - List and input components
- `github.com/charmbracelet/bubbletea` - TUI framework
- `github.com/charmbracelet/lipgloss` - Styling

## Production Use

These components are used in [shelfctl](https://github.com/blackwell-systems/shelfctl), a personal library manager that organizes PDFs using GitHub Release assets.

## Contributing

Contributions welcome! Please open an issue before starting work on major changes.

## License

MIT License - see [LICENSE](LICENSE) for details.
