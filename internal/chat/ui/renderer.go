// Copyright 2025 Antenore Gatta
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package ui

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/antenore/deecli/internal/config"
	"github.com/charmbracelet/lipgloss"
)

// Renderer handles message formatting and display content
type Renderer struct {
	configManager *config.Manager
	viewportWidth int
	sidebarVisible bool
	syntaxHighlightEnabled bool
	rawCodeMode bool // Toggle for raw code display (no borders/formatting)
}

// NewRenderer creates a new renderer
func NewRenderer(configManager *config.Manager) *Renderer {
	// Default to disabled syntax highlighting for better copying
	syntaxHighlight := false
	if configManager != nil {
		syntaxHighlight = configManager.GetSyntaxHighlightEnabled()
	}

	return &Renderer{
		configManager: configManager,
		syntaxHighlightEnabled: syntaxHighlight,
		rawCodeMode: true, // Start in raw mode for easy copying
	}
}

// SetViewportWidth updates the viewport width for text wrapping
func (r *Renderer) SetViewportWidth(width int, sidebarVisible bool) {
	r.viewportWidth = width
	r.sidebarVisible = sidebarVisible
}

// SetSyntaxHighlightEnabled updates the syntax highlighting setting
func (r *Renderer) SetSyntaxHighlightEnabled(enabled bool) {
	r.syntaxHighlightEnabled = enabled
}

// ToggleRawCodeMode toggles between raw and formatted code display
func (r *Renderer) ToggleRawCodeMode() bool {
	r.rawCodeMode = !r.rawCodeMode
	return r.rawCodeMode
}

// GetRawCodeMode returns the current raw code mode state
func (r *Renderer) GetRawCodeMode() bool {
	return r.rawCodeMode
}

// FormatMessage formats a message with proper styling and wrapping
func (r *Renderer) FormatMessage(role, content string) string {
	var style lipgloss.Style
	var prefix string

	switch role {
	case "user":
		style = lipgloss.NewStyle().Foreground(lipgloss.Color("220")).Bold(true)
		userName := "You"
		if r.configManager != nil {
			userName = r.configManager.GetUserName()
		}
		prefix = userName + ": "
	case "assistant":
		style = lipgloss.NewStyle().Foreground(lipgloss.Color("86")).Bold(true)
		prefix = "DeeCLI: "
	case "system":
		style = lipgloss.NewStyle().Foreground(lipgloss.Color("244"))
		prefix = "System: "
	}

	// Calculate available width for content
	availableWidth := r.viewportWidth - len(prefix) - 2 // Account for prefix and some padding
	if r.sidebarVisible {
		// Adjust for sidebar taking up space
		availableWidth = r.viewportWidth - 30 // Account for sidebar width
	}
	if availableWidth < 20 {
		availableWidth = 20 // Minimum readable width
	}

	// Format content with code block handling
	formattedContent := r.formatContentWithCodeBlocks(content, availableWidth)

	return style.Render(prefix) + formattedContent
}

// FormatInitialContent creates the welcome message
func (r *Renderer) FormatInitialContent() string {
	// Get current working directory
	cwd, err := os.Getwd()
	if err != nil {
		cwd = "unknown"
	}

	// Get current key bindings for display
	newlineKey := "Ctrl+J"
	historyBackKey := "Ctrl+P"
	historyForwardKey := "Ctrl+N"
	if r.configManager != nil {
		newlineKey = r.formatKeyForDisplay(r.configManager.GetNewlineKey())
		historyBackKey = r.formatKeyForDisplay(r.configManager.GetHistoryBackKey())
		historyForwardKey = r.formatKeyForDisplay(r.configManager.GetHistoryForwardKey())
	}

	// Compact welcome screen
	welcomeContent := fmt.Sprintf(`🐉 DeeCLI - AI Code Assistant | %s

Essential Commands: /load <file> /unload <pattern> /list /clear /analyze /config /history /help
Quick Keys: Tab=complete/focus %s=newline ↑/↓ or %s/%s=history F1=help F2=files F3=format

💡 Start by loading files: /load *.go or /load main.go
   Code is raw by default (copy-friendly). Press F3 for formatted view`,
		filepath.Base(cwd), newlineKey, historyBackKey, historyForwardKey)

	return welcomeContent
}

