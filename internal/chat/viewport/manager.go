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

package viewport

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/antenore/deecli/internal/api"
	"github.com/antenore/deecli/internal/chat/ui"
	"github.com/antenore/deecli/internal/config"
	"github.com/antenore/deecli/internal/sessions"
	"github.com/charmbracelet/bubbles/viewport"
	"github.com/charmbracelet/lipgloss"
)

// Manager handles viewport and message management
type Manager struct {
	viewport         *viewport.Model
	renderer         *ui.Renderer
	layoutManager    *ui.Layout
	configManager    *config.Manager
	sessionManager   *sessions.Manager
	currentSession   *sessions.Session
	spinner          *ui.Spinner
	// Use pointers to the actual model data instead of copies
	messages         *[]string
	apiMessages      *[]api.Message
	filesWidgetVisible *bool
	isLoading        *bool
	loadingMsg       *string
}

// Dependencies contains all dependencies needed by the viewport manager
type Dependencies struct {
	Viewport         *viewport.Model
	Renderer         *ui.Renderer
	LayoutManager    *ui.Layout
	ConfigManager    *config.Manager
	SessionManager   *sessions.Manager
	CurrentSession   *sessions.Session
	Spinner          *ui.Spinner
	Messages         *[]string
	APIMessages      *[]api.Message
	FilesWidgetVisible *bool
	IsLoading        *bool
	LoadingMsg       *string
}

// NewManager creates a new viewport manager
func NewManager(deps Dependencies) *Manager {
	return &Manager{
		viewport:         deps.Viewport,
		renderer:         deps.Renderer,
		layoutManager:    deps.LayoutManager,
		configManager:    deps.ConfigManager,
		sessionManager:   deps.SessionManager,
		currentSession:   deps.CurrentSession,
		spinner:          deps.Spinner,
		// Store pointers to actual model data
		messages:         deps.Messages,
		apiMessages:      deps.APIMessages,
		filesWidgetVisible: deps.FilesWidgetVisible,
		isLoading:        deps.IsLoading,
		loadingMsg:       deps.LoadingMsg,
	}
}

// AddMessage adds a message to the viewport
func (m *Manager) AddMessage(role, content string, aiOperations interface{}) {
	// Update renderer with current viewport dimensions
	if m.renderer != nil {
		m.renderer.SetViewportWidth(m.viewport.Width, *m.filesWidgetVisible)
	}

	// Save to session database
	if m.sessionManager != nil && m.currentSession != nil && role != "system" {
		m.sessionManager.SaveMessage(m.currentSession.ID, role, content)
	}

	// Store in API format for conversation context (exclude system messages)
	if role != "system" {
		*m.apiMessages = append(*m.apiMessages, api.Message{
			Role:    role,
			Content: content,
		})
		// Sync with AI operations (using interface{} to avoid circular imports)
		if aiOps, ok := aiOperations.(interface{ SetAPIMessages([]api.Message) }); ok {
			aiOps.SetAPIMessages(*m.apiMessages)
		}
	}

	// Use renderer to format the message
	var formattedContent string
	if m.renderer != nil {
		formattedContent = m.renderer.FormatMessage(role, content)
	} else {
		// Fallback if renderer is not available
		formattedContent = fmt.Sprintf("%s: %s", role, content)
	}

	// Add to message history
	*m.messages = append(*m.messages, formattedContent)

	// Rebuild full content from all messages
	fullContent := strings.Join(*m.messages, "\n\n")
	m.viewport.SetContent(fullContent)
	m.viewport.GotoBottom()
}

