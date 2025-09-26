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

package functions

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestListFilesTool_Execute(t *testing.T) {
	// Create a temporary directory structure for testing
	tempDir := t.TempDir()
	
	// Create test structure:
	// tempDir/
	//   ├── file1.txt
	//   ├── file2.go
	//   ├── subdir/
	//   │   ├── file3.txt
	//   │   └── file4.py
	//   └── .hidden
	
	// Create files
	files := []string{
		"file1.txt",
		"file2.go",
		".hidden",
	}
	
	for _, file := range files {
		filePath := filepath.Join(tempDir, file)
		err := os.WriteFile(filePath, []byte("test content"), 0644)
		if err != nil {
			t.Fatalf("Failed to create test file %s: %v", file, err)
		}
	}
	
	// Create subdirectory and files
	subDir := filepath.Join(tempDir, "subdir")
	err := os.Mkdir(subDir, 0755)
	if err != nil {
		t.Fatalf("Failed to create subdirectory: %v", err)
	}
	
	subFiles := []string{"file3.txt", "file4.py"}
	for _, file := range subFiles {
		filePath := filepath.Join(subDir, file)
		err := os.WriteFile(filePath, []byte("sub content"), 0644)
		if err != nil {
			t.Fatalf("Failed to create sub file %s: %v", file, err)
		}
	}
	
	tool := &ListFiles{}
	
	tests := []struct {
		name           string
		args           map[string]interface{}
		wantSuccess    bool
		wantContains   []string
		wantNotContain []string
		wantError      string
	}{
		{
			name: "list current directory (empty args)",
			args: map[string]interface{}{},
			wantSuccess: true,
			// This will list the current working directory, which varies
		},
		{
			name: "list specific directory",
			args: map[string]interface{}{
				"path": tempDir,
			},
			wantSuccess: true,
			wantContains: []string{
				"file1.txt",
				"file2.go", 
				"subdir",
			},
			wantNotContain: []string{
				"file3.txt", // Should not be in non-recursive listing
				"file4.py",
			},
		},
		{
			name: "list with recursive=false",
			args: map[string]interface{}{
				"path":      tempDir,
				"recursive": false,
			},
			wantSuccess: true,
			wantContains: []string{
				"file1.txt",
				"file2.go",
				"subdir",
			},
			wantNotContain: []string{
				"file3.txt",
				"file4.py",
			},
		},
		{
			name: "list with recursive=true",
			args: map[string]interface{}{
				"path":      tempDir,
				"recursive": true,
			},
			wantSuccess: true,
			wantContains: []string{
				"file1.txt",
				"file2.go",
				"subdir",
				"file3.txt",
				"file4.py",
			},
		},
		{
			name: "nonexistent directory",
			args: map[string]interface{}{
				"path": filepath.Join(tempDir, "nonexistent"),
			},
			wantSuccess: false,
			wantError:   "no such file or directory",
		},
		{
			name: "invalid path type",
			args: map[string]interface{}{
				"path": 123,
			},
			wantSuccess: false,
			wantError:   "path must be a string",
		},
		{
			name: "invalid recursive type",
			args: map[string]interface{}{
				"path":      tempDir,
				"recursive": "invalid",
			},
			wantSuccess: false,
			wantError:   "recursive must be a boolean",
		},
	}
	
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			argsJSON, err := json.Marshal(tt.args)
			if err != nil {
				t.Fatalf("Failed to marshal args: %v", err)
			}
			
			result, err := tool.Execute(context.Background(), json.RawMessage(argsJSON))
			
			if tt.wantSuccess {
				if err != nil {
					t.Errorf("Execute() error = %v, want nil", err)
					return
				}
				
				if result == "" {
					t.Errorf("Execute() result = empty, want non-empty")
					return
				}

				output := result
				
				// Check that wanted files are contained
				for _, want := range tt.wantContains {
					if !strings.Contains(output, want) {
						t.Errorf("Execute() result missing %q", want)
						t.Logf("Output: %s", output)
					}
				}
				
				// Check that unwanted files are not contained
				for _, unwant := range tt.wantNotContain {
					if strings.Contains(output, unwant) {
						t.Errorf("Execute() result unexpectedly contains %q", unwant)
						t.Logf("Output: %s", output)
					}
				}
				
			} else {
				if err == nil && (result == "" || err == nil) {
					t.Errorf("Execute() should have failed but didn't")
					return
				}
				
				var errorMsg string
				if err != nil {
					errorMsg = err.Error()
				} else if result != "" && err != nil {
					errorMsg = err.Error()
				}
				
				if tt.wantError != "" && errorMsg == "" {
					t.Errorf("Execute() no error message, want %q", tt.wantError)
				}
				
				if tt.wantError != "" && errorMsg != "" {
					if !containsIgnoreCase(errorMsg, tt.wantError) {
						t.Errorf("Execute() error = %q, want to contain %q", errorMsg, tt.wantError)
					}
				}
			}
		})
	}
}

