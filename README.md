# Bubble Tea Components

> **This repository has been split into individual packages.** Each component now lives in its own repo for independent versioning and lighter dependency trees.

## Components

| Component | Repository | Description |
|-----------|-----------|-------------|
| **Base Picker** | [bubbletea-picker](https://github.com/blackwell-systems/bubbletea-picker) | Foundation for picker components — handles key bindings, window resizing, border rendering, and selection logic |
| **Multi-Select** | [bubbletea-multiselect](https://github.com/blackwell-systems/bubbletea-multiselect) | Generic multi-selection wrapper with checkbox UI and persistent selection state |
| **Miller Columns** | [bubbletea-millercolumns](https://github.com/blackwell-systems/bubbletea-millercolumns) | Hierarchical navigation layout inspired by macOS Finder |
| **Carousel** | [bubbletea-carousel](https://github.com/blackwell-systems/bubbletea-carousel) | Peeking single-row card layout with centered active card and adjacent previews |
| **Command Palette** | [bubbletea-commandpalette](https://github.com/blackwell-systems/bubbletea-commandpalette) | Fuzzy-search overlay for actions — VS Code Ctrl+P style |

## Migration

Replace your import:

```go
// Before
import "github.com/blackwell-systems/bubbletea-components/picker"

// After
import "github.com/blackwell-systems/bubbletea-picker"
```

Then run:

```bash
go get github.com/blackwell-systems/bubbletea-picker@v0.1.0
```

Repeat for each component you use.

## License

MIT License - see [LICENSE](LICENSE) for details.
