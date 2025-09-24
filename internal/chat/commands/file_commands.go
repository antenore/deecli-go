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
	"github.com/antenore/deecli/internal/files"
)

// FileCommands handles file-related chat commands
type FileCommands struct {
	deps Dependencies
}

// NewFileCommands creates a new file commands handler
func NewFileCommands(deps Dependencies) *FileCommands {
	return &FileCommands{deps: deps}
}

// Load handles the /load command - now additive by default
func (fc *FileCommands) Load(args []string) tea.Cmd {
	if len(args) < 1 {
		fc.deps.MessageLogger("system", "Usage: /load <filepath>. Examples: /load *.go, /load main.go, /load src/**/*.py")
		fc.deps.MessageLogger("system", "Use --all flag to bypass .gitignore: /load --all *.js")
		return nil
	}

	// Check for --all flag
	respectGitignore := true
	patterns := args
	if len(args) > 0 && args[0] == "--all" {
		respectGitignore = false
		patterns = args[1:]
		if len(patterns) == 0 {
			fc.deps.MessageLogger("system", "Usage: /load --all <filepath>. Examples: /load --all *.js, /load --all node_modules/**/*.js")
			return nil
		}
	}

	// Temporarily set a different loader if --all is specified
	originalLoader := fc.deps.FileContext.Loader
	if !respectGitignore {
		fc.deps.FileContext.Loader = files.NewFileLoaderWithOptions(false)
		defer func() { fc.deps.FileContext.Loader = originalLoader }()
		fc.deps.MessageLogger("system", "Loading files with --all flag (ignoring .gitignore)")
	}

	err := fc.deps.FileContext.LoadFiles(patterns)
	if err != nil {
		fc.deps.MessageLogger("system", fmt.Sprintf("❌ %v", err))
	} else {
		fc.deps.MessageLogger("system", fc.deps.FileContext.GetInfo())
		fc.deps.RefreshUI()
	}
	return nil
}

// Add handles the /add command
func (fc *FileCommands) Add(args []string) tea.Cmd {
	if len(args) < 1 {
		fc.deps.MessageLogger("system", "Usage: /add <filepath>. Note: /add is deprecated, use /load instead. Examples: /load *.go")
		return nil
	}

	patterns := args
	err := fc.deps.FileContext.LoadFiles(patterns)
	if err != nil {
		fc.deps.MessageLogger("system", fmt.Sprintf("❌ %v", err))
	} else {
		fc.deps.MessageLogger("system", fc.deps.FileContext.GetInfo())
		fc.deps.RefreshUI()
	}
	return nil
}

// List handles the /list command
func (fc *FileCommands) List(args []string) tea.Cmd {
	if len(fc.deps.FileContext.Files) == 0 {
		fc.deps.MessageLogger("system", "No files loaded. Try: /load *.go or /load <filename>")
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

// Unload handles the /unload command for selective file removal
func (fc *FileCommands) Unload(args []string) tea.Cmd {
	if len(args) < 1 {
		fc.deps.MessageLogger("system", "Usage: /unload <pattern>. Examples: /unload *.test.go, /unload temp, /unload *")
		return nil
	}

	pattern := args[0]
	removed := fc.deps.FileContext.UnloadFiles(pattern)
	if removed > 0 {
		fc.deps.MessageLogger("system", fmt.Sprintf("✓ Removed %d file(s) matching '%s'", removed, pattern))
		fc.deps.MessageLogger("system", fc.deps.FileContext.GetInfo())
		fc.deps.RefreshUI()
	} else {
		fc.deps.MessageLogger("system", fmt.Sprintf("No files found matching pattern '%s'. Use /list to see loaded files", pattern))
	}
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
			fc.deps.MessageLogger("system", "No files loaded to reload. Use /load <files> first")
		} else {
			fc.deps.MessageLogger("system", fmt.Sprintf("No matching loaded files found for pattern: %s. Use /list to see loaded files", strings.Join(patterns, ", ")))
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