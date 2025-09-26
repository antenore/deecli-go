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

	"github.com/antenore/deecli/internal/ai"
	"github.com/antenore/deecli/internal/api"
	"github.com/antenore/deecli/internal/chat/ui"
	"github.com/antenore/deecli/internal/config"
	"github.com/antenore/deecli/internal/files"
	"github.com/antenore/deecli/internal/permissions"
	"github.com/antenore/deecli/internal/tools"
)

// mockTool for testing
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


func setupTestManager() (*Manager, *tools.Registry, *ai.Operations) {
	registry := tools.NewRegistry()

	// Register a mock tool
	mockTool := &mockTool{
		name:        "test_read_file",
		description: "Read a test file",
		parameters: map[string]interface{}{
			"type": "object",
			"properties": map[string]interface{}{
				"path": map[string]interface{}{
					"type":        "string",
					"description": "File path to read",
				},
			},
			"required": []string{"path"},
		},
	}
	registry.Register(mockTool)

	// Create proper instances for testing
	configManager := config.NewManager()
	approvalHandler := ui.NewApprovalHandler()
	permManager := permissions.NewManager(configManager, approvalHandler)
	executor := tools.NewExecutor(registry, permManager)

	manager := NewManager(Dependencies{
		ToolsRegistry:     registry,
		ToolsExecutor:     executor,
		PermissionManager: permManager,
		ApprovalHandler:   approvalHandler,
	})

	// Create a real ai.Operations instance for testing
	deepSeekClient := api.NewDeepSeekClient("test-key", "deepseek-chat", 0.7, 4000)
	apiClient := api.NewService(deepSeekClient)
	fileContext := &files.FileContext{}
	aiOps := ai.NewOperations(apiClient, fileContext, configManager)

	return manager, registry, aiOps
}

func TestManager_HandleToolCallsResponse(t *testing.T) {
	manager, _, _ := setupTestManager()
	
	toolCalls := []api.ToolCall{
		{
			ID:   "call_1",
			Type: "function",
			Function: struct {
				Name      string `json:"name"`
				Arguments string `json:"arguments"`
			}{
				Name:      "test_read_file",
				Arguments: `{"path": "test.go"}`,
			},
		},
	}
	
	msg := ai.ToolCallsResponseMsg{
		ToolCalls: toolCalls,
	}
	
	cmd := manager.HandleToolCallsResponse(msg)
	
	if cmd == nil {
		t.Errorf("HandleToolCallsResponse() returned nil command, want non-nil")
	}
	
	if !manager.IsShowingApproval() {
		t.Errorf("HandleToolCallsResponse() should set showing approval to true")
	}
	
	if len(manager.pendingToolCalls) != 1 {
		t.Errorf("HandleToolCallsResponse() pending tool calls = %d, want 1", len(manager.pendingToolCalls))
	}
}

func TestManager_ExecuteApprovedTool(t *testing.T) {
	manager, _, _ := setupTestManager()
	
	// Set up pending tool call
	manager.pendingToolCalls = []api.ToolCall{
		{
			ID:   "call_1",
			Type: "function",
			Function: struct {
				Name      string `json:"name"`
				Arguments string `json:"arguments"`
			}{
				Name:      "test_read_file",
				Arguments: `{"path": "test.go"}`,
			},
		},
	}
	
	response := tools.ApprovalResponse{
		Approved: true,
	}
	
	cmd := manager.ExecuteApprovedTool(response)
	
	if cmd == nil {
		t.Errorf("ExecuteApprovedTool() returned nil command, want non-nil")
	}
	
	// Execute the command to get the result
	result := cmd()
	
	execMsg, ok := result.(ToolExecutionCompleteMsg)
	if !ok {
		t.Errorf("ExecuteApprovedTool() result type = %T, want ToolExecutionCompleteMsg", result)
		return
	}
	
	if execMsg.Error != nil {
		t.Errorf("ExecuteApprovedTool() execution error = %v, want nil", execMsg.Error)
	}
	
	if execMsg.Result == nil {
		t.Errorf("ExecuteApprovedTool() result = nil, want non-nil")
		return
	}
	
	if !execMsg.Result.Success {
		t.Errorf("ExecuteApprovedTool() result success = false, want true")
	}
}