func TestListFilesTool_Metadata(t *testing.T) {
	tool := &ListFiles{}
	
	if tool.Name() != "list_files" {
		t.Errorf("Name() = %q, want %q", tool.Name(), "list_files")
	}
	
	description := tool.Description()
	if description == "" {
		t.Errorf("Description() = empty, want non-empty")
	}
	
	parameters := tool.Parameters()
	if parameters == nil {
		t.Errorf("Parameters() = nil, want non-nil")
	}
	
	// Check parameter structure
	if parameters["type"] != "object" {
		t.Errorf("Parameters type = %v, want %q", parameters["type"], "object")
	}
	
	properties, ok := parameters["properties"].(map[string]interface{})
	if !ok {
		t.Errorf("Parameters properties not a map")
		return
	}
	
	// Check path parameter (should be optional)
	if pathParam, exists := properties["path"]; exists {
		pathParamMap, ok := pathParam.(map[string]interface{})
		if !ok {
			t.Errorf("Path parameter not a map")
		} else if pathParamMap["type"] != "string" {
			t.Errorf("Path parameter type = %v, want %q", pathParamMap["type"], "string")
		}
	}
	
	// Check recursive parameter
	recursiveParam, ok := properties["recursive"]
	if !ok {
		t.Errorf("Parameters missing 'recursive' property")
		return
	}
	
	recursiveParamMap, ok := recursiveParam.(map[string]interface{})
	if !ok {
		t.Errorf("Recursive parameter not a map")
		return
	}
	
	if recursiveParamMap["type"] != "boolean" {
		t.Errorf("Recursive parameter type = %v, want %q", recursiveParamMap["type"], "boolean")
	}
}

func TestListFilesTool_HiddenFiles(t *testing.T) {
	// Create a temporary directory with hidden files
	tempDir := t.TempDir()
	
	// Create regular and hidden files
	files := []string{"visible.txt", ".hidden", ".gitignore"}
	
	for _, file := range files {
		filePath := filepath.Join(tempDir, file)
		err := os.WriteFile(filePath, []byte("test"), 0644)
		if err != nil {
			t.Fatalf("Failed to create test file %s: %v", file, err)
		}
	}
	
	tool := &ListFiles{}
	
	args := map[string]interface{}{
		"path": tempDir,
	}
	
	argsJSON, err := json.Marshal(args)
	if err != nil {
		t.Fatalf("Failed to marshal args: %v", err)
	}
	
	result, err := tool.Execute(context.Background(), json.RawMessage(argsJSON))
	
	if err != nil {
		t.Errorf("Execute() error = %v, want nil", err)
		return
	}
	
	if result == "" || err != nil {
		t.Errorf("Execute() should succeed for directory with hidden files")
		return
	}
	
	output := result
	
	// Should contain visible file
	if !strings.Contains(output, "visible.txt") {
		t.Errorf("Execute() result missing visible file")
	}
	
	// Behavior with hidden files may vary by implementation
	// This test just ensures it doesn't crash
	t.Logf("Output with hidden files: %s", output)
}