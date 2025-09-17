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

	"github.com/antenore/deecli/internal/config"
	"github.com/charmbracelet/lipgloss"
)

// Layout handles terminal layout calculations and header rendering
type Layout struct {
	configManager *config.Manager
}

// NewLayout creates a new layout manager
func NewLayout(configManager *config.Manager) *Layout {
	return &Layout{configManager: configManager}
}

// CalculateViewportDimensions calculates viewport height and positioning
func (l *Layout) CalculateViewportDimensions(terminalHeight int, showCompletions bool) (height, yPosition int) {
	// Calculate available space
	headerHeight := 1    // Header line
	separatorHeight := 1 // Separator line
	inputHeight := 3     // Textarea height
	completionHeight := 0

	if showCompletions {
		completionHeight = 1
	}

	// Calculate viewport height - subtract an extra 1 for safety
	viewportHeight := terminalHeight - headerHeight - separatorHeight - inputHeight - completionHeight - 1
	if viewportHeight < 3 {
		viewportHeight = 3
	}

	return viewportHeight, headerHeight
}

// CalculateTextareaWidth calculates textarea width based on layout
func (l *Layout) CalculateTextareaWidth(terminalWidth int, sidebarVisible bool) int {
	textareaWidth := terminalWidth - 4
	if sidebarVisible {
		textareaWidth = terminalWidth - 30 // Account for sidebar
	}
	if textareaWidth < 20 {
		textareaWidth = 20 // Minimum width
	}
	return textareaWidth
}

// RenderHeader creates the application header
func (l *Layout) RenderHeader(filesCount int, focusMode string) string {
	headerStyle := lipgloss.NewStyle().
		Background(lipgloss.Color("62")).
		Foreground(lipgloss.Color("230")).
		Padding(0, 1)

	focusIndicator := ""
	switch focusMode {
	case "viewport":
		focusIndicator = " | 📜 CHAT"
	case "sidebar":
		focusIndicator = " | 📁 FILES"
	default:
		focusIndicator = " | ✏️ INPUT"
	}

	// Get newline key display
	newlineKeyDisplay := "Ctrl+J" // Default
	if l.configManager != nil {
		key := l.configManager.GetNewlineKey()
		// Format key for display (e.g., "ctrl+j" -> "Ctrl+J")
		newlineKeyDisplay = l.FormatKeyForDisplay(key)
	}

	header := headerStyle.Render(fmt.Sprintf("DeeCLI | F: %d | NL: %s | F1 | F2 | C-W%s",
		filesCount, newlineKeyDisplay, focusIndicator))

	return header
}

// RenderMainContent creates the main content area with optional sidebar
func (l *Layout) RenderMainContent(chatContent, sidebarContent string, terminalWidth int, sidebarVisible bool, focusMode string) string {
	if !sidebarVisible {
		// Single column: just the viewport
		return chatContent
	}

	// Two-column layout: chat viewport + files sidebar
	sidebarWidth := 25
	chatWidth := terminalWidth - sidebarWidth - 3 // -3 for " │ " separator
	if chatWidth < 40 {
		chatWidth = 40
	}

	// Create styles with focus indicators
	chatBorderColor := lipgloss.Color("244")
	sidebarBorderColor := lipgloss.Color("244")

	if focusMode == "viewport" {
		chatBorderColor = lipgloss.Color("220") // Yellow for focused
	} else if focusMode == "sidebar" {
		sidebarBorderColor = lipgloss.Color("220") // Yellow for focused
	}

	chatStyle := lipgloss.NewStyle().
		Width(chatWidth).
		MaxWidth(chatWidth).
		BorderStyle(lipgloss.RoundedBorder()).
		BorderRight(true).
		BorderForeground(chatBorderColor)

	sidebarStyle := lipgloss.NewStyle().
		Width(sidebarWidth).
		MaxWidth(sidebarWidth).
		BorderStyle(lipgloss.RoundedBorder()).
		BorderLeft(true).
		BorderForeground(sidebarBorderColor)

	// Render both columns with proper width constraints
	chatColumn := chatStyle.Render(chatContent)
	sidebarColumn := sidebarStyle.Render(sidebarContent)

	// Use lipgloss to join them horizontally
	return lipgloss.JoinHorizontal(
		lipgloss.Top,
		chatColumn,
		sidebarColumn,
	)
}

// RenderFooter creates the footer with input area and completions
func (l *Layout) RenderFooter(inputContent string, completions []string, completionIndex int, terminalWidth int) string {
	var footerContent strings.Builder

	// Separator
	separator := strings.Repeat("─", terminalWidth)
	footerContent.WriteString(separator + "\n")

	// Input area
	footerContent.WriteString(inputContent)

	// Add completions if visible
	if len(completions) > 0 {
		completionStyle := lipgloss.NewStyle().
			Foreground(lipgloss.Color("244")).
			Background(lipgloss.Color("235")).
			Padding(0, 1)

		completionList := fmt.Sprintf("Completions (%d/%d): ", completionIndex+1, len(completions))
		for i, comp := range completions {
			if i > 0 {
				completionList += "  "
			}
			if i >= 10 {
				completionList += fmt.Sprintf("... +%d more", len(completions)-10)
				break
			}

			// Highlight the currently selected completion
			if i == completionIndex {
				highlightStyle := lipgloss.NewStyle().
					Foreground(lipgloss.Color("220")).
					Background(lipgloss.Color("235")).
					Bold(true)
				completionList += highlightStyle.Render(comp)
			} else {
				completionList += comp
			}
		}
		footerContent.WriteString("\n" + completionStyle.Render(completionList))
	}

	return footerContent.String()
}

// FormatKeyForDisplay formats a key string for user-friendly display
func (l *Layout) FormatKeyForDisplay(key string) string {
	if key == "" {
		return "Ctrl+J" // Default
	}

	// Split by + and capitalize each part
	parts := strings.Split(key, "+")
	for i, part := range parts {
		switch strings.ToLower(part) {
		case "ctrl":
			parts[i] = "Ctrl"
		case "alt":
			parts[i] = "Alt"
		case "shift":
			parts[i] = "Shift"
		case "enter":
			parts[i] = "Enter"
		default:
			// Uppercase single letters (j -> J, m -> M)
			if len(part) == 1 {
				parts[i] = strings.ToUpper(part)
			} else {
				// Capitalize first letter of words
				parts[i] = strings.Title(part)
			}
		}
	}
	return strings.Join(parts, "+")
}