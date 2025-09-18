# TODO.md

## 🎯 Current Priorities

### Priority 1: Core Functionality
- [x] **Configuration Completion**: Unified `/config` command for chat

### Priority 2: User Experience
- [x] **Streaming API Responses**: Real-time output as model generates (see DEEPSEEK_IMPROVEMENTS.md #1)
- [ ] **Session Management**: `/session list`, `/session load`, `/session save` commands
- [ ] **Assistant Identity**: Change "assistant:" to "DeeCLI:" in chat interface
- [ ] **User Name Configuration**: Set user display name during `/config init` or first-time setup (instead of "You:")
- [ ] **Visual Thinking Indicator**: Add visual effect/spinner when DeeCLI is processing requests
- [ ] **File Discovery**: `/explore` command with interactive permission prompts
- [ ] **Output Formatting**: Copy-ready code blocks with clear markers
- [ ] **Enhanced Config Validation**: Model whitelist and key binding validation with helpful error messages
- [ ] **File Change Detection**: Auto-reload files when modified externally

## ✅ Recently Completed (max 2 to 4 items)
- **Smart Tab Navigation**: Context-aware Tab for completion and focus switching with full cycle support
- **Standard Text Editing Shortcuts**: Enabled all Bubbletea textarea shortcuts including Ctrl+W word deletion
- **Arrow Key History Navigation**: Up/down arrows navigate command history for single-line input
- **Enhanced /edit Command**: Smart file detection without arguments, prioritizes loaded files from AI context

## 📋 Near-Term Roadmap

### File Management
- [ ] `/edit file:42` - Open at specific line if editor supports it
- [ ] `.gitignore` respect for file operations

### Performance & Reliability
- [ ] **Testing Infrastructure**: Create comprehensive unit tests for critical modules
- [ ] **Smart Context Management**: Implement file prioritization for large codebases (sort by recency, relevance)
- [ ] **Memory Optimization**: Prune conversation history after 1000 messages to prevent memory issues
- [ ] **Token Usage Display**: Show current token count in interface
- [ ] **Token Estimation**: Warn users before hitting limits (see DEEPSEEK_IMPROVEMENTS.md #3)
- [ ] Context window management
- [ ] Circuit breaker pattern for API failures
- [ ] **API Health Checks**: Implement periodic connection health monitoring
- [ ] **Rate Limiting**: Add rate limit detection and backoff handling

### API Optimization Strategies
- [ ] **Connection Optimization**: Investigate if connection warming is working properly (should be implemented but may be ineffective)
- [ ] **Response Caching**: Implement caching for repeated requests (can reduce costs by 74%)
- [ ] **Request Compression**: Consider request compression for better performance
- [ ] **JSON Mode**: Structured output for `/analyze` command (see DEEPSEEK_IMPROVEMENTS.md #4)
- [ ] **Stop Sequences**: Control output boundaries for code generation (see DEEPSEEK_IMPROVEMENTS.md #5)
- [ ] **FIM Completions**: Code completion for `/edit` (see DEEPSEEK_IMPROVEMENTS.md #2)
- [ ] **Cost Tracking**: Enhanced `/balance` and `/cost` commands for detailed API usage/cost breakdown (see DEEPSEEK_IMPROVEMENTS.md #6)
- [ ] **Request ID Tracking**: Add unique request IDs for debugging API issues
- [ ] **Performance Metrics**: Track and log slow API calls (>2s) for monitoring

### Git Integration (Basic)
- [ ] `/git status` - Show git status in chat
- [ ] `/git diff` - Show changes inline
- [ ] Git-aware file completion (only tracked files)

## 🔮 Future Considerations

### When Core is Stable
- [ ] Mouse support for pane switching
- [ ] Advanced session management (export/delete)
- [ ] Patch file generation (`.patch` format)
- [ ] Multi-file dependency analysis

### Technical Debt & Improvements
- [ ] Improve text selection and copying in viewport
- [ ] Memory clearing for API keys after use
- [ ] Encrypted storage in config files
- [ ] **Code Review**: Final review for any remaining utility functions that could be shared
- [ ] **Completion Engine Enhancement**: Consider further modularization of complex completion logic if needed

## 🚧 Deferred Features
*The following items are intentionally deferred until core functionality is stable:*
- Bang commands for shell execution (`! ls -la`)
- `/run` command for code execution
- `/test` command for running tests
- Markdown rendering with syntax highlighting
- Complex session export formats

---

**Note**: This roadmap focuses on delivering a stable, reliable core experience first. Features are prioritized based on user value and implementation
complexity.

*Last updated: September 2025*
