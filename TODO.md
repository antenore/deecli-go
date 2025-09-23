# TODO.md

Focus: Keep It Simple (KISS) and SOLID. Ship a fast, reliable TUI first; advanced features are optional and opt‑in.

## P0 — Must Have (Stability & Core UX)
- Multi-round conversation context bug (HIGH PRIORITY)
  - Context accumulates incorrectly: loading file A → ask about A → load B → ask about B results in answer for both A and B
  - Each new file load adds to context instead of replacing it (A → A+B → A+B+C)
  - Fix: ensure /load replaces file context rather than appending; maintain proper context isolation per query
- Streaming reliability
  - Maintain spinner until meaningful content arrives; never show empty assistant messages.
  - Keep message manager and viewport in sync during streaming and on completion.
  - Model‑aware timeouts (reasoner slower than chat); clear cancellation/errors.
- Context window management
  - Enforce formatted context cap from config; stream under the cap, fall back with a helpful message when exceeded.
  - Trim/truncate large file content when needed; show context summary (size, files) in errors.
  - Optionally separate “files context” into its own message to keep user prompt clean.
- Config correctness
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
- Tools layer (safe file/list/search/diff/patch/test) with preview/confirm flows.
- Request IDs and simple retry/backoff; slow‑call logging.
- Session commands: `/session list|load|save`.

## P3 — Nice to Dream About (Complex/Optional)
- AST/LSP assistance (opt‑in):
  - Start Go‑only (std `go/ast`) for outlines and targeted spans.
  - Optional LSP adapter (`gopls`, pyright, tsserver) with graceful fallback.
  - Warning: adds dependencies, complexity, and indexing overhead; keep disabled by default.
- Response caching and cost tracking.

## Done Recently
- Visual spinner and “thinking” flow improvements.
- Auto‑reload robustness (rename events) and notices.
- Enhanced config validation and history/input handling.
- Partial architecture clean‑up (extracted managers; continue reducing large files).

Last updated: 2025‑09‑23
