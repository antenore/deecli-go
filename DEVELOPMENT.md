# DEVELOPMENT.md

Stricly follow **Contributing guidelines**: CONTRIBUTING.md

## Architecture Overview

DeeCLI is built with a clean, modular architecture designed for maintainability and extensibility. The system follows a strict separation of concerns
with well-defined interfaces between components.

### Core Architecture Principles

1. **Single Responsibility**: Each module handles one specific concern
2. **Dependency Injection**: Configuration is injected, not hardcoded
3. **No Code Duplication**: Shared functionality is centralized
4. **Testability**: Components are designed for easy testing

## Module Structure

### `internal/api/` - DeepSeek API Integration

```go
// Core components:
- Client: Main API client with retry logic
- RequestBuilder: Constructs API requests with proper formatting
- ResponseHandler: Processes and validates API responses
- ErrorHandler: Manages API errors with exponential backoff
```

**Key Features:**
- Full conversation context support
- Streaming response capability (future)
- Comprehensive error handling with retries
- API key security and masking
- Token counting and context window management

### `internal/chat/` - TUI Implementation (Bubbletea)

```go
// Main components:
- Model: Central state management for the TUI
- CompletionEngine: Tab completion with file path cycling
- InputHandler: Multi-line input with Ctrl+Enter support
- ViewportManager: Scrollable chat history with unlimited content
- FocusManager: Pane switching with Ctrl+W
```

**Input System Details:**

