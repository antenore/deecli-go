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
	"os"
	"os/exec"
	"strings"

	"github.com/antenore/deecli/internal/config"
	tea "github.com/charmbracelet/bubbletea"
)

// ConfigCommands handles configuration-related chat commands
type ConfigCommands struct {
	deps Dependencies
}

// NewConfigCommands creates a new config commands handler
func NewConfigCommands(deps Dependencies) *ConfigCommands {
	return &ConfigCommands{deps: deps}
}

// Config handles the /config command
func (cc *ConfigCommands) Config(args []string) tea.Cmd {
	if len(args) < 1 {
		// No arguments - show current configuration
		cc.showConfig()
		return nil
	}

	cc.handleConfigCommand(args)
	return nil
}

// KeySetup handles the /keysetup command
func (cc *ConfigCommands) KeySetup(args []string) tea.Cmd {
	if len(args) > 0 {
		keyType := args[0]
		switch keyType {
		case "newline", "break":
			cc.deps.SetKeyDetection(true, "newline")
		case "history-back", "back", "previous":
			cc.deps.SetKeyDetection(true, "history-back")
		case "history-forward", "forward", "next":
			cc.deps.SetKeyDetection(true, "history-forward")
		default:
			cc.deps.MessageLogger("system", "Unknown key type: "+keyType)
			cc.deps.MessageLogger("system", "Usage: /keysetup [newline|history-back|history-forward]")
		}
	} else {
		// No argument - show current key bindings
		cc.showKeyBindings()
	}
	return nil
}

// History handles the /history command
func (cc *ConfigCommands) History(args []string) tea.Cmd {
	if len(args) > 0 {
		subCmd := args[0]
		switch subCmd {
		case "clear":
			cc.deps.InputHistory = cc.deps.InputHistory[:0] // Clear slice
			if cc.deps.HistoryManager != nil {
				if err := cc.deps.HistoryManager.Clear(); err != nil {
					cc.deps.MessageLogger("system", fmt.Sprintf("❌ Failed to clear persistent history: %v", err))
				} else {
					cc.deps.MessageLogger("system", "✅ History cleared (both in-memory and persistent)")
				}
			} else {
				cc.deps.MessageLogger("system", "✅ In-memory history cleared")
			}
		case "show", "list":
			cc.deps.ShowHistory()
		default:
			cc.deps.MessageLogger("system", "Unknown history command: "+subCmd)
			cc.deps.MessageLogger("system", "Usage: /history [show|clear]")
		}
	} else {
		// No argument - show history
		cc.deps.ShowHistory()
	}
	return nil
}

// handleConfigCommand processes specific config subcommands
func (cc *ConfigCommands) handleConfigCommand(args []string) {
	if len(args) == 0 {
		return
	}

	switch args[0] {
	case "show":
		cc.showConfig()
	case "init":
		cc.handleConfigInit()
	case "set":
		if len(args) < 3 {
			cc.deps.MessageLogger("system", "Usage: /config set <key> <value> [--global|--project]")
			cc.deps.MessageLogger("system", "Keys: api-key, model, user-name, temperature, max-tokens, auto-reload-files, auto-reload-debounce, show-reload-notices")
			return
		}
		cc.handleConfigSet(args[1], args[2], args[3:])
	case "get":
		if len(args) < 2 {
			cc.deps.MessageLogger("system", "Usage: /config get <key>")
			cc.deps.MessageLogger("system", "Keys: api-key, model, user-name, temperature, max-tokens, auto-reload-files, auto-reload-debounce, show-reload-notices")
			return
		}
		cc.handleConfigGet(args[1])
	case "editor":
		cc.handleEditorConfig(args[1:])
	case "model":
		if len(args) < 2 {
			cc.deps.MessageLogger("system", "Usage: /config model <model_name> [--global|--project]")
			cc.deps.MessageLogger("system", "Common models: deepseek-chat, deepseek-reasoner")
		} else {
			cc.handleConfigSet("model", args[1], args[2:])
		}
	case "temperature":
		if len(args) < 2 {
			cc.deps.MessageLogger("system", "Usage: /config temperature <0.0-2.0> [--global|--project]")
			cc.deps.MessageLogger("system", "Examples: 0.1 (focused), 0.7 (balanced), 1.5 (creative)")
		} else {
			cc.handleConfigSet("temperature", args[1], args[2:])
		}
	case "max-tokens":
		if len(args) < 2 {
			cc.deps.MessageLogger("system", "Usage: /config max-tokens <number> [--global|--project]")
			cc.deps.MessageLogger("system", "Examples: 2048 (default), 8192 (chat max), 32768 (reasoner default), 65536 (reasoner max)")
		} else {
			cc.handleConfigSet("max-tokens", args[1], args[2:])
		}
	case "help":
		cc.showConfigHelp()
	default:
		cc.deps.MessageLogger("system", fmt.Sprintf("Unknown config command: %s", args[0]))
		cc.showConfigHelp()
	}
}

