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
	"testing"

	"github.com/charmbracelet/bubbles/textarea"
)

// MockCompletionEngine for testing
type MockCompletionEngine struct {
	completions []string
	baseInput   string
}

func (m *MockCompletionEngine) Complete(input string, cursorPos int) ([]string, string) {
	return m.completions, m.baseInput
}

func (m *MockCompletionEngine) ApplyCompletion(baseInput string, cursorPos int, completion string) (string, int) {
	return completion, len(completion)
}

func TestHandleHistoryBack(t *testing.T) {
	tests := []struct {
		name           string
		history        []string
		currentInput   string
		expectedResult string
		expectedOk     bool
	}{
		{
			name:           "Empty history",
			history:        []string{},
			currentInput:   "test",
			expectedResult: "test",
			expectedOk:     false,
		},
		{
			name:           "Navigate to previous item",
			history:        []string{"first", "second", "third"},
			currentInput:   "current",
			expectedResult: "third",
			expectedOk:     true,
		},
		{
			name:           "Navigate through multiple items",
			history:        []string{"first", "second", "third"},
			currentInput:   "",
			expectedResult: "third", // Will go to third, test only checks first navigation
			expectedOk:     true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ta := textarea.New()
			ta.SetValue(tt.currentInput)

			mgr := NewManager(tt.history, nil, &MockCompletionEngine{},
				func(string, string) {}, func() {})

			ok := mgr.HandleHistoryBack(&ta)
			if ok != tt.expectedOk {
				t.Errorf("HandleHistoryBack() = %v, want %v", ok, tt.expectedOk)
			}

			if ta.Value() != tt.expectedResult {
				t.Errorf("textarea value = %q, want %q", ta.Value(), tt.expectedResult)
			}
		})
	}
}

func TestHandleHistoryForward(t *testing.T) {
	tests := []struct {
		name           string
		history        []string
		setupFunc      func(*Manager, *textarea.Model)
		expectedResult string
		expectedOk     bool
	}{
		{
			name:    "No history navigation active",
			history: []string{"first", "second"},
			setupFunc: func(m *Manager, ta *textarea.Model) {
				// No setup, historyIndex is -1
			},
			expectedResult: "",
			expectedOk:     false,
		},
		{
			name:    "Navigate forward from history",
			history: []string{"first", "second", "third"},
			setupFunc: func(m *Manager, ta *textarea.Model) {
				// Navigate back twice to be at "second"
				m.HandleHistoryBack(ta)
				m.HandleHistoryBack(ta)
			},
			expectedResult: "third",
			expectedOk:     true,
		},
		{
			name:    "Navigate to original input",
			history: []string{"first", "second"},
			setupFunc: func(m *Manager, ta *textarea.Model) {
				ta.SetValue("original")
				m.HandleHistoryBack(ta) // Go to "second"
			},
			expectedResult: "original",
			expectedOk:     true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ta := textarea.New()
			ta.SetValue("")

			mgr := NewManager(tt.history, nil, &MockCompletionEngine{},
				func(string, string) {}, func() {})

			if tt.setupFunc != nil {
				tt.setupFunc(mgr, &ta)
			}

			ok := mgr.HandleHistoryForward(&ta)
			if ok != tt.expectedOk {
				t.Errorf("HandleHistoryForward() = %v, want %v", ok, tt.expectedOk)
			}

			if ok && ta.Value() != tt.expectedResult {
				t.Errorf("textarea value = %q, want %q", ta.Value(), tt.expectedResult)
			}
		})
	}
}

