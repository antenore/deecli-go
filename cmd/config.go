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

package cmd

import (
	"bufio"
	"fmt"
	"os"
	"strings"
	"syscall"

	"github.com/antenore/deecli/internal/config"
	"github.com/spf13/cobra"
	"golang.org/x/term"
)

var configCmd = &cobra.Command{
	Use:   "config",
	Short: "Manage DeeCLI configuration",
	Long:  `Initialize and manage DeeCLI configuration including API keys and model settings.`,
}

var configInitCmd = &cobra.Command{
	Use:   "init",
	Short: "Initialize DeeCLI configuration",
	Long:  `Set up your DeeCLI configuration with API key and default settings.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		return runConfigInit()
	},
}

var configShowCmd = &cobra.Command{
	Use:   "show",
	Short: "Show current configuration",
	Long:  `Display the current merged configuration from all sources.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		return runConfigShow()
	},
}

var configSetCmd = &cobra.Command{
	Use:   "set <key> <value>",
	Short: "Set a configuration value",
	Long:  `Set a configuration value. Valid keys: api-key, model, user-name, temperature, max-tokens`,
	Args:  cobra.ExactArgs(2),
	RunE: func(cmd *cobra.Command, args []string) error {
		return runConfigSet(args[0], args[1])
	},
}

var configModelCmd = &cobra.Command{
	Use:   "model <model>",
	Short: "Set the default model",
	Long:  `Set the default model to use for API requests.`,
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		return runConfigSet("model", args[0])
	},
}

var configEditorCmd = &cobra.Command{
	Use:   "editor <editor>",
	Short: "Set the default editor",
	Long:  `Set the default editor for file editing operations.`,
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		return runConfigSet("editor", args[0])
	},
}

var configTempCmd = &cobra.Command{
	Use:   "temperature <value>",
	Short: "Set the temperature value",
	Long:  `Set the temperature value for API requests (0.0-1.0).`,
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		return runConfigSet("temperature", args[0])
	},
}

var configTokensCmd = &cobra.Command{
	Use:   "max-tokens <value>",
	Short: "Set the max tokens value",
	Long:  `Set the maximum tokens for API requests.`,
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		return runConfigSet("max-tokens", args[0])
	},
}

func init() {
	rootCmd.AddCommand(configCmd)
	configCmd.AddCommand(configInitCmd)
	configCmd.AddCommand(configShowCmd)
	configCmd.AddCommand(configSetCmd)
	configCmd.AddCommand(configModelCmd)
	configCmd.AddCommand(configEditorCmd)
	configCmd.AddCommand(configTempCmd)
	configCmd.AddCommand(configTokensCmd)
}

func runConfigInit() error {
	reader := bufio.NewReader(os.Stdin)
	
	// Check if config already exists
	if configManager.GlobalConfigExists() {
		fmt.Print("Global configuration already exists. Overwrite? (y/N): ")
		response, _ := reader.ReadString('\n')
		response = strings.TrimSpace(strings.ToLower(response))
		if response != "y" && response != "yes" {
			fmt.Println("Configuration unchanged.")
			return nil
		}
	}

	fmt.Println("🔧 DeeCLI Configuration Setup")
	fmt.Println("=============================")
	fmt.Println()

	// Get user name for chat display
	fmt.Print("Enter your display name for chat (default: You): ")
	userNameInput, _ := reader.ReadString('\n')
	userNameInput = strings.TrimSpace(userNameInput)
	if userNameInput == "" {
		userNameInput = "You"
	}

	// Get API key
	fmt.Print("Enter your DeepSeek API key: ")
	bytePassword, err := term.ReadPassword(int(syscall.Stdin))
	if err != nil {
		return fmt.Errorf("failed to read API key: %w", err)
	}
	apiKeyInput := strings.TrimSpace(string(bytePassword))
	fmt.Println() // New line after password input

	if apiKeyInput == "" {
		return fmt.Errorf("API key cannot be empty")
	}

	// Get model preference
	fmt.Printf("Select model (default: deepseek-chat): ")
	modelInput, _ := reader.ReadString('\n')
	modelInput = strings.TrimSpace(modelInput)
	if modelInput == "" {
		modelInput = "deepseek-chat"
	}

	// Get temperature
	fmt.Printf("Temperature (0.0-1.0, default: 0.1): ")
	tempInput, _ := reader.ReadString('\n')
	tempInput = strings.TrimSpace(tempInput)
	tempValue := 0.1
	if tempInput != "" {
		fmt.Sscanf(tempInput, "%f", &tempValue)
	}

	// Get max tokens
	fmt.Printf("Max tokens (default: 2048): ")
	tokensInput, _ := reader.ReadString('\n')
	tokensInput = strings.TrimSpace(tokensInput)
	tokensValue := 2048
	if tokensInput != "" {
		fmt.Sscanf(tokensInput, "%d", &tokensValue)
	}

	// Save configuration
	cfg := &config.Config{
		APIKey:      apiKeyInput,
		Model:       modelInput,
		Temperature: tempValue,
		MaxTokens:   tokensValue,
		UserName:    userNameInput,
		Profiles:    make(map[string]config.Profile),
	}

	if err := configManager.SaveGlobal(cfg); err != nil {
		return fmt.Errorf("failed to save configuration: %w", err)
	}

	fmt.Println()
	fmt.Println("✅ Configuration saved successfully!")
	fmt.Printf("   Location: ~/.deecli/config.yaml\n")
	fmt.Println()
	fmt.Println("You can now use DeeCLI without setting environment variables.")
	fmt.Println("To create project-specific settings, run 'deecli config init' in your project directory.")

	return nil
}

