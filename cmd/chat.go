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
	"github.com/antenore/deecli/internal/chat"
	"github.com/spf13/cobra"
)

var continueSession bool

// chatCmd represents the chat command
var chatCmd = &cobra.Command{
	Use:   "chat",
	Short: "Start interactive coding chat session",
	Long: `Start an interactive chat session with DeepSeek AI for discussing code,
getting explanations, and iterative development assistance.

Features:
- File loading with glob pattern support
- Command history and completion  
- Session persistence
- Professional TUI with Bubbletea`,
	Run: func(cmd *cobra.Command, args []string) {
		// Use configuration values
		chatApp := chat.NewChatApp()
		if continueSession {
			if err := chatApp.StartContinueWithConfig(configManager, apiKey, model, temperature, maxTokens); err != nil {
				cmd.PrintErrf("Chat error: %v\n", err)
			}
		} else {
			if err := chatApp.StartNewWithConfig(configManager, apiKey, model, temperature, maxTokens); err != nil {
				cmd.PrintErrf("Chat error: %v\n", err)
			}
		}
	},
}

func init() {
	rootCmd.AddCommand(chatCmd)
	chatCmd.Flags().BoolVar(&continueSession, "continue", false, "Continue previous chat session")
}