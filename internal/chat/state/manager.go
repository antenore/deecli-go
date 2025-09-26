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

package state

import (
	"context"

	"github.com/antenore/deecli/internal/chat/input"
	"github.com/antenore/deecli/internal/chat/ui"
	"github.com/antenore/deecli/internal/config"
	"github.com/charmbracelet/bubbles/textarea"
	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
)

// Manager handles TUI state management including viewports, focus modes, and layout
type Manager struct {
	viewport        viewport.Model
	sidebarViewport viewport.Model
	textarea        textarea.Model
	layoutManager   *ui.Layout
	inputManager    *input.Manager
	width           int
	height          int
	ready           bool
	helpVisible     bool
	filesWidgetVisible bool
	focusMode       string // "input", "viewport", or "sidebar"
	isLoading       bool
	loadingMsg      string
	apiCancel       context.CancelFunc
}

// Dependencies contains the dependencies needed by the state manager
type Dependencies struct {
	LayoutManager *ui.Layout
	InputManager  *input.Manager
}

// NewManager creates a new TUI state manager
func NewManager(deps Dependencies) *Manager {
	return &Manager{
		layoutManager: deps.LayoutManager,
		inputManager:  deps.InputManager,
		focusMode:     "input", // Start with input focused
	}
}

// InitializeViewports initializes the viewports with proper size and positioning
func (s *Manager) InitializeViewports(width, height int) {
	s.width = width
	s.height = height
	
	// Initialize viewports with proper size and positioning
	s.viewport = viewport.New(width, 10) // Initial size, will be set by layout
	s.viewport.YPosition = 1  // Start after header line

	// Initialize sidebar viewport
	s.sidebarViewport = viewport.New(25, 10) // Initial size, will be set by layout
	s.sidebarViewport.YPosition = 1  // Start after header line

	// Set proper layout (this will correct the sizes and positions)
	s.updateLayout()
	
	s.ready = true
}

// CreateTextarea creates and configures a textarea component
func (s *Manager) CreateTextarea(width int, configManager *config.Manager) textarea.Model {
	ta := textarea.New()
	ta.Placeholder = "Type your message... (Enter to send, Ctrl+Enter for new line)"
	ta.ShowLineNumbers = false
	ta.SetHeight(3)
	ta.SetWidth(width - 4)
	ta.CharLimit = 0
	ta.Focus()
	ta.Prompt = "┃ "

	// Customize KeyMap: Use default fallback key for newlines
	keyMap := textarea.DefaultKeyMap
	keyMap.InsertNewline.SetKeys("ctrl+j") // Default fallback
	ta.KeyMap = keyMap

	s.textarea = ta
	return ta
}

// updateLayout calculates and sets proper dimensions for all components
func (s *Manager) updateLayout() {
	if s.layoutManager == nil {
		return
	}
	
	// Calculate viewport dimensions using layout manager
	hasCompletions := false
	if s.inputManager != nil {
		completions, _, showCompletions := s.inputManager.GetCompletionState()
		hasCompletions = showCompletions && len(completions) > 0
	}
	viewportHeight, yPosition := s.layoutManager.CalculateViewportDimensions(s.height, hasCompletions)

	// Update viewports with proper Y position
	s.viewport.Height = viewportHeight
	s.viewport.YPosition = yPosition

	s.sidebarViewport.Height = viewportHeight
	s.sidebarViewport.YPosition = yPosition

	// Update textarea width using layout manager
	textareaWidth := s.layoutManager.CalculateTextareaWidth(s.width, s.filesWidgetVisible)
	s.textarea.SetWidth(textareaWidth)
}

// HandleResize handles window resize events
func (s *Manager) HandleResize(width, height int) {
	s.width = width
	s.height = height
	
	// Update viewport width and recalculate layout
	s.viewport.Width = width
	s.updateLayout()
}

// GetViewport returns the main viewport
func (s *Manager) GetViewport() *viewport.Model {
	return &s.viewport
}

// GetSidebarViewport returns the sidebar viewport
func (s *Manager) GetSidebarViewport() *viewport.Model {
	return &s.sidebarViewport
}

// GetTextarea returns the textarea
func (s *Manager) GetTextarea() *textarea.Model {
	return &s.textarea
}

// IsReady returns true if the state manager is ready
func (s *Manager) IsReady() bool {
	return s.ready
}

// GetFocusMode returns the current focus mode
func (s *Manager) GetFocusMode() string {
	return s.focusMode
}