func TestManager_ExecuteApprovedTool_Cancelled(t *testing.T) {
	manager, _, _ := setupTestManager()
	
	// Set up pending tool call
	manager.pendingToolCalls = []api.ToolCall{
		{
			ID:   "call_1",
			Type: "function",
			Function: struct {
				Name      string `json:"name"`
				Arguments string `json:"arguments"`
			}{
				Name:      "test_read_file",
				Arguments: `{"path": "test.go"}`,
			},
		},
	}
	
	response := tools.ApprovalResponse{
		Approved: false,
	}
	
	cmd := manager.ExecuteApprovedTool(response)
	
	if cmd == nil {
		t.Errorf("ExecuteApprovedTool() returned nil command, want non-nil for cancellation")
	}
	
	// Execute the command to get the result
	result := cmd()
	
	if _, ok := result.(error); !ok {
		t.Errorf("ExecuteApprovedTool() cancelled result should be error, got %T", result)
	}
	
	if manager.pendingToolCalls != nil {
		t.Errorf("ExecuteApprovedTool() cancelled should clear pending tool calls")
	}
}

func TestManager_HandleToolExecutionComplete(t *testing.T) {
	manager, _, aiOps := setupTestManager()
	
	toolCall := api.ToolCall{
		ID:   "call_1",
		Type: "function",
		Function: struct {
				Name      string `json:"name"`
				Arguments string `json:"arguments"`
			}{
			Name:      "test_read_file",
			Arguments: `{"path": "test.go"}`,
		},
	}
	
	result := &tools.ExecutionResult{
		Success: true,
		Output:  "file content here",
	}
	
	msg := ToolExecutionCompleteMsg{
		ToolCall: toolCall,
		Result:   result,
		Error:    nil,
	}
	
	cmd, success := manager.HandleToolExecutionComplete(msg, aiOps)
	
	if !success {
		t.Errorf("HandleToolExecutionComplete() success = false, want true")
	}
	
	if cmd == nil {
		t.Errorf("HandleToolExecutionComplete() returned nil command, want non-nil")
	}
	
	// Check that suppress flag is set
	if !manager.ShouldSuppressToolCalls() {
		t.Errorf("HandleToolExecutionComplete() should set suppressNextToolCalls to true")
	}
}

func TestManager_HandleToolExecutionComplete_Error(t *testing.T) {
	manager, _, aiOps := setupTestManager()
	
	msg := ToolExecutionCompleteMsg{
		ToolCall: api.ToolCall{},
		Result:   nil,
		Error:    fmt.Errorf("execution failed"),
	}
	
	cmd, success := manager.HandleToolExecutionComplete(msg, aiOps)
	
	if success {
		t.Errorf("HandleToolExecutionComplete() success = true, want false for error")
	}
	
	if cmd != nil {
		t.Errorf("HandleToolExecutionComplete() returned command for error case, want nil")
	}
}

func TestManager_HandleToolExecutionComplete_MultiplePendingTools(t *testing.T) {
	manager, _, aiOps := setupTestManager()
	
	// Set up multiple pending tool calls
	manager.pendingToolCalls = []api.ToolCall{
		{
			ID:   "call_1",
			Type: "function",
			Function: struct {
				Name      string `json:"name"`
				Arguments string `json:"arguments"`
			}{
				Name:      "test_read_file",
				Arguments: `{"path": "file1.go"}`,
			},
		},
		{
			ID:   "call_2",
			Type: "function",
			Function: struct {
				Name      string `json:"name"`
				Arguments string `json:"arguments"`
			}{
				Name:      "test_read_file",
				Arguments: `{"path": "file2.go"}`,
			},
		},
	}
	
	toolCall := api.ToolCall{
		ID:   "call_1",
		Type: "function",
		Function: struct {
				Name      string `json:"name"`
				Arguments string `json:"arguments"`
			}{
			Name:      "test_read_file",
			Arguments: `{"path": "file1.go"}`,
		},
	}
	
	result := &tools.ExecutionResult{
		Success: true,
		Output:  "file1 content",
	}
	
	msg := ToolExecutionCompleteMsg{
		ToolCall: toolCall,
		Result:   result,
		Error:    nil,
	}
	
	cmd, success := manager.HandleToolExecutionComplete(msg, aiOps)
	
	if !success {
		t.Errorf("HandleToolExecutionComplete() success = false, want true")
	}
	
	if cmd == nil {
		t.Errorf("HandleToolExecutionComplete() returned nil command, want non-nil")
	}
	
	// Execute the command to get the result
	result2 := cmd()
	
	reqMsg, ok := result2.(RequestToolApprovalMsg)
	if !ok {
		t.Errorf("HandleToolExecutionComplete() result type = %T, want RequestToolApprovalMsg", result2)
		return
	}
	
	if reqMsg.ToolCall.ID != "call_2" {
		t.Errorf("HandleToolExecutionComplete() next tool ID = %s, want call_2", reqMsg.ToolCall.ID)
	}
	
	// Should not set suppress flag when more tools pending
	if manager.ShouldSuppressToolCalls() {
		t.Errorf("HandleToolExecutionComplete() should not set suppressNextToolCalls when more tools pending")
	}
}

func TestManager_SuppressToolCalls(t *testing.T) {
	manager, _, _ := setupTestManager()
	
	// Initially should not suppress
	if manager.ShouldSuppressToolCalls() {
		t.Errorf("Initial ShouldSuppressToolCalls() = true, want false")
	}
	
	// Set suppress
	manager.SetSuppressToolCalls(true)
	if !manager.ShouldSuppressToolCalls() {
		t.Errorf("ShouldSuppressToolCalls() after set = false, want true")
	}
	
	// Clear suppress
	manager.ClearSuppressToolCalls()
	if manager.ShouldSuppressToolCalls() {
		t.Errorf("ShouldSuppressToolCalls() after clear = true, want false")
	}
}

func TestManager_ApprovalDialog(t *testing.T) {
	manager, _, _ := setupTestManager()
	
	// Initially no dialog
	if manager.IsShowingApproval() {
		t.Errorf("Initial IsShowingApproval() = true, want false")
	}
	
	if manager.GetApprovalDialog() != nil {
		t.Errorf("Initial GetApprovalDialog() = non-nil, want nil")
	}
	
	// Set showing approval
	manager.SetShowingApproval(true)
	if !manager.IsShowingApproval() {
		t.Errorf("IsShowingApproval() after set = false, want true")
	}
	
	// Create approval dialog
	req := tools.ApprovalRequest{
		FunctionName: "test_tool",
		Description:  "Test tool description",
		Arguments:    map[string]interface{}{"path": "test.go"},
	}
	
	manager.CreateApprovalDialog(req, 80, 24)
	
	dialog := manager.GetApprovalDialog()
	if dialog == nil {
		t.Errorf("GetApprovalDialog() after create = nil, want non-nil")
	}
	
	// Clear dialog
	manager.ClearApprovalDialog()
	if manager.GetApprovalDialog() != nil {
		t.Errorf("GetApprovalDialog() after clear = non-nil, want nil")
	}
}

// TestManager_IntegrationFlow tests the complete tool execution flow
func TestManager_IntegrationFlow(t *testing.T) {
	manager, _, aiOps := setupTestManager()
	
	// Step 1: Handle tool calls response from AI
	toolCalls := []api.ToolCall{
		{
			ID:   "call_1",
			Type: "function",
			Function: struct {
				Name      string `json:"name"`
				Arguments string `json:"arguments"`
			}{
				Name:      "test_read_file",
				Arguments: `{"path": "integration_test.go"}`,
			},
		},
	}
	
	toolMsg := ai.ToolCallsResponseMsg{ToolCalls: toolCalls}
	cmd1 := manager.HandleToolCallsResponse(toolMsg)
	
	if cmd1 == nil {
		t.Fatal("Step 1: HandleToolCallsResponse() returned nil")
	}
	
	// Execute command to get approval dialog creation message
	result1 := cmd1()
	createMsg, ok := result1.(CreateApprovalDialogMsg)
	if !ok {
		t.Fatalf("Step 1: Expected CreateApprovalDialogMsg, got %T", result1)
	}
	
	// Step 2: Create approval dialog and simulate approval
	manager.CreateApprovalDialog(createMsg.ApprovalRequest, 80, 24)
	
	approvalResponse := tools.ApprovalResponse{Approved: true}
	cmd2 := manager.ExecuteApprovedTool(approvalResponse)
	
	if cmd2 == nil {
		t.Fatal("Step 2: ExecuteApprovedTool() returned nil")
	}
	
	// Step 3: Execute tool and get completion
	result2 := cmd2()
	execMsg, ok := result2.(ToolExecutionCompleteMsg)
	if !ok {
		t.Fatalf("Step 3: Expected ToolExecutionCompleteMsg, got %T", result2)
	}
	
	if execMsg.Error != nil {
		t.Fatalf("Step 3: Tool execution failed: %v", execMsg.Error)
	}
	
	// Step 4: Handle tool execution completion
	cmd3, success := manager.HandleToolExecutionComplete(execMsg, aiOps)
	
	if !success {
		t.Fatal("Step 4: HandleToolExecutionComplete() failed")
	}
	
	if cmd3 == nil {
		t.Fatal("Step 4: HandleToolExecutionComplete() returned nil command")
	}
	
	// Execute final command to get follow-up trigger
	result3 := cmd3()
	_, ok = result3.(TriggerFollowupMsg)
	if !ok {
		t.Fatalf("Step 4: Expected TriggerFollowupMsg, got %T", result3)
	}
	
	// Verify final state
	if !manager.ShouldSuppressToolCalls() {
		t.Error("Final state: ShouldSuppressToolCalls() = false, want true")
	}
	
	if len(manager.pendingToolCalls) != 0 {
		t.Errorf("Final state: pending tool calls = %d, want 0", len(manager.pendingToolCalls))
	}
}