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

package ui

import (
	"fmt"
	"strings"

	"github.com/antenore/deecli/internal/files"
	"github.com/charmbracelet/lipgloss"
)

// Sidebar handles the files sidebar rendering
type Sidebar struct{}

// NewSidebar creates a new sidebar
func NewSidebar() *Sidebar {
	return &Sidebar{}
}

// RenderFilesSidebar creates the files sidebar content
func (s *Sidebar) RenderFilesSidebar(fileContext *files.FileContext) string {
	var sb strings.Builder

	// Sidebar title
	titleStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("212")).
		Bold(true)
	sb.WriteString(titleStyle.Render("Files") + "\n")
	sb.WriteString(strings.Repeat("─", 22) + "\n")

	if len(fileContext.Files) == 0 {
		sb.WriteString(lipgloss.NewStyle().
			Foreground(lipgloss.Color("244")).
			Render("No files loaded") + "\n")
		sb.WriteString("\n")
		sb.WriteString(lipgloss.NewStyle().
			Foreground(lipgloss.Color("244")).
			Render("Use /load <file>") + "\n")
	} else {
		// List ALL files with icons and sizes (no limit for scrolling)
		totalSize := int64(0)
		for i, file := range fileContext.Files {
			// Get file type icon
			icon := s.GetFileTypeIcon(file.Language)

			// Format file size
			sizeStr := s.FormatFileSize(file.Size)

			// File name (truncate if too long for sidebar width)
			fileName := file.RelPath
			if len(fileName) > 18 {
				fileName = fileName[:15] + "..."
			}

			// File number for future selection
			numberStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("240"))
			fileStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("252"))

			// File entry with number
			sb.WriteString(fmt.Sprintf("%s %s %s\n",
				numberStyle.Render(fmt.Sprintf("%2d.", i+1)),
				icon,
				fileStyle.Render(fileName)))

			// Size and language (indented)
			detailStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("244"))
			sb.WriteString(detailStyle.Render(fmt.Sprintf("     %s • %s", file.Language, sizeStr)) + "\n")

			if i < len(fileContext.Files)-1 {
				sb.WriteString("\n")
			}

			totalSize += file.Size
		}

		// Total context size at bottom
		sb.WriteString("\n")
		sb.WriteString(strings.Repeat("─", 22) + "\n")
		totalStr := s.FormatFileSize(totalSize)
		sb.WriteString(lipgloss.NewStyle().
			Foreground(lipgloss.Color("208")).
			Bold(true).
			Render(fmt.Sprintf("Total: %s", totalStr)) + "\n")
	}

	return sb.String()
}

// GetFileTypeIcon returns an icon for the given file language
func (s *Sidebar) GetFileTypeIcon(language string) string {
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

// FormatFileSize formats bytes into human-readable format
func (s *Sidebar) FormatFileSize(bytes int64) string {
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