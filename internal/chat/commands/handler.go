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
	"strings"

	tea "github.com/charmbracelet/bubbletea"
)

// Handler manages chat command parsing and routing
type Handler struct {
	fileCommands   *FileCommands
	aiCommands     *AICommands
	configCommands *ConfigCommands
	systemCommands *SystemCommands
}

// NewHandler creates a new command handler
func NewHandler(deps Dependencies) *Handler {
	return &Handler{
		fileCommands:   NewFileCommands(deps),
		aiCommands:     NewAICommands(deps),
		configCommands: NewConfigCommands(deps),
		systemCommands: NewSystemCommands(deps),
	}
}

// Handle processes a chat command and returns appropriate tea.Cmd
func (h *Handler) Handle(input string) tea.Cmd {
	parts := strings.Fields(input)
	if len(parts) == 0 {
		return nil
	}

	command := parts[0]
	args := parts[1:]

	switch command {
	// File commands
	case "/load":
		return h.fileCommands.Load(args)
	case "/add":
		return h.fileCommands.Add(args)
	case "/list":
		return h.fileCommands.List(args)
	case "/clear":
		return h.fileCommands.Clear(args)
	case "/reload":
		return h.fileCommands.Reload(args)

	// AI commands
	case "/analyze":
		return h.aiCommands.Analyze(args)
	case "/explain":
		return h.aiCommands.Explain(args)
	case "/improve":
		return h.aiCommands.Improve(args)
	case "/edit":
		return h.aiCommands.Edit(args)

	// Config commands
	case "/config":
		return h.configCommands.Config(args)
	case "/keysetup":
		return h.configCommands.KeySetup(args)
	case "/history":
		return h.configCommands.History(args)

	// System commands
	case "/help":
		return h.systemCommands.Help(args)
	case "/quit", "/exit":
		return h.systemCommands.Quit(args)
	case "/create":
		return h.systemCommands.Create(args)

	default:
		h.systemCommands.ShowUnknownCommand(command)
		return nil
	}
}