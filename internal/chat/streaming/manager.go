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
	"unicode"

	"github.com/antenore/deecli/internal/ai"
	"github.com/antenore/deecli/internal/api"
	"github.com/antenore/deecli/internal/chat/ui"
	tea "github.com/charmbracelet/bubbletea"
)

// Manager handles streaming operations and state
type Manager struct {
	streamReader         api.StreamReader
	streamContent        string
	isActive             bool
	messageAdded         bool // Track if assistant message has been added yet
}

// NewManager creates a new streaming manager
func NewManager() *Manager {
	return &Manager{
		streamContent: "",
		isActive:      false,
		messageAdded:  false,
	}
}

// StartStream handles the start of a new stream
func (sm *Manager) StartStream(msg ai.StreamStartedMsg, renderer interface{}, messages *[]string) tea.Cmd {
	sm.streamReader = msg.Stream
	sm.streamContent = ""
	sm.isActive = true
	sm.messageAdded = false

	// Don't add assistant message yet - wait for meaningful content
	// This prevents empty assistant messages from showing

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
	if *isLoading && sm.hasMeaningfulContent() {
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
	// Keep streamContent for final message processing
	finalContent := sm.streamContent
	sm.streamContent = ""

	// Return completion message with final content
	return func() tea.Msg {
		return StreamCompleteInternalMsg{
			Content:      content,
			FinalContent: finalContent, // Include accumulated content for proper message sync
			MessageAdded: sm.messageAdded, // Track if message was added during streaming
			Err:          err,
		}
	}
}

// UpdateDisplay updates the streaming display with accumulated content
func (sm *Manager) UpdateDisplay(content string, renderer interface{}, messages *[]string, viewport ViewportInterface) {
	// Add assistant message only when we have meaningful content for the first time
	if !sm.messageAdded && sm.hasMeaningfulContent() {
		if r, ok := renderer.(interface{ FormatMessage(string, string) string }); ok {
			*messages = append(*messages, r.FormatMessage("assistant", content))
			sm.messageAdded = true
		}
	} else if sm.messageAdded && len(*messages) > 0 {
		// Update the last message (which should be our streaming assistant message)
		lastIdx := len(*messages) - 1
		if r, ok := renderer.(interface{ FormatMessage(string, string) string }); ok {
			(*messages)[lastIdx] = r.FormatMessage("assistant", content)
		}
	}

	// Update viewport content only if we have messages
	if len(*messages) > 0 {
		viewport.SetContent(strings.Join(*messages, "\n\n"))
		_ = viewport.GotoBottom() // Ignore return value
	}
}

// ViewportInterface defines required viewport methods
type ViewportInterface interface {
	SetContent(string)
	GotoBottom() []string
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

// GetStream returns the current stream reader as api.StreamReader
func (sm *Manager) GetStream() api.StreamReader {
	if reader, ok := sm.streamReader.(api.StreamReader); ok {
		return reader
	}
	return nil
}

// AppendContent appends content to the stream
func (sm *Manager) AppendContent(content string) {
	sm.streamContent += content
}

// IsActive returns whether streaming is currently active
func (sm *Manager) IsActive() bool {
	return sm.isActive
}

// Reset resets the streaming state
func (sm *Manager) Reset() {
	sm.streamContent = ""
	sm.isActive = false
	sm.messageAdded = false
	if sm.streamReader != nil {
		sm.streamReader.Close()
		sm.streamReader = nil
	}
}

// hasMeaningfulContent checks if the content has substantial text (not just whitespace/tokens)
func (sm *Manager) hasMeaningfulContent() bool {
	trimmed := strings.TrimSpace(sm.streamContent)
	if len(trimmed) < 5 {
		return false // Too short to be meaningful
	}

	// Count actual letters and meaningful characters
	letterCount := 0
	for _, r := range trimmed {
		if unicode.IsLetter(r) || unicode.IsDigit(r) {
			letterCount++
		}
	}

	// Require at least 3 letters/digits and content longer than just tokens
	return letterCount >= 3 && len(trimmed) >= 8
}

// StreamCompleteInternalMsg is used internally for stream completion
type StreamCompleteInternalMsg struct {
	Content      string
	FinalContent string // Final accumulated content from streaming
	MessageAdded bool   // Whether assistant message was added during streaming
	Err          error
}