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
	"encoding/json"
	"fmt"
	"strings"

	"github.com/antenore/deecli/internal/tools"
	"github.com/charmbracelet/lipgloss"
)

// ApprovalDialog represents the tool approval UI component
type ApprovalDialog struct {
	request       tools.ApprovalRequest
	selectedIndex int
	options       []approvalOption
	width         int
	height        int
}

type approvalOption struct {
	label string
	level tools.PermissionLevel
}

// NewApprovalDialog creates a new approval dialog
func NewApprovalDialog(request tools.ApprovalRequest, width, height int) *ApprovalDialog {
	return &ApprovalDialog{
		request: request,
		width:   width,
		height:  height,
		options: []approvalOption{
			{"Approve Once", tools.PermissionOnce},
			{"Always Approve (This Project)", tools.PermissionAlways},
			{"Never (Block in This Project)", tools.PermissionNever},
		},
		selectedIndex: 0,
	}
}

// Update handles key events for the dialog
func (d *ApprovalDialog) Update(key string) (bool, *tools.ApprovalResponse) {
	switch key {
	case "up", "k":
		d.selectedIndex--
		if d.selectedIndex < 0 {
			d.selectedIndex = len(d.options) - 1
		}

	case "down", "j":
		d.selectedIndex++
		if d.selectedIndex >= len(d.options) {
			d.selectedIndex = 0
		}

	case "enter":
		selected := d.options[d.selectedIndex]
		response := &tools.ApprovalResponse{
			Approved: selected.level != tools.PermissionNever,
			Level:    selected.level,
		}
		return true, response

	case "esc", "q":
		// Cancel/deny
		response := &tools.ApprovalResponse{
			Approved: false,
			Level:    tools.PermissionOnce,
		}
		return true, response
	}

	return false, nil
}

// View renders the approval dialog
func (d *ApprovalDialog) View() string {
	// Styles
	titleStyle := lipgloss.NewStyle().
		Bold(true).
		Foreground(lipgloss.Color("226")).
		MarginBottom(1)

	functionStyle := lipgloss.NewStyle().
		Bold(true).
		Foreground(lipgloss.Color("87"))

	descStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("245")).
		MarginBottom(1)

	paramStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("251")).
		MarginLeft(2)

	optionStyle := lipgloss.NewStyle().
		MarginLeft(2).
		MarginTop(1)

	selectedStyle := lipgloss.NewStyle().
		Bold(true).
		Foreground(lipgloss.Color("226")).
		Background(lipgloss.Color("235"))

	borderStyle := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(lipgloss.Color("240")).
		Padding(1, 2).
		MaxWidth(d.width - 4)

	// Build content
	var content strings.Builder

	// Title
	content.WriteString(titleStyle.Render("🔧 Tool Approval Request"))
	content.WriteString("\n\n")

	// Function info
	content.WriteString(functionStyle.Render("Function: " + d.request.FunctionName))
	content.WriteString("\n")
	content.WriteString(descStyle.Render(d.request.Description))
	content.WriteString("\n")

	// Parameters
	if len(d.request.Arguments) > 0 {
		content.WriteString("\nParameters:\n")

		// Pretty print JSON arguments
		jsonBytes, err := json.MarshalIndent(d.request.Arguments, "", "  ")
		if err != nil {
			content.WriteString(paramStyle.Render(fmt.Sprintf("%v", d.request.Arguments)))
		} else {
			content.WriteString(paramStyle.Render(string(jsonBytes)))
		}
		content.WriteString("\n")
	}

	// Options
	content.WriteString("\nChoose an option:\n")
	for i, option := range d.options {
		var optionText string
		if i == d.selectedIndex {
			optionText = selectedStyle.Render(fmt.Sprintf("▶ %s", option.label))
		} else {
			optionText = optionStyle.Render(fmt.Sprintf("  %s", option.label))
		}
		content.WriteString(optionText + "\n")
	}

	// Help text
	helpStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("241")).
		MarginTop(1)

	helpText := "↑/↓ or j/k: Navigate • Enter: Select • Esc/q: Cancel"
	content.WriteString("\n" + helpStyle.Render(helpText))

	// Apply border
	return borderStyle.Render(content.String())
}

// GetSelectedOption returns the currently selected option
func (d *ApprovalDialog) GetSelectedOption() approvalOption {
	return d.options[d.selectedIndex]
}