func runConfigShow() error {
	// Reload config to get latest
	if err := configManager.Load(); err != nil {
		return fmt.Errorf("failed to load configuration: %w", err)
	}

	cfg := configManager.Get()

	fmt.Println("📋 Current Configuration")
	fmt.Println("========================")
	fmt.Println()
	
	// Show config sources
	if configManager.GlobalConfigExists() {
		fmt.Println("✓ Global config: ~/.deecli/config.yaml")
	}
	if configManager.ProjectConfigExists() {
		fmt.Println("✓ Project config: ./.deecli/config.yaml")
	}
	if os.Getenv("DEEPSEEK_API_KEY") != "" {
		fmt.Println("✓ Environment: DEEPSEEK_API_KEY")
	}
	
	fmt.Println()
	fmt.Println("Merged Configuration:")
	fmt.Println("--------------------")
	
	// Hide API key for security
	apiKeyDisplay := cfg.APIKey
	if len(apiKeyDisplay) > 8 {
		apiKeyDisplay = apiKeyDisplay[:4] + "..." + apiKeyDisplay[len(apiKeyDisplay)-4:]
	}
	
	fmt.Printf("User Name:    %s\n", cfg.UserName)
	fmt.Printf("API Key:      %s\n", apiKeyDisplay)
	fmt.Printf("Model:        %s\n", cfg.Model)
	fmt.Printf("Temperature:  %.2f\n", cfg.Temperature)
	fmt.Printf("Max Tokens:   %d\n", cfg.MaxTokens)
	
	if cfg.ActiveProfile != "" {
		fmt.Printf("Active Profile: %s\n", cfg.ActiveProfile)
	}
	
	if len(cfg.Profiles) > 0 {
		fmt.Println("\nProfiles:")
		for name := range cfg.Profiles {
			fmt.Printf("  - %s\n", name)
		}
	}

	return nil
}

func runConfigSet(key, value string) error {
	// Load current config
	if err := configManager.Load(); err != nil {
		return fmt.Errorf("failed to load configuration: %w", err)
	}

	cfg := configManager.Get()
	
	// Create a copy to modify
	newCfg := *cfg
	
	switch key {
	case "api-key":
		newCfg.APIKey = value
		fmt.Printf("✅ API key updated (showing first 4 characters: %s...)\n", value[:min(4, len(value))])
	case "model":
		newCfg.Model = value
		fmt.Printf("✅ Model set to: %s\n", value)
	case "user-name":
		newCfg.UserName = value
		fmt.Printf("✅ User name set to: %s\n", value)
	case "temperature":
		var temp float64
		if _, err := fmt.Sscanf(value, "%f", &temp); err != nil {
			return fmt.Errorf("invalid temperature value: %s", value)
		}
		if temp < 0.0 || temp > 1.0 {
			return fmt.Errorf("temperature must be between 0.0 and 1.0")
		}
		newCfg.Temperature = temp
		fmt.Printf("✅ Temperature set to: %.2f\n", temp)
	case "max-tokens":
		var tokens int
		if _, err := fmt.Sscanf(value, "%d", &tokens); err != nil {
			return fmt.Errorf("invalid max-tokens value: %s", value)
		}
		if tokens <= 0 {
			return fmt.Errorf("max-tokens must be positive")
		}
		newCfg.MaxTokens = tokens
		fmt.Printf("✅ Max tokens set to: %d\n", tokens)
	case "editor":
		// For now, we'll just acknowledge the editor setting
		// In future, this could be stored in a separate editor config field
		fmt.Printf("✅ Default editor preference noted: %s\n", value)
		fmt.Println("   (Editor integration will use this preference in future versions)")
		return nil
	default:
		return fmt.Errorf("unknown config key: %s. Valid keys: api-key, model, user-name, temperature, max-tokens, editor", key)
	}

	// Determine where to save
	if configManager.ProjectConfigExists() {
		if err := configManager.SaveProject(&newCfg); err != nil {
			return fmt.Errorf("failed to save project configuration: %w", err)
		}
		fmt.Println("   Saved to project config: ./.deecli/config.yaml")
	} else if configManager.GlobalConfigExists() {
		if err := configManager.SaveGlobal(&newCfg); err != nil {
			return fmt.Errorf("failed to save global configuration: %w", err)
		}
		fmt.Println("   Saved to global config: ~/.deecli/config.yaml")
	} else {
		if err := configManager.SaveGlobal(&newCfg); err != nil {
			return fmt.Errorf("failed to save global configuration: %w", err)
		}
		fmt.Println("   Created global config: ~/.deecli/config.yaml")
	}

	return nil
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}