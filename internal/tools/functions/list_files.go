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

package functions

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

// ListFiles implements file listing tool function
type ListFiles struct{}

// Name returns the function name
func (l *ListFiles) Name() string {
	return "list_files"
}

// Description returns what this function does
func (l *ListFiles) Description() string {
	return "List files in a directory. Examples: {} lists current dir, {\"recursive\":true} lists all files recursively, {\"path\":\"internal\",\"recursive\":true} lists internal/ recursively"
}

// Parameters returns the JSON schema for parameters
func (l *ListFiles) Parameters() map[string]interface{} {
	return map[string]interface{}{
		"type": "object",
		"properties": map[string]interface{}{
			"path": map[string]interface{}{
				"type":        "string",
				"description": "Directory path to list (default: current directory '.')",
				"default":     ".",
			},
			"pattern": map[string]interface{}{
				"type":        "string",
				"description": "Glob pattern to filter files (e.g., '*.go', '*.md')",
			},
			"recursive": map[string]interface{}{
				"type":        "boolean",
				"description": "List files recursively in all subdirectories (default: false)",
				"default":     false,
			},
		},
		"required": []string{},
		"additionalProperties": false,
	}
}

// Execute lists files in the specified directory
func (l *ListFiles) Execute(ctx context.Context, args json.RawMessage) (string, error) {
	// Parse arguments
	var params struct {
		Path      string `json:"path"`
		Pattern   string `json:"pattern"`
		Recursive bool   `json:"recursive"`
	}

	// Handle empty or invalid arguments by using defaults
	if len(args) == 0 || string(args) == "" || string(args) == "{}" || string(args) == "null" {
		// Use default values
		params.Path = "."
		params.Recursive = false
	} else if err := json.Unmarshal(args, &params); err != nil {
		// Try to be helpful with malformed JSON
		fmt.Fprintf(os.Stderr, "[DEBUG] Failed to parse list_files args: %s, error: %v\n", string(args), err)
		return "", fmt.Errorf("invalid arguments. Use: {} for current dir, {\"recursive\":true} for recursive, or {\"path\":\"dir\",\"recursive\":true}")
	}

	// Default to current directory
	if params.Path == "" {
		params.Path = "."
	}

	// Check if path exists
	info, err := os.Stat(params.Path)
	if err != nil {
		return "", fmt.Errorf("cannot access path %s: %w", params.Path, err)
	}

	// If it's a file, return just that file
	if !info.IsDir() {
		return params.Path, nil
	}

	var files []string

	if params.Recursive {
		// Walk directory tree
		err = filepath.Walk(params.Path, func(path string, info os.FileInfo, err error) error {
			if err != nil {
				return nil // Skip inaccessible paths
			}

			// Skip hidden directories
			if info.IsDir() && strings.HasPrefix(info.Name(), ".") && path != params.Path {
				return filepath.SkipDir
			}

			// Skip directories unless they match pattern
			if info.IsDir() {
				return nil
			}

			// Apply pattern filter if specified
			if params.Pattern != "" {
				matched, err := filepath.Match(params.Pattern, info.Name())
				if err != nil || !matched {
					return nil
				}
			}

			files = append(files, path)
			return nil
		})
		if err != nil {
			return "", fmt.Errorf("failed to walk directory: %w", err)
		}
	} else {
		// List single directory
		entries, err := os.ReadDir(params.Path)
		if err != nil {
			return "", fmt.Errorf("failed to read directory: %w", err)
		}

		for _, entry := range entries {
			// Skip hidden files
			if strings.HasPrefix(entry.Name(), ".") {
				continue
			}

			// Apply pattern filter if specified
			if params.Pattern != "" {
				matched, err := filepath.Match(params.Pattern, entry.Name())
				if err != nil || !matched {
					continue
				}
			}

			path := filepath.Join(params.Path, entry.Name())
			if entry.IsDir() {
				path += "/"
			}
			files = append(files, path)
		}
	}

	// Sort files
	sort.Strings(files)

	if len(files) == 0 {
		if params.Pattern != "" {
			return fmt.Sprintf("No files matching pattern '%s' in %s", params.Pattern, params.Path), nil
		}
		return fmt.Sprintf("No files in %s", params.Path), nil
	}

	return strings.Join(files, "\n"), nil
}