// FormatHelpContent creates the detailed help content
func (r *Renderer) FormatHelpContent() string {
	// Get current key bindings
	newlineKey := "Ctrl+J"
	historyBackKey := "Ctrl+P"
	historyForwardKey := "Ctrl+N"

	if r.configManager != nil {
		newlineKey = r.formatKeyForDisplay(r.configManager.GetNewlineKey())
		historyBackKey = r.formatKeyForDisplay(r.configManager.GetHistoryBackKey())
		historyForwardKey = r.formatKeyForDisplay(r.configManager.GetHistoryForwardKey())
	}

	return fmt.Sprintf(`🐉 DeeCLI Help

=== Multi-line Input ===
• Enter: Send message
• %s: New line in message
• Type naturally across multiple lines

=== History Navigation ===
• %s: Previous command/message
• %s: Next command/message

=== Chat Commands ===
/load <file>    Load files (additive - adds to existing)
/unload <pattern> Remove files matching pattern
/add <file>     Same as /load (deprecated)
/list           List all loaded files
/clear          Clear all loaded files
/analyze        Analyze loaded files
/improve        Get improvement suggestions
/explain        Explain loaded code
/edit           AI suggests which files to edit based on conversation
/edit <file>    Open specific file in editor
/config         View/manage configuration settings
/keysetup       Configure key bindings
/history        View/manage command history
/help           Show this help
/quit           Exit the application

=== Keyboard Shortcuts ===
Tab             Smart: show/accept completions OR switch focus
Enter           Send message
%s         New line in message
↑ or %s    Previous history (single-line input only)
↓ or %s    Next history (single-line input only)
F1              Toggle this help
F2              Toggle files sidebar
F3              Toggle code format (raw/bordered) for new messages
Esc             Cancel ongoing AI response
Ctrl+C          Exit application
Ctrl+W          Delete word backward
Ctrl+U/K        Delete to line start/end
Alt+Backspace   Delete word backward (alternative)

=== Focus Modes ===
✏️ INPUT        Type messages and commands
📜 CHAT         Scroll through chat history
📁 FILES        Browse loaded files (when F2 open)

Tab cycles focus: Input → Chat → Files (if open) → Input

=== Navigation ===
↑/↓             Scroll in viewport/sidebar OR history in input (single-line)
PgUp/PgDn       Page up/down
Ctrl+U/Ctrl+D   Half page up/down
Home/End        Jump to top/bottom
Esc/Enter       Return to input mode

Tip: Yellow border shows which pane has focus!

=== File Patterns ===
You can use glob patterns to load multiple files:
  /load *.go           Load all .go files
  /load src/**/*.go    Load all .go files in src
  /load {*.go,*.md}    Load all .go and .md files

=== Tips ===
• Multi-line messages: Use %s to add new lines
• Quick submit: Just press Enter to send your message
• Press Tab (when no completions) to switch between panes
• Standard text editing shortcuts work (Ctrl+W, Ctrl+U, Ctrl+K, etc.)
• Yellow border shows which pane has focus
• Tab shows completions, use ↑↓ arrows to cycle, Tab/Enter to accept, Esc to cancel
• Arrow keys scroll in focused panes
• Press Esc to quickly return to input mode

Press F1 to close this help`, newlineKey, historyBackKey, historyForwardKey,
		newlineKey, historyBackKey, historyForwardKey, newlineKey)
}

// FormatLoadingMessage creates a loading message with cancel hint
func (r *Renderer) FormatLoadingMessage(loadingMsg string) string {
	// Add loading indicator with static fallback
	loadingStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("220")).Bold(true)
	loadingText := loadingStyle.Render("🔄 " + loadingMsg)

	// Add hint about cancellation
	hintStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("244"))
	hintText := hintStyle.Render("Press Esc to cancel")

	return loadingText + "\n" + hintText
}

// FormatLoadingMessageWithSpinner creates a loading message with animated spinner
func (r *Renderer) FormatLoadingMessageWithSpinner(loadingMsg string, spinnerFrame string) string {
	// Add loading indicator with animated spinner
	loadingStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("220")).Bold(true)
	spinnerText := spinnerFrame
	if spinnerText == "" {
		spinnerText = "🔄" // Fallback if spinner is not active
	}
	loadingText := loadingStyle.Render(spinnerText + " " + loadingMsg)

	// Add hint about cancellation
	hintStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("244"))
	hintText := hintStyle.Render("Press Esc to cancel")

	return loadingText + "\n" + hintText
}