// showConfig displays current configuration
func (cc *ConfigCommands) showConfig() {
	cc.deps.MessageLogger("system", "📋 Current Configuration:")

	if cc.deps.ConfigManager != nil {
		// Reload config to get latest
		if err := cc.deps.ConfigManager.Load(); err != nil {
			cc.deps.MessageLogger("system", fmt.Sprintf("⚠️ Warning: Failed to load config: %v", err))
		}

		cfg := cc.deps.ConfigManager.Get()

		// Show config sources
		if cc.deps.ConfigManager.GlobalConfigExists() {
			cc.deps.MessageLogger("system", "  ✓ Global config: ~/.deecli/config.yaml")
		}
		if cc.deps.ConfigManager.ProjectConfigExists() {
			cc.deps.MessageLogger("system", "  ✓ Project config: ./.deecli/config.yaml")
		}
		if os.Getenv("DEEPSEEK_API_KEY") != "" {
			cc.deps.MessageLogger("system", "  ✓ Environment: DEEPSEEK_API_KEY")
		}

		cc.deps.MessageLogger("system", "")

		// Show merged configuration with proper masking
		apiKeyDisplay := cfg.APIKey
		if len(apiKeyDisplay) > 8 {
			apiKeyDisplay = apiKeyDisplay[:4] + "..." + apiKeyDisplay[len(apiKeyDisplay)-4:]
		} else if apiKeyDisplay != "" {
			apiKeyDisplay = "****"
		} else {
			apiKeyDisplay = "Not set"
		}

		cc.deps.MessageLogger("system", fmt.Sprintf("  User Name: %s", cfg.UserName))
		cc.deps.MessageLogger("system", fmt.Sprintf("  API Key: %s", apiKeyDisplay))
		cc.deps.MessageLogger("system", fmt.Sprintf("  Model: %s", cfg.Model))
		cc.deps.MessageLogger("system", fmt.Sprintf("  Temperature: %.2f", cfg.Temperature))
		cc.deps.MessageLogger("system", fmt.Sprintf("  Max Tokens: %d", cfg.MaxTokens))
		cc.deps.MessageLogger("system", "")
		cc.deps.MessageLogger("system", "File Auto-Reload:")
		cc.deps.MessageLogger("system", fmt.Sprintf("  Enabled: %t", cfg.AutoReloadFiles))
		cc.deps.MessageLogger("system", fmt.Sprintf("  Debounce: %dms", cfg.AutoReloadDebounce))
		cc.deps.MessageLogger("system", fmt.Sprintf("  Notifications: %t", cfg.ShowReloadNotices))
	} else {
		cc.deps.MessageLogger("system", "⚠️ Config manager not available")
	}

	// Show editor configuration
	editor := os.Getenv("EDITOR")
	if editor == "" {
		editor = os.Getenv("VISUAL")
	}
	if editor == "" {
		editor = "auto-detect"
	}
	cc.deps.MessageLogger("system", fmt.Sprintf("  Editor: %s", editor))
}

