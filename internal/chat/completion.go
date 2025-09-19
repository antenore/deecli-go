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
	"os"
	"path/filepath"
	"strings"
)

type CompletionEngine struct {
	commands []string
}

func NewCompletionEngine() *CompletionEngine {
	return &CompletionEngine{
		commands: []string{
			"/load",
			"/add",
			"/list",
			"/clear",
			"/reload",
			"/analyze",
			"/edit",
			"/create",
			"/improve",
			"/explain",
			"/history",
			"/keysetup",
			"/config",
			"/help",
			"/quit",
			"/exit",
			"/sessions",
		},
	}
}

func (ce *CompletionEngine) Complete(input string, cursorPos int) ([]string, string) {
	if cursorPos > len(input) {
		cursorPos = len(input)
	}

	prefix := input[:cursorPos]

	if strings.HasPrefix(prefix, "/") {
		parts := strings.Fields(prefix)
		if len(parts) == 0 {
			return ce.completeCommands("/"), "/"
		}

		if len(parts) == 1 && !strings.Contains(prefix, " ") {
			return ce.completeCommands(parts[0]), parts[0]
		}

		cmd := parts[0]

		// Handle /config subcommands
		if cmd == "/config" {
			if len(parts) == 1 && strings.HasSuffix(prefix, " ") {
				// After "/config "
				return ce.completeConfigSubcommands(""), ""
			} else if len(parts) == 2 && !strings.HasSuffix(prefix, " ") {
				// Typing the subcommand
				return ce.completeConfigSubcommands(parts[1]), parts[1]
			} else if len(parts) == 2 && strings.HasSuffix(prefix, " ") {
				// After "/config <subcommand> "
				subcmd := parts[1]
				if subcmd == "get" || subcmd == "set" {
					return ce.completeConfigKeys(""), ""
				} else if subcmd == "model" {
					return ce.completeModels(""), ""
				}
			} else if len(parts) == 3 && !strings.HasSuffix(prefix, " ") {
				// Typing the third argument
				subcmd := parts[1]
				if subcmd == "get" || subcmd == "set" {
					return ce.completeConfigKeys(parts[2]), parts[2]
				} else if subcmd == "model" {
					return ce.completeModels(parts[2]), parts[2]
				}
			} else if len(parts) == 3 && strings.HasSuffix(prefix, " ") && parts[1] == "set" {
				// After "/config set <key> " - suggest example values
				key := parts[2]
				return ce.completeConfigValues(key, ""), ""
			} else if len(parts) == 4 && !strings.HasSuffix(prefix, " ") && parts[1] == "set" {
				// Typing the value for set
				key := parts[2]
				return ce.completeConfigValues(key, parts[3]), parts[3]
			}
		}

		if cmd == "/load" || cmd == "/add" || cmd == "/reload" || cmd == "/edit" || cmd == "/create" {
			// Find the current word being typed at cursor position
			currentWord, wordStart := ce.getCurrentWord(input, cursorPos)
			if wordStart > 0 { // We're after the command
				return ce.completeFilePath(currentWord), currentWord
			} else if strings.HasSuffix(prefix, " ") {
				return ce.completeFilePath(""), ""
			}
		}
	}

	return nil, ""
}

func (ce *CompletionEngine) completeCommands(prefix string) []string {
	var matches []string
	for _, cmd := range ce.commands {
		if strings.HasPrefix(cmd, prefix) {
			matches = append(matches, cmd)
		}
	}
	return matches
}

func (ce *CompletionEngine) completeFilePath(prefix string) []string {
	originalPrefix := prefix
	
	// Handle empty prefix specially - don't treat it as "."
	if prefix == "" {
		dir := "."
		showDotFiles := false
		
		entries, err := os.ReadDir(dir)
		if err != nil {
			return nil
		}

		var matches []string
		for _, entry := range entries {
			name := entry.Name()
			
			if strings.HasPrefix(name, ".") && !showDotFiles {
				continue
			}

			fullPath := name
			if entry.IsDir() {
				fullPath += "/"
			}
			
			matches = append(matches, fullPath)
		}
		
		return matches
	}

	// Handle non-empty prefix
	dir := filepath.Dir(prefix)
	base := filepath.Base(prefix)

	if strings.HasSuffix(prefix, "/") {
		dir = prefix
		base = ""
	}

	if dir == "" {
		dir = "."
	}
	
	// If user didn't type anything (empty original prefix), don't show dot files
	showDotFiles := originalPrefix != "" && (strings.HasPrefix(base, ".") || strings.HasPrefix(originalPrefix, "."))

	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil
	}

	var matches []string
	for _, entry := range entries {
		name := entry.Name()
		
		if strings.HasPrefix(name, ".") && !showDotFiles {
			continue
		}

		if base == "" || strings.HasPrefix(name, base) {
			fullPath := filepath.Join(dir, name)
			if dir == "." {
				fullPath = name
			}
			
			if entry.IsDir() {
				fullPath += "/"
			}
			
			matches = append(matches, fullPath)
		}
	}

	if len(matches) == 0 && base != "" {
		pattern := filepath.Join(dir, base+"*")
		globMatches, err := filepath.Glob(pattern)
		if err == nil {
			for _, match := range globMatches {
				info, err := os.Stat(match)
				if err == nil {
					if info.IsDir() {
						match += "/"
					}
					matches = append(matches, match)
				}
			}
		}
	}

	return matches
}

