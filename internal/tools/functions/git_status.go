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

// GitStatus implements git status tool function
type GitStatus struct{}

// Name returns the function name
func (g *GitStatus) Name() string {
	return "git_status"
}

// Description returns what this function does
func (g *GitStatus) Description() string {
	return "Get the current git repository status"
}

// Parameters returns the JSON schema for parameters
func (g *GitStatus) Parameters() map[string]interface{} {
	return map[string]interface{}{
		"type": "object",
		"properties": map[string]interface{}{
			"short": map[string]interface{}{
				"type":        "boolean",
				"description": "Show status in short format",
			},
		},
		"required": []string{},
	}
}

// Execute runs git status command
func (g *GitStatus) Execute(ctx context.Context, args json.RawMessage) (string, error) {
	// Parse arguments
	var params struct {
		Short bool `json:"short"`
	}
	if err := json.Unmarshal(args, &params); err != nil {
		return "", fmt.Errorf("failed to parse arguments: %w", err)
	}

	// Build command
	cmdArgs := []string{"status"}
	if params.Short {
		cmdArgs = append(cmdArgs, "--short")
	}

	cmd := exec.CommandContext(ctx, "git", cmdArgs...)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return "", fmt.Errorf("git status failed: %w\n%s", err, output)
	}

	return strings.TrimSpace(string(output)), nil
}