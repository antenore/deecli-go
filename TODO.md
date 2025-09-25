# TODO.md

Focus: Keep It Simple (KISS). Ship working tools that developers actually need.

## P0 — Core Stability (✅ DONE)
- ✅ Multi-round conversation context
- ✅ Streaming reliability  
- ✅ Context window management
- ✅ File loading UX (additive /load, /unload, .gitignore respect)
- ✅ Terminal-friendly output (raw code by default, F3 toggles)
- ✅ Basic file operations (/edit file:line)

## P1 — Essential Writing Tools (NEXT PRIORITY)
**Goal**: Let AI actually modify code, not just read it

### Phase 1: Basic Write Operations (Start Here)
- **write_file**: Create new files or overwrite existing
- **edit_file**: Apply text replacements to existing files  
- **append_file**: Add content to end of file
- Keep using simple approval system: [Once] [Always] [Never]

### Phase 2: Smart Code Tools  
- **grep_code**: Use ripgrep first, fallback to grep if not available
  - Pattern matching in codebase with context lines
  - Respect .gitignore by default
- **find_symbol**: Find function/class/variable definitions
  - Start with simple regex patterns
  - Later: optional Go AST for .go files (stdlib go/ast)
- **apply_patch**: Apply unified diff patches
  - Use system `patch` command
  - Show preview before applying

### Phase 3: Refactoring Tools
- **rename_symbol**: Rename across multiple files
- **extract_function**: Extract code into new function
- **move_file**: Move and update imports

## P2 — Developer Experience 
### Logging (Start Simple)
- Add structured logging with `slog` (Go 1.21+ stdlib)
  - Debug/Info/Warn/Error levels
  - Log to ~/.deecli/logs/ with rotation
  - Flag: --log-level=debug|info|warn|error
- Log tool executions for debugging (not "audit" - just troubleshooting)

### Session Management
- `/session save <name>` - save conversation
- `/session load <name>` - restore conversation  
- `/session list` - show saved sessions
- Store in SQLite (already have the dependency)

### Better Shell Integration
- **run_command**: Execute shell commands with approval
  - Show command before execution
  - Capture stdout/stderr
  - Working directory awareness

## P3 — Nice to Have (Later)
- Advanced permissions (per-tool, per-project)
- Cost tracking and response caching
- LSP integration (gopls, pyright) - optional, off by default
- Workflow templates for common tasks

## Anti-Patterns to Avoid
- ❌ Over-engineering permission systems before having write tools
- ❌ Building complex audit logs when simple debug logs would suffice
- ❌ Reimplementing grep/ripgrep/patch - use what exists
- ❌ Adding dependencies for problems we don't have yet

## Development Principles
- Use system tools first (ripgrep, grep, patch, diff)
- Implement fallbacks only when truly needed
- Keep dependencies minimal and proven
- Test with real developer workflows
- Ship incrementally - working features over perfect architecture

---

**Current Status**: Read-only tools work. Next: Add basic write tools so AI can actually help code.

Last updated: 2025-01-27