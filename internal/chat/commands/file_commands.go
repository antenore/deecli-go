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
	"strings"

	tea "github.com/charmbracelet/bubbletea"
)

// FileCommands handles file-related chat commands
type FileCommands struct {
	deps Dependencies
}

// NewFileCommands creates a new file commands handler
func NewFileCommands(deps Dependencies) *FileCommands {
	return &FileCommands{deps: deps}
}

// Load handles the /load command
func (fc *FileCommands) Load(args []string) tea.Cmd {
	if len(args) < 1 {
		fc.deps.MessageLogger("system", "Usage: /load <filepath>")
		return nil
	}

	// Clear existing file context first
	fc.deps.FileContext.Clear()

	patterns := args
	err := fc.deps.FileContext.LoadFiles(patterns)
	if err != nil {
		fc.deps.MessageLogger("system", fmt.Sprintf("Error loading files: %v", err))
	} else {
		fc.deps.MessageLogger("system", fc.deps.FileContext.GetInfo())
		fc.deps.RefreshUI()
	}
	return nil
}

// Add handles the /add command
func (fc *FileCommands) Add(args []string) tea.Cmd {
	if len(args) < 1 {
		fc.deps.MessageLogger("system", "Usage: /add <filepath>")
		return nil
	}

	patterns := args
	err := fc.deps.FileContext.LoadFiles(patterns)
	if err != nil {
		fc.deps.MessageLogger("system", fmt.Sprintf("Error adding files: %v", err))
	} else {
		fc.deps.MessageLogger("system", fc.deps.FileContext.GetInfo())
		fc.deps.RefreshUI()
	}
	return nil
}

// List handles the /list command
func (fc *FileCommands) List(args []string) tea.Cmd {
	if len(fc.deps.FileContext.Files) == 0 {
		fc.deps.MessageLogger("system", "No files loaded")
	} else {
		fc.deps.MessageLogger("system", fc.deps.FileContext.GetInfo())
	}
	return nil
}

// Clear handles the /clear command
func (fc *FileCommands) Clear(args []string) tea.Cmd {
	fc.deps.FileContext.Clear()
	fc.deps.MessageLogger("system", "All files cleared")
	fc.deps.RefreshUI()
	return nil
}

// Reload handles the /reload command
func (fc *FileCommands) Reload(args []string) tea.Cmd {
	var patterns []string
	if len(args) > 0 {
		patterns = args
	}

	results, err := fc.deps.FileContext.ReloadFiles(patterns)
	if err != nil {
		fc.deps.MessageLogger("system", fmt.Sprintf("❌ Error reloading files: %v", err))
		return nil
	}

	if len(results) == 0 {
		if len(patterns) == 0 {
			fc.deps.MessageLogger("system", "No files loaded to reload")
		} else {
			fc.deps.MessageLogger("system", "No matching loaded files found")
		}
		return nil
	}

	// Show results with changes
	var msg strings.Builder
	changedCount := 0
	unchangedCount := 0
	errorCount := 0

	for _, result := range results {
		if result.Status == "error" {
			errorCount++
		} else if result.Status == "changed" {
			changedCount++
		} else {
			unchangedCount++
		}
	}

	if changedCount > 0 && unchangedCount > 0 {
		msg.WriteString(fmt.Sprintf("✅ Reloaded %d files (%d changed, %d unchanged):\n", len(results), changedCount, unchangedCount))
	} else if changedCount > 0 {
		msg.WriteString(fmt.Sprintf("✅ Reloaded %d files (all changed):\n", len(results)))
	} else if unchangedCount > 0 {
		msg.WriteString(fmt.Sprintf("✅ Reloaded %d files (no changes detected):\n", len(results)))
	}

	for _, result := range results {
		icon := fc.getFileTypeIcon(result.Language)
		if result.Status == "error" {
			msg.WriteString(fmt.Sprintf("  ❌ %s %s - Error: %s\n", icon, result.Path, result.Error))
		} else if result.Status == "changed" {
			oldSizeStr := fc.formatFileSize(result.OldSize)
			newSizeStr := fc.formatFileSize(result.NewSize)
			msg.WriteString(fmt.Sprintf("  %s %s (%s → %s)\n", icon, result.Path, oldSizeStr, newSizeStr))
		} else {
			sizeStr := fc.formatFileSize(result.NewSize)
			msg.WriteString(fmt.Sprintf("  %s %s (%s)\n", icon, result.Path, sizeStr))
		}
	}

	fc.deps.MessageLogger("system", strings.TrimSuffix(msg.String(), "\n"))
	fc.deps.RefreshUI()
	return nil
}

// Helper functions for file operations
func (fc *FileCommands) getFileTypeIcon(language string) string {
	iconMap := map[string]string{
		"go":         "🐹",
		"javascript": "🟨",
		"typescript": "🔷",
		"python":     "🐍",
		"rust":       "🦀",
		"java":       "☕",
		"c":          "⚡",
		"cpp":        "⚡",
		"html":       "🌐",
		"css":        "🎨",
		"json":       "📋",
		"yaml":       "📝",
		"markdown":   "📖",
		"sql":        "🗃️",
		"dockerfile": "🐳",
		"makefile":   "🔨",
		"bash":       "🖥️",
		"text":       "📄",
	}

	if icon, ok := iconMap[language]; ok {
		return icon
	}
	return "📄"
}

func (fc *FileCommands) formatFileSize(bytes int64) string {
	const unit = 1024
	if bytes < unit {
		return fmt.Sprintf("%db", bytes)
	}

	div, exp := int64(unit), 0
	for n := bytes / unit; n >= unit; n /= unit {
		div *= unit
		exp++
	}

	units := []string{"KB", "MB", "GB", "TB"}
	return fmt.Sprintf("%.1f%s", float64(bytes)/float64(div), units[exp])
}