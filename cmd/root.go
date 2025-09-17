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
	"fmt"
	"os"

	"github.com/spf13/cobra"
	"github.com/antenore/deecli/internal/config"
)

var (
	// Config flags
	apiKey      string
	model       string
	temperature float64
	maxTokens   int
	verbose     bool
	quiet       bool

	// Config manager
	configManager *config.Manager
)

// rootCmd represents the base command when called without any subcommands
var rootCmd = &cobra.Command{
	Use:   "deecli",
	Short: "🐉 AI-powered code assistant using DeepSeek models",
	Long: `DeeCLI is a professional AI-powered code assistant using DeepSeek models,
focusing on excellent terminal UX, session persistence, and extensibility.

Built with Go, Cobra, and Bubbletea for maximum performance and reliability.`,
	Version: "0.1.0",
}

// Execute adds all child commands to the root command and sets flags appropriately.
// This is called by main.main(). It only needs to happen once to the rootCmd.
func Execute() error {
	return rootCmd.Execute()
}

func init() {
	cobra.OnInitialize(initConfig)

	// Global flags
	rootCmd.PersistentFlags().StringVar(&apiKey, "api-key", "", "DeepSeek API key (overrides config)")
	rootCmd.PersistentFlags().StringVar(&model, "model", "", "Model to use (overrides config)")
	rootCmd.PersistentFlags().Float64Var(&temperature, "temperature", 0, "Temperature for generation (overrides config)")
	rootCmd.PersistentFlags().IntVar(&maxTokens, "max-tokens", 0, "Maximum tokens to generate (overrides config)")
	rootCmd.PersistentFlags().BoolVar(&verbose, "verbose", false, "Enable verbose output")
	rootCmd.PersistentFlags().BoolVar(&quiet, "quiet", false, "Quiet mode")

	// Hide help command since we have custom help
	rootCmd.SetHelpCommand(&cobra.Command{
		Use:    "no-help",
		Hidden: true,
	})
}

func initConfig() {
	configManager = config.NewManager()
	
	// Load configuration files
	if err := configManager.Load(); err != nil {
		if verbose {
			fmt.Fprintf(os.Stderr, "Warning: Failed to load config: %v\n", err)
		}
	}

	// Apply command-line overrides
	cfg := configManager.Get()
	
	// Command-line flags take precedence
	if apiKey != "" {
		cfg.APIKey = apiKey
	}
	if model != "" {
		cfg.Model = model
	}
	if temperature != 0 {
		cfg.Temperature = temperature
	}
	if maxTokens != 0 {
		cfg.MaxTokens = maxTokens
	}

	// Update the flag values with config values if not set
	if apiKey == "" {
		apiKey = cfg.APIKey
	}
	if model == "" {
		model = cfg.Model
	}
	if temperature == 0 {
		temperature = cfg.Temperature
	}
	if maxTokens == 0 {
		maxTokens = cfg.MaxTokens
	}

	// Check if API key is set when needed
	if !isConfigCommand() && apiKey == "" && !configManager.GlobalConfigExists() {
		fmt.Fprintln(os.Stderr, "❌ No API key found. Please run 'deecli config init' or set DEEPSEEK_API_KEY environment variable.")
		os.Exit(1)
	}
}

func isConfigCommand() bool {
	// Check if the command is a config command (we'll implement this command next)
	args := os.Args[1:]
	return len(args) > 0 && args[0] == "config"
}