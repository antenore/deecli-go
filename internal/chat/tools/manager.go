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

	"github.com/antenore/deecli/internal/ai"
	"github.com/antenore/deecli/internal/api"
	"github.com/antenore/deecli/internal/chat/ui"
	"github.com/antenore/deecli/internal/debug"
	"github.com/antenore/deecli/internal/permissions"
	"github.com/antenore/deecli/internal/tools"
	tea "github.com/charmbracelet/bubbletea"
)

// Manager handles tool execution flow, approval dialogs, and result processing
type Manager struct {
	toolsRegistry      *tools.Registry
	toolsExecutor      *tools.Executor
	permissionManager  *permissions.Manager
	approvalHandler    *ui.ApprovalHandler
	approvalDialog     *ui.ApprovalDialog
	showingApproval    bool
	pendingToolCalls   []api.ToolCall
	// Guard to avoid loops when DeepSeek returns tool-call markers
	// even after we request a follow-up with tool_choice="none".
	// When true, the next non-stream response will not trigger tool parsing.
	suppressNextToolCalls bool
}

// Dependencies contains the dependencies needed by the tool manager
type Dependencies struct {
	ToolsRegistry     *tools.Registry
	ToolsExecutor     *tools.Executor
	PermissionManager *permissions.Manager
	ApprovalHandler   *ui.ApprovalHandler
}

// NewManager creates a new tool manager with the given dependencies
func NewManager(deps Dependencies) *Manager {
	return &Manager{
		toolsRegistry:     deps.ToolsRegistry,
		toolsExecutor:     deps.ToolsExecutor,
		permissionManager: deps.PermissionManager,
		approvalHandler:   deps.ApprovalHandler,
	}
}

// ToolExecutionCompleteMsg represents completion of tool execution
type ToolExecutionCompleteMsg struct {
	ToolCall api.ToolCall
	Result   *tools.ExecutionResult
	Error    error
}

// HandleToolCallsResponse handles AI responses that request tool executions
func (m *Manager) HandleToolCallsResponse(msg ai.ToolCallsResponseMsg) tea.Cmd {
	if m.toolsExecutor == nil {
		return func() tea.Msg {
			return fmt.Errorf("tools not available in this session")
		}
	}

	// Store the pending tool calls
	m.pendingToolCalls = msg.ToolCalls

	// Show the first tool call for approval
	if len(msg.ToolCalls) > 0 {
		return m.requestToolApproval(msg.ToolCalls[0])
	}

	return nil
}

// requestToolApproval shows approval dialog for a tool call
func (m *Manager) requestToolApproval(toolCall api.ToolCall) tea.Cmd {
	if m.approvalHandler == nil {
		return func() tea.Msg {
			return fmt.Errorf("approval system not available")
		}
	}

	// Enhanced debug logging
	debug.Printf("\n[DEBUG] ========== Tool Approval Request ==========\n")
	debug.Printf("[DEBUG] Tool: %s\n", toolCall.Function.Name)
	debug.Printf("[DEBUG] Raw arguments from AI: %q\n", toolCall.Function.Arguments)
	debug.Printf("[DEBUG] Arguments length: %d\n", len(toolCall.Function.Arguments))
	
	// Parse tool call arguments
	var args map[string]interface{}

	// Handle empty or invalid arguments
	if toolCall.Function.Arguments == "" || toolCall.Function.Arguments == "null" {
		// Use empty object for tools that don't require arguments
		debug.Printf("[DEBUG] Empty/null arguments detected, defaulting to empty map\n")
		args = map[string]interface{}{}
	} else if err := json.Unmarshal([]byte(toolCall.Function.Arguments), &args); err != nil {
		debug.Printf("[DEBUG] JSON parse error: %v\n", err)
		debug.Printf("[DEBUG] Attempting to use empty map as fallback\n")
		// If parsing fails and it's not empty, try to use defaults
		args = map[string]interface{}{}
	} else {
		debug.Printf("[DEBUG] Successfully parsed arguments: %+v\n", args)
	}
	debug.Printf("[DEBUG] ==========================================\n\n")

	// Get tool description
	description := fmt.Sprintf("Execute %s", toolCall.Function.Name)
	if tool, exists := m.toolsRegistry.Get(toolCall.Function.Name); exists {
		description = tool.Description()
	}

	// Create approval request
	approvalReq := tools.ApprovalRequest{
		FunctionName: toolCall.Function.Name,
		Description:  description,
		Arguments:    args,
	}

	// Show approval dialog - dimensions will be set by caller
	m.showingApproval = true
	
	// Return message to create approval dialog with proper dimensions
	return func() tea.Msg {
		return CreateApprovalDialogMsg{
			ApprovalRequest: approvalReq,
			ToolCall: toolCall,
		}
	}
}

