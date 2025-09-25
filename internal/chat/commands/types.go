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
	"context"

	"github.com/antenore/deecli/internal/api"
	"github.com/antenore/deecli/internal/chat/tracker"
	"github.com/antenore/deecli/internal/config"
	"github.com/antenore/deecli/internal/files"
	"github.com/antenore/deecli/internal/history"
	"github.com/antenore/deecli/internal/sessions"
	"github.com/antenore/deecli/internal/tools"
	tea "github.com/charmbracelet/bubbletea"
)

// Dependencies defines what each command handler needs from the main model
type Dependencies struct {
	// Core components
	FileContext      *files.FileContext
	APIClient        *api.Service
	ConfigManager    *config.Manager
	SessionManager   *sessions.Manager
	CurrentSession   *sessions.Session
	HistoryManager   *history.Manager
	FileTracker      *tracker.FileTracker
	ToolsRegistry    *tools.Registry

	// UI state
	Messages     []string
	APIMessages  []api.Message
	InputHistory []string
	HelpVisible  bool

	// State management
	MessageLogger func(role, content string)
	SetLoading    func(bool, string) tea.Cmd
	SetCancel     func(context.CancelFunc)
	RefreshUI     func()
	ShowHistory   func() // Show input history

	// AI operations
	AnalyzeFiles func() tea.Cmd
	ExplainFiles func() tea.Cmd
	ImproveFiles func() tea.Cmd
	GenerateEditSuggestions func() tea.Cmd

	// UI control
	SetHelpVisible  func(bool)
	SetKeyDetection func(bool, string)
}