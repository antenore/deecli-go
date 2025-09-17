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

package editor

import (
	"fmt"
	"os"
	"os/exec"
	"strings"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/antenore/deecli/internal/utils"
)

// EditorFinishedMsg represents an editor closing event
type EditorFinishedMsg struct {
	Error error
}

// Config holds configuration for editor operations
type Config struct {
	// MessageProvider provides recent chat messages for instruction files
	MessageProvider func() []string
	// MessageLogger logs messages to the chat interface
	MessageLogger func(role, content string)
}

// OpenFileWithInstructions opens a file in the editor with AI-generated instruction file
func OpenFileWithInstructions(filepath string, config Config) tea.Cmd {
	// Create instruction file with context from last messages
	instructionFile := createInstructionFile(filepath, config.MessageProvider)
	
	// Auto-create directories if they don't exist
	if err := ensureDirectoryExists(filepath, config.MessageLogger); err != nil {
		config.MessageLogger("system", fmt.Sprintf("❌ Failed to create directory: %v", err))
		return nil
	}
	
	// Check if this is a new file creation
	if _, err := os.Stat(filepath); os.IsNotExist(err) {
		// Create the new file with basic template
		createNewFileWithTemplate(filepath)
	}
	
	// Find editor with interactive fallback
	editor := findEditor(config.MessageLogger)
	if editor == "" {
		return nil
	}
	
	// Build command - simple approach for any editor
	var c *exec.Cmd
	// Get editor base name for switching logic
	editorParts := strings.Split(editor, "/")
	editorBase := editorParts[len(editorParts)-1]
	
	// Handle different editors with two-file opening
	switch {
	case strings.Contains(editorBase, "vim") || editorBase == "nvim":
		// Vim/NVim: vertical split with target file on left, suggestions on right
		c = exec.Command(editor, "-O", filepath, instructionFile)
	case editorBase == "code":
		// VSCode: open target file first, then suggestions
		c = exec.Command(editor, filepath, instructionFile)
	case editorBase == "emacs":
		// Emacs: open target file first, then suggestions
		c = exec.Command(editor, filepath, instructionFile)
	default:
		// Other editors: target file first, suggestions second
		c = exec.Command(editor, filepath, instructionFile)
	}
	
	config.MessageLogger("system", fmt.Sprintf("📝 Opening %s with instructions in %s", filepath, editor))
	
	return tea.ExecProcess(c, func(err error) tea.Msg {
		// Clean up instruction file
		if instructionFile != "" {
			os.Remove(instructionFile)
		}
		if err != nil {
			return EditorFinishedMsg{Error: err}
		}
		return EditorFinishedMsg{}
	})
}

// CreateAndEditNewFile creates a new file with template and opens it for editing
func CreateAndEditNewFile(filepath string, config Config) tea.Cmd {
	// Always create instruction file for new files
	instructionFile := createInstructionFile(filepath, config.MessageProvider)
	
	// Create the new file with template
	if err := createNewFileWithTemplate(filepath); err != nil {
		config.MessageLogger("system", fmt.Sprintf("❌ Failed to create file: %v", err))
		return nil
	}
	
	config.MessageLogger("system", fmt.Sprintf("✓ Creating new file: %s", filepath))
	
	// Find editor
	editor := os.Getenv("EDITOR")
	if editor == "" {
		editor = os.Getenv("VISUAL")
	}
	if editor == "" {
		// Try to find a common editor
		for _, e := range []string{"nvim", "vim", "vi", "nano", "emacs"} {
			if _, err := exec.LookPath(e); err == nil {
				editor = e
				break
			}
		}
	}
	if editor == "" {
		config.MessageLogger("system", "❌ No editor found. Please set $EDITOR environment variable")
		return nil
	}
	
	// Build command based on editor
	var c *exec.Cmd
	if strings.Contains(editor, "vim") || editor == "nvim" {
		// Use vertical split for vim/nvim
		if instructionFile != "" {
			c = exec.Command(editor, "-O", instructionFile, filepath)
		} else {
			c = exec.Command(editor, filepath)
		}
	} else {
		// For other editors, just open the file
		c = exec.Command(editor, filepath)
	}
	
	return tea.ExecProcess(c, func(err error) tea.Msg {
		// Clean up instruction file
		if instructionFile != "" {
			os.Remove(instructionFile)
		}
		if err != nil {
			return EditorFinishedMsg{Error: err}
		}
		return EditorFinishedMsg{}
	})
}

