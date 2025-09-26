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
	"testing"
)

// mockTool implements the ToolFunction interface for testing
type mockTool struct {
	name        string
	description string
	parameters  map[string]interface{}
	executeFunc func(context.Context, json.RawMessage) (string, error)
}

func (m *mockTool) Name() string        { return m.name }
func (m *mockTool) Description() string { return m.description }
func (m *mockTool) Parameters() map[string]interface{} { return m.parameters }
func (m *mockTool) Execute(ctx context.Context, args json.RawMessage) (string, error) {
	if m.executeFunc != nil {
		return m.executeFunc(ctx, args)
	}
	return "mock output", nil
}

// mockPermissionManager implements PermissionManager interface
type mockPermissionManager struct {
	allowAll bool
}

func (m *mockPermissionManager) CheckPermission(functionName, projectPath string) (PermissionLevel, error) {
	if m.allowAll {
		return PermissionAlways, nil
	}
	return PermissionNever, nil
}

func (m *mockPermissionManager) SetPermission(functionName, projectPath string, level PermissionLevel) error {
	return nil
}

func (m *mockPermissionManager) RequestApproval(request ApprovalRequest) (ApprovalResponse, error) {
	return ApprovalResponse{
		Approved: m.allowAll,
		Level:    PermissionOnce,
	}, nil
}

func TestExecutor_Execute(t *testing.T) {
	registry := NewRegistry()
	
	// Register a mock tool
	mockTool := &mockTool{
		name:        "test_tool",
		description: "A test tool",
		parameters:  map[string]interface{}{},
	}
	registry.Register(mockTool)
	
	permManager := &mockPermissionManager{allowAll: true}
	executor := NewExecutor(registry, permManager)
	
	tests := []struct {
		name        string
		request     ExecutionRequest
		projectPath string
		wantSuccess bool
		wantError   bool
	}{
		{
			name: "successful execution",
			request: ExecutionRequest{
				FunctionName: "test_tool",
				Arguments:    json.RawMessage(`{}`),
				RequestID:    "test_1",
			},
			projectPath: "/test",
			wantSuccess: true,
			wantError:   false,
		},
		{
			name: "nonexistent tool",
			request: ExecutionRequest{
				FunctionName: "nonexistent_tool",
				Arguments:    json.RawMessage(`{}`),
				RequestID:    "test_2",
			},
			projectPath: "/test",
			wantSuccess: false,
			wantError:   true,
		},
		{
			name: "empty function name",
			request: ExecutionRequest{
				FunctionName: "",
				Arguments:    json.RawMessage(`{}`),
				RequestID:    "test_3",
			},
			projectPath: "/test",
			wantSuccess: false,
			wantError:   true,
		},
	}
	
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := executor.Execute(context.Background(), tt.request, tt.projectPath)
			
			if tt.wantError {
				if err == nil {
					t.Errorf("Execute() error = nil, want error")
				}
				return
			}
			
			if err != nil {
				t.Errorf("Execute() error = %v, want nil", err)
				return
			}
			
			if result == nil {
				t.Errorf("Execute() result = nil, want non-nil")
				return
			}
			
			if result.Success != tt.wantSuccess {
				t.Errorf("Execute() result.Success = %v, want %v", result.Success, tt.wantSuccess)
			}
		})
	}
}

func TestExecutor_ExecuteWithoutPermission(t *testing.T) {
	registry := NewRegistry()
	
	// Register a mock tool
	mockTool := &mockTool{
		name:        "test_tool",
		description: "A test tool",
		parameters:  map[string]interface{}{},
	}
	registry.Register(mockTool)
	
	permManager := &mockPermissionManager{allowAll: false} // Deny permission
	executor := NewExecutor(registry, permManager)
	
	// This should succeed even though permission manager denies
	request := ExecutionRequest{
		FunctionName: "test_tool",
		Arguments:    json.RawMessage(`{}`),
		RequestID:    "test_direct",
	}
	result, err := executor.ExecuteWithoutPermission(context.Background(), request.FunctionName, request.Arguments)
	
	if err != nil {
		t.Errorf("ExecuteWithoutPermission() error = %v, want nil", err)
	}
	
	if result == nil || !result.Success {
		t.Errorf("ExecuteWithoutPermission() should succeed regardless of permissions")
	}
}

func TestExecutor_ToolError(t *testing.T) {
	registry := NewRegistry()
	
	// Register a mock tool that returns an error
	mockTool := &mockTool{
		name:        "error_tool",
		description: "A tool that errors",
		parameters:  map[string]interface{}{},
		executeFunc: func(ctx context.Context, args json.RawMessage) (string, error) {
			return "", fmt.Errorf("tool execution failed")
		},
	}
	registry.Register(mockTool)
	
	permManager := &mockPermissionManager{allowAll: true}
	executor := NewExecutor(registry, permManager)
	
	request := ExecutionRequest{
		FunctionName: "error_tool",
		Arguments:    json.RawMessage(`{}`),
		RequestID:    "test_error",
	}
	result, err := executor.Execute(context.Background(), request, "/test")
	
	if err == nil {
		t.Errorf("Execute() error = nil, want error")
	}
	
	if result != nil {
		t.Errorf("Execute() result = %v, want nil when tool errors", result)
	}
}

func TestExecutor_ToolFailure(t *testing.T) {
	registry := NewRegistry()
	
	// Register a mock tool that returns a failure result
	mockTool := &mockTool{
		name:        "failing_tool",
		description: "A tool that fails",
		parameters:  map[string]interface{}{},
		executeFunc: func(ctx context.Context, args json.RawMessage) (string, error) {
			return "", fmt.Errorf("tool operation failed")
		},
	}
	registry.Register(mockTool)
	
	permManager := &mockPermissionManager{allowAll: true}
	executor := NewExecutor(registry, permManager)
	
	request := ExecutionRequest{
		FunctionName: "failing_tool",
		Arguments:    json.RawMessage(`{}`),
		RequestID:    "test_fail",
	}
	result, err := executor.Execute(context.Background(), request, "/test")
	
	if err == nil {
		t.Errorf("Execute() error = nil, want error for failing tool")
	}

	if result != nil {
		t.Errorf("Execute() result = %v, want nil when tool fails", result)
	}
}