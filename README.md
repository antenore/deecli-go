# DeeCLI

![Test Suite](https://github.com/antenore/deecli-go/actions/workflows/test.yml/badge.svg)

A terminal-based AI assistant for code, built with Go and DeepSeek models.

## What it does

DeeCLI helps you work with code through chat and command-line interfaces. It loads files, analyzes code, and integrates with external editors.

## Quick start

## Prerequisites

- Go 1.18 or higher installed on your system.
- A valid DeepSeek API key.

```bash
# Build it
go build -o deecli main.go

# Set up your API key (you will be prompted to enter it)
./deecli config init

# Alternatively, you can set it via an environment variable
export DEEPSEEK_API_KEY='your-key-here'

# Start chatting
./deecli chat

# Or analyze code directly
./deecli analyze main.go
```

## Usage

### Chat Interface Navigation

**Smart Tab**: Context-aware behavior
- Show/accept file completions when available
- Switch focus between panes when no completions
- Focus cycle: Input → Chat History → Files Sidebar → Input

**Function Keys**:
- `F1` - Toggle help
- `F2` - Toggle files sidebar
- `F3` - Toggle code formatting (raw/bordered) for new messages

**Focus & Navigation**:
- `Esc` / `Enter` - Return to input mode from any pane
- `↑/↓` - Scroll in focused pane OR navigate history (single-line input only)
- `PgUp/PgDn` - Page up/down in viewports
- `Ctrl+U/D` - Half page up/down in viewports
- `Home/End` - Jump to top/bottom in viewports

### Text Editing Shortcuts

**Word Operations**:
- `Ctrl+W` - Delete word backward
- `Alt+Backspace` - Delete word backward (alternative)
- `Alt+D` - Delete word forward
- `Alt+F/B` - Move word forward/backward

**Line Operations**:
- `Ctrl+A` - Jump to line start
- `Ctrl+E` - Jump to line end
- `Ctrl+K` - Delete from cursor to line end
- `Ctrl+U` - Delete from cursor to line start

**Character Operations**:
- `Ctrl+F/B` - Move character forward/backward
- `Ctrl+H` - Delete character backward
- `Ctrl+D` - Delete character forward
- `Ctrl+T` - Transpose characters

**Text Transformation**:
- `Alt+C` - Capitalize word forward
- `Alt+L` - Lowercase word forward
- `Alt+U` - Uppercase word forward

**Multi-line Input**:
- Configurable newline key (default: `Ctrl+J`)
- `Enter` - Send message
- `Ctrl+V` - Paste

**History Navigation**:
- `↑/↓` - Navigate command history (single-line input only)
- `Ctrl+P/N` - Alternative history navigation keys

### Chat Commands

All commands start with `/` and support tab completion:

**File Management**:
- `/load <file>` - Load files additively (supports glob patterns like `*.go`, `**/*.py`)
- `/load --all <file>` - Load files ignoring .gitignore (includes node_modules, etc.)
- `/unload <pattern>` - Remove files matching pattern (supports wildcards)
- `/add <file>` - Same as `/load` (deprecated, kept for compatibility)
- `/reload` - Refresh files from disk
- `/edit <file>` - Open file in external editor
- `/edit <file:line>` - Jump to specific line in editor (e.g., `/edit main.go:42`)
- `/list` - Show loaded files
- `/clear` - Clear all context

**Smart File Loading**:
- Respects `.gitignore` by default (skips node_modules, build artifacts, etc.)
- Pattern validation with helpful error messages and suggestions
- Supports complex patterns: `src/**/*.go`, `{*.js,*.ts}`, etc.
- File size limits with clear feedback

**Session Management**:
- `/history` - Show command history
- `/help` - Show detailed help
- `/quit` - Exit application

**Configuration**:
- `/config show` - Display current settings
- `/config init` - Initialize configuration
- `/keysetup <key>` - Configure keyboard shortcuts

**AI Operations**:
- `/analyze` - Analyze loaded code
- Type any message to chat with the AI about your code

### Main Features

**Chat Interface**:
- Tab completion for files and commands
- Multi-line input support
- Scrollable chat history
- File sidebar with loaded files
- Terminal-friendly code output (raw by default for easy copying)
- Optional syntax highlighting and bordered code blocks

### File handling
- Load files with patterns: `*.go`, `**/*.go`, `{*.go,*.md}`
- Auto-reload after external edits
- Shows which files changed

### Commands
```
/load <file>     - Load files (respects .gitignore)
/load --all <file> - Load files ignoring .gitignore
/add <file>      - Add more files
/reload          - Refresh from disk
/edit <file>     - Open in editor
/edit <file:line> - Jump to specific line
/list            - Show loaded files
/clear           - Clear context
/config show     - Show settings
/help            - Show help
/quit            - Exit
```

### CLI commands
```
deecli chat              - Start interactive chat
deecli analyze <file>    - Analyze code
deecli improve <file>    - Get improvements
deecli explain <file>    - Explain code
deecli config <command>  - Manage settings
```

## Configuration

Settings are stored in `~/.deecli/config.yaml` or `./.deecli/config.yaml`. Environment variables take priority.

Here's the configuration persistence priority order (from lowest to highest priority - higher priority wins):

1. Default config (hardcoded defaults)
2. ~/.deecli/config.yaml (global/user config)
3. ./.deecli/config.yaml (project/local config)
4. Active profile (if set, from either global or project)
5. Environment variables (DEEPSEEK_API_KEY)

## How it's built

DeeCLI is built with **Go** and features a clean architecture with separate modules for core logic, commands, and the UI.

- **TUI Framework:** [Bubble Tea](https://github.com/charmbracelet/bubbletea) for the interactive chat interface
- **CLI Framework:** [Cobra](https://github.com/spf13/cobra) for the command-line commands
- **Storage:** SQLite for persisting chat sessions and context

## Development

### Testing

DeeCLI follows Go's standard testing practices:

```bash
# Run all tests
make test

# Run with coverage report (HTML)
make test-coverage

# Run unit tests only
make test-unit

# Run tests with race detection
make test-race

# See all test commands
make help
```

Tests are located alongside source code (`*_test.go` files) following Go conventions.

See [TESTING.md](TESTING.md) for testing documentation.

### Building

```bash
# Build for current platform
make build

# Build for all platforms
make build-all
```

## Current status

Alpha - core functionality works, focusing on stability and essential features.

## Future development & contributing

### AST Integration Roadmap
DeeCLI now uses text-based analysis for simplicity and performance. Full Abstract Syntax Tree (AST) integration is planned for future versions
when the core experience is stable.

**Why text-based first?**
- Faster startup and lower memory usage
- Works across all programming languages immediately
- More reliable with partial or malformed code
- Easier to maintain and debug

**Interested in AST development?**

Check `DEVELOPMENT.md` for technical details on the planned architecture. The codebase is structured to make AST integration straightforward when the
community is ready to support it.

### Contributing Guidelines

See CONTRIBUTING.md and remember, the focus is on delivering a stable, professional-grade tool first, then adding advanced features like AST analysis.

## License

Apache 2.0 - see LICENSE file

## Author

Antenore Gatta

---

*For detailed development info, see DEVELOPMENT.md. For current priorities, see TODO.md.*

