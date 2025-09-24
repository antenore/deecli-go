# Next Session Notes

This file captures what matters so we can resume fast.

## Current Setup
- Preferred model: `deepseek-chat` (works smoother right now)
- Alternative: `deepseek-reasoner` (latency/regressions since 2025‑09‑22). Use small context and longer timeouts.
- Suggested max_tokens: 1024–2048 for responsiveness
- Context cap: uses `max_context_size` from config (default ~100 KB)

## Changes Landed Today
- Streaming
  - Threshold now respects config `max_context_size`
  - Spinner stays until meaningful content; assistant message added only with real text
  - Message manager stays in sync during streaming and on completion
- Timeouts
  - Reasoner gets 300s timeouts (HTTP + call)
- Debug
  - `DEECLI_DEBUG=1` enables size/token logs (to stderr-friendly code paths only)

## Known Issues / Watchlist
- Reasoner model sometimes delays first tokens for several seconds
- Large contexts still cost time/money; trim files when possible
- We still concatenate file context into the user message; consider splitting into a dedicated context message

## Next Tasks (Ordered)
1) Context-as-message
   - Send file context as a separate message (system or labeled) before user input
2) Token feedback (lightweight)
   - Show approx context chars/tokens in header or after `/list`
3) Tools layer (phase 1)
   - Safe `fs.list`, `fs.read`, `fs.search`, `git.diff`, `fs.patch` (preview + confirm)
4) Git/.gitignore niceties
   - Respect `.gitignore` on `/load` and tab completion

## How to Verify Quickly
- Build: `make build` (or `go build -o deecli main.go`)
- Run chat: `./deecli chat`
- Small context test: load 1–2 files, ask a question, confirm streaming starts quickly
- Debug (optional): `DEECLI_DEBUG=1 ./deecli analyze README.md` for non‑TUI logs

## Notes
- Keep KISS/SOLID. Prefer small, focused changes. Advanced AST/LSP stays optional and off by default.