// OpenFile opens a file directly without instruction files (simple version)
func OpenFile(filepath string, config Config) tea.Cmd {
	// Parse file:line format
	file, line := parseFileAndLine(filepath)
	
	// Find editor
	editor := findEditor(config.MessageLogger)
	if editor == "" {
		return nil
	}
	
	var c *exec.Cmd
	editorParts := strings.Split(editor, "/")
	editorBase := editorParts[len(editorParts)-1]
	
	// Handle line number navigation
	switch {
	case strings.Contains(editorBase, "vim") || editorBase == "nvim":
		if line > 0 {
			c = exec.Command(editor, fmt.Sprintf("+%d", line), file)
		} else {
			c = exec.Command(editor, file)
		}
	case editorBase == "code":
		if line > 0 {
			c = exec.Command(editor, "--goto", fmt.Sprintf("%s:%d", file, line))
		} else {
			c = exec.Command(editor, file)
		}
	case editorBase == "emacs" || editorBase == "nano":
		if line > 0 {
			c = exec.Command(editor, fmt.Sprintf("+%d", line), file)
		} else {
			c = exec.Command(editor, file)
		}
	default:
		// Generic fallback
		c = exec.Command(editor, file)
	}
	
	config.MessageLogger("system", fmt.Sprintf("📝 Opening %s in %s", file, editor))
	
	return tea.ExecProcess(c, func(err error) tea.Msg {
		if err != nil {
			return EditorFinishedMsg{Error: err}
		}
		return EditorFinishedMsg{}
	})
}

// createInstructionFile creates a temporary markdown file with AI suggestions and editing tips
func createInstructionFile(filepath string, messageProvider func() []string) string {
	if messageProvider == nil {
		return ""
	}
	
	// Create temp file for instructions
	tmpfile, err := os.CreateTemp("", "deecli_instructions_*.md")
	if err != nil {
		return ""
	}
	defer tmpfile.Close()
	
	instructions := ""
	instructions += fmt.Sprintf("# DeeCLI Edit Instructions for %s\n\n", filepath)
	
	// Add helpful editor shortcuts
	instructions += "## Quick Editor Tips:\n"
	instructions += "- **Vim/NVim**: Switch panes with `Ctrl+W W`, copy with `yip`, save+exit with `:wq`\n"
	instructions += "- **VSCode**: Use split view, copy suggestions, then edit\n\n"
	
	instructions += "## AI Suggestions:\n\n"
	
	// Get last few relevant messages (clean of ANSI codes)
	messages := messageProvider()
	if len(messages) > 0 {
		// Get last AI response (usually has the suggestions)
		for i := len(messages) - 1; i >= 0 && i >= len(messages)-5; i-- {
			msg := messages[i]
			if strings.Contains(msg, "DeeCLI:") || strings.Contains(msg, "assistant:") {
				// Strip ANSI codes and clean up the content
				cleanMsg := utils.StripANSI(msg)
				// Trim trailing spaces from each line
				lines := strings.Split(cleanMsg, "\n")
				for j, line := range lines {
					lines[j] = strings.TrimRight(line, " \t")
				}
				cleanMsg = strings.Join(lines, "\n")
				instructions += "```\n"
				instructions += cleanMsg + "\n"
				instructions += "```\n\n"
				break
			}
		}
	}
	
	instructions += "## Next Steps:\n"
	instructions += "1. Review the suggestions above\n"
	instructions += "2. Copy relevant code/instructions\n"
	instructions += "3. Switch to the other pane and make changes\n"
	instructions += "4. Save and exit to return to chat\n\n"
	instructions += "---\n"
	instructions += "*This instruction file will be automatically deleted when you close the editor.*\n"
	
	tmpfile.WriteString(instructions)
	return tmpfile.Name()
}