- Uses Bubbletea's textarea component for natural input handling
- Ctrl+Enter inserts newlines (terminal limitation: can't detect Shift+Enter)
- Eliminated complex manual input parsing
- Professional-grade tab completion with cycling through options

### `internal/config/` - Configuration System

```go
// Configuration hierarchy (highest priority first):
1. Environment variables (DEEPSEEK_API_KEY, etc.)
2. Project config (./.deecli/config.yaml)
3. Global config (~/.deecli/config.yaml)
4. Default values

// Configurable options:
- API keys with secure storage
- Model selection (deepseek-coder, deepseek-chat, etc.)
- Temperature and max-tokens
- Editor preferences
- Session persistence settings
```

### `internal/files/` - File Management

```go
// Features:
- Glob pattern support: *.go, **/*.go, {*.go,*.md}
- Auto-reload after external edits
- Binary file detection and exclusion
- Change detection showing modified files
- Context tracking with file metadata

// File loading process:
1. Pattern expansion and validation
2. Binary file filtering
3. Content reading with proper encoding
4. Metadata collection (size, mod time)
5. Context integration with chat
```

### `internal/sessions/` - Session Persistence

```go
// SQLite-based session storage:
- Automatic session creation and updating
- Resume previous sessions with --continue flag
- Efficient storage with proper indexing
- Cleanup of old sessions

// Session structure:
- Conversation history with timestamps
- Loaded files and their context
- Configuration state
- API usage metrics
```

### `internal/editor/` - External Editor Integration
```go

// Supported editors:
- System default editor detection
- Configurable editor preference
- Line number support (file:42 syntax)
- Multiple file editing capability

// Integration process:
1. Editor detection and validation
2. Temporary file creation
3. Process execution and monitoring
4. File change detection and reload
```

## Key Technical Solutions

### Input System Architecture

The input system was completely refactored to use Bubbletea's built-in textarea component, which provides:
- Natural multi-line input handling
- Proper cursor navigation
- Built-in scrolling and viewport management
- Elimination of complex manual input parsing

**Why Ctrl+Enter for newlines:**
Terminals cannot distinguish between Enter and Shift+Enter, so we use Ctrl+Enter as the newline mechanism while Enter sends the message.

### Configuration System Design
The git-like configuration hierarchy ensures flexibility while maintaining security:
```yaml
# Environment variables take highest precedence
# Project config allows per-project settings
# Global config provides user defaults
# Built-in defaults ensure basic functionality
```

### File Context Management
Files are managed with smart change detection:
- Automatic reloading after external edits
- `/reload` command for manual synchronization
- Visual indicators of modified files
- Efficient content caching with validation

## Development Guidelines

### Code Style
- **Go formatting**: Always run `gofmt` or `goimports`
- **Naming conventions**: Clear, descriptive names
- **Error handling**: Proper error wrapping and context
- **Documentation**: Comprehensive godoc comments

### Testing Strategy
1. **Unit tests**: For isolated components
2. **Integration tests**: For module interactions
3. **Terminal testing**: Always test in real terminal environments
4. **SSH testing**: Verify functionality over remote connections

### Adding New Features
1. **Research existing patterns** before implementation
2. **Follow established architecture** and interfaces
3. **Consider both chat and CLI** implementations
4. **Test in real environments** before finalizing

### Performance Considerations
- **Memory usage**: Be mindful of large file handling
- **API efficiency**: Minimize unnecessary API calls
- **Response time**: Optimize for interactive use
- **Resource cleanup**: Properly close files and connections

## Build and Deployment

### Development Build
```bash
go build -o deecli main.go
```

### Production Build
```bash
# Use makefile for standardized builds
make build
```

### Cross-Platform Builds
```bash
make build-all  # Builds for all supported platforms
```

### Testing Commands
```bash
go test ./...           # Run all tests
go test -v ./internal/chat  # Verbose testing for specific module
go test -race ./...     # Race condition detection
```

## Troubleshooting Common Issues

### Terminal Compatibility
- Test over SSH connections
- Verify functionality in different terminal emulators
- Check for ANSI color support issues

### API Integration
- Validate API key configuration
- Check network connectivity
- Monitor rate limiting and quotas

### File System Issues
- Verify file permissions
- Check for path resolution problems
- Validate glob pattern matching

## Future Architecture Considerations

### Planned Enhancements
- **Streaming responses**: Real-time API response handling
- **Advanced caching**: Response and file content caching
- **Plugin system**: Extensible functionality
- **Advanced analytics**: Usage tracking and optimization

### Technical Debt Items
- **Viewport text selection**: Improve copy/paste functionality
- **Mouse support**: Enhanced interaction capabilities
- **Memory optimization**: Better large file handling
- **Concurrency**: Improved parallel processing

## AST Integration - Future Considerations

### Why Not Yet Implemented
Full Abstract Syntax Tree (AST) integration is a complex feature that would significantly increase:
- **Memory usage** (ASTs consume substantial memory for large codebases)
- **Language support complexity** (each language needs its own parser)
- **Maintenance burden** (keeping ASTs synchronized with file changes)
- **Edge case handling** (partial files, syntax errors, mixed languages)

### Architecture Preparedness
The current architecture is designed to support AST integration when ready:
- Clean module separation allows adding `internal/ast/` package
- File loading system provides content and language detection
- Configuration system can handle AST-specific settings
- Session persistence can store AST metadata

### For Contributors Interested in AST
If you want to work on AST integration:

1. **Start small** with Go-only AST support (Go's AST library is excellent)
2. **Follow the existing patterns** - create `internal/ast/` with clear interfaces
3. **Consider incremental approach**:
   - Phase 1: AST-based function navigation (`/ast functions`)
   - Phase 2: Simple refactoring (rename local variables)
   - Phase 3: Cross-file analysis
   - Phase 4: Multi-language support

4. **Key technical challenges to address**:
   - Memory management for large ASTs
   - Real-time AST synchronization
   - Error recovery for malformed code
   - Language parser consistency

### Example Starting Point
```go
// Simple AST exploration for Go files
func exploreGoAST(content string) {
    fset := token.NewFileSet()
    file, err := parser.ParseFile(fset, "", content, parser.ParseComments)
    if err != nil {
        return // Handle parse errors gracefully
    }

    // Basic AST traversal
    ast.Inspect(file, func(n ast.Node) bool {
        if fn, ok := n.(*ast.FuncDecl); ok {
            // Found a function declaration
            fmt.Printf("Function: %s at line %d\n",
                fn.Name.Name,
                fset.Position(fn.Pos()).Line)
        }
        return true
    })
}
```

### Testing Strategy

AST features should include:
- Unit tests for individual AST operations
- Integration tests with real code samples
- Performance testing for memory usage
- Cross-platform compatibility testing

---

*This document provides implementation details for developers. For user documentation, see README.md. For project roadmap, see TODO.md.*

