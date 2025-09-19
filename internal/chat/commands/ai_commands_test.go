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
	"testing"

	"github.com/antenore/deecli/internal/files"
)

func TestGetFileFromRecentContext(t *testing.T) {
	tests := []struct {
		name         string
		messages     []string
		loadedFiles  []files.LoadedFile
		expectedFile string
	}{
		{
			name:         "no messages",
			messages:     []string{},
			loadedFiles:  []files.LoadedFile{},
			expectedFile: "",
		},
		{
			name:     "no loaded files",
			messages: []string{"user: Can you help with main.go?"},
			loadedFiles: []files.LoadedFile{},
			expectedFile: "",
		},
		{
			name:     "file mentioned in recent message",
			messages: []string{"user: Can you help with main.go?"},
			loadedFiles: []files.LoadedFile{
				{RelPath: "main.go", Path: "/path/to/main.go"},
				{RelPath: "README.md", Path: "/path/to/README.md"},
			},
			expectedFile: "main.go",
		},
		{
			name: "skip AI responses, find user message",
			messages: []string{
				"DeeCLI: I can help with that",
				"user: Let's look at TODO.md",
				"system: File loaded",
			},
			loadedFiles: []files.LoadedFile{
				{RelPath: "TODO.md", Path: "/path/to/TODO.md"},
				{RelPath: "main.go", Path: "/path/to/main.go"},
			},
			expectedFile: "TODO.md",
		},
		{
			name: "most recent user message takes priority",
			messages: []string{
				"user: First let's check main.go",
				"DeeCLI: Looking at main.go...",
				"user: Actually, let's edit README.md instead",
			},
			loadedFiles: []files.LoadedFile{
				{RelPath: "main.go", Path: "/path/to/main.go"},
				{RelPath: "README.md", Path: "/path/to/README.md"},
			},
			expectedFile: "README.md",
		},
		{
			name: "basename matching works",
			messages: []string{"user: Can you edit config.go?"},
			loadedFiles: []files.LoadedFile{
				{RelPath: "internal/config/config.go", Path: "/path/to/internal/config/config.go"},
			},
			expectedFile: "internal/config/config.go",
		},
		{
			name: "partial path matching",
			messages: []string{"user: Let's look at chat/model.go"},
			loadedFiles: []files.LoadedFile{
				{RelPath: "internal/chat/model.go", Path: "/path/to/internal/chat/model.go"},
			},
			expectedFile: "internal/chat/model.go",
		},
		{
			name: "no matching file in loaded files",
			messages: []string{"user: Can you help with server.go?"},
			loadedFiles: []files.LoadedFile{
				{RelPath: "main.go", Path: "/path/to/main.go"},
				{RelPath: "README.md", Path: "/path/to/README.md"},
			},
			expectedFile: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create AI commands with mock dependencies
			deps := Dependencies{
				Messages: tt.messages,
				FileContext: &files.FileContext{
					Files: tt.loadedFiles,
				},
			}
			ai := &AICommands{deps: deps}

			result := ai.getFileFromRecentContext()

			if result != tt.expectedFile {
				t.Errorf("getFileFromRecentContext() = %q, want %q", result, tt.expectedFile)
			}
		})
	}
}

func TestExtractFileFromMessage(t *testing.T) {
	tests := []struct {
		name        string
		message     string
		loadedFiles []files.LoadedFile
		expectedFile string
	}{
		{
			name:        "simple file mention",
			message:     "Can you help with main.go?",
			loadedFiles: []files.LoadedFile{{RelPath: "main.go", Path: "/path/to/main.go"}},
			expectedFile: "main.go",
		},
		{
			name:        "file with path",
			message:     "Let's check src/utils/helper.js",
			loadedFiles: []files.LoadedFile{{RelPath: "src/utils/helper.js", Path: "/path/to/src/utils/helper.js"}},
			expectedFile: "src/utils/helper.js",
		},
		{
			name:        "multiple files, returns first match",
			message:     "Compare main.go and test.go",
			loadedFiles: []files.LoadedFile{
				{RelPath: "main.go", Path: "/path/to/main.go"},
				{RelPath: "test.go", Path: "/path/to/test.go"},
			},
			expectedFile: "main.go",
		},
		{
			name:        "no file extension",
			message:     "Check the readme file",
			loadedFiles: []files.LoadedFile{{RelPath: "README.md", Path: "/path/to/README.md"}},
			expectedFile: "",
		},
		{
			name:        "file not in loaded files",
			message:     "Look at server.go",
			loadedFiles: []files.LoadedFile{{RelPath: "main.go", Path: "/path/to/main.go"}},
			expectedFile: "",
		},
		{
			name:        "basename matching",
			message:     "Edit config.go",
			loadedFiles: []files.LoadedFile{{RelPath: "internal/config/config.go", Path: "/path/to/internal/config/config.go"}},
			expectedFile: "internal/config/config.go",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			deps := Dependencies{
				FileContext: &files.FileContext{
					Files: tt.loadedFiles,
				},
			}
			ai := &AICommands{deps: deps}

			result := ai.extractFileFromMessage(tt.message)

			if result != tt.expectedFile {
				t.Errorf("extractFileFromMessage() = %q, want %q", result, tt.expectedFile)
			}
		})
	}
}