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

package tools

import (
	"context"
	"encoding/json"
)

// ToolFunction represents a function that can be called by the AI
type ToolFunction interface {
	// Name returns the function name
	Name() string

	// Description returns what this function does
	Description() string

	// Parameters returns the JSON schema for parameters
	Parameters() map[string]interface{}

	// Execute runs the function with given arguments
	Execute(ctx context.Context, args json.RawMessage) (string, error)
}

// PermissionLevel represents the permission level for a tool
type PermissionLevel string

const (
	PermissionOnce   PermissionLevel = "once"   // Approve for this invocation only
	PermissionAlways PermissionLevel = "always" // Auto-approve in this project
	PermissionNever  PermissionLevel = "never"  // Block in this project
)

// ToolPermission represents permission settings for a tool
type ToolPermission struct {
	FunctionName string          `json:"function_name"`
	ProjectPath  string          `json:"project_path"`
	Level        PermissionLevel `json:"level"`
	UpdatedAt    int64           `json:"updated_at"`
}

// ExecutionRequest represents a request to execute a tool function
type ExecutionRequest struct {
	FunctionName string          `json:"function_name"`
	Arguments    json.RawMessage `json:"arguments"`
	RequestID    string          `json:"request_id"`
}

// ExecutionResult represents the result of a tool execution
type ExecutionResult struct {
	Success bool   `json:"success"`
	Output  string `json:"output"`
	Error   string `json:"error,omitempty"`
}

// ApprovalRequest represents a request for user approval
type ApprovalRequest struct {
	FunctionName string                 `json:"function_name"`
	Description  string                 `json:"description"`
	Arguments    map[string]interface{} `json:"arguments"`
}

// ApprovalResponse represents user's approval decision
type ApprovalResponse struct {
	Approved bool            `json:"approved"`
	Level    PermissionLevel `json:"level"`
}