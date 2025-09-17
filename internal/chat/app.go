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
	"fmt"
	"github.com/antenore/deecli/internal/config"
	tea "github.com/charmbracelet/bubbletea"
)

// ChatApp represents the main chat application
type ChatApp struct {
	program *tea.Program
}

// NewChatApp creates a new chat application
func NewChatApp() *ChatApp {
	return &ChatApp{}
}

// Start initializes and starts the chat application (legacy method)
func (app *ChatApp) Start() error {
	m := newChatModel()
	
	// Try with alt screen first, fallback to normal mode if TTY issues
	app.program = tea.NewProgram(m, tea.WithAltScreen())
	
	if _, err := app.program.Run(); err != nil {
		// Fallback to basic mode without alt screen
		fmt.Println("Falling back to basic mode...")
		app.program = tea.NewProgram(m)
		_, err = app.program.Run()
		return err
	}
	
	return nil
}

// StartNew initializes and starts the new chat application
func (app *ChatApp) StartNew() error {
	m := newChatModel()
	
	// Use alt screen for full terminal control with proper input handling
	app.program = tea.NewProgram(m, 
		tea.WithAltScreen(),
	)
	
	if _, err := app.program.Run(); err != nil {
		// Fallback to basic mode without alt screen
		fmt.Println("Falling back to basic mode...")
		app.program = tea.NewProgram(m)
		_, err = app.program.Run()
		return err
	}
	
	return nil
}

// StartNewWithConfig initializes and starts the chat application with specific configuration
func (app *ChatApp) StartNewWithConfig(configManager *config.Manager, apiKey, model string, temperature float64, maxTokens int) error {
	m := newChatModelWithConfig(configManager, apiKey, model, temperature, maxTokens)
	
	// Use alt screen for full terminal control with proper input handling
	app.program = tea.NewProgram(m, 
		tea.WithAltScreen(),
	)
	
	if _, err := app.program.Run(); err != nil {
		// Fallback to basic mode without alt screen
		fmt.Println("Falling back to basic mode...")
		app.program = tea.NewProgram(m)
		_, err = app.program.Run()
		return err
	}
	
	return nil
}

// StartContinueWithConfig continues previous session with specific configuration
func (app *ChatApp) StartContinueWithConfig(configManager *config.Manager, apiKey, model string, temperature float64, maxTokens int) error {
	m := newChatModelWithConfig(configManager, apiKey, model, temperature, maxTokens)
	
	// Load previous session messages
	if err := m.loadPreviousSession(); err != nil {
		fmt.Printf("Could not load previous session: %v\n", err)
		fmt.Println("Starting new session instead...")
	}
	
	// Use alt screen for full terminal control with proper input handling
	app.program = tea.NewProgram(m, 
		tea.WithAltScreen(),
	)
	
	if _, err := app.program.Run(); err != nil {
		// Fallback to basic mode without alt screen
		fmt.Println("Falling back to basic mode...")
		app.program = tea.NewProgram(m)
		_, err = app.program.Run()
		return err
	}
	
	return nil
}