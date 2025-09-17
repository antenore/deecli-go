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

package sessions

import (
	"fmt"
	"strings"

	"github.com/antenore/deecli/internal/api"
)

// LoaderDependencies contains the dependencies needed for session loading
type LoaderDependencies struct {
	SessionManager     *Manager
	CurrentSession     *Session
	Renderer           interface {
		FormatMessage(role, content string) string
		SetViewportWidth(width int, filesVisible bool)
	}
	Viewport           interface {
		SetContent(content string)
		GotoBottom() []string
	}
	ViewportWidth      int
	FilesWidgetVisible bool
	FormatInitialContent func() string
}

// Loader handles session loading operations
type Loader struct {
	deps *LoaderDependencies
}

// NewLoader creates a new session loader
func NewLoader(deps *LoaderDependencies) *Loader {
	return &Loader{
		deps: deps,
	}
}

// LoadSession loads the previous session and returns messages and apiMessages
func (l *Loader) LoadSession() ([]string, []api.Message, error) {
	if l.deps.SessionManager == nil || l.deps.CurrentSession == nil {
		return nil, nil, fmt.Errorf("no session manager available")
	}

	messages, err := l.deps.SessionManager.GetSessionMessages(l.deps.CurrentSession.ID)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to load session messages: %w", err)
	}

	var displayMessages []string
	var apiMessages []api.Message

	// Initialize with formatted initial content
	if l.deps.FormatInitialContent != nil {
		displayMessages = append(displayMessages, l.deps.FormatInitialContent())
	}

	for _, msg := range messages {
		// Store in API format for context (exclude system messages)
		if msg.Role != "system" {
			apiMessages = append(apiMessages, api.Message{
				Role:    msg.Role,
				Content: msg.Content,
			})
		}

		// Update renderer with current viewport dimensions
		if l.deps.Renderer != nil {
			l.deps.Renderer.SetViewportWidth(l.deps.ViewportWidth, l.deps.FilesWidgetVisible)
		}

		// Use renderer to format the message
		var formattedContent string
		if l.deps.Renderer != nil {
			formattedContent = l.deps.Renderer.FormatMessage(msg.Role, msg.Content)
		} else {
			// Fallback if renderer is not available
			formattedContent = fmt.Sprintf("%s: %s", msg.Role, msg.Content)
		}
		displayMessages = append(displayMessages, formattedContent)
	}

	// Update viewport with content
	if l.deps.Viewport != nil {
		fullContent := strings.Join(displayMessages, "\n\n")
		l.deps.Viewport.SetContent(fullContent)
		l.deps.Viewport.GotoBottom()
	}

	return displayMessages, apiMessages, nil
}