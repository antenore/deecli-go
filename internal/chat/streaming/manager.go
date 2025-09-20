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

package streaming

import (
	"strings"

	"github.com/antenore/deecli/internal/ai"
	"github.com/antenore/deecli/internal/api"
	"github.com/antenore/deecli/internal/chat/ui"
	tea "github.com/charmbracelet/bubbletea"
)

// Manager handles streaming operations and state
type Manager struct {
	streamReader  api.StreamReader
	streamContent string
	isActive      bool
}

// NewManager creates a new streaming manager
func NewManager() *Manager {
	return &Manager{
		streamContent: "",
		isActive:      false,
	}
}

// StartStream handles the start of a new stream
func (sm *Manager) StartStream(msg ai.StreamStartedMsg, renderer interface{}, messages *[]string) tea.Cmd {
	sm.streamReader = msg.Stream
	sm.streamContent = ""
	sm.isActive = true

	// Add initial placeholder assistant message - this will be updated during streaming
	if r, ok := renderer.(interface{ FormatMessage(string, string) string }); ok {
		*messages = append(*messages, r.FormatMessage("assistant", ""))
	}

	// Start reading the first chunk
	return ai.ReadNextChunk(msg.Stream, sm.streamContent)
}

// HandleChunk processes a streaming chunk
func (sm *Manager) HandleChunk(msg ai.StreamChunkMsg, spinner *ui.Spinner, isLoading *bool, setLoadingFn func(bool, string) tea.Cmd) (tea.Cmd, []tea.Cmd) {
	var cmds []tea.Cmd

	if msg.Err != nil {
		return sm.completeStream(sm.streamContent, msg.Err), nil
	}

	// Append chunk content first
	sm.streamContent += msg.Content

	// Stop spinner only when we have accumulated meaningful content
	// This ensures the spinner stays visible during the "thinking" phase
	// and initial token streaming before substantial text appears
	if *isLoading && len(strings.TrimSpace(sm.streamContent)) >= 10 {
		if cmd := setLoadingFn(false, ""); cmd != nil {
			cmds = append(cmds, cmd)
		}
	}

	// Continue reading next chunk
	if sm.streamReader != nil {
		nextCmd := ai.ReadNextChunk(sm.streamReader, sm.streamContent)
		if len(cmds) > 0 {
			cmds = append(cmds, nextCmd)
			return nil, cmds
		}
		return nextCmd, cmds
	}

	return nil, cmds
}

// CompleteStream handles stream completion
func (sm *Manager) CompleteStream(msg ai.StreamCompleteMsg) tea.Cmd {
	return sm.completeStream(msg.TotalContent, msg.Err)
}

// completeStream internal completion handler
func (sm *Manager) completeStream(content string, err error) tea.Cmd {
	sm.isActive = false
	sm.streamReader = nil
	sm.streamContent = ""

	// Return completion message
	return func() tea.Msg {
		return StreamCompleteInternalMsg{
			Content: content,
			Err:     err,
		}
	}
}

// UpdateDisplay updates the streaming display with accumulated content
func (sm *Manager) UpdateDisplay(content string, renderer interface{}, messages *[]string, viewport ViewportInterface) {
	// Only update if we have messages (allow updates during entire streaming process)
	if len(*messages) == 0 {
		return
	}

	// Update the last message (which should be our streaming assistant message)
	lastIdx := len(*messages) - 1
	if r, ok := renderer.(interface{ FormatMessage(string, string) string }); ok {
		(*messages)[lastIdx] = r.FormatMessage("assistant", content)
	}

	// Update viewport content
	viewport.SetContent(strings.Join(*messages, "\n\n"))
	viewport.GotoBottom()
}

// ViewportInterface defines required viewport methods
type ViewportInterface interface {
	SetContent(string)
	GotoBottom()
}

// GetStreamContent returns the current accumulated stream content
func (sm *Manager) GetStreamContent() string {
	return sm.streamContent
}

// AddContent adds content to the stream (for direct manipulation)
func (sm *Manager) AddContent(content string) {
	sm.streamContent += content
}

// GetStreamReader returns the current stream reader
func (sm *Manager) GetStreamReader() interface{} {
	return sm.streamReader
}

// IsActive returns whether streaming is currently active
func (sm *Manager) IsActive() bool {
	return sm.isActive
}

// Reset resets the streaming state
func (sm *Manager) Reset() {
	sm.streamContent = ""
	sm.isActive = false
	if sm.streamReader != nil {
		sm.streamReader.Close()
		sm.streamReader = nil
	}
}

// StreamCompleteInternalMsg is used internally for stream completion
type StreamCompleteInternalMsg struct {
	Content string
	Err     error
}