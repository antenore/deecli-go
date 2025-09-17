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

package history

import (
	"bufio"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"
)

// Entry represents a single history entry
type Entry struct {
	Command   string    `json:"command"`
	Timestamp time.Time `json:"timestamp"`
}

// Manager handles project-specific history persistence
type Manager struct {
	projectDir  string
	historyFile string
	maxEntries  int
}

// NewManager creates a new history manager for the current project
func NewManager() (*Manager, error) {
	// Get current working directory as project root
	projectDir, err := os.Getwd()
	if err != nil {
		return nil, fmt.Errorf("failed to get current directory: %w", err)
	}

	// Create .deecli directory if it doesn't exist
	deecliDir := filepath.Join(projectDir, ".deecli")
	if err := os.MkdirAll(deecliDir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create .deecli directory: %w", err)
	}

	return &Manager{
		projectDir:  projectDir,
		historyFile: filepath.Join(deecliDir, "history.jsonl"),
		maxEntries:  1000, // Keep last 1000 entries
	}, nil
}

// Load reads history from the project-specific file
func (m *Manager) Load() ([]string, error) {
	// Check if history file exists
	if _, err := os.Stat(m.historyFile); os.IsNotExist(err) {
		return []string{}, nil // No history yet
	}

	file, err := os.Open(m.historyFile)
	if err != nil {
		return nil, fmt.Errorf("failed to open history file: %w", err)
	}
	defer file.Close()

	var commands []string
	scanner := bufio.NewScanner(file)

	for scanner.Scan() {
		var entry Entry
		if err := json.Unmarshal(scanner.Bytes(), &entry); err != nil {
			// Skip malformed lines
			continue
		}
		commands = append(commands, entry.Command)
	}

	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("failed to read history file: %w", err)
	}

	return commands, nil
}

// Add appends a new command to history
func (m *Manager) Add(command string) error {
	// Don't save empty commands or duplicates of the last command
	if command == "" {
		return nil
	}

	// Check last entry to avoid consecutive duplicates
	lastCommand, _ := m.GetLast()
	if lastCommand == command {
		return nil
	}

	// Open file in append mode
	file, err := os.OpenFile(m.historyFile, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0644)
	if err != nil {
		return fmt.Errorf("failed to open history file: %w", err)
	}
	defer file.Close()

	entry := Entry{
		Command:   command,
		Timestamp: time.Now(),
	}

	data, err := json.Marshal(entry)
	if err != nil {
		return fmt.Errorf("failed to marshal history entry: %w", err)
	}

	if _, err := file.Write(append(data, '\n')); err != nil {
		return fmt.Errorf("failed to write history entry: %w", err)
	}

	// Trim history if it's too long
	return m.trimHistory()
}

// GetLast returns the last command in history
func (m *Manager) GetLast() (string, error) {
	commands, err := m.Load()
	if err != nil {
		return "", err
	}

	if len(commands) == 0 {
		return "", nil
	}

	return commands[len(commands)-1], nil
}

// trimHistory keeps only the last maxEntries in the file
func (m *Manager) trimHistory() error {
	commands, err := m.Load()
	if err != nil {
		return err
	}

	if len(commands) <= m.maxEntries {
		return nil // No trimming needed
	}

	// Keep only the last maxEntries
	commands = commands[len(commands)-m.maxEntries:]

	// Rewrite the file
	file, err := os.Create(m.historyFile)
	if err != nil {
		return fmt.Errorf("failed to create history file: %w", err)
	}
	defer file.Close()

	for _, cmd := range commands {
		entry := Entry{
			Command:   cmd,
			Timestamp: time.Now(), // We lose original timestamps during trim
		}
		data, err := json.Marshal(entry)
		if err != nil {
			continue
		}
		file.Write(append(data, '\n'))
	}

	return nil
}

// Clear removes all history
func (m *Manager) Clear() error {
	return os.Remove(m.historyFile)
}