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
	"fmt"
	"sync"

	"github.com/antenore/deecli/internal/api"
)

// Registry manages available tool functions
type Registry struct {
	mu    sync.RWMutex
	tools map[string]ToolFunction
}

// NewRegistry creates a new tool registry
func NewRegistry() *Registry {
	return &Registry{
		tools: make(map[string]ToolFunction),
	}
}

// Register adds a tool function to the registry
func (r *Registry) Register(tool ToolFunction) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	name := tool.Name()
	if _, exists := r.tools[name]; exists {
		return fmt.Errorf("tool %s already registered", name)
	}

	r.tools[name] = tool
	return nil
}

// Get retrieves a tool function by name
func (r *Registry) Get(name string) (ToolFunction, bool) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	tool, exists := r.tools[name]
	return tool, exists
}

// GetAll returns all registered tools
func (r *Registry) GetAll() []ToolFunction {
	r.mu.RLock()
	defer r.mu.RUnlock()

	tools := make([]ToolFunction, 0, len(r.tools))
	for _, tool := range r.tools {
		tools = append(tools, tool)
	}
	return tools
}

// GetAPITools converts registered tools to API format
func (r *Registry) GetAPITools() []api.Tool {
	r.mu.RLock()
	defer r.mu.RUnlock()

	apiTools := make([]api.Tool, 0, len(r.tools))
	for _, tool := range r.tools {
		apiTools = append(apiTools, api.Tool{
			Type: "function",
			Function: api.Function{
				Name:        tool.Name(),
				Description: tool.Description(),
				Parameters:  tool.Parameters(),
			},
		})
	}
	return apiTools
}

// DefaultRegistry is the global tool registry
var DefaultRegistry = NewRegistry()

// Register adds a tool to the default registry
func Register(tool ToolFunction) error {
	return DefaultRegistry.Register(tool)
}

// Get retrieves a tool from the default registry
func Get(name string) (ToolFunction, bool) {
	return DefaultRegistry.Get(name)
}