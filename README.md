# DeeCLI

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

## Main features

### Chat interface
- Tab completion for files and commands
- Multi-line input (Ctrl+Enter for new lines)
- Scrollable history
- File sidebar (F2 to toggle)

### File handling
- Load files with patterns: `*.go`, `**/*.go`, `{*.go,*.md}`
- Auto-reload after external edits
- Shows which files changed

### Commands
```
/load <file>     - Load files
/add <file>      - Add more files
/reload          - Refresh from disk
/edit <file>     - Open in editor
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

```bash
# Run tests
go test ./...

# Build for all platforms
make build-all
```

## Current status

Alpha - core functionality works, focusing on stability and essential features.

## Future development & contributing

### AST Integration Roadmap
DeeCLI currently uses text-based analysis for simplicity and performance. Full Abstract Syntax Tree (AST) integration is planned for future versions
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

