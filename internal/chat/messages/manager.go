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

package messages

import (
	"fmt"
	"strings"

	"github.com/antenore/deecli/internal/ai"
	"github.com/antenore/deecli/internal/api"
	"github.com/antenore/deecli/internal/chat/ui"
	"github.com/antenore/deecli/internal/sessions"
	"github.com/charmbracelet/lipgloss"
)

// Dependencies required by the message manager
type Dependencies struct {
	Renderer        *ui.Renderer
	Spinner         *ui.Spinner
	SessionManager  *sessions.Manager
	CurrentSession  *sessions.Session
	AIOperations    *ai.Operations
}

// Manager handles message storage, formatting, and display
type Manager struct {
	messages       []string       // Formatted messages for display
	apiMessages    []api.Message  // Raw API messages for conversation context
	deps           Dependencies
}

// NewManager creates a new message manager
func NewManager(deps Dependencies) *Manager {
	return &Manager{
		messages:    []string{},
		apiMessages: []api.Message{},
		deps:        deps,
	}
}

// AddMessage adds a new message to the conversation
func (mm *Manager) AddMessage(role, content string, viewport ViewportInterface, filesWidgetVisible bool) {
	// Update renderer with current viewport dimensions
	if mm.deps.Renderer != nil {
		mm.deps.Renderer.SetViewportWidth(viewport.GetWidth(), filesWidgetVisible)
	}

	// Save to session database
	if mm.deps.SessionManager != nil && mm.deps.CurrentSession != nil && role != "system" {
		mm.deps.SessionManager.SaveMessage(mm.deps.CurrentSession.ID, role, content)
	}

	// Store in API format for conversation context (exclude system messages)
	if role != "system" {
		mm.apiMessages = append(mm.apiMessages, api.Message{
			Role:    role,
			Content: content,
		})
		// Sync with AI operations
		if mm.deps.AIOperations != nil {
			mm.deps.AIOperations.SetAPIMessages(mm.apiMessages)
		}
	}

	// Use renderer to format the message
	var formattedContent string
	if mm.deps.Renderer != nil {
		formattedContent = mm.deps.Renderer.FormatMessage(role, content)
	} else {
		// Fallback if renderer is not available
		formattedContent = fmt.Sprintf("%s: %s", role, content)
	}

	// Add to message history
	mm.messages = append(mm.messages, formattedContent)

	// Rebuild full content from all messages
	mm.updateViewport(viewport, false, "")
}

// RefreshViewport rebuilds the viewport display
func (mm *Manager) RefreshViewport(viewport ViewportInterface, isLoading bool, loadingMsg string) {
	mm.updateViewport(viewport, isLoading, loadingMsg)
}

// updateViewport internal method to update viewport content
func (mm *Manager) updateViewport(viewport ViewportInterface, isLoading bool, loadingMsg string) {
	if isLoading {
		// Use renderer with animated spinner
		var loadingDisplay string
		if mm.deps.Renderer != nil {
			loadingDisplay = mm.deps.Renderer.FormatLoadingMessageWithSpinner(loadingMsg, mm.deps.Spinner.Frame())
		} else {
			// Fallback if renderer is not available
			loadingStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("220")).Bold(true)
			spinnerFrame := mm.deps.Spinner.Frame()
			if spinnerFrame == "" {
				spinnerFrame = "🔄"
			}
			loadingDisplay = loadingStyle.Render(spinnerFrame + " " + loadingMsg)
		}

		// Show all messages plus loading indicator
		allContent := strings.Join(mm.messages, "\n\n")
		if allContent != "" {
			viewport.SetContent(allContent + "\n\n" + loadingDisplay)
		} else {
			viewport.SetContent(loadingDisplay)
		}
		viewport.GotoBottom()
	} else {
		// Just show all messages
		fullContent := strings.Join(mm.messages, "\n\n")
		viewport.SetContent(fullContent)
	}
}

// GetMessages returns the formatted messages
func (mm *Manager) GetMessages() []string {
	return mm.messages
}

// SetMessages sets the formatted messages (for session loading)
func (mm *Manager) SetMessages(messages []string) {
	mm.messages = messages
}

// GetAPIMessages returns the API messages
func (mm *Manager) GetAPIMessages() []api.Message {
	return mm.apiMessages
}

// SetAPIMessages sets the API messages (for session loading)
func (mm *Manager) SetAPIMessages(apiMessages []api.Message) {
	mm.apiMessages = apiMessages
	// Sync with AI operations
	if mm.deps.AIOperations != nil {
		mm.deps.AIOperations.SetAPIMessages(mm.apiMessages)
	}
}

// ViewportInterface defines the required viewport methods
type ViewportInterface interface {
	SetContent(string)
	GotoBottom()
	GetWidth() int
}