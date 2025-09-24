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

## P1 — High‑Impact Developer Workflow (Small, Safe Wins)
- Git integration: `/git diff`, `/git status` (read‑only commands)
- Session management: `/session save <name>`, `/session load <name>`
- Tab completion improvements respecting `.gitignore`
- Better file pattern completion (exclude build artifacts by default)

## P2 — Roadmap (When Core Is Solid)
- Tools layer (THE REAL SOLUTION for context management)
  - Implement safe file reading tools (`/read`, `/search`, `/diff`, `/list`)
  - Let AI request file content on-demand instead of pre-loading everything
  - Preview/confirm flows for file operations
  - This solves context bloat and scales to large codebases
- Request IDs and simple retry/backoff; slow‑call logging.
- Context optimization commands: `/compact`, `/summarize` for conversation cleanup.

## P3 — Nice to Dream About (Complex/Optional)
- Config system refinements (env overrides, profiles) - developers don't need this daily
- AST/LSP assistance (opt‑in):
  - Start Go‑only (std `go/ast`) for outlines and targeted spans.
  - Optional LSP adapter (`gopls`, pyright, tsserver`) with graceful fallback.
  - Warning: adds dependencies, complexity, and indexing overhead; keep disabled by default.
- Response caching and cost tracking.
- Advanced configuration correctness (single source of truth, predictable behavior)

## Done Recently
- ✅ **Basic File Operations COMPLETE** - All P0 file operation improvements shipped
  - `/edit file:line` support with comprehensive testing and documentation
  - Enhanced error messages with actionable suggestions for all file operations
  - Pattern pre-validation to catch problematic patterns early with helpful guidance
  - .gitignore support by default with `--all` flag override, using battle-tested `go-gitignore` library
- ✅ **Terminal-Friendly Output Formatting** - RAW code by default for instant copying, F3 toggles formatting, KISS approach
- ✅ **Streaming reliability FIX** - Eliminated empty assistant messages and improved content detection
- ✅ **Context window management FIX** - Added comprehensive UI monitoring and smart truncation
- ✅ **Multi-round conversation context bug FIX** - Files now remain accessible throughout conversations
- ✅ **Context management research** - Analyzed approaches, implemented working solution
- Visual spinner and "thinking" flow improvements.
- Auto‑reload robustness (rename events) and notices.
- Enhanced config validation and history/input handling.
- Partial architecture clean‑up (extracted managers; continue reducing large files).

## Testing Priority
- Unit tests for file loading logic (gitignore, additive behavior)
- Integration tests for real developer workflows
- Keep Makefile targets green; avoid flaky tests.

---

**Developer Reality Check**: Focus on daily pain points that make developers choose other tools. The goal is making DeeCLI faster and more convenient than "Claude Web + copy/paste" for code work.

Last updated: 2025‑01‑27