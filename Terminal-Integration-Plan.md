# Terminal Integration Plan for DeeCLI

## Overview

Plan to implement industry-standard Shift+Enter support for newlines in DeeCLI, making it more intuitive for users coming from other development tools.

## How It Works

### Detection Strategy
- Check if terminal supports modern keyboard protocols (Kitty/CSI u protocol)
- Parse escape sequences like `\e[13;2u` for Shift+Enter
- Fall back to manual terminal configuration for older terminals

### Implementation Approach

```go
// Enhanced key detection in Bubbletea
func detectShiftEnter(msg tea.KeyMsg) bool {
    // Modern terminal: check for Shift+Enter escape sequence
    if msg.String() == "\x1b[13;2u" {
        return true
    }
    // Legacy: check configured alternatives
    return false
}
```

## Terminal Setup Command

### DeeCLI `/terminal-setup` Command
- Detect terminal type (iTerm2, VS Code, Warp, etc.)
- Generate appropriate configuration snippets
- Provide copy-paste instructions for manual setup

### Configuration Files We'd Generate

**iTerm2**:
```json
{"Key": "0xd-0x20000", "Action": 10, "Text": "\\e[13;2u"}
```

**VS Code settings.json**:
```json
"terminal.integrated.sendKeybindingsToShell": false,
"terminal.integrated.commandsToSkipShell": ["shift+enter"]
```

**Kitty terminal**:
```
map shift+enter send_text all \e[13;2u
```

**Warp terminal**:
```yaml
keybindings:
  - key: shift+enter
    command: send_text
    args: ["\e[13;2u"]
```

## Challenges for SSH/Remote Usage

1. **Terminal passthrough**: SSH needs to forward the escape sequences
2. **Client-side config**: The terminal setup must happen on the client machine
3. **Protocol support**: Both client terminal and SSH server need compatible protocols

## Implementation Phases

### Phase 1: Basic Support
- Add `/terminal-setup` command to DeeCLI
- Detect common terminals and provide setup instructions
- Support modern keyboard protocols where available

### Phase 2: Enhanced Detection
- Implement escape sequence parsing in our Bubbletea key handler
- Add fallback mechanisms for various terminal types
- Test across SSH scenarios

### Phase 3: Auto-Configuration
- Try to auto-detect and configure some terminals
- Provide better error messages and fallback options

## Benefits

- Industry-standard UX (Shift+Enter for newlines)
- Better experience for users coming from other tools
- Maintains current Ctrl+J fallback for reliability
- Works well over SSH with proper configuration

## Technical Implementation

### Code Changes Required
1. Add terminal detection utilities
2. Enhance key handling in model.go
3. Add `/terminal-setup` command
4. Update configuration system to store terminal preferences

### Key Handler Enhancement
```go
// In model.go Update method
case tea.KeyMsg:
    // Check for Shift+Enter first
    if detectShiftEnter(msg) {
        // Insert newline
        m.textarea.InsertString("\n")
        return m, nil
    }
    // ... existing key handling
```

### Terminal Detection
```go
func detectTerminal() string {
    // Check environment variables
    if term := os.Getenv("TERM_PROGRAM"); term != "" {
        return term // "iTerm.app", "vscode", etc.
    }
    // Check other indicators
    return "unknown"
}
```

## Configuration Integration

### Global Config Addition
```yaml
terminal:
  type: "auto"  # auto-detect or specific type
  newline_key: "shift+enter"  # preferred newline key
  escape_sequence: "\e[13;2u"  # custom escape sequence
```

### Per-Terminal Profiles
- Store terminal-specific configurations
- Allow users to override detection
- Provide manual configuration options

## Testing Strategy

1. **Local testing**: Various terminal emulators
2. **SSH testing**: Through different SSH clients
3. **Protocol testing**: Modern vs legacy terminal protocols
4. **Fallback testing**: Ensure Ctrl+J always works

## Migration Path

1. Keep current Ctrl+J as default
2. Add Shift+Enter as optional enhancement
3. Gradual rollout with user feedback
4. Eventually make Shift+Enter the recommended default

This implementation would provide industry-standard keyboard behavior while maintaining reliability across all terminal environments.