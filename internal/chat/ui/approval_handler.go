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

package ui

import (
	"github.com/antenore/deecli/internal/tools"
	tea "github.com/charmbracelet/bubbletea"
)

// ApprovalHandler handles tool approval requests in the TUI
type ApprovalHandler struct {
	responseChan chan tools.ApprovalResponse
	requestChan  chan tools.ApprovalRequest
}

// NewApprovalHandler creates a new approval handler
func NewApprovalHandler() *ApprovalHandler {
	return &ApprovalHandler{
		responseChan: make(chan tools.ApprovalResponse, 1),
		requestChan:  make(chan tools.ApprovalRequest, 1),
	}
}

// RequestApproval requests user approval for a tool function call
func (h *ApprovalHandler) RequestApproval(request tools.ApprovalRequest) (tools.ApprovalResponse, error) {
	// Send request to UI
	h.requestChan <- request

	// Wait for response
	response := <-h.responseChan

	return response, nil
}

// GetRequestChannel returns the channel for receiving approval requests
func (h *ApprovalHandler) GetRequestChannel() <-chan tools.ApprovalRequest {
	return h.requestChan
}

// SendResponse sends the user's approval response
func (h *ApprovalHandler) SendResponse(response tools.ApprovalResponse) {
	select {
	case h.responseChan <- response:
	default:
		// Channel might be full, ignore
	}
}

// ApprovalRequestMsg is sent to the TUI when approval is needed
type ApprovalRequestMsg struct {
	Request tools.ApprovalRequest
	Handler *ApprovalHandler
}

// ApprovalCompleteMsg is sent when approval is complete
type ApprovalCompleteMsg struct {
	Response tools.ApprovalResponse
}

// HandleApprovalUpdate processes approval dialog updates in the TUI
func HandleApprovalUpdate(dialog *ApprovalDialog, msg tea.Msg, handler *ApprovalHandler) (tea.Cmd, bool) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		done, response := dialog.Update(msg.String())
		if done && response != nil {
			handler.SendResponse(*response)
			return func() tea.Msg {
				return ApprovalCompleteMsg{Response: *response}
			}, true
		}
	}
	return nil, false
}