// handleEditorConfig processes editor configuration
func (cc *ConfigCommands) handleEditorConfig(args []string) {
	if len(args) == 0 {
		cc.deps.MessageLogger("system", "Usage: /config editor <editor_name> [--save|--global]")
		cc.deps.MessageLogger("system", "Options:")
		cc.deps.MessageLogger("system", "  (no flag)  - Set for current session only")
		cc.deps.MessageLogger("system", "  --save     - Save to project config (./.deecli/config.yaml)")
		cc.deps.MessageLogger("system", "  --global   - Save to global config (~/.deecli/config.yaml)")
		return
	}

	// Parse flags
	editorParts := []string{}
	saveFlag := ""

	for _, arg := range args {
		if arg == "--save" {
			saveFlag = "project"
		} else if arg == "--global" {
			saveFlag = "global"
		} else {
			editorParts = append(editorParts, arg)
		}
	}

	if len(editorParts) == 0 {
		cc.deps.MessageLogger("system", "❌ Editor name required")
		return
	}

	newEditor := strings.Join(editorParts, " ")

	// Verify editor exists
	if _, err := exec.LookPath(strings.Fields(newEditor)[0]); err != nil {
		cc.deps.MessageLogger("system", fmt.Sprintf("⚠️ Warning: Editor '%s' not found in PATH", newEditor))
	}

	// Set for current session
	os.Setenv("EDITOR", newEditor)

	// Handle persistence if requested
	if saveFlag != "" && cc.deps.ConfigManager != nil {
		if saveFlag == "global" {
			cc.deps.MessageLogger("system", fmt.Sprintf("✅ Editor set to: %s", newEditor))
			cc.deps.MessageLogger("system", "💡 Note: Editor setting is environment-based. To persist globally, add to ~/.bashrc:")
			cc.deps.MessageLogger("system", fmt.Sprintf("     export EDITOR=\"%s\"", newEditor))
		} else {
			cc.deps.MessageLogger("system", fmt.Sprintf("✅ Editor set to: %s", newEditor))
			cc.deps.MessageLogger("system", "💡 Note: Editor setting is environment-based. To persist in project, add to shell config or .env")
		}
	} else {
		cc.deps.MessageLogger("system", fmt.Sprintf("✅ Editor set to: %s (current session only)", newEditor))
	}
}

// showKeyBindings displays current key bindings
func (cc *ConfigCommands) showKeyBindings() {
	cc.deps.MessageLogger("system", "🎹 Current Key Bindings:")

	if cc.deps.ConfigManager != nil {
		newlineKey := cc.formatKeyForDisplay(cc.deps.ConfigManager.GetNewlineKey())
		historyBackKey := cc.formatKeyForDisplay(cc.deps.ConfigManager.GetHistoryBackKey())
		historyForwardKey := cc.formatKeyForDisplay(cc.deps.ConfigManager.GetHistoryForwardKey())

		cc.deps.MessageLogger("system", fmt.Sprintf("  • Newline:         %s", newlineKey))
		cc.deps.MessageLogger("system", fmt.Sprintf("  • History Back:    %s", historyBackKey))
		cc.deps.MessageLogger("system", fmt.Sprintf("  • History Forward: %s", historyForwardKey))
	} else {
		cc.deps.MessageLogger("system", "  • Newline:         Ctrl+J (default)")
		cc.deps.MessageLogger("system", "  • History Back:    Ctrl+P (default)")
		cc.deps.MessageLogger("system", "  • History Forward: Ctrl+N (default)")
	}

	cc.deps.MessageLogger("system", "")
	cc.deps.MessageLogger("system", "To change a key binding:")
	cc.deps.MessageLogger("system", "  /keysetup newline        - Configure newline key")
	cc.deps.MessageLogger("system", "  /keysetup history-back   - Configure history back key")
	cc.deps.MessageLogger("system", "  /keysetup history-forward - Configure history forward key")
}