func TestHistoryNavigation(t *testing.T) {
	// Test full navigation flow
	ta := textarea.New()
	ta.SetValue("current input")

	history := []string{"cmd1", "cmd2", "cmd3"}
	mgr := NewManager(history, nil, &MockCompletionEngine{},
		func(string, string) {}, func() {})

	// Navigate back through history
	if !mgr.HandleHistoryBack(&ta) {
		t.Error("First HandleHistoryBack should return true")
	}
	if ta.Value() != "cmd3" {
		t.Errorf("Expected 'cmd3', got %q", ta.Value())
	}

	if !mgr.HandleHistoryBack(&ta) {
		t.Error("Second HandleHistoryBack should return true")
	}
	if ta.Value() != "cmd2" {
		t.Errorf("Expected 'cmd2', got %q", ta.Value())
	}

	if !mgr.HandleHistoryBack(&ta) {
		t.Error("Third HandleHistoryBack should return true")
	}
	if ta.Value() != "cmd1" {
		t.Errorf("Expected 'cmd1', got %q", ta.Value())
	}

	// At the end of history, should not navigate further
	if mgr.HandleHistoryBack(&ta) {
		t.Error("HandleHistoryBack at end of history should return false")
	}

	// Navigate forward
	if !mgr.HandleHistoryForward(&ta) {
		t.Error("HandleHistoryForward should return true")
	}
	if ta.Value() != "cmd2" {
		t.Errorf("Expected 'cmd2', got %q", ta.Value())
	}

	if !mgr.HandleHistoryForward(&ta) {
		t.Error("HandleHistoryForward should return true")
	}
	if ta.Value() != "cmd3" {
		t.Errorf("Expected 'cmd3', got %q", ta.Value())
	}

	// Navigate forward to original input
	if !mgr.HandleHistoryForward(&ta) {
		t.Error("HandleHistoryForward to original should return true")
	}
	if ta.Value() != "current input" {
		t.Errorf("Expected 'current input', got %q", ta.Value())
	}
}

func TestAddToHistory(t *testing.T) {
	history := []string{"existing"}
	mgr := NewManager(history, nil, &MockCompletionEngine{},
		func(string, string) {}, func() {})

	// Add new entry
	mgr.AddToHistory("new command")

	inputHistory := mgr.GetInputHistory()
	if len(inputHistory) != 2 {
		t.Errorf("Expected 2 history items, got %d", len(inputHistory))
	}
	if inputHistory[1] != "new command" {
		t.Errorf("Expected 'new command', got %q", inputHistory[1])
	}

	// Verify history index is reset
	if mgr.historyIndex != -1 {
		t.Errorf("Expected historyIndex to be -1, got %d", mgr.historyIndex)
	}
}

func TestTabCompletion(t *testing.T) {
	completions := []string{"file1.go", "file2.go", "file3.go"}
	mockEngine := &MockCompletionEngine{
		completions: completions,
		baseInput:   "/load ",
	}

	mgr := NewManager([]string{}, nil, mockEngine,
		func(string, string) {}, func() {})

	// Initiate tab completion
	if !mgr.HandleTabCompletion("/load fi") {
		t.Error("HandleTabCompletion should return true when completions available")
	}

	comps, idx, show := mgr.GetCompletionState()
	if !show {
		t.Error("Completions should be shown")
	}
	if len(comps) != 3 {
		t.Errorf("Expected 3 completions, got %d", len(comps))
	}
	if idx != 0 {
		t.Errorf("Expected completion index 0, got %d", idx)
	}

	// Test navigation
	ta := textarea.New()
	mgr.HandleCompletionNavigation("down", &ta)
	_, idx, _ = mgr.GetCompletionState()
	if idx != 1 {
		t.Errorf("Expected completion index 1 after down, got %d", idx)
	}

	mgr.HandleCompletionNavigation("up", &ta)
	_, idx, _ = mgr.GetCompletionState()
	if idx != 0 {
		t.Errorf("Expected completion index 0 after up, got %d", idx)
	}

	// Test wrap around
	mgr.HandleCompletionNavigation("up", &ta)
	_, idx, _ = mgr.GetCompletionState()
	if idx != 2 {
		t.Errorf("Expected completion index 2 after wrap up, got %d", idx)
	}

	// Clear completions
	mgr.ClearCompletions()
	_, _, show = mgr.GetCompletionState()
	if show {
		t.Error("Completions should not be shown after clear")
	}
}