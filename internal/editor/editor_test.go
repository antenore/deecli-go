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

package editor

import (
	"os"
	"testing"
)

func TestParseFileAndLine(t *testing.T) {
	// Create a temporary test file
	tmpFile, err := os.CreateTemp("", "test_*.go")
	if err != nil {
		t.Fatal(err)
	}
	defer os.Remove(tmpFile.Name())
	tmpFile.Close()

	tests := []struct {
		name         string
		input        string
		expectedFile string
		expectedLine int
	}{
		{
			name:         "file with line number",
			input:        tmpFile.Name() + ":42",
			expectedFile: tmpFile.Name(),
			expectedLine: 42,
		},
		{
			name:         "file without line number",
			input:        tmpFile.Name(),
			expectedFile: tmpFile.Name(),
			expectedLine: 0,
		},
		{
			name:         "non-existent file with line number",
			input:        "/nonexistent/file.go:123",
			expectedFile: "/nonexistent/file.go",
			expectedLine: 123,
		},
		{
			name:         "file with colon but no valid line",
			input:        tmpFile.Name() + ":abc",
			expectedFile: tmpFile.Name(),
			expectedLine: 0,
		},
		{
			name:         "file with multiple colons - should parse line number correctly",
			input:        "/path/with:colon/file.go:100",
			expectedFile: "/path/with:colon/file.go",
			expectedLine: 100,
		},
		{
			name:         "TODO.md with line number (reported bug)",
			input:        "TODO.md:45",
			expectedFile: "TODO.md",
			expectedLine: 45,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			file, line := ParseFileAndLine(tt.input)
			if file != tt.expectedFile {
				t.Errorf("parseFileAndLine(%q) file = %q, want %q", tt.input, file, tt.expectedFile)
			}
			if line != tt.expectedLine {
				t.Errorf("parseFileAndLine(%q) line = %d, want %d", tt.input, line, tt.expectedLine)
			}
		})
	}
}