// formatKeyForDisplay formats a key string for user-friendly display
func (cc *ConfigCommands) formatKeyForDisplay(key string) string {
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

// handleConfigInit handles interactive configuration initialization
func (cc *ConfigCommands) handleConfigInit() {
	cc.deps.MessageLogger("system", "🔧 Configuration Initialization")
	cc.deps.MessageLogger("system", "================================")
	cc.deps.MessageLogger("system", "")
	cc.deps.MessageLogger("system", "This will set up your DeeCLI configuration.")
	cc.deps.MessageLogger("system", "")
	cc.deps.MessageLogger("system", "For interactive setup, use the CLI command:")
	cc.deps.MessageLogger("system", "  deecli config init")
	cc.deps.MessageLogger("system", "")
	cc.deps.MessageLogger("system", "Or use /config set commands to configure individual settings:")
	cc.deps.MessageLogger("system", "  /config set api-key <your-key>")
	cc.deps.MessageLogger("system", "  /config set model deepseek-chat")
	cc.deps.MessageLogger("system", "  /config set temperature 0.7")
	cc.deps.MessageLogger("system", "  /config set max-tokens 2048")
}

// handleConfigSet sets a configuration value
func (cc *ConfigCommands) handleConfigSet(key, value string, flags []string) {
	if cc.deps.ConfigManager == nil {
		cc.deps.MessageLogger("system", "❌ Configuration manager not available")
		return
	}

	// Parse flags
	scope := ""
	for _, flag := range flags {
		if flag == "--global" {
			scope = "global"
		} else if flag == "--project" {
			scope = "project"
		}
	}

	// Load current config
	if err := cc.deps.ConfigManager.Load(); err != nil {
		cc.deps.MessageLogger("system", fmt.Sprintf("❌ Failed to load configuration: %v", err))
		return
	}

	cfg := cc.deps.ConfigManager.Get()
	newCfg := *cfg

	// Update the specific field
	var displayValue string
	switch key {
	case "api-key":
		// Validate API key before setting
		if err := config.ValidateAPIKey(value); err != nil {
			cc.deps.MessageLogger("system", fmt.Sprintf("❌ %v", err))
			cc.deps.MessageLogger("system", "   DeepSeek API keys should start with 'sk-'")
			return
		}
		newCfg.APIKey = value
		if len(value) > 8 {
			displayValue = value[:4] + "..." + value[len(value)-4:]
		} else {
			displayValue = "****"
		}
		cc.deps.MessageLogger("system", fmt.Sprintf("✅ API key updated: %s", displayValue))

	case "model":
		// Validate model before setting
		if err := config.ValidateModel(value); err != nil {
			cc.deps.MessageLogger("system", fmt.Sprintf("❌ %v", err))
			return
		}
		newCfg.Model = value
		cc.deps.MessageLogger("system", fmt.Sprintf("✅ Model set to: %s", value))

	case "temperature":
		var temp float64
		if _, err := fmt.Sscanf(value, "%f", &temp); err != nil {
			cc.deps.MessageLogger("system", fmt.Sprintf("❌ Invalid temperature value: %s", value))
			cc.deps.MessageLogger("system", "   Temperature should be a decimal number (e.g., 0.7)")
			return
		}
		if err := config.ValidateTemperature(temp); err != nil {
			cc.deps.MessageLogger("system", fmt.Sprintf("❌ %v", err))
			cc.deps.MessageLogger("system", "   Lower values (0.0-0.5) = more focused, deterministic")
			cc.deps.MessageLogger("system", "   Higher values (0.5-2.0) = more creative, varied")
			return
		}
		newCfg.Temperature = temp
		cc.deps.MessageLogger("system", fmt.Sprintf("✅ Temperature set to: %.2f", temp))

	case "max-tokens":
		var tokens int
		if _, err := fmt.Sscanf(value, "%d", &tokens); err != nil {
			cc.deps.MessageLogger("system", fmt.Sprintf("❌ Invalid max-tokens value: %s", value))
			cc.deps.MessageLogger("system", "   Max tokens should be a positive integer")
			return
		}
		if err := config.ValidateMaxTokens(tokens); err != nil {
			cc.deps.MessageLogger("system", fmt.Sprintf("❌ %v", err))
			cc.deps.MessageLogger("system", "   Limits: deepseek-chat (max 8192), deepseek-reasoner (max 65536)")
			cc.deps.MessageLogger("system", "   Recommended: 8192 for chat, 32768 for reasoner")
			return
		}
		newCfg.MaxTokens = tokens
		cc.deps.MessageLogger("system", fmt.Sprintf("✅ Max tokens set to: %d", tokens))

	case "user-name":
		if err := config.ValidateUserName(value); err != nil {
			cc.deps.MessageLogger("system", fmt.Sprintf("❌ %v", err))
			return
		}
		newCfg.UserName = value
		cc.deps.MessageLogger("system", fmt.Sprintf("✅ User name set to: %s", value))

	case "auto-reload-files":
		var enabled bool
		if value == "true" || value == "1" || value == "yes" || value == "on" {
			enabled = true
		} else if value == "false" || value == "0" || value == "no" || value == "off" {
			enabled = false
		} else {
			cc.deps.MessageLogger("system", fmt.Sprintf("❌ Invalid auto-reload-files value: %s (use true/false)", value))
			return
		}
		newCfg.AutoReloadFiles = enabled
		cc.deps.MessageLogger("system", fmt.Sprintf("✅ Auto-reload files set to: %t", enabled))

	case "auto-reload-debounce":
		var debounce int
		if _, err := fmt.Sscanf(value, "%d", &debounce); err != nil {
			cc.deps.MessageLogger("system", fmt.Sprintf("❌ Invalid auto-reload-debounce value: %s", value))
			cc.deps.MessageLogger("system", "   Debounce should be a number in milliseconds")
			return
		}
		if err := config.ValidateAutoReloadDebounce(debounce); err != nil {
			cc.deps.MessageLogger("system", fmt.Sprintf("❌ %v", err))
			cc.deps.MessageLogger("system", "   Recommended: 100-500ms for most editors")
			return
		}
		newCfg.AutoReloadDebounce = debounce
		cc.deps.MessageLogger("system", fmt.Sprintf("✅ Auto-reload debounce set to: %dms", debounce))

	case "show-reload-notices":
		var show bool
		if value == "true" || value == "1" || value == "yes" || value == "on" {
			show = true
		} else if value == "false" || value == "0" || value == "no" || value == "off" {
			show = false
		} else {
			cc.deps.MessageLogger("system", fmt.Sprintf("❌ Invalid show-reload-notices value: %s (use true/false)", value))
			return
		}
		newCfg.ShowReloadNotices = show
		cc.deps.MessageLogger("system", fmt.Sprintf("✅ Show reload notices set to: %t", show))

	default:
		cc.deps.MessageLogger("system", fmt.Sprintf("❌ Unknown config key: %s", key))
		cc.deps.MessageLogger("system", "Valid keys: api-key, model, temperature, max-tokens, auto-reload-files, auto-reload-debounce, show-reload-notices")
		return
	}

	// Determine where to save
	var err error
	if scope == "global" || (!cc.deps.ConfigManager.ProjectConfigExists() && scope != "project") {
		err = cc.deps.ConfigManager.SaveGlobal(&newCfg)
		if err == nil {
			cc.deps.MessageLogger("system", "   Saved to global config: ~/.deecli/config.yaml")
		}
	} else {
		err = cc.deps.ConfigManager.SaveProject(&newCfg)
		if err == nil {
			cc.deps.MessageLogger("system", "   Saved to project config: ./.deecli/config.yaml")
		}
	}

	if err != nil {
		cc.deps.MessageLogger("system", fmt.Sprintf("❌ Failed to save configuration: %v", err))
		return
	}

	// Reload to apply changes
	if err := cc.deps.ConfigManager.Load(); err != nil {
		cc.deps.MessageLogger("system", fmt.Sprintf("⚠️ Configuration saved but reload failed: %v", err))
	} else {
		cc.deps.MessageLogger("system", "   Configuration reloaded and applied")
	}
}

// handleConfigGet retrieves a configuration value
func (cc *ConfigCommands) handleConfigGet(key string) {
	if cc.deps.ConfigManager == nil {
		cc.deps.MessageLogger("system", "❌ Configuration manager not available")
		return
	}

	// Reload config to get latest
	if err := cc.deps.ConfigManager.Load(); err != nil {
		cc.deps.MessageLogger("system", fmt.Sprintf("⚠️ Warning: Failed to load config: %v", err))
	}

	cfg := cc.deps.ConfigManager.Get()

	switch key {
	case "api-key":
		apiKeyDisplay := cfg.APIKey
		if len(apiKeyDisplay) > 8 {
			apiKeyDisplay = apiKeyDisplay[:4] + "..." + apiKeyDisplay[len(apiKeyDisplay)-4:]
		} else if apiKeyDisplay != "" {
			apiKeyDisplay = "****"
		} else {
			apiKeyDisplay = "Not set"
		}
		cc.deps.MessageLogger("system", fmt.Sprintf("API Key: %s", apiKeyDisplay))

	case "model":
		cc.deps.MessageLogger("system", fmt.Sprintf("Model: %s", cfg.Model))

	case "temperature":
		cc.deps.MessageLogger("system", fmt.Sprintf("Temperature: %.2f", cfg.Temperature))

	case "max-tokens":
		cc.deps.MessageLogger("system", fmt.Sprintf("Max Tokens: %d", cfg.MaxTokens))

	case "user-name":
		cc.deps.MessageLogger("system", fmt.Sprintf("User Name: %s", cfg.UserName))

	case "auto-reload-files":
		cc.deps.MessageLogger("system", fmt.Sprintf("Auto-reload Files: %t", cfg.AutoReloadFiles))

	case "auto-reload-debounce":
		cc.deps.MessageLogger("system", fmt.Sprintf("Auto-reload Debounce: %dms", cfg.AutoReloadDebounce))

	case "show-reload-notices":
		cc.deps.MessageLogger("system", fmt.Sprintf("Show Reload Notices: %t", cfg.ShowReloadNotices))

	default:
		cc.deps.MessageLogger("system", fmt.Sprintf("❌ Unknown config key: %s", key))
		cc.deps.MessageLogger("system", "Valid keys: api-key, model, user-name, temperature, max-tokens, auto-reload-files, auto-reload-debounce, show-reload-notices")
	}
}

// showConfigHelp displays help for config command
func (cc *ConfigCommands) showConfigHelp() {
	cc.deps.MessageLogger("system", "📋 Config Command Help")
	cc.deps.MessageLogger("system", "======================")
	cc.deps.MessageLogger("system", "")
	cc.deps.MessageLogger("system", "Usage: /config [command] [args]")
	cc.deps.MessageLogger("system", "")
	cc.deps.MessageLogger("system", "Commands:")
	cc.deps.MessageLogger("system", "  /config                  - Show current configuration")
	cc.deps.MessageLogger("system", "  /config show             - Show detailed configuration")
	cc.deps.MessageLogger("system", "  /config init             - Initialize configuration")
	cc.deps.MessageLogger("system", "  /config get <key>        - Get a specific config value")
	cc.deps.MessageLogger("system", "  /config set <key> <val>  - Set a config value")
	cc.deps.MessageLogger("system", "")
	cc.deps.MessageLogger("system", "Shortcuts:")
	cc.deps.MessageLogger("system", "  /config model <name>     - Set model quickly")
	cc.deps.MessageLogger("system", "  /config temperature <val> - Set temperature (0.0-2.0)")
	cc.deps.MessageLogger("system", "  /config max-tokens <num>  - Set max tokens")
	cc.deps.MessageLogger("system", "")
	cc.deps.MessageLogger("system", "Flags:")
	cc.deps.MessageLogger("system", "  --global                 - Save to global config")
	cc.deps.MessageLogger("system", "  --project                - Save to project config")
	cc.deps.MessageLogger("system", "")
	cc.deps.MessageLogger("system", "Examples:")
	cc.deps.MessageLogger("system", "  /config model deepseek-reasoner --global")
	cc.deps.MessageLogger("system", "  /config set temperature 0.7")
	cc.deps.MessageLogger("system", "  /config get model")
}