// RefreshViewport refreshes the viewport content
func (m *Manager) RefreshViewport() {
	// Rebuild viewport from message history
	if *m.isLoading {
		// Use renderer with animated spinner
		var loadingDisplay string
		if m.renderer != nil && m.spinner != nil {
			loadingDisplay = m.renderer.FormatLoadingMessageWithSpinner(*m.loadingMsg, m.spinner.Frame())
		} else {
			// Fallback if renderer or spinner is not available
			loadingStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("220")).Bold(true)
			spinnerFrame := "🔄"
			if m.spinner != nil {
				frame := m.spinner.Frame()
				if frame != "" {
					spinnerFrame = frame
				}
			}
			loadingDisplay = loadingStyle.Render(spinnerFrame + " " + *m.loadingMsg)

			// Add hint about cancellation
			hintStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("244"))
			hintText := hintStyle.Render("Press Esc to cancel")
			loadingDisplay += "\n" + hintText
		}

		// Show all messages plus loading indicator
		allContent := strings.Join(*m.messages, "\n\n")
		if allContent != "" {
			m.viewport.SetContent(allContent + "\n\n" + loadingDisplay)
		} else {
			m.viewport.SetContent(loadingDisplay)
		}
		m.viewport.GotoBottom()
	} else {
		// Just show all messages
		fullContent := strings.Join(*m.messages, "\n\n")
		m.viewport.SetContent(fullContent)
	}
}

// GetTextWidth calculates available text width for messages
func (m *Manager) GetTextWidth() int {
	availableWidth := m.viewport.Width - 10 // Base padding
	if *m.filesWidgetVisible {
		// Account for sidebar width and borders
		availableWidth = m.viewport.Width - 35
	}
	if availableWidth < 20 {
		availableWidth = 20 // Minimum readable width
	}
	return availableWidth
}

// FormatInitialContent generates the initial welcome content
func (m *Manager) FormatInitialContent() string {
	// Get current working directory
	cwd, err := os.Getwd()
	if err != nil {
		cwd = "unknown"
	}
	
	// Get current key bindings for display
	newlineKey := "Ctrl+J"
	historyBackKey := "Ctrl+P"
	historyForwardKey := "Ctrl+N"
	if m.configManager != nil {
		newlineKey = m.layoutManager.FormatKeyForDisplay(m.configManager.GetNewlineKey())
		historyBackKey = m.layoutManager.FormatKeyForDisplay(m.configManager.GetHistoryBackKey())
		historyForwardKey = m.layoutManager.FormatKeyForDisplay(m.configManager.GetHistoryForwardKey())
	}

	// Compact welcome screen
	welcomeContent := fmt.Sprintf(`🐉 DeeCLI - AI Code Assistant | %s

Essential Commands: /load <file> /unload <pattern> /list /clear /analyze /history /keysetup /help
Quick Keys: Tab=complete/focus %s=newline ↑/↓ or %s/%s=history F1=help F2=files

💡 Start by loading files: /load *.go or /load main.go
   Press F1 for detailed help, Tab for completion`,
		filepath.Base(cwd), newlineKey, historyBackKey, historyForwardKey)
	
	return welcomeContent
}

// HelpContent generates the help content
func (m *Manager) HelpContent() string {
	// Get current key bindings
	newlineKey := "Ctrl+J"
	historyBackKey := "Ctrl+P"
	historyForwardKey := "Ctrl+N"

	if m.configManager != nil {
		newlineKey = m.layoutManager.FormatKeyForDisplay(m.configManager.GetNewlineKey())
		historyBackKey = m.layoutManager.FormatKeyForDisplay(m.configManager.GetHistoryBackKey())
		historyForwardKey = m.layoutManager.FormatKeyForDisplay(m.configManager.GetHistoryForwardKey())
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
/load --all <file> Load files ignoring .gitignore
/unload <pattern> Remove files matching pattern
/add <file>     Same as /load (deprecated)
/list           List all loaded files
/clear          Clear all loaded files
/analyze        Analyze loaded files
/improve        Get improvement suggestions
/explain        Explain loaded code
/edit           AI suggests which files to edit based on conversation
/edit <file>    Open specific file in editor
/edit <file:line> Jump to specific line in file
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


// StripANSI removes ANSI color codes from text
func StripANSI(s string) string {
	ansiRegex := regexp.MustCompile(`\x1b\[[0-9;]*[a-zA-Z]`)
	return ansiRegex.ReplaceAllString(s, "")
}