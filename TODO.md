# TODO.md

Focus: Keep It Simple (KISS) and SOLID. Ship a fast, reliable TUI first; advanced features are optional and opt‑in.

## P0 — Must Have (Daily Usability Blockers)
- ✅ Multi-round conversation context bug (FIXED)
  - Implemented simple working solution: all loaded files sent with every request
  - Ensures files remain accessible throughout conversation history
- ✅ Streaming reliability (FIXED)
  - Implemented meaningful content detection to prevent empty assistant messages
  - Enhanced message manager and viewport synchronization during streaming
  - Improved spinner timing with intelligent content analysis
  - Model‑aware timeouts and clear error handling already working properly
- ✅ Context window management (FIXED)
  - Implemented context size monitoring with color-coded warnings in UI
  - Added smart context truncation with helpful error messages when exceeded
  - Show token usage and approximate context size to user in header and sidebar
  - Enhanced file content handling with size limits and usage indicators
- ✅ File Loading UX Fixes (FIXED)
  - Made `/load` additive by default (no longer destructive)
  - Added `/unload <pattern>` for selective file removal with glob pattern support
  - Updated tab completion and help text to reflect new behavior
  - Deprecated `/add` command (kept for backward compatibility)
  - TODO: Respect `.gitignore` by default with `--all` flag to override
  - TODO: Smart pattern loading that excludes build artifacts (node_modules, target/, dist/, etc.)
- ✅ Terminal-Friendly Output Formatting (FIXED)
  - RAW code by default - instantly copyable without any borders
  - F3 toggles formatting for new messages (raw vs bordered)
  - Fixed /edit instruction file to strip box-drawing characters
  - Configuration options: syntax_highlight and code_block_style in config.yaml
  - KISS: Prioritized copying over pretty formatting
- ✅ Basic File Operations (COMPLETED)
  - ✅ `/edit <file>:line` support - fully working and documented
  - ✅ Better error messages for file operations with suggested fixes
  - ✅ Validate file patterns before attempting to load
  - ✅ Basic .gitignore support with `--all` flag to override
  - ✅ Refactored to use proven `go-gitignore` library (180+ lines → 15 lines)

## P1 — AI Function Calling System (THE BREAKTHROUGH FEATURE)
**Vision**: AI can autonomously execute commands with user approval, becoming a true coding partner

**Why P1**: This transforms DeeCLI from "chat interface" to "autonomous AI assistant" - major differentiator

### Phase 1: Infrastructure & Read-Only Operations (PARTIALLY COMPLETE)
- ✅ **DeepSeek Function Calling Integration**: Implement tools/function calling API support
- ✅ **Permission Framework**: Basic approve/deny system with project-level persistence
- ✅ **Read-Only Tool Functions**: `git_status()`, `git_diff()`, `list_files()`, `read_file()`
- ✅ **Approval UI**: Simple TUI prompts: `[Approve Once] [Always Approve] [Never]`

**CRITICAL ISSUE**: Context re-execution bug - AI re-runs previously executed tools on new requests
- Status: Under investigation
- Impact: Tools work but execute multiple times unnecessarily
- Priority: Must fix before Phase 2
- Analysis: See TOOL_EXECUTION_ANALYSIS.md for detailed technical analysis

### Phase 2: Enhanced Permission System (Medium Complexity)
- **Granular Permissions**: Per-command, per-project, command-category approvals
- **Safety Classifications**: Automatic read-only vs write operation detection
- **Permission Management**: `/permissions` command to view/edit approval settings
- **Audit Log**: Track all AI-initiated commands for security

### Phase 3: Write Operations (High Value, Needs Care)
- **Safe Write Functions**: `edit_file()`, `create_file()` with enhanced approval flows
- **Preview Mode**: Show exactly what AI wants to change before execution
- **Rollback Support**: Easy undo for AI-initiated file changes
- **Sandbox Mode**: Option to run AI commands in isolated environment

### Phase 4: Advanced Automation (Future Vision)
- **Workflow Templates**: Pre-approved command sequences for common tasks
- **Smart Context**: AI automatically loads relevant files based on conversation
- **Background Monitoring**: AI watches for file changes and offers relevant help

## P2 — Traditional Developer Workflow (When P1 Phase 1 is Stable)
- Session management: `/session save <name>`, `/session load <name>`
- Tab completion improvements respecting `.gitignore`
- Better file pattern completion (exclude build artifacts by default)
- Request IDs and simple retry/backoff; slow‑call logging
- Context optimization commands: `/compact`, `/summarize` for conversation cleanup

## P3 — Advanced Features (Complex/Optional)
- Config system refinements (env overrides, profiles) - developers don't need this daily
- AST/LSP assistance (opt‑in):
  - Start Go‑only (std `go/ast`) for outlines and targeted spans.
  - Optional LSP adapter (`gopls`, pyright, tsserver`) with graceful fallback.
  - Warning: adds dependencies, complexity, and indexing overhead; keep disabled by default.
- Response caching and cost tracking.
- Advanced configuration correctness (single source of truth, predictable behavior)

## Done Recently
- ✅ **AI Function Calling System INFRASTRUCTURE** - Complete foundation for autonomous AI assistant
  - Full DeepSeek API integration with streaming and non-streaming support
  - Tool registry with 4 read-only tools: git_status, git_diff, list_files, read_file
  - Complete approval system with project-scoped permissions
  - Professional TUI approval dialog with keyboard navigation
  - Empty argument handling and robust error management
  - **BUG FIX**: Double execution eliminated with proper queue validation
  - **ONGOING**: Context re-execution investigation (tools work but repeat unnecessarily)
- ✅ **Basic File Operations COMPLETE** - All P0 file operation improvements shipped
  - `/edit file:line` support with comprehensive testing and documentation
  - **BUG FIX**: `/edit file:line` now works correctly with AI instructions (was creating literal "file:line" files)
  - Enhanced error messages with actionable suggestions for all file operations
  - Pattern pre-validation to catch problematic patterns early with helpful guidance
  - .gitignore support by default with `--all` flag override, using battle-tested `go-gitignore` library
- ✅ **Terminal-Friendly Output Formatting** - RAW code by default for instant copying, F3 toggles formatting, KISS approach
- ✅ **Streaming reliability FIX** - Eliminated empty assistant messages and improved content detection
- ✅ **Context window management FIX** - Added comprehensive UI monitoring and smart truncation
- ✅ **Multi-round conversation context bug FIX** - Files now remain accessible throughout conversations

## Testing Priority
- Unit tests for file loading logic (gitignore, additive behavior)
- Integration tests for real developer workflows
- Keep Makefile targets green; avoid flaky tests.

---

**Developer Reality Check**: Core P0 stability achieved. Next major feature: AI Function Calling System.

User workflow becomes:
- User: "What changed recently?"
- AI: [requests git_status] → [approved] → "3 files modified. Main changes in parser.go..."

Focus: Build this system to be safe and reliable.

Last updated: 2025‑01‑27