# TODO.md

Focus: Keep It Simple (KISS) and SOLID. Ship a fast, reliable TUI first; advanced features are optional and opt‑in.

## P0 — Must Have (Stability & Core UX)
- ✅ Multi-round conversation context bug (FIXED)
  - Implemented simple working solution: all loaded files sent with every request
  - Ensures files remain accessible throughout conversation history
  - See CONTEXT_MANAGEMENT_PLAN.md for research and future optimization strategy
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
- Config correctness (NEXT PRIORITY)
  - Single source of truth (env overrides, profiles) and predictable behavior.
- Tests & CI
  - Unit tests for streaming (chunking, completion), message sync, file watcher debounce.
  - Keep Makefile targets green; avoid flaky tests.

## P1 — High‑Impact UX (Small, Safe Wins)
- Output formatting improvements (code blocks, wrapping, copyability).
- Token/size feedback: show approximate context size and a gentle warning when near the cap.
- File operations: respect `.gitignore` for `/load` and completions; `/edit file:line` if the editor supports it.
- Basic Git integration: `/git diff`, `/git status` (read‑only).

## P2 — Roadmap (When Core Is Solid)
- Tools layer (THE REAL SOLUTION for context management)
  - Implement safe file reading tools (`/read`, `/search`, `/diff`, `/list`)
  - Let AI request file content on-demand instead of pre-loading everything
  - Preview/confirm flows for file operations
  - This solves context bloat and scales to large codebases
- Request IDs and simple retry/backoff; slow‑call logging.
- Session commands: `/session list|load|save`.
- Context optimization commands: `/compact`, `/summarize` for conversation cleanup.

## P3 — Nice to Dream About (Complex/Optional)
- AST/LSP assistance (opt‑in):
  - Start Go‑only (std `go/ast`) for outlines and targeted spans.
  - Optional LSP adapter (`gopls`, pyright, tsserver) with graceful fallback.
  - Warning: adds dependencies, complexity, and indexing overhead; keep disabled by default.
- Response caching and cost tracking.

## Done Recently
- ✅ **Streaming reliability FIX** - Eliminated empty assistant messages and improved content detection
- ✅ **Context window management FIX** - Added comprehensive UI monitoring and smart truncation
- ✅ **Multi-round conversation context bug FIX** - Files now remain accessible throughout conversations
- ✅ **Context management research** - Analyzed Claude Code, OpenAI Codex, DeepSeek approaches
- ✅ **Implementation plan** - Created CONTEXT_MANAGEMENT_PLAN.md with detailed future roadmap
- Visual spinner and "thinking" flow improvements.
- Auto‑reload robustness (rename events) and notices.
- Enhanced config validation and history/input handling.
- Partial architecture clean‑up (extracted managers; continue reducing large files).

Last updated: 2025‑01‑27
