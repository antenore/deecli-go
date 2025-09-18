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

package input

import (
	"fmt"
	"github.com/antenore/deecli/internal/history"
	"github.com/charmbracelet/bubbles/textarea"
)

// Manager handles input history navigation and completion state
type Manager struct {
	// History navigation
	inputHistory  []string
	historyIndex  int
	tempInput     string
	historyMgr    *history.Manager

	// Completion state
	completions         []string
	showCompletions     bool
	completionIndex     int
	originalInput       string
	completionBaseInput string

	// Dependencies
	completionEngine CompletionEngine
	messageLogger    func(string, string)
	refreshViewport  func()
}

// CompletionEngine interface for completion functionality
type CompletionEngine interface {
	Complete(input string, cursorPos int) ([]string, string)
	ApplyCompletion(baseInput string, cursorPos int, completion string) (string, int)
}

// NewManager creates a new input manager
func NewManager(historyData []string, historyMgr *history.Manager, completionEngine CompletionEngine, messageLogger func(string, string), refreshViewport func()) *Manager {
	return &Manager{
		inputHistory:     historyData,
		historyIndex:     -1,
		historyMgr:       historyMgr,
		completionEngine: completionEngine,
		messageLogger:    messageLogger,
		refreshViewport:  refreshViewport,
	}
}

// HandleHistoryBack navigates backward in history
func (m *Manager) HandleHistoryBack(textarea *textarea.Model) bool {
	if len(m.inputHistory) == 0 {
		return false
	}

	// Save current input if starting history navigation
	if m.historyIndex == -1 {
		m.tempInput = textarea.Value()
	}

	// Move backward in history
	if m.historyIndex < len(m.inputHistory)-1 {
		m.historyIndex++
		textarea.SetValue(m.inputHistory[len(m.inputHistory)-1-m.historyIndex])
		return true
	}
	return false
}

// HandleHistoryForward navigates forward in history
func (m *Manager) HandleHistoryForward(textarea *textarea.Model) bool {
	if m.historyIndex <= -1 {
		return false
	}

	m.historyIndex--

	if m.historyIndex == -1 {
		// Restore original input
		textarea.SetValue(m.tempInput)
	} else {
		// Show history item
		textarea.SetValue(m.inputHistory[len(m.inputHistory)-1-m.historyIndex])
	}
	return true
}

// AddToHistory adds a new entry to input history
func (m *Manager) AddToHistory(input string) {
	m.inputHistory = append(m.inputHistory, input)
	m.historyIndex = -1
	m.tempInput = ""

	// Persist to disk if manager available
	if m.historyMgr != nil {
		m.historyMgr.Add(input)
	}
}

// ShowHistory displays the command history
func (m *Manager) ShowHistory() {
	m.messageLogger("system", "📜 Command History:")

	if len(m.inputHistory) == 0 {
		m.messageLogger("system", "  No history yet")
		m.refreshViewport()
		return
	}

	// Show last 20 entries to avoid overwhelming the display
	start := 0
	if len(m.inputHistory) > 20 {
		start = len(m.inputHistory) - 20
		m.messageLogger("system", fmt.Sprintf("  Showing last 20 of %d entries:", len(m.inputHistory)))
	} else {
		m.messageLogger("system", fmt.Sprintf("  %d entries:", len(m.inputHistory)))
	}

	for i := start; i < len(m.inputHistory); i++ {
		entry := m.inputHistory[i]
		// Truncate long entries
		if len(entry) > 80 {
			entry = entry[:77] + "..."
		}
		m.messageLogger("system", fmt.Sprintf("  %d. %s", i+1, entry))
	}

	historyFile := "in-memory only"
	if m.historyMgr != nil {
		historyFile = ".deecli/history.jsonl"
	}
	m.messageLogger("system", fmt.Sprintf("  History file: %s", historyFile))
	m.refreshViewport()
}

// HandleTabCompletion initiates tab completion
func (m *Manager) HandleTabCompletion(input string) bool {
	cursorPos := len(input)
	completions, originalText := m.completionEngine.Complete(input, cursorPos)

	if len(completions) > 0 {
		m.showCompletions = true
		m.completions = completions
		m.completionIndex = 0
		m.originalInput = originalText
		m.completionBaseInput = input
		return true
	} else {
		m.ClearCompletions()
		return false
	}
}

// HandleCompletionNavigation handles up/down navigation in completions
func (m *Manager) HandleCompletionNavigation(direction string, textarea *textarea.Model) bool {
	if !m.showCompletions || len(m.completions) == 0 {
		return false
	}

	switch direction {
	case "down", "ctrl+n":
		m.completionIndex = (m.completionIndex + 1) % len(m.completions)
		m.updateCompletionPreview(textarea)
	case "up", "ctrl+p":
		if m.completionIndex == 0 {
			m.completionIndex = len(m.completions) - 1
		} else {
			m.completionIndex--
		}
		m.updateCompletionPreview(textarea)
	}
	return true
}

// AcceptCompletion applies the current completion
func (m *Manager) AcceptCompletion(textarea *textarea.Model) bool {
	if !m.showCompletions || m.completionIndex >= len(m.completions) {
		return false
	}

	completion := m.completions[m.completionIndex]
	baseInput := m.completionBaseInput
	newInput, _ := m.completionEngine.ApplyCompletion(baseInput, len(baseInput), completion)
	textarea.SetValue(newInput)

	m.ClearCompletions()
	return true
}

// ClearCompletions clears completion state
func (m *Manager) ClearCompletions() {
	m.showCompletions = false
	m.completions = nil
	m.completionIndex = 0
	m.completionBaseInput = ""
}

// GetCompletionState returns current completion state
func (m *Manager) GetCompletionState() ([]string, int, bool) {
	return m.completions, m.completionIndex, m.showCompletions
}

// GetInputHistory returns the current input history
func (m *Manager) GetInputHistory() []string {
	return m.inputHistory
}

// GetHistoryManager returns the history manager for persistence operations
func (m *Manager) GetHistoryManager() *history.Manager {
	return m.historyMgr
}

// updateCompletionPreview updates the textarea with completion preview
func (m *Manager) updateCompletionPreview(textarea *textarea.Model) {
	if m.completionIndex < len(m.completions) {
		completion := m.completions[m.completionIndex]
		baseInput := m.completionBaseInput
		cursorPos := len(baseInput)
		previewText, _ := m.completionEngine.ApplyCompletion(baseInput, cursorPos, completion)
		textarea.SetValue(previewText)
	}
}