// formatContentWithCodeBlocks processes content to format code blocks with clear boundaries
func (r *Renderer) formatContentWithCodeBlocks(content string, width int) string {
	// Regular expression to match code blocks with optional language
	codeBlockRegex := regexp.MustCompile("(?s)```([a-zA-Z0-9_+-]*)\n(.*?)```")

	var result strings.Builder
	lastEnd := 0

	// Find all code blocks
	matches := codeBlockRegex.FindAllStringSubmatchIndex(content, -1)

	for _, match := range matches {
		// Add text before code block
		if match[0] > lastEnd {
			textBefore := content[lastEnd:match[0]]
			// Wrap non-code text
			wrapper := lipgloss.NewStyle().Width(width)
			result.WriteString(wrapper.Render(strings.TrimSpace(textBefore)))
			if strings.TrimSpace(textBefore) != "" {
				result.WriteString("\n")
			}
		}

		// Extract language and code
		language := ""
		if match[3] > match[2] {
			language = strings.TrimSpace(content[match[2]:match[3]])
		}
		code := content[match[4]:match[5]]

		// Format the code block
		result.WriteString(r.formatCodeBlock(code, language, width))

		lastEnd = match[1]
	}

	// Add remaining text after last code block
	if lastEnd < len(content) {
		remainingText := content[lastEnd:]
		if strings.TrimSpace(remainingText) != "" {
			wrapper := lipgloss.NewStyle().Width(width)
			result.WriteString("\n")
			result.WriteString(wrapper.Render(strings.TrimSpace(remainingText)))
		}
	}

	return result.String()
}

// formatCodeBlock formats a single code block with clear boundaries
func (r *Renderer) formatCodeBlock(code, language string, width int) string {
	// If raw mode is enabled, return code as-is with minimal formatting
	if r.rawCodeMode {
		var block strings.Builder
		block.WriteString("\n")
		// Just the raw code, nothing else - perfect for copying
		block.WriteString(code)
		if !strings.HasSuffix(code, "\n") {
			block.WriteString("\n")
		}
		block.WriteString("\n")
		return block.String()
	}

	// Get code block style from config
	style := "bordered"
	if r.configManager != nil {
		style = r.configManager.GetCodeBlockStyle()
	}

	// Apply syntax highlighting if enabled (not in raw mode)
	highlightedCode := HighlightCode(code, language, r.syntaxHighlightEnabled && !r.rawCodeMode)

	var block strings.Builder

	if style == "simple" {
		// Simple style with just language indicator and indentation
		if language != "" {
			block.WriteString(fmt.Sprintf("\n[%s]\n", language))
		} else {
			block.WriteString("\n")
		}
		// Code content with simple indentation
		lines := strings.Split(strings.TrimRight(highlightedCode, "\n"), "\n")
		for _, line := range lines {
			block.WriteString("  " + line + "\n")
		}
		block.WriteString("\n")
	} else {
		// Bordered style (default)
		separator := strings.Repeat("─", min(width, 80))

		// Top border with language indicator
		if language != "" {
			block.WriteString(fmt.Sprintf("\n┌─ %s %s\n", language, separator[:max(0, len(separator)-len(language)-4)]))
		} else {
			block.WriteString("\n┌" + separator[:min(len(separator), width-1)] + "\n")
		}

		// Code content (preserve exact formatting)
		lines := strings.Split(strings.TrimRight(highlightedCode, "\n"), "\n")
		for _, line := range lines {
			block.WriteString("│ " + line + "\n")
		}

		// Bottom border
		block.WriteString("└" + separator[:min(len(separator), width-1)] + "\n")
	}

	return block.String()
}

// min returns the minimum of two integers
func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

// max returns the maximum of two integers
func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}

// formatKeyForDisplay formats a key string for user-friendly display
func (r *Renderer) formatKeyForDisplay(key string) string {
	if key == "" {
		return "Ctrl+J" // Default
	}

	// Split by + and capitalize each part
	parts := strings.Split(key, "+")
	for i, part := range parts {
		switch strings.ToLower(part) {
		case "ctrl":
			parts[i] = "Ctrl"
		case "alt":
			parts[i] = "Alt"
		case "shift":
			parts[i] = "Shift"
		case "enter":
			parts[i] = "Enter"
		default:
			// Uppercase single letters (j -> J, m -> M)
			if len(part) == 1 {
				parts[i] = strings.ToUpper(part)
			} else {
				// Capitalize first letter of words
				parts[i] = strings.Title(part)
			}
		}
	}
	return strings.Join(parts, "+")
}