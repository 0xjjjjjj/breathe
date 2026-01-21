# breathe

> Let your disk breathe. A fast, interactive disk space analyzer and file organizer.

**breathe** helps you reclaim disk space by finding what's eating your storage and organizing messy folders like Downloads.

## Features

- **Interactive TUI** - Navigate your filesystem, see sizes at a glance, delete with `d`
- **Junk Detection** - Automatically finds `node_modules`, build artifacts, browser caches
- **Smart Organize** - Sort Downloads into Documents, Pictures, etc. with configurable rules
- **Safe by Default** - Moves to Trash (reversible), validates paths, tracks history
- **AI-Friendly** - JSON output mode for scripting or LLM integration

## Installation

```bash
# Clone and build
git clone https://github.com/0xjjjjjj/breathe.git
cd breathe
go build -o breathe ./cmd/breathe

# Or install directly
go install github.com/0xjjjjjj/breathe/cmd/breathe@latest
```

Requires Go 1.21+

## Quick Start

```bash
# Interactive space explorer
breathe scan ~/Downloads

# Quick top-level overview (fast for huge directories)
breathe scan ~ --top

# Find junk (node_modules, caches, build artifacts)
breathe scan ~/projects --json | jq '.junk'

# Organize Downloads folder (dry run first!)
breathe organize --dry-run
breathe organize --apply

# View operation history
breathe history

# Undo a move
breathe undo 42
```

## TUI Controls

| Key | Action |
|-----|--------|
| `j/k` or `↑/↓` | Navigate |
| `Enter` | Open directory |
| `h` or `Backspace` | Go back |
| `d` | Delete (moves to Trash) |
| `Space` | Select multiple items |
| `Tab` | Toggle Junk view |
| `q` | Quit |

## Configuration

Config file: `~/.config/breathe/config.yaml`

```yaml
# Junk patterns to detect
junk_patterns:
  - name: "node_modules"
    pattern: "**/node_modules"
    safe: true
  - name: "Python cache"
    pattern: "**/__pycache__"
    safe: true

# File organization rules
organize_rules:
  - match: "*.pdf"
    dest: "~/Documents"
  - match: "*.{jpg,png,gif}"
    dest: "~/Pictures"
  - match: "*"
    dest: "~/Downloads/Unsorted"
```

## Safety

- **Trash by default**: Deletions go to `~/.Trash`, not permanent delete
- **Protected paths**: Refuses to delete `/`, `/usr`, home directory, etc.
- **Operation history**: Every move/delete is logged to SQLite for undo
- **Dry run mode**: Preview changes before applying

## JSON Output

For scripting or AI integration:

```bash
# Get top space consumers
breathe scan ~/work --json | jq '.children | sort_by(-.size) | .[0:5]'

# Find all junk with sizes
breathe scan . --json | jq '.junk[] | {name, total: .total, count: (.paths | length)}'
```

## Development

```bash
# Run tests
go test ./...

# Build
go build -o breathe ./cmd/breathe
```

## License

MIT License - see [LICENSE](LICENSE)

## Contributing

Issues and PRs welcome! This is a young project - feedback appreciated.
