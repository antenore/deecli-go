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
	"testing"
)

func TestReadFileTool_Execute(t *testing.T) {
	// Create a temporary directory for testing
	tempDir := t.TempDir()
	
	// Create test files
	testFile := filepath.Join(tempDir, "test.txt")
	testContent := "Hello, World!\nThis is a test file.\n"
	
	err := os.WriteFile(testFile, []byte(testContent), 0644)
	if err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}
	
	// Create empty file
	emptyFile := filepath.Join(tempDir, "empty.txt")
	err = os.WriteFile(emptyFile, []byte(""), 0644)
	if err != nil {
		t.Fatalf("Failed to create empty test file: %v", err)
	}
	
	tool := &ReadFile{}
	
	tests := []struct {
		name        string
		args        map[string]interface{}
		wantSuccess bool
		wantContent string
		wantError   string
	}{
		{
			name: "successful read",
			args: map[string]interface{}{
				"path": testFile,
			},
			wantSuccess: true,
			wantContent: testContent,
		},
		{
			name: "read empty file",
			args: map[string]interface{}{
				"path": emptyFile,
			},
			wantSuccess: true,
			wantContent: "",
		},
		{
			name: "nonexistent file",
			args: map[string]interface{}{
				"path": filepath.Join(tempDir, "nonexistent.txt"),
			},
			wantSuccess: false,
			wantError:   "no such file or directory",
		},
		{
			name: "missing path argument",
			args: map[string]interface{}{},
			wantSuccess: false,
			wantError:   "path is required",
		},
		{
			name: "empty path argument",
			args: map[string]interface{}{
				"path": "",
			},
			wantSuccess: false,
			wantError:   "path cannot be empty",
		},
		{
			name: "invalid path type",
			args: map[string]interface{}{
				"path": 123,
			},
			wantSuccess: false,
			wantError:   "path must be a string",
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
					t.Errorf("Execute() result = nil, want non-nil")
					return
				}
				
				if err != nil {
					t.Errorf("Execute() err == nil = false, want true")
				}
				
				if result != tt.wantContent {
					t.Errorf("Execute() result = %q, want %q", result, tt.wantContent)
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
					// Check if error message contains expected substring
					if !containsIgnoreCase(errorMsg, tt.wantError) {
						t.Errorf("Execute() error = %q, want to contain %q", errorMsg, tt.wantError)
					}
				}
			}
		})
	}
}

func TestReadFileTool_Metadata(t *testing.T) {
	tool := &ReadFile{}
	
	if tool.Name() != "read_file" {
		t.Errorf("Name() = %q, want %q", tool.Name(), "read_file")
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
	
	pathParam, ok := properties["path"]
	if !ok {
		t.Errorf("Parameters missing 'path' property")
		return
	}
	
	pathParamMap, ok := pathParam.(map[string]interface{})
	if !ok {
		t.Errorf("Path parameter not a map")
		return
	}
	
	if pathParamMap["type"] != "string" {
		t.Errorf("Path parameter type = %v, want %q", pathParamMap["type"], "string")
	}
}

func TestReadFileTool_BinaryFile(t *testing.T) {
	// Create a temporary directory for testing
	tempDir := t.TempDir()
	
	// Create a binary file (with null bytes)
	binaryFile := filepath.Join(tempDir, "binary.bin")
	binaryContent := []byte{0x00, 0x01, 0x02, 0xFF, 0xFE}
	
	err := os.WriteFile(binaryFile, binaryContent, 0644)
	if err != nil {
		t.Fatalf("Failed to create binary test file: %v", err)
	}
	
	tool := &ReadFile{}
	
	args := map[string]interface{}{
		"path": binaryFile,
	}
	
	argsJSON, err := json.Marshal(args)
	if err != nil {
		t.Fatalf("Failed to marshal args: %v", err)
	}
	
	result, err := tool.Execute(context.Background(), json.RawMessage(argsJSON))
	
	// Should succeed but may have different handling for binary content
	if err != nil {
		t.Errorf("Execute() error = %v, want nil for binary file", err)
	}
	
	if result == "" {
		t.Errorf("Execute() result = nil, want non-nil")
		return
	}
	
	if err != nil {
		t.Errorf("Execute() err == nil = false, want true for binary file")
	}
	
	// Binary content should be readable (may be converted to string representation)
	if result == "" {
		t.Errorf("Execute() result = empty, want non-empty for binary file")
	}
}

// containsIgnoreCase checks if s contains substr (case-insensitive)
func containsIgnoreCase(s, substr string) bool {
	return len(s) >= len(substr) && 
		   (s == substr || 
		    len(substr) == 0 || 
		    (len(s) > 0 && len(substr) > 0 && 
		     containsIgnoreCaseHelper(s, substr)))
}

func containsIgnoreCaseHelper(s, substr string) bool {
	sLower := toLower(s)
	substrLower := toLower(substr)
	
	for i := 0; i <= len(sLower)-len(substrLower); i++ {
		if sLower[i:i+len(substrLower)] == substrLower {
			return true
		}
	}
	return false
}

func toLower(s string) string {
	result := make([]byte, len(s))
	for i, b := range []byte(s) {
		if b >= 'A' && b <= 'Z' {
			result[i] = b + 32
		} else {
			result[i] = b
		}
	}
	return string(result)
}