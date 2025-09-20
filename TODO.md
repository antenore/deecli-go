# TODO.md

## 🎯 Current Focus: Stability & Core UX

*These items are critical for a reliable, professional-grade user experience and should be addressed first.*

### P0: Critical Bugs & Must-Haves
- [x] **Bug: `/edit` command lacks context awareness.** FIXED: `/edit` now analyzes recent conversation context to identify the file being discussed, with interactive fallback for multiple files
- [x] **Testing Infrastructure**: COMPLETED - Enhanced testing workflow with:
  - [x] **Professional Makefile**: Comprehensive test commands (`make test-coverage`, `make test-bench`, `make test-race`)
  - [x] **Coverage Reporting**: HTML coverage reports with 88.4% tracker, 60.7% input, 24.6% API coverage
  - [x] **GitHub Actions CI**: Automated testing pipeline with coverage uploads
  - [x] **Documentation**: Complete TESTING.md guide for developers
- [x] **Visual Thinking Indicator**: COMPLETED - Added animated spinner with multiple styles (Braille dots, line, bounce, circle) that displays during AI processing
- [x] **Assistant Identity**: VERIFIED - Already consistently named "DeeCLI" across the UI, commands, and prompts
- [x] **File Change Detection**: FIXED - Auto-reload now works correctly with all editors (handles RENAME events from editor saves)
- [x] **Enhanced Config Validation**: COMPLETED - Model whitelists, key bindings, and all config fields now validated with helpful error messages

### P1: High-Impact User Experience
- [x] **User Name Configuration**: COMPLETED - Users can now set a display name during setup and via /config command, replacing the generic "You:" in chat
- [ ] **Bug: Thinking Spinner Duration**: PARTIAL FIX - Spinner now shows longer but still stops before content appears. There's a gap of several seconds between spinner stopping and text appearing. Need to investigate why empty chunks trigger spinner stop.
- [ ] **Output Formatting**: Ensure code blocks in responses are well-formatted and easy to copy.
- [ ] **Token Usage Display**: Show the current token count in the interface to help users manage context.
- [ ] **Token Estimation & Warnings**: Warn users before they hit API token limits.
- [ ] **Multi-platform Support**: Solidify support for Linux, Windows, and macOS. (Currently Linux-focused).

---

## 🚀 Near-Term Roadmap (P2)

*These features will significantly enhance functionality once the core is stable.*

### Reliability & Performance
- [ ] **Circuit Breaker Pattern**: Implement robust handling for API failures to prevent cascading errors.
- [ ] **API Health Checks**: Add periodic connection health monitoring.
- [ ] **Rate Limiting**: Implement detection and backoff handling for API rate limits.
- [ ] **Memory Optimization**: Prune conversation history after a large number of messages (~1000) to prevent memory bloat.
- [ ] **Context Window Management**: Smartly manage the context window to prioritize recent and relevant information.

### API Optimization & Cost Control
- [ ] **Connection Optimization**: Verify and improve connection pooling/warming for faster API calls.
- [ ] **Response Caching**: Cache repeated AI requests to reduce costs and latency.
- [ ] **Cost Tracking**: Enhance `/balance` and `/cost` commands for a detailed API usage and cost breakdown.
- [ ] **Request ID Tracking**: Add unique IDs to API requests for easier debugging.
- [ ] **Performance Metrics**: Track and log slow API calls (e.g., >2s) for monitoring.

### Enhanced File & Workflow Integration
- [ ] **`/edit file:42`**: Open a file at a specific line number if the user's editor supports it.
- [ ] **Respect `.gitignore`**: Filter file operations (load, add, explore) based on `.gitignore` rules.
- [ ] **Git Integration (Basic)**: Implement `/git status` and `/git diff` to show version control info in the chat.
- [ ] **Git-aware File Completion**: Prioritize files tracked by git in autocompletion.

### Session & Configuration
- [ ] **Session Management**: Implement `/session list`, `/session load`, `/session save` commands.
- [ ] **File Discovery**: Create an `/explore` command with interactive permission prompts for navigating directories.

---

## 🔮 Future Considerations (P3 / Backlog)

*Valuable features that depend on a stable core or are more complex to implement. Re-evaluate after P0/P1 are complete.*

- [ ] **JSON Mode**: Enable structured JSON output for the `/analyze` command.
- [ ] **Stop Sequences**: Implement control sequences to better manage code generation boundaries.
- [ ] **FIM Completions**: Add fill-in-the-middle code completion for the `/edit` command.
- [ ] **Request Compression**: Investigate compressing requests to the API for better performance.
- [ ] **Tunnel/Proxy Support**: Add support for network proxies for users in restricted environments.
- [ ] **Mouse Support**: Allow pane switching with the mouse.
- [ ] **Advanced Session Management**: Features like exporting or deleting sessions.
- [ ] **Patch File Generation**: Allow the AI to generate changes in `.patch` format.
- [ ] **Multi-file Dependency Analysis**: Basic analysis of relationships between files.

---

## ⏳ Deferred / Icebox

*These are explicitly deferred. They are often complex, pose security risks, or are outside the core mission of a code analysis assistant. Revisit only after the above categories are mature.*

- [ ] **Bang Commands** (`! ls -la`): Shell command execution.
- [ ] **`/run` Command**: Code execution.
- [ ] **`/test` Command**: Running test suites.
- [ ] **Markdown Rendering**: Full syntax highlighting and rich rendering of markdown.
- [ ] **Complex Session Export Formats**: Exporting to non-standard formats.
- [ ] **Encrypted Storage**: For config files (adds significant complexity; environment variables are a good alternative for secrets).
- [ ] **Memory Clearing for API Keys**: (The OS manages process memory; this is often an over-optimization for a local CLI tool).

---

## ✅ Recently Completed

*Keep this section to celebrate progress and provide context.* (Keep at max 6 entries)
- **User Name Configuration** - Custom display names in chat, configurable during setup or via /config
- **Enhanced Config Validation** - All config fields validated with helpful error messages
- **Fixed File Auto-reload** - Now handles RENAME events from editors, watches files correctly
- **Visual Thinking Indicator** - Animated spinner with multiple styles for AI processing feedback
- **Enhanced Testing Infrastructure** - Professional Makefile, coverage reports, CI/CD pipeline
- **Fixed `/edit` Context Awareness Bug** - Conversation context detection with interactive fallback

---

**Note on Prioritization:** This list prioritizes **stability** (bugs, tests), **core user experience** (clear output, token management), and **reliability** (API error handling) above all new features. This approach ensures DeeCLI becomes a robust and trustworthy tool before expanding its feature set.

*Last updated: September 20 2025*