// SetFocusMode sets the current focus mode
func (s *Manager) SetFocusMode(mode string) {
	s.focusMode = mode
	
	// Update focus state of components
	switch mode {
	case "input":
		s.textarea.Focus()
	case "viewport", "sidebar":
		s.textarea.Blur()
	}
}

// CycleFocus cycles through focus modes
func (s *Manager) CycleFocus() string {
	switch s.focusMode {
	case "input":
		s.SetFocusMode("viewport")
	case "viewport":
		if s.filesWidgetVisible {
			s.SetFocusMode("sidebar")
		} else {
			s.SetFocusMode("input")
		}
	case "sidebar":
		s.SetFocusMode("input")
	default:
		s.SetFocusMode("input")
	}
	
	return s.focusMode
}

// IsHelpVisible returns true if help is visible
func (s *Manager) IsHelpVisible() bool {
	return s.helpVisible
}

// SetHelpVisible sets help visibility
func (s *Manager) SetHelpVisible(visible bool) {
	s.helpVisible = visible
}

// IsFilesWidgetVisible returns true if files widget is visible
func (s *Manager) IsFilesWidgetVisible() bool {
	return s.filesWidgetVisible
}

// SetFilesWidgetVisible sets files widget visibility and updates layout
func (s *Manager) SetFilesWidgetVisible(visible bool) {
	s.filesWidgetVisible = visible
	s.updateLayout()
}

// ToggleFilesWidget toggles files widget visibility
func (s *Manager) ToggleFilesWidget() bool {
	s.filesWidgetVisible = !s.filesWidgetVisible
	s.updateLayout()
	return s.filesWidgetVisible
}

// IsLoading returns true if in loading state
func (s *Manager) IsLoading() bool {
	return s.isLoading
}

// GetLoadingMessage returns the loading message
func (s *Manager) GetLoadingMessage() string {
	return s.loadingMsg
}

// SetLoading sets the loading state
func (s *Manager) SetLoading(loading bool, message string) {
	s.isLoading = loading
	s.loadingMsg = message
}

// GetAPICancel returns the API cancel function
func (s *Manager) GetAPICancel() context.CancelFunc {
	return s.apiCancel
}

// SetAPICancel sets the API cancel function
func (s *Manager) SetAPICancel(cancel context.CancelFunc) {
	s.apiCancel = cancel
}

// ClearAPICancel clears the API cancel function
func (s *Manager) ClearAPICancel() {
	s.apiCancel = nil
}

// GetDimensions returns the current dimensions
func (s *Manager) GetDimensions() (int, int) {
	return s.width, s.height
}

// HandleViewportScroll handles viewport scrolling when viewport has focus
func (s *Manager) HandleViewportScroll(key string) bool {
	if s.focusMode != "viewport" {
		return false
	}
	
	switch key {
	case "up", "down", "pgup", "pgdown", "ctrl+u", "ctrl+d", "home", "end":
		s.viewport, _ = s.viewport.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune(key)})
		return true
	case "tab":
		s.CycleFocus()
		return true
	case "enter", "esc":
		s.SetFocusMode("input")
		return true
	}
	
	return false
}

// HandleSidebarScroll handles sidebar scrolling when sidebar has focus
func (s *Manager) HandleSidebarScroll(key string) bool {
	if s.focusMode != "sidebar" {
		return false
	}
	
	switch key {
	case "up", "down", "pgup", "pgdown", "ctrl+u", "ctrl+d", "home", "end":
		s.sidebarViewport, _ = s.sidebarViewport.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune(key)})
		// Note: cmd would need to be handled by the caller
		return true
	case "tab":
		s.SetFocusMode("input")
		return true
	case "enter", "esc":
		s.SetFocusMode("input")
		return true
	}
	
	return false
}

// SetSidebarContent sets the sidebar content
func (s *Manager) SetSidebarContent(content string) {
	s.sidebarViewport.SetContent(content)
	if s.focusMode == "sidebar" {
		s.sidebarViewport.GotoTop()
	}
}

// SetViewportContent sets the main viewport content
func (s *Manager) SetViewportContent(content string) {
	s.viewport.SetContent(content)
}

// GoToBottom scrolls the main viewport to the bottom
func (s *Manager) GoToBottom() {
	s.viewport.GotoBottom()
}

// GoToTop scrolls the main viewport to the top
func (s *Manager) GoToTop() {
	s.viewport.GotoTop()
}

// GetViewportContent returns the current viewport content
func (s *Manager) GetViewportContent() string {
	return s.viewport.View()
}

// GetSidebarContent returns the current sidebar content
func (s *Manager) GetSidebarContent() string {
	return s.sidebarViewport.View()
}