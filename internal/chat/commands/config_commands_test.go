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
	"testing"

	"github.com/antenore/deecli/internal/config"
)

// TestConfigCommands_Config tests the /config command without arguments
func TestConfigCommands_Config(t *testing.T) {
	var messages []string
	messageLogger := func(role, content string) {
		messages = append(messages, content)
	}

	configManager := config.NewManager()

	deps := Dependencies{
		ConfigManager: configManager,
		MessageLogger: messageLogger,
	}

	cc := NewConfigCommands(deps)

	// Test /config with no arguments (should show current config)
	cc.Config([]string{})

	// Check that it shows configuration
	if len(messages) == 0 {
		t.Error("Expected messages to be logged")
	}

	// Check for expected content in messages
	foundConfig := false
	for _, msg := range messages {
		if strings.Contains(msg, "Current Configuration") ||
			strings.Contains(msg, "API Key:") ||
			strings.Contains(msg, "Model:") {
			foundConfig = true
			break
		}
	}

	if !foundConfig {
		t.Error("Expected configuration information to be displayed")
	}
}

// TestConfigCommands_HandleConfigSet tests the config set functionality
func TestConfigCommands_HandleConfigSet(t *testing.T) {
	var messages []string
	messageLogger := func(role, content string) {
		messages = append(messages, content)
	}

	configManager := config.NewManager()

	deps := Dependencies{
		ConfigManager: configManager,
		MessageLogger: messageLogger,
	}

	cc := NewConfigCommands(deps)

	tests := []struct {
		name        string
		key         string
		value       string
		flags       []string
		shouldError bool
	}{
		{"set model", "model", "deepseek-chat", []string{}, false},
		{"set temperature", "temperature", "0.7", []string{}, false},
		{"set invalid temperature", "temperature", "3.0", []string{}, true},
		{"set max-tokens", "max-tokens", "2048", []string{}, false},
		{"set invalid max-tokens", "max-tokens", "-100", []string{}, true},
		{"set unknown key", "unknown", "value", []string{}, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			messages = nil // Clear messages
			cc.handleConfigSet(tt.key, tt.value, tt.flags)

			hasError := false
			hasSuccess := false
			for _, msg := range messages {
				if strings.Contains(msg, "❌") {
					hasError = true
				}
				if strings.Contains(msg, "✅") {
					hasSuccess = true
				}
			}

			if tt.shouldError && !hasError {
				t.Errorf("%s: expected error message but got none", tt.name)
			}
			if !tt.shouldError && !hasSuccess {
				t.Errorf("%s: expected success message but got none", tt.name)
			}
		})
	}
}

// TestConfigCommands_ShowConfigHelp tests the help display
func TestConfigCommands_ShowConfigHelp(t *testing.T) {
	var messages []string
	messageLogger := func(role, content string) {
		messages = append(messages, content)
	}

	deps := Dependencies{
		MessageLogger: messageLogger,
	}

	cc := NewConfigCommands(deps)

	// Test help command
	cc.showConfigHelp()

	// Check that help was shown
	if len(messages) == 0 {
		t.Error("Expected help messages to be logged")
	}

	// Check for expected help content
	foundHelp := false
	for _, msg := range messages {
		if strings.Contains(msg, "Config Command Help") ||
			strings.Contains(msg, "Usage:") ||
			strings.Contains(msg, "Commands:") {
			foundHelp = true
			break
		}
	}

	if !foundHelp {
		t.Error("Expected help information to be displayed")
	}
}
