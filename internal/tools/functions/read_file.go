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
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"os"
	"strings"
)

// ReadFile implements file reading tool function
type ReadFile struct{}

// Name returns the function name
func (r *ReadFile) Name() string {
	return "read_file"
}

// Description returns what this function does
func (r *ReadFile) Description() string {
	return "Read a file. Examples: {\"path\":\"TODO.md\"}, {\"path\":\"main.go\"}, {\"path\":\"internal/api/client.go\"}"
}

// Parameters returns the JSON schema for parameters
func (r *ReadFile) Parameters() map[string]interface{} {
	return map[string]interface{}{
		"type": "object",
		"properties": map[string]interface{}{
			"path": map[string]interface{}{
				"type":        "string",
				"description": "File path to read (required). Examples: 'TODO.md', 'main.go', 'internal/api/client.go'",
			},
			"startLine": map[string]interface{}{
				"type":        "integer",
				"description": "Starting line number (1-based, optional)",
				"minimum":     1,
			},
			"endLine": map[string]interface{}{
				"type":        "integer",
				"description": "Ending line number (1-based, optional)",
				"minimum":     1,
			},
		},
		"required": []string{"path"},
		"additionalProperties": false,
	}
}

// Execute reads the specified file
func (r *ReadFile) Execute(ctx context.Context, args json.RawMessage) (string, error) {
	// Parse arguments
	var params struct {
		Path      string `json:"path"`
		StartLine int    `json:"startLine"`
		EndLine   int    `json:"endLine"`
	}

	// Handle empty or invalid arguments - provide clear guidance
	argStr := string(args)
	if argStr == "" || argStr == "null" || argStr == "{}" {
		return "", fmt.Errorf("path is required. Use: {\"path\":\"filename\"} e.g., {\"path\":\"TODO.md\"}")
	}

	if err := json.Unmarshal(args, &params); err != nil {
		// Try to be helpful with common mistakes
		fmt.Fprintf(os.Stderr, "[DEBUG] Failed to parse read_file args: %s, error: %v\n", argStr, err)
		
		// Check if AI sent just a string instead of JSON object
		trimmed := strings.Trim(argStr, `"' `)
		if !strings.Contains(trimmed, "{") && !strings.Contains(trimmed, "}") {
			// Likely just sent "filename" instead of {"path": "filename"}
			return "", fmt.Errorf("invalid format. Use: {\"path\":\"%s\"} not just \"%s\"", trimmed, trimmed)
		}
		return "", fmt.Errorf("invalid JSON format. Use: {\"path\":\"filename\"} e.g., {\"path\":\"TODO.md\"}")
	}

	if params.Path == "" {
		return "", fmt.Errorf("path is required. Use: {\"path\":\"filename\"} e.g., {\"path\":\"TODO.md\"}")
	}

	// Open the file
	file, err := os.Open(params.Path)
	if err != nil {
		// Provide helpful suggestions for common issues
		if os.IsNotExist(err) {
			return "", fmt.Errorf("file not found: %s. Use list_files to see available files", params.Path)
		}
		return "", fmt.Errorf("cannot open %s: %w", params.Path, err)
	}
	defer file.Close()

	// Check if file is too large (limit to 1MB for safety)
	info, err := file.Stat()
	if err != nil {
		return "", fmt.Errorf("cannot stat file: %w", err)
	}
	if info.Size() > 1024*1024 {
		return "", fmt.Errorf("file too large (%d bytes), limit is 1MB. Consider using startLine/endLine to read portions", info.Size())
	}

	// Read file with optional line range
	scanner := bufio.NewScanner(file)
	var lines []string
	lineNum := 0

	for scanner.Scan() {
		lineNum++

		// Skip lines before start
		if params.StartLine > 0 && lineNum < params.StartLine {
			continue
		}

		// Stop at end line
		if params.EndLine > 0 && lineNum > params.EndLine {
			break
		}

		// Add line with number
		lines = append(lines, fmt.Sprintf("%4d: %s", lineNum, scanner.Text()))
	}

	if err := scanner.Err(); err != nil {
		return "", fmt.Errorf("error reading file: %w", err)
	}

	if len(lines) == 0 {
		if params.StartLine > 0 || params.EndLine > 0 {
			return fmt.Sprintf("No content in specified range (lines %d-%d)", params.StartLine, params.EndLine), nil
		}
		return "File is empty", nil
	}

	return strings.Join(lines, "\n"), nil
}