func (ce *CompletionEngine) ApplyCompletion(input string, cursorPos int, completion string) (string, int) {
	if cursorPos > len(input) {
		cursorPos = len(input)
	}

	if strings.HasPrefix(input, "/") {
		parts := strings.Fields(input)
		
		// Command completion (no space after command yet)
		if len(parts) <= 1 && !strings.Contains(input, " ") {
			newInput := completion + " "
			return newInput, len(completion) + 1
		}

		// File path completion - find the current word and replace it
		currentWord, wordStart := ce.getCurrentWord(input, cursorPos)
		if wordStart >= 0 {
			// Replace the current word with the completion
			before := input[:wordStart]
			after := ""
			
			// Find where current word ends
			wordEnd := wordStart + len(currentWord)
			if wordEnd < len(input) {
				after = input[wordEnd:]
			}
			
			newInput := before + completion + after
			newCursorPos := wordStart + len(completion)
			return newInput, newCursorPos
		}
	}

	return completion, len(completion)
}

func (ce *CompletionEngine) GetCommonPrefix(completions []string) string {
	if len(completions) == 0 {
		return ""
	}
	if len(completions) == 1 {
		return completions[0]
	}

	prefix := completions[0]
	for _, comp := range completions[1:] {
		for !strings.HasPrefix(comp, prefix) && len(prefix) > 0 {
			prefix = prefix[:len(prefix)-1]
		}
		if prefix == "" {
			break
		}
	}
	
	return prefix
}

// getCurrentWord finds the word being typed at the cursor position
func (ce *CompletionEngine) getCurrentWord(input string, cursorPos int) (string, int) {
	if cursorPos > len(input) {
		cursorPos = len(input)
	}

	// Find word boundaries around cursor
	wordStart := cursorPos
	wordEnd := cursorPos

	// Find start of current word (go backwards until space or start)
	for wordStart > 0 && input[wordStart-1] != ' ' {
		wordStart--
	}

	// Find end of current word (go forwards until space or end)
	for wordEnd < len(input) && input[wordEnd] != ' ' {
		wordEnd++
	}

	// Return the current word and its start position
	if wordStart < len(input) {
		return input[wordStart:wordEnd], wordStart
	}

	return "", wordStart
}

// completeConfigSubcommands returns available config subcommands
func (ce *CompletionEngine) completeConfigSubcommands(prefix string) []string {
	subcommands := []string{
		"show", "init", "get", "set",
		"model", "temperature", "max-tokens", "help",
	}

	var matches []string
	for _, subcmd := range subcommands {
		if strings.HasPrefix(subcmd, prefix) {
			matches = append(matches, subcmd)
		}
	}
	return matches
}

// completeConfigKeys returns available configuration keys
func (ce *CompletionEngine) completeConfigKeys(prefix string) []string {
	keys := []string{
		"api-key", "model", "temperature", "max-tokens",
	}

	var matches []string
	for _, key := range keys {
		if strings.HasPrefix(key, prefix) {
			matches = append(matches, key)
		}
	}
	return matches
}

// completeModels returns available model names
func (ce *CompletionEngine) completeModels(prefix string) []string {
	models := []string{
		"deepseek-chat", "deepseek-reasoner",
	}

	var matches []string
	for _, model := range models {
		if strings.HasPrefix(model, prefix) {
			matches = append(matches, model)
		}
	}
	return matches
}

// completeConfigValues suggests values based on the config key
func (ce *CompletionEngine) completeConfigValues(key, prefix string) []string {
	switch key {
	case "model":
		return ce.completeModels(prefix)
	case "temperature":
		values := []string{"0.1", "0.3", "0.5", "0.7", "1.0", "1.5"}
		var matches []string
		for _, val := range values {
			if strings.HasPrefix(val, prefix) {
				matches = append(matches, val)
			}
		}
		return matches
	case "max-tokens":
		values := []string{"1024", "2048", "4096", "8192"}
		var matches []string
		for _, val := range values {
			if strings.HasPrefix(val, prefix) {
				matches = append(matches, val)
			}
		}
		return matches
	default:
		return nil
	}
}
