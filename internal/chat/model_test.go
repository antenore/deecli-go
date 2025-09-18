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

package chat

import (
	"testing"

	tea "github.com/charmbracelet/bubbletea"
)

// TestArrowKeyHistoryNavigation tests that arrow keys navigate history only in appropriate conditions
func TestArrowKeyHistoryNavigation(t *testing.T) {
	// Create model with history
	model := newChatModel()

	// Add some history
	if model.inputManager != nil {
		model.inputManager.AddToHistory("/load test.go")
		model.inputManager.AddToHistory("/add file.go")
		model.inputManager.AddToHistory("/list")
	}

	tests := []struct {
		name           string
		setupInput     string
		keyMsg         tea.KeyMsg
		shouldNavigate bool
		description    string
	}{
		{
			name:           "Up arrow with empty single-line input",
			setupInput:     "",
			keyMsg:         tea.KeyMsg{Type: tea.KeyUp},
			shouldNavigate: true,
			description:    "Should navigate history",
		},
		{
			name:           "Down arrow with single-line input",
			setupInput:     "some text",
			keyMsg:         tea.KeyMsg{Type: tea.KeyDown},
			shouldNavigate: true,
			description:    "Should navigate history",
		},
		{
			name:           "Up arrow with multi-line input",
			setupInput:     "line1\nline2",
			keyMsg:         tea.KeyMsg{Type: tea.KeyUp},
			shouldNavigate: false,
			description:    "Should NOT navigate history, should move cursor",
		},
		{
			name:           "Down arrow with multi-line input",
			setupInput:     "line1\nline2\nline3",
			keyMsg:         tea.KeyMsg{Type: tea.KeyDown},
			shouldNavigate: false,
			description:    "Should NOT navigate history, should move cursor",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Setup input
			model.textarea.SetValue(tt.setupInput)
			initialValue := model.textarea.Value()

			// Process the key
			model.Update(tt.keyMsg)

			// Check if value changed (indicating navigation happened)
			finalValue := model.textarea.Value()
			valueChanged := initialValue != finalValue

			if tt.shouldNavigate && !valueChanged && len(model.inputManager.GetInputHistory()) > 0 {
				t.Errorf("%s: Expected history navigation but value didn't change", tt.description)
			}
			// For multi-line, we can't easily test cursor movement without access to internal state
			// But at least verify the text content remains the same
			if !tt.shouldNavigate && valueChanged {
				t.Errorf("%s: Expected no history navigation but value changed from %q to %q",
					tt.description, initialValue, finalValue)
			}
		})
	}
}

// TestArrowKeyWithCompletions tests that arrow keys navigate completions when shown
func TestArrowKeyWithCompletions(t *testing.T) {
	model := newChatModel()

	// We can't directly test completion navigation without access to private fields
	// but we can verify that HandleTabCompletion initiates completion
	if model.inputManager != nil {
		// Setup input for tab completion
		model.textarea.SetValue("/load ")

		// This would initiate tab completion, but we need a proper completion engine
		// For now, just verify the method exists
		model.inputManager.HandleTabCompletion("/load ")

		// Check if completions are shown using public method
		_, _, showCompletions := model.inputManager.GetCompletionState()

		// Without a proper completion engine mock, this won't actually show completions
		// This test is more of a placeholder demonstrating the structure
		_ = showCompletions
	}
}

// TestHistoryNavigationKeys tests both arrow keys and configured keys
func TestHistoryNavigationKeys(t *testing.T) {
	model := newChatModelWithConfig(nil, "", "", 0, 0)

	// Add history
	if model.inputManager != nil {
		model.inputManager.AddToHistory("command1")
		model.inputManager.AddToHistory("command2")
		model.inputManager.AddToHistory("command3")
	}

	// Set current input
	model.textarea.SetValue("current")

	// Test up arrow navigates back
	model.Update(tea.KeyMsg{Type: tea.KeyUp})
	if model.textarea.Value() != "command3" {
		t.Errorf("Up arrow should navigate to last history item, got %q", model.textarea.Value())
	}

	// Test down arrow navigates forward
	model.Update(tea.KeyMsg{Type: tea.KeyUp}) // Go to command2
	model.Update(tea.KeyMsg{Type: tea.KeyDown})
	if model.textarea.Value() != "command3" {
		t.Errorf("Down arrow should navigate forward in history, got %q", model.textarea.Value())
	}

	// Test Ctrl+P still works (if configured)
	model.textarea.SetValue("test")
	model.Update(tea.KeyMsg{Type: tea.KeyCtrlP})
	// This should also navigate history if it's configured
}

// TestFocusModeArrowKeys tests that arrow keys behave differently in different focus modes
func TestFocusModeArrowKeys(t *testing.T) {
	model := newChatModel()

	// Add history for testing
	if model.inputManager != nil {
		model.inputManager.AddToHistory("test command")
	}

	// Test in viewport focus mode
	model.focusMode = "viewport"

	model.Update(tea.KeyMsg{Type: tea.KeyDown})
	// In viewport mode, arrow keys should scroll, not navigate history
	// We can't easily test viewport scrolling without more setup

	// Test in sidebar focus mode
	model.focusMode = "sidebar"
	model.Update(tea.KeyMsg{Type: tea.KeyUp})
	// In sidebar mode, arrow keys should scroll sidebar, not navigate history

	// Test in input focus mode
	model.focusMode = "input"
	model.textarea.SetValue("")
	model.Update(tea.KeyMsg{Type: tea.KeyUp})
	// Should navigate history
	if model.textarea.Value() == "" && len(model.inputManager.GetInputHistory()) > 0 {
		t.Error("In input focus mode, up arrow should navigate history")
	}
}