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

package permissions

import (
	"time"

	"github.com/antenore/deecli/internal/config"
	"github.com/antenore/deecli/internal/tools"
)

// Manager handles permission management for tool functions
type Manager struct {
	configManager   *config.Manager
	approvalHandler ApprovalHandler
}

// ApprovalHandler interface for UI approval requests
type ApprovalHandler interface {
	RequestApproval(request tools.ApprovalRequest) (tools.ApprovalResponse, error)
}

// NewManager creates a new permission manager
func NewManager(configManager *config.Manager, approvalHandler ApprovalHandler) *Manager {
	return &Manager{
		configManager:   configManager,
		approvalHandler: approvalHandler,
	}
}

// CheckPermission checks the permission level for a function in the current project
func (m *Manager) CheckPermission(functionName, projectPath string) (tools.PermissionLevel, error) {
	cfg := m.configManager.Get()
	if cfg.ToolPermissions == nil {
		return "", nil // No permission set
	}

	permission, exists := cfg.ToolPermissions[functionName]
	if !exists {
		return "", nil // No permission set for this function
	}

	return tools.PermissionLevel(permission.Level), nil
}

// SetPermission sets the permission level for a function in the current project
func (m *Manager) SetPermission(functionName, projectPath string, level tools.PermissionLevel) error {
	cfg := m.configManager.Get()

	// Initialize map if needed
	if cfg.ToolPermissions == nil {
		cfg.ToolPermissions = make(map[string]config.ToolPermission)
	}

	// Set the permission
	cfg.ToolPermissions[functionName] = config.ToolPermission{
		Level:     string(level),
		UpdatedAt: time.Now().Unix(),
	}

	// Save to project config (permissions are project-specific)
	return m.configManager.SaveProject(cfg)
}

// RequestApproval requests user approval for a function call
func (m *Manager) RequestApproval(request tools.ApprovalRequest) (tools.ApprovalResponse, error) {
	if m.approvalHandler == nil {
		// If no handler, deny by default
		return tools.ApprovalResponse{
			Approved: false,
			Level:    tools.PermissionNever,
		}, nil
	}

	return m.approvalHandler.RequestApproval(request)
}

// GetAllPermissions returns all permissions for the current project
func (m *Manager) GetAllPermissions() []tools.ToolPermission {
	cfg := m.configManager.Get()
	if cfg.ToolPermissions == nil {
		return []tools.ToolPermission{}
	}

	permissions := make([]tools.ToolPermission, 0, len(cfg.ToolPermissions))
	for funcName, perm := range cfg.ToolPermissions {
		permissions = append(permissions, tools.ToolPermission{
			FunctionName: funcName,
			ProjectPath:  "", // Not needed in YAML-based system
			Level:        tools.PermissionLevel(perm.Level),
			UpdatedAt:    perm.UpdatedAt,
		})
	}

	return permissions
}

// ClearPermission removes a permission setting
func (m *Manager) ClearPermission(functionName string) error {
	cfg := m.configManager.Get()
	if cfg.ToolPermissions == nil {
		return nil // Nothing to clear
	}

	delete(cfg.ToolPermissions, functionName)

	// Save to project config
	return m.configManager.SaveProject(cfg)
}

// ClearAllPermissions removes all permission settings
func (m *Manager) ClearAllPermissions() error {
	cfg := m.configManager.Get()
	cfg.ToolPermissions = make(map[string]config.ToolPermission)

	// Save to project config
	return m.configManager.SaveProject(cfg)
}