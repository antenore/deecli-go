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
	"os/exec"
	"strings"
)

// GitDiff implements git diff tool function
type GitDiff struct{}

// Name returns the function name
func (g *GitDiff) Name() string {
	return "git_diff"
}

// Description returns what this function does
func (g *GitDiff) Description() string {
	return "Show changes between commits, commit and working tree, etc"
}

// Parameters returns the JSON schema for parameters
func (g *GitDiff) Parameters() map[string]interface{} {
	return map[string]interface{}{
		"type": "object",
		"properties": map[string]interface{}{
			"file": map[string]interface{}{
				"type":        "string",
				"description": "Specific file to diff",
			},
			"staged": map[string]interface{}{
				"type":        "boolean",
				"description": "Show staged changes",
			},
			"nameOnly": map[string]interface{}{
				"type":        "boolean",
				"description": "Show only file names",
			},
		},
		"required": []string{},
	}
}

// Execute runs git diff command
func (g *GitDiff) Execute(ctx context.Context, args json.RawMessage) (string, error) {
	// Parse arguments
	var params struct {
		File     string `json:"file"`
		Staged   bool   `json:"staged"`
		NameOnly bool   `json:"nameOnly"`
	}
	if err := json.Unmarshal(args, &params); err != nil {
		return "", fmt.Errorf("failed to parse arguments: %w", err)
	}

	// Build command
	cmdArgs := []string{"diff"}

	if params.Staged {
		cmdArgs = append(cmdArgs, "--cached")
	}

	if params.NameOnly {
		cmdArgs = append(cmdArgs, "--name-only")
	}

	if params.File != "" {
		cmdArgs = append(cmdArgs, "--", params.File)
	}

	cmd := exec.CommandContext(ctx, "git", cmdArgs...)
	output, err := cmd.CombinedOutput()
	if err != nil {
		// git diff returns exit code 1 when there are differences, which is not an error
		if exitErr, ok := err.(*exec.ExitError); ok && exitErr.ExitCode() == 1 {
			// This is fine, there are differences
			return strings.TrimSpace(string(output)), nil
		}
		return "", fmt.Errorf("git diff failed: %w\n%s", err, output)
	}

	result := strings.TrimSpace(string(output))
	if result == "" {
		return "No changes detected", nil
	}

	return result, nil
}