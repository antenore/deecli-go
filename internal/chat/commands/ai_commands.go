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

package commands

import (
	"fmt"
	"os"
	"strings"

	"github.com/antenore/deecli/internal/chat/tracker"
	"github.com/antenore/deecli/internal/editor"
	tea "github.com/charmbracelet/bubbletea"
)

// AICommands handles AI-related chat commands
type AICommands struct {
	deps Dependencies
}

// NewAICommands creates a new AI commands handler
func NewAICommands(deps Dependencies) *AICommands {
	return &AICommands{deps: deps}
}

// Analyze handles the /analyze command
func (ai *AICommands) Analyze(args []string) tea.Cmd {
	if len(ai.deps.FileContext.Files) == 0 {
		ai.deps.MessageLogger("system", "No files loaded. Use /load to load files first.")
		return nil
	}

	if ai.deps.APIClient == nil {
		ai.deps.MessageLogger("system", "Please set DEEPSEEK_API_KEY environment variable")
		return nil
	}

	ai.deps.SetLoading(true, "Analyzing files...")
	ai.deps.RefreshUI()
	return ai.deps.AnalyzeFiles()
}

// Explain handles the /explain command
func (ai *AICommands) Explain(args []string) tea.Cmd {
	if len(ai.deps.FileContext.Files) == 0 {
		ai.deps.MessageLogger("system", "No files loaded. Use /load to load files first.")
		return nil
	}

	if ai.deps.APIClient == nil {
		ai.deps.MessageLogger("system", "Please set DEEPSEEK_API_KEY environment variable")
		return nil
	}

	ai.deps.SetLoading(true, "Explaining code...")
	ai.deps.RefreshUI()
	return ai.deps.ExplainFiles()
}

// Improve handles the /improve command
func (ai *AICommands) Improve(args []string) tea.Cmd {
	if len(ai.deps.FileContext.Files) == 0 {
		ai.deps.MessageLogger("system", "No files loaded. Use /load to load files first.")
		return nil
	}

	if ai.deps.APIClient == nil {
		ai.deps.MessageLogger("system", "Please set DEEPSEEK_API_KEY environment variable")
		return nil
	}

	ai.deps.SetLoading(true, "Generating improvement suggestions...")
	ai.deps.RefreshUI()
	return ai.deps.ImproveFiles()
}

// Edit handles the /edit command (both with and without arguments)
func (ai *AICommands) Edit(args []string) tea.Cmd {
	if len(args) < 1 {
		// Check if we have cached file suggestions from previous AI responses
		if ai.deps.FileTracker != nil && ai.deps.FileTracker.HasSuggestions() {
			suggestions := ai.deps.FileTracker.GetEditSuggestions()

			// Prioritize files that are currently loaded or exist
			var bestFile *tracker.TrackedFile

			// First, look for files that are currently loaded in the context
			for _, suggestion := range suggestions {
				for _, loadedFile := range ai.deps.FileContext.Files {
					if suggestion.Path == loadedFile.RelPath ||
					   strings.HasSuffix(loadedFile.RelPath, suggestion.Path) {
						bestFile = &suggestion
						break
					}
				}
				if bestFile != nil {
					break
				}
			}

			// If no loaded file matches, check for files that exist on disk
			if bestFile == nil {
				for _, suggestion := range suggestions {
					if fileExists(suggestion.Path) {
						bestFile = &suggestion
						break
					}
				}
			}

			// If still no match, take the most recent suggestion
			if bestFile == nil && len(suggestions) > 0 {
				bestFile = &suggestions[0]
			}

			if bestFile != nil {
				config := editor.Config{
					MessageProvider: func() []string { return ai.deps.Messages },
					MessageLogger:   ai.deps.MessageLogger,
				}
				ai.deps.MessageLogger("system", fmt.Sprintf("📝 Opening suggested file: %s", bestFile.Path))
				if bestFile.Description != "" {
					ai.deps.MessageLogger("system", fmt.Sprintf("   Reason: %s", bestFile.Description))
				}
				return editor.OpenFileWithInstructions(bestFile.Path, config)
			}
		}

		// Fallback to generating new suggestions if no cached files
		if len(ai.deps.FileContext.Files) == 0 {
			ai.deps.MessageLogger("system", "No files loaded. Use /load <file> to load files first, or specify a file with /edit <filepath>")
			return nil
		}

		if ai.deps.APIClient == nil {
			ai.deps.MessageLogger("system", "Please set DEEPSEEK_API_KEY environment variable")
			return nil
		}

		ai.deps.SetLoading(true, "Analyzing conversation for edit suggestions...")
		ai.deps.RefreshUI()
		return ai.deps.GenerateEditSuggestions()
	}

	// Open specific file in editor
	config := editor.Config{
		MessageProvider: func() []string { return ai.deps.Messages },
		MessageLogger:   ai.deps.MessageLogger,
	}
	return editor.OpenFileWithInstructions(args[0], config)
}

// fileExists checks if a file exists on disk
func fileExists(filename string) bool {
	_, err := os.Stat(filename)
	return !os.IsNotExist(err)
}