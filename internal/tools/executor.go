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
	"fmt"
	"time"
)

// Executor handles safe execution of tool functions
type Executor struct {
	registry    *Registry
	permissions PermissionManager
}

// PermissionManager interface for managing tool permissions
type PermissionManager interface {
	CheckPermission(functionName, projectPath string) (PermissionLevel, error)
	SetPermission(functionName, projectPath string, level PermissionLevel) error
	RequestApproval(request ApprovalRequest) (ApprovalResponse, error)
}

// NewExecutor creates a new tool executor
func NewExecutor(registry *Registry, permissions PermissionManager) *Executor {
	return &Executor{
		registry:    registry,
		permissions: permissions,
	}
}

// Execute runs a tool function with permission checks
func (e *Executor) Execute(ctx context.Context, request ExecutionRequest, projectPath string) (*ExecutionResult, error) {
	// Get the tool function
	tool, exists := e.registry.Get(request.FunctionName)
	if !exists {
		return &ExecutionResult{
			Success: false,
			Error:   fmt.Sprintf("tool function %s not found", request.FunctionName),
		}, nil
	}

	// Check permissions
	permission, err := e.permissions.CheckPermission(request.FunctionName, projectPath)
	if err != nil {
		return nil, fmt.Errorf("failed to check permissions: %w", err)
	}

	// Handle permission levels
	switch permission {
	case PermissionNever:
		return &ExecutionResult{
			Success: false,
			Error:   fmt.Sprintf("function %s is blocked in this project", request.FunctionName),
		}, nil

	case PermissionOnce, "": // Empty means no permission set yet
		// Request approval from user
		var args map[string]interface{}
		// Handle empty or invalid arguments
		argStr := string(request.Arguments)
		if argStr == "" || argStr == "null" || argStr == "{}" {
			args = map[string]interface{}{}
		} else if err := json.Unmarshal(request.Arguments, &args); err != nil {
			// Use empty args for invalid JSON
			args = map[string]interface{}{}
		}

		approvalReq := ApprovalRequest{
			FunctionName: request.FunctionName,
			Description:  tool.Description(),
			Arguments:    args,
		}

		approval, err := e.permissions.RequestApproval(approvalReq)
		if err != nil {
			return nil, fmt.Errorf("failed to request approval: %w", err)
		}

		if !approval.Approved {
			return &ExecutionResult{
				Success: false,
				Error:   "function call not approved by user",
			}, nil
		}

		// Save permission if not "once"
		if approval.Level != PermissionOnce {
			if err := e.permissions.SetPermission(request.FunctionName, projectPath, approval.Level); err != nil {
				// Log error but continue with execution
				fmt.Printf("Warning: failed to save permission: %v\n", err)
			}
		}

	case PermissionAlways:
		// Approved, continue with execution
	}

	// Execute the function with timeout
	execCtx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()

	output, err := tool.Execute(execCtx, request.Arguments)
	if err != nil {
		return &ExecutionResult{
			Success: false,
			Error:   err.Error(),
		}, nil
	}

	return &ExecutionResult{
		Success: true,
		Output:  output,
	}, nil
}

// ExecuteWithoutPermission runs a tool function without permission checks (for testing)
func (e *Executor) ExecuteWithoutPermission(ctx context.Context, functionName string, args json.RawMessage) (*ExecutionResult, error) {
	tool, exists := e.registry.Get(functionName)
	if !exists {
		return &ExecutionResult{
			Success: false,
			Error:   fmt.Sprintf("tool function %s not found", functionName),
		}, nil
	}

	output, err := tool.Execute(ctx, args)
	if err != nil {
		return &ExecutionResult{
			Success: false,
			Error:   err.Error(),
		}, nil
	}

	return &ExecutionResult{
		Success: true,
		Output:  output,
	}, nil
}