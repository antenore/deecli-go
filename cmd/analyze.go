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
	"github.com/antenore/deecli/internal/api"
	"github.com/antenore/deecli/internal/files"
)

// analyzeCmd represents the analyze command
var analyzeCmd = &cobra.Command{
	Use:   "analyze <file>",
	Short: "Analyze a code file and get suggestions",
	Long: `Analyze a code file and get AI-powered suggestions for improvements,
potential issues, and best practices.`,
	Args: cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		filepath := args[0]
		
		// Check if API key is available
		cfg := configManager.Get()
		if cfg.APIKey == "" {
			fmt.Fprintf(os.Stderr, "❌ No API key found. Please run 'deecli config init' or set DEEPSEEK_API_KEY environment variable.\n")
			os.Exit(1)
		}
		
		fmt.Printf("🔍 Analyzing %s...\n", filepath)
		
		// Load the file
		loader := files.NewFileLoader()
		fileInfo, err := loader.LoadFile(filepath)
		if err != nil {
			fmt.Fprintf(os.Stderr, "❌ Failed to load file: %v\n", err)
			os.Exit(1)
		}
		
		// Create API service
		service := api.NewDeepSeekService(cfg.APIKey, cfg.Model, cfg.Temperature, cfg.MaxTokens)
		
		// Analyze the code
		analysis, err := service.AnalyzeCode(fileInfo.Content, fileInfo.RelPath)
		if err != nil {
			fmt.Fprintf(os.Stderr, "❌ Analysis failed: %v\n", err)
			os.Exit(1)
		}
		
		fmt.Printf("\n📊 Analysis of %s:\n\n%s\n", fileInfo.RelPath, analysis)
	},
}

func init() {
	rootCmd.AddCommand(analyzeCmd)
}