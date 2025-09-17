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

	"github.com/antenore/deecli/internal/editor"
	tea "github.com/charmbracelet/bubbletea"
)

// SystemCommands handles system-related chat commands
type SystemCommands struct {
	deps Dependencies
}

// NewSystemCommands creates a new system commands handler
func NewSystemCommands(deps Dependencies) *SystemCommands {
	return &SystemCommands{deps: deps}
}

// Help handles the /help command
func (sc *SystemCommands) Help(args []string) tea.Cmd {
	// Toggle help visibility
	sc.deps.SetHelpVisible(!sc.deps.HelpVisible)
	return nil
}

// Quit handles the /quit and /exit commands
func (sc *SystemCommands) Quit(args []string) tea.Cmd {
	return tea.Quit
}

// Create handles the /create command
func (sc *SystemCommands) Create(args []string) tea.Cmd {
	if len(args) < 1 {
		sc.deps.MessageLogger("system", "Usage: /create <filepath> - Creates a new file with AI suggestions")
		return nil
	}

	// Use the editor module for new file creation
	config := editor.Config{
		MessageProvider: func() []string { return sc.deps.Messages },
		MessageLogger:   sc.deps.MessageLogger,
	}
	return editor.CreateAndEditNewFile(args[0], config)
}

// ShowUnknownCommand handles unknown commands
func (sc *SystemCommands) ShowUnknownCommand(command string) {
	sc.deps.MessageLogger("system", fmt.Sprintf("Unknown command: %s. Type /help for available commands.", command))
}