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
	"testing"
)

func TestRegistry_Register(t *testing.T) {
	registry := NewRegistry()
	
	tool := &mockTool{
		name:        "test_tool",
		description: "A test tool",
		parameters:  map[string]interface{}{"test": "param"},
	}
	
	err := registry.Register(tool)
	if err != nil {
		t.Errorf("Register() error = %v, want nil", err)
	}
	
	// Test duplicate registration
	err = registry.Register(tool)
	if err == nil {
		t.Errorf("Register() error = nil, want error for duplicate registration")
	}
}

func TestRegistry_Get(t *testing.T) {
	registry := NewRegistry()
	
	tool := &mockTool{
		name:        "test_tool",
		description: "A test tool",
		parameters:  map[string]interface{}{},
	}
	
	// Test getting non-existent tool
	_, exists := registry.Get("nonexistent")
	if exists {
		t.Errorf("Get() exists = true, want false for non-existent tool")
	}
	
	// Register and test getting existing tool
	registry.Register(tool)
	
	retrieved, exists := registry.Get("test_tool")
	if !exists {
		t.Errorf("Get() exists = false, want true for registered tool")
	}
	
	if retrieved.Name() != tool.Name() {
		t.Errorf("Get() returned tool name = %s, want %s", retrieved.Name(), tool.Name())
	}
}

func TestRegistry_GetAPITools(t *testing.T) {
	registry := NewRegistry()
	
	// Test empty registry
	apiTools := registry.GetAPITools()
	if len(apiTools) != 0 {
		t.Errorf("GetAPITools() length = %d, want 0 for empty registry", len(apiTools))
	}
	
	// Add tools and test
	tool1 := &mockTool{
		name:        "tool1",
		description: "First tool",
		parameters: map[string]interface{}{
			"type": "object",
			"properties": map[string]interface{}{
				"param1": map[string]interface{}{
					"type":        "string",
					"description": "A parameter",
				},
			},
		},
	}
	
	tool2 := &mockTool{
		name:        "tool2",
		description: "Second tool",
		parameters:  map[string]interface{}{},
	}
	
	registry.Register(tool1)
	registry.Register(tool2)
	
	apiTools = registry.GetAPITools()
	if len(apiTools) != 2 {
		t.Errorf("GetAPITools() length = %d, want 2", len(apiTools))
	}
	
	// Check first tool structure
	foundTool1 := false
	for _, apiTool := range apiTools {
		if apiTool.Function.Name == "tool1" {
			foundTool1 = true
			if apiTool.Type != "function" {
				t.Errorf("API tool type = %s, want 'function'", apiTool.Type)
			}
			if apiTool.Function.Description != "First tool" {
				t.Errorf("API tool description = %s, want 'First tool'", apiTool.Function.Description)
			}
		}
	}
	
	if !foundTool1 {
		t.Errorf("tool1 not found in API tools")
	}
}

func TestRegistry_List(t *testing.T) {
	registry := NewRegistry()
	
	// Test empty registry
	names := registry.List()
	if len(names) != 0 {
		t.Errorf("List() length = %d, want 0 for empty registry", len(names))
	}
	
	// Add tools and test
	tool1 := &mockTool{name: "tool1", description: "First tool"}
	tool2 := &mockTool{name: "tool2", description: "Second tool"}
	
	registry.Register(tool1)
	registry.Register(tool2)
	
	names = registry.List()
	if len(names) != 2 {
		t.Errorf("List() length = %d, want 2", len(names))
	}
	
	expectedNames := map[string]bool{"tool1": true, "tool2": true}
	for _, name := range names {
		if !expectedNames[name] {
			t.Errorf("List() contains unexpected name: %s", name)
		}
		delete(expectedNames, name)
	}
	
	if len(expectedNames) > 0 {
		t.Errorf("List() missing names: %v", expectedNames)
	}
}

func TestAPITool_Conversion(t *testing.T) {
	tool := &mockTool{
		name:        "conversion_test",
		description: "Test conversion to API format",
		parameters: map[string]interface{}{
			"type": "object",
			"properties": map[string]interface{}{
				"path": map[string]interface{}{
					"type":        "string",
					"description": "File path",
				},
				"recursive": map[string]interface{}{
					"type":        "boolean",
					"description": "Recursive flag",
					"default":     false,
				},
			},
			"required": []string{"path"},
		},
	}
	
	registry := NewRegistry()
	registry.Register(tool)
	
	apiTools := registry.GetAPITools()
	if len(apiTools) != 1 {
		t.Fatalf("GetAPITools() length = %d, want 1", len(apiTools))
	}
	
	apiTool := apiTools[0]
	
	// Verify API tool structure
	if apiTool.Type != "function" {
		t.Errorf("API tool type = %s, want 'function'", apiTool.Type)
	}
	
	if apiTool.Function.Name != "conversion_test" {
		t.Errorf("API tool name = %s, want 'conversion_test'", apiTool.Function.Name)
	}
	
	if apiTool.Function.Description != "Test conversion to API format" {
		t.Errorf("API tool description = %s, want 'Test conversion to API format'", apiTool.Function.Description)
	}
	
	// Verify parameters are properly converted
	params := apiTool.Function.Parameters
	if params == nil {
		t.Errorf("API tool parameters = nil, want non-nil")
		return
	}
	
	// The parameters should be preserved as-is from the tool
	paramsMap := params
	
	if paramsMap["type"] != "object" {
		t.Errorf("Parameters type = %v, want 'object'", paramsMap["type"])
	}
}