// ExecuteApprovedTool executes a tool after user approval
func (m *Manager) ExecuteApprovedTool(response tools.ApprovalResponse) tea.Cmd {
	if !response.Approved || len(m.pendingToolCalls) == 0 {
		m.pendingToolCalls = nil
		return func() tea.Msg {
			return fmt.Errorf("tool execution cancelled")
		}
	}

	// Get the first pending tool call
	toolCall := m.pendingToolCalls[0]
	m.pendingToolCalls = m.pendingToolCalls[1:] // Remove from queue

	// Execute the tool
	return func() tea.Msg {
		// Parse arguments
		var args json.RawMessage
		// Handle empty or malformed arguments
		if toolCall.Function.Arguments == "" || toolCall.Function.Arguments == "null" {
			args = []byte("{}")
		} else {
			args = []byte(toolCall.Function.Arguments)
		}

		// Execute the tool
		result, err := m.toolsExecutor.ExecuteWithoutPermission(context.Background(), toolCall.Function.Name, args)
		if err != nil {
			return ToolExecutionCompleteMsg{
				ToolCall: toolCall,
				Result:   nil,
				Error:    err,
			}
		}

		return ToolExecutionCompleteMsg{
			ToolCall: toolCall,
			Result:   result,
			Error:    nil,
		}
	}
}

// HandleToolExecutionComplete handles the completion of tool execution
func (m *Manager) HandleToolExecutionComplete(msg ToolExecutionCompleteMsg, aiOperations *ai.Operations) (tea.Cmd, bool) {
	if msg.Error != nil {
		return nil, false
	}

	if msg.Result == nil {
		return nil, false
	}

	if !msg.Result.Success {
		return nil, false
	}

	// Return success info for the caller to handle display and API sync
	return m.handleSuccessfulToolCompletion(msg, aiOperations), true
}

// handleSuccessfulToolCompletion processes successful tool execution
func (m *Manager) handleSuccessfulToolCompletion(msg ToolExecutionCompleteMsg, aiOperations *ai.Operations) tea.Cmd {
	// If there are more pending tool calls, process the next one
	if len(m.pendingToolCalls) > 0 {
		// Validate the next tool call before processing
		nextTool := m.pendingToolCalls[0]
		if nextTool.Function.Name != "" && nextTool.ID != "" {
			// Note: This method signature needs to be updated to receive width/height
			// For now, we'll return a command that can be handled by the caller
			return func() tea.Msg {
				return RequestToolApprovalMsg{ToolCall: nextTool}
			}
		} else {
			// Clear invalid tool call from queue
			m.pendingToolCalls = m.pendingToolCalls[1:]
			// Check if there are more valid tools
			if len(m.pendingToolCalls) > 0 {
				return func() tea.Msg {
					return RequestToolApprovalMsg{ToolCall: m.pendingToolCalls[0]}
				}
			}
		}
	}

	// Clear the pending tool calls queue when all tools are complete
	m.pendingToolCalls = nil

	// Set the flag to suppress tool parsing in the follow-up response
	m.suppressNextToolCalls = true

	// Return command to trigger follow-up API call
	return func() tea.Msg {
		return TriggerFollowupMsg{}
	}
}

// RequestToolApprovalMsg represents a request to show tool approval dialog
type RequestToolApprovalMsg struct {
	ToolCall api.ToolCall
}

// TriggerFollowupMsg represents a request to trigger follow-up API call
type TriggerFollowupMsg struct{}

// CreateApprovalDialogMsg represents a request to create approval dialog
type CreateApprovalDialogMsg struct {
	ApprovalRequest tools.ApprovalRequest
	ToolCall       api.ToolCall
}

// IsShowingApproval returns true if approval dialog is currently showing
func (m *Manager) IsShowingApproval() bool {
	return m.showingApproval
}

// GetApprovalDialog returns the current approval dialog
func (m *Manager) GetApprovalDialog() *ui.ApprovalDialog {
	return m.approvalDialog
}

// SetShowingApproval sets the approval dialog state
func (m *Manager) SetShowingApproval(showing bool) {
	m.showingApproval = showing
}

// ClearApprovalDialog clears the approval dialog
func (m *Manager) ClearApprovalDialog() {
	m.approvalDialog = nil
}

// ShouldSuppressToolCalls returns true if tool call parsing should be suppressed
func (m *Manager) ShouldSuppressToolCalls() bool {
	return m.suppressNextToolCalls
}

// ClearSuppressToolCalls clears the tool call suppression flag
func (m *Manager) ClearSuppressToolCalls() {
	m.suppressNextToolCalls = false
}

// SetSuppressToolCalls sets the tool call suppression flag
func (m *Manager) SetSuppressToolCalls(suppress bool) {
	m.suppressNextToolCalls = suppress
}

// CreateApprovalDialog creates an approval dialog with the given dimensions
func (m *Manager) CreateApprovalDialog(req tools.ApprovalRequest, width, height int) {
	m.approvalDialog = ui.NewApprovalDialog(req, width, height)
}

// UpdateApprovalDialog processes approval dialog input
func (m *Manager) UpdateApprovalDialog(input string) (bool, *tools.ApprovalResponse) {
	if m.approvalDialog == nil {
		return false, nil
	}
	return m.approvalDialog.Update(input)
}

// GetApprovalDialogView returns the approval dialog view
func (m *Manager) GetApprovalDialogView() string {
	if m.approvalDialog == nil {
		return ""
	}
	return m.approvalDialog.View()
}