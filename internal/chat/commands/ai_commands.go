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
	"path/filepath"
	"regexp"
	"strconv"
	"strings"

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

	loadingCmd := ai.deps.SetLoading(true, "Analyzing files...")
	ai.deps.RefreshUI()
	return tea.Batch(loadingCmd, ai.deps.AnalyzeFiles())
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

	loadingCmd := ai.deps.SetLoading(true, "Explaining code...")
	ai.deps.RefreshUI()
	return tea.Batch(loadingCmd, ai.deps.ExplainFiles())
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

	loadingCmd := ai.deps.SetLoading(true, "Generating improvement suggestions...")
	ai.deps.RefreshUI()
	return tea.Batch(loadingCmd, ai.deps.ImproveFiles())
}

// getFileFromRecentContext analyzes recent user messages to find the most recently mentioned loaded file
func (ai *AICommands) getFileFromRecentContext() string {
	if len(ai.deps.Messages) == 0 || len(ai.deps.FileContext.Files) == 0 {
		return ""
	}

	// Look at the last 5 user messages for file mentions
	messageCount := 0
	for i := len(ai.deps.Messages) - 1; i >= 0 && messageCount < 5; i-- {
		message := ai.deps.Messages[i]

		// Skip AI responses (they typically start with system indicators or have certain patterns)
		if strings.HasPrefix(message, "DeeCLI:") || strings.HasPrefix(message, "🤖") ||
		   strings.Contains(message, "📝") || strings.Contains(message, "system:") {
			continue
		}

		messageCount++

		// Look for file mentions in user messages
		if filePath := ai.extractFileFromMessage(message); filePath != "" {
			return filePath
		}
	}

	return ""
}

// extractFileFromMessage extracts a loaded file path mentioned in a message
func (ai *AICommands) extractFileFromMessage(message string) string {
	// Pattern to match file paths with extensions
	filePattern := regexp.MustCompile(`\b([a-zA-Z0-9_\-./]+\.[a-zA-Z0-9]+)\b`)
	matches := filePattern.FindAllString(message, -1)

	for _, match := range matches {
		// Clean the match
		cleanMatch := strings.TrimSpace(match)

		// Check if this file is currently loaded
		for _, loadedFile := range ai.deps.FileContext.Files {
			// Direct match
			if loadedFile.RelPath == cleanMatch || loadedFile.Path == cleanMatch {
				return loadedFile.RelPath
			}

			// Basename match
			if filepath.Base(loadedFile.RelPath) == filepath.Base(cleanMatch) {
				return loadedFile.RelPath
			}

			// Contains match (for partial paths)
			if strings.Contains(loadedFile.RelPath, cleanMatch) {
				return loadedFile.RelPath
			}
		}
	}

	return ""
}

// showInteractiveFileSelection displays a numbered list of loaded files for user selection
func (ai *AICommands) showInteractiveFileSelection() tea.Cmd {
	if len(ai.deps.FileContext.Files) == 0 {
		ai.deps.MessageLogger("system", "No files loaded. Use /load <file> to load files first, or specify a file with /edit <filepath>")
		return nil
	}

	if len(ai.deps.FileContext.Files) == 1 {
		// Only one file loaded, use it directly
		file := ai.deps.FileContext.Files[0]
		config := editor.Config{
			MessageProvider: func() []string { return ai.deps.Messages },
			MessageLogger:   ai.deps.MessageLogger,
		}
		ai.deps.MessageLogger("system", fmt.Sprintf("📝 Opening only loaded file: %s", file.RelPath))
		return editor.OpenFileWithInstructions(file.RelPath, config)
	}

	// Multiple files - show selection menu
	var fileList strings.Builder
	fileList.WriteString("📝 Edit which file?\n")

	for i, file := range ai.deps.FileContext.Files {
		fileList.WriteString(fmt.Sprintf("[%d] %s\n", i+1, file.RelPath))
	}

	fileList.WriteString("Enter number (1-")
	fileList.WriteString(fmt.Sprintf("%d", len(ai.deps.FileContext.Files)))
	fileList.WriteString(") or filename:")

	ai.deps.MessageLogger("system", fileList.String())
	return nil
}

// Edit handles the /edit command (both with and without arguments)
func (ai *AICommands) Edit(args []string) tea.Cmd {
	if len(args) < 1 {
		// First, try to find a file from recent conversation context
		if contextFile := ai.getFileFromRecentContext(); contextFile != "" {
			config := editor.Config{
				MessageProvider: func() []string { return ai.deps.Messages },
				MessageLogger:   ai.deps.MessageLogger,
			}
			ai.deps.MessageLogger("system", fmt.Sprintf("📝 Opening file from context: %s", contextFile))
			return editor.OpenFileWithInstructions(contextFile, config)
		}

		// If no context, show interactive file selection
		return ai.showInteractiveFileSelection()
	}

	// Check if the argument is a number (for file selection)
	if fileIndex, err := strconv.Atoi(args[0]); err == nil {
		if fileIndex >= 1 && fileIndex <= len(ai.deps.FileContext.Files) {
			selectedFile := ai.deps.FileContext.Files[fileIndex-1]
			config := editor.Config{
				MessageProvider: func() []string { return ai.deps.Messages },
				MessageLogger:   ai.deps.MessageLogger,
			}
			ai.deps.MessageLogger("system", fmt.Sprintf("📝 Opening selected file [%d]: %s", fileIndex, selectedFile.RelPath))
			return editor.OpenFileWithInstructions(selectedFile.RelPath, config)
		} else {
			ai.deps.MessageLogger("system", fmt.Sprintf("Invalid file number. Please use 1-%d", len(ai.deps.FileContext.Files)))
			return nil
		}
	}

	// Open specific file in editor
	config := editor.Config{
		MessageProvider: func() []string { return ai.deps.Messages },
		MessageLogger:   ai.deps.MessageLogger,
	}
	return editor.OpenFileWithInstructions(args[0], config)
}