// createNewFileWithTemplate creates a new file with appropriate template based on file extension
func createNewFileWithTemplate(filepath string) error {
	// Determine file type and create appropriate template
	var content string
	
	if strings.HasSuffix(filepath, ".go") {
		content = "package main\n\n// TODO: Implement based on AI suggestions\n"
	} else if strings.HasSuffix(filepath, ".py") {
		content = "#!/usr/bin/env python3\n\n# TODO: Implement based on AI suggestions\n"
	} else if strings.HasSuffix(filepath, ".js") {
		content = "// TODO: Implement based on AI suggestions\n"
	} else if strings.HasSuffix(filepath, ".sh") {
		content = "#!/bin/bash\n\n# TODO: Implement based on AI suggestions\n"
	} else {
		content = "# New file created by DeeCLI\n# TODO: Implement based on AI suggestions\n"
	}
	
	// Create directory if needed
	if idx := strings.LastIndex(filepath, "/"); idx > 0 {
		dir := filepath[:idx]
		if err := os.MkdirAll(dir, 0755); err != nil {
			return fmt.Errorf("failed to create directory: %w", err)
		}
	}
	
	return os.WriteFile(filepath, []byte(content), 0644)
}

// findEditor attempts to find an available editor with interactive fallback
func findEditor(messageLogger func(role, content string)) string {
	// Try environment variables first
	editor := os.Getenv("EDITOR")
	if editor == "" {
		editor = os.Getenv("VISUAL")
	}
	
	// If found, verify it exists
	if editor != "" {
		if _, err := exec.LookPath(editor); err == nil {
			return editor
		}
		// Editor set but not found - inform user and fall back
		messageLogger("system", fmt.Sprintf("⚠️ Editor '%s' not found, searching for alternatives...", editor))
	}
	
	// Try to find a common editor
	commonEditors := []string{"nvim", "vim", "vi", "nano", "emacs", "code"}
	for _, e := range commonEditors {
		if _, err := exec.LookPath(e); err == nil {
			messageLogger("system", fmt.Sprintf("📝 Using editor: %s", e))
			return e
		}
	}
	
	// No editor found - provide helpful message
	messageLogger("system", "❌ No editor found. Please:")
	messageLogger("system", "   1. Install an editor: sudo apt install vim (or nvim, nano, etc.)")
	messageLogger("system", "   2. Set EDITOR environment variable: export EDITOR=vim")
	messageLogger("system", "   3. Or use /config editor <editor_name> (future feature)")
	return ""
}

// ensureDirectoryExists creates parent directories if they don't exist
func ensureDirectoryExists(filepath string, messageLogger func(role, content string)) error {
	lastSlashIndex := strings.LastIndex(filepath, "/")
	if lastSlashIndex == -1 {
		// No directory component, file is in current directory
		return nil
	}
	
	dir := filepath[:lastSlashIndex]
	if dir != "" && dir != filepath {
		if err := os.MkdirAll(dir, 0755); err != nil {
			return fmt.Errorf("failed to create directory %s: %w", dir, err)
		}
		messageLogger("system", fmt.Sprintf("📁 Created directory: %s", dir))
	}
	return nil
}

// parseFileAndLine parses "file:line" format and returns file path and line number
func parseFileAndLine(input string) (string, int) {
	lastColon := strings.LastIndex(input, ":")
	if lastColon == -1 {
		return input, 0
	}
	
	file := input[:lastColon]
	lineStr := input[lastColon+1:]
	
	// Try to parse line number
	var line int
	fmt.Sscanf(lineStr, "%d", &line)
	
	// Check if the file part exists
	if _, err := os.Stat(file); err == nil {
		return file, line
	}
	
	// File doesn't exist, maybe the colon is part of the filename
	return input, 0
}

