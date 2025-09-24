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

package files

import (
	"os"
	"strings"
	"testing"
)

func TestFileLoaderErrorMessages(t *testing.T) {
	loader := NewFileLoader()

	t.Run("file not found error message", func(t *testing.T) {
		_, err := loader.LoadFile("/nonexistent/file.go")
		if err == nil {
			t.Fatal("expected error for non-existent file")
		}

		errMsg := err.Error()
		if !strings.Contains(errMsg, "file not found") {
			t.Errorf("expected 'file not found' in error, got: %s", errMsg)
		}
		if !strings.Contains(errMsg, "Try: /load *.go") {
			t.Errorf("expected helpful suggestion in error, got: %s", errMsg)
		}
	})

	t.Run("directory instead of file error message", func(t *testing.T) {
		// Create a temporary directory
		tmpDir, err := os.MkdirTemp("", "test_dir")
		if err != nil {
			t.Fatal(err)
		}
		defer os.RemoveAll(tmpDir)

		_, err = loader.LoadFile(tmpDir)
		if err == nil {
			t.Fatal("expected error for directory")
		}

		errMsg := err.Error()
		if !strings.Contains(errMsg, "is a directory, not a file") {
			t.Errorf("expected directory error message, got: %s", errMsg)
		}
		if !strings.Contains(errMsg, "Try: /load") {
			t.Errorf("expected helpful suggestion in error, got: %s", errMsg)
		}
	})

	t.Run("no files matching pattern error message", func(t *testing.T) {
		_, err := loader.LoadFiles([]string{"nonexistent_file.xyz"})
		if err == nil {
			t.Fatal("expected error for non-matching pattern")
		}

		errMsg := err.Error()
		if !strings.Contains(errMsg, "no files matching") {
			t.Errorf("expected pattern matching error, got: %s", errMsg)
		}
		if !strings.Contains(errMsg, "Try: /load") {
			t.Errorf("expected helpful suggestion in error, got: %s", errMsg)
		}
	})

	t.Run("file limit exceeded error message", func(t *testing.T) {
		// Set a very low limit for testing
		loader.MaxFiles = 1
		defer func() { loader.MaxFiles = 100 }()

		// Create multiple temp files
		tmpFile1, err := os.CreateTemp("", "test1_*.go")
		if err != nil {
			t.Fatal(err)
		}
		defer os.Remove(tmpFile1.Name())
		tmpFile1.Close()

		tmpFile2, err := os.CreateTemp("", "test2_*.go")
		if err != nil {
			t.Fatal(err)
		}
		defer os.Remove(tmpFile2.Name())
		tmpFile2.Close()

		_, err = loader.LoadFiles([]string{tmpFile1.Name(), tmpFile2.Name()})
		if err == nil {
			t.Fatal("expected error for file limit exceeded")
		}

		errMsg := err.Error()
		if !strings.Contains(errMsg, "exceeds maximum limit") {
			t.Errorf("expected file limit error, got: %s", errMsg)
		}
		if !strings.Contains(errMsg, "Use more specific patterns") {
			t.Errorf("expected helpful suggestion in error, got: %s", errMsg)
		}
	})
}

func TestPatternValidation(t *testing.T) {
	loader := NewFileLoader()

	tests := []struct {
		name          string
		pattern       string
		expectError   bool
		errorContains string
	}{
		{
			name:        "valid simple pattern",
			pattern:     "*.go",
			expectError: false,
		},
		{
			name:        "valid specific file",
			pattern:     "main.go",
			expectError: false,
		},
		{
			name:        "valid recursive pattern",
			pattern:     "src/**/*.go",
			expectError: false,
		},
		{
			name:          "empty pattern",
			pattern:       "",
			expectError:   true,
			errorContains: "empty pattern not allowed",
		},
		{
			name:          "too broad star pattern",
			pattern:       "*",
			expectError:   true,
			errorContains: "matches all files, which may be too broad",
		},
		{
			name:          "problematic recursive pattern",
			pattern:       "src/**/*",
			expectError:   true,
			errorContains: "may match too many files",
		},
		{
			name:          "invalid double star pattern",
			pattern:       "src/**/**/*.go",
			expectError:   true,
			errorContains: "invalid ** pattern",
		},
		{
			name:          "node_modules pattern",
			pattern:       "node_modules/**/*.js",
			expectError:   true,
			errorContains: "includes 'node_modules' directory",
		},
		{
			name:          "git directory pattern",
			pattern:       ".git/**/*",
			expectError:   true,
			errorContains: "includes '.git' directory",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := loader.validatePattern(tt.pattern)
			if tt.expectError {
				if err == nil {
					t.Errorf("expected error for pattern %q, got none", tt.pattern)
				} else if !strings.Contains(err.Error(), tt.errorContains) {
					t.Errorf("expected error containing %q, got: %s", tt.errorContains, err.Error())
				}
			} else {
				if err != nil {
					t.Errorf("expected no error for pattern %q, got: %s", tt.pattern, err.Error())
				}
			}
		})
	}
}