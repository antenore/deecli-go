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

package tracker

import (
	"strings"
	"testing"
	"time"
)

func TestNewFileTracker(t *testing.T) {
	ft := NewFileTracker()
	if ft == nil {
		t.Fatal("NewFileTracker returned nil")
	}
	if len(ft.files) != 0 {
		t.Errorf("Expected empty files list, got %d files", len(ft.files))
	}
}

func TestExtractFilesFromResponse_CodeBlocks(t *testing.T) {
	ft := NewFileTracker()

	response := `
Here's the code:

` + "```go:internal/main.go" + `
package main

func main() {
	fmt.Println("Hello")
}
` + "```" + `

And another file:

` + "```js:src/app.js" + `
console.log("test");
` + "```"

	extracted := ft.ExtractFilesFromResponse(response)

	if len(extracted) != 2 {
		t.Errorf("Expected 2 files, got %d", len(extracted))
	}

	if extracted[0].Path != "internal/main.go" {
		t.Errorf("Expected path 'internal/main.go', got '%s'", extracted[0].Path)
	}

	if extracted[1].Path != "src/app.js" {
		t.Errorf("Expected path 'src/app.js', got '%s'", extracted[1].Path)
	}
}

func TestExtractFilesFromResponse_FileMentions(t *testing.T) {
	ft := NewFileTracker()

	response := `
Let me explain the file structure:
- The main.go file contains the entry point
- Configuration is in config/app.yaml
- Tests are in main_test.go
`

	extracted := ft.ExtractFilesFromResponse(response)

	expectedFiles := []string{"main.go", "config/app.yaml", "main_test.go"}

	if len(extracted) != len(expectedFiles) {
		t.Errorf("Expected %d files, got %d", len(expectedFiles), len(extracted))
	}

	for i, expected := range expectedFiles {
		if i < len(extracted) && extracted[i].Path != expected {
			t.Errorf("Expected file %s, got %s", expected, extracted[i].Path)
		}
	}
}

func TestExtractFilesFromResponse_BulletPoints(t *testing.T) {
	ft := NewFileTracker()

	response := `
📝 Edit Suggestions:
• **main.go** - Add error handling for file operations
• **utils/helper.js** - Optimize the search algorithm
• **test.py** - Fix the broken unit tests
`

	extracted := ft.ExtractFilesFromResponse(response)

	if len(extracted) != 3 {
		t.Errorf("Expected 3 files, got %d", len(extracted))
	}

	// Check first suggestion
	if extracted[0].Path != "main.go" {
		t.Errorf("Expected path 'main.go', got '%s'", extracted[0].Path)
	}
	if extracted[0].Source != "edit_suggestion" {
		t.Errorf("Expected source 'edit_suggestion', got '%s'", extracted[0].Source)
	}
	if !strings.Contains(extracted[0].Description, "error handling") {
		t.Errorf("Expected description to contain 'error handling', got '%s'", extracted[0].Description)
	}
}

func TestGetRecentFiles(t *testing.T) {
	ft := NewFileTracker()

	// Add some files
	ft.files = []TrackedFile{
		{Path: "file1.go", Timestamp: time.Now().Add(-3 * time.Minute)},
		{Path: "file2.go", Timestamp: time.Now().Add(-2 * time.Minute)},
		{Path: "file3.go", Timestamp: time.Now().Add(-1 * time.Minute)},
	}

	recent := ft.GetRecentFiles(2)

	if len(recent) != 2 {
		t.Errorf("Expected 2 recent files, got %d", len(recent))
	}

	// Should return in reverse order (most recent first)
	if recent[0].Path != "file3.go" {
		t.Errorf("Expected most recent file 'file3.go', got '%s'", recent[0].Path)
	}
	if recent[1].Path != "file2.go" {
		t.Errorf("Expected second recent file 'file2.go', got '%s'", recent[1].Path)
	}
}

func TestGetEditSuggestions(t *testing.T) {
	ft := NewFileTracker()

	// Add mixed files
	ft.files = []TrackedFile{
		{Path: "file1.go", Source: "ai_response"},
		{Path: "file2.go", Source: "edit_suggestion", Description: "Fix bug"},
		{Path: "file3.go", Source: "user_mention"},
		{Path: "file4.go", Source: "edit_suggestion", Description: "Add feature"},
	}

	suggestions := ft.GetEditSuggestions()

	if len(suggestions) != 2 {
		t.Errorf("Expected 2 edit suggestions, got %d", len(suggestions))
	}

	// Should return in reverse order
	if suggestions[0].Path != "file4.go" {
		t.Errorf("Expected 'file4.go', got '%s'", suggestions[0].Path)
	}
	if suggestions[1].Path != "file2.go" {
		t.Errorf("Expected 'file2.go', got '%s'", suggestions[1].Path)
	}
}

func TestHasSuggestions(t *testing.T) {
	ft := NewFileTracker()

	// Initially no suggestions
	if ft.HasSuggestions() {
		t.Error("Expected no suggestions initially")
	}

	// Add non-suggestion file
	ft.files = append(ft.files, TrackedFile{
		Path:   "test.go",
		Source: "ai_response",
	})

	if ft.HasSuggestions() {
		t.Error("Expected no suggestions with only ai_response files")
	}

	// Add edit suggestion
	ft.files = append(ft.files, TrackedFile{
		Path:   "fix.go",
		Source: "edit_suggestion",
	})

	if !ft.HasSuggestions() {
		t.Error("Expected to have suggestions after adding edit_suggestion")
	}
}

func TestClear(t *testing.T) {
	ft := NewFileTracker()

	// Add files
	ft.files = []TrackedFile{
		{Path: "file1.go"},
		{Path: "file2.go"},
	}

	ft.Clear()

	if len(ft.files) != 0 {
		t.Errorf("Expected empty files after Clear(), got %d files", len(ft.files))
	}
}

func TestMaxFileLimit(t *testing.T) {
	ft := NewFileTracker()

	// Add 60 files (more than the 50 limit)
	for i := 0; i < 60; i++ {
		response := "Check file" + strings.TrimSpace(strings.Repeat("x", i%10)) + ".go"
		ft.ExtractFilesFromResponse(response)
	}

	if len(ft.files) > 50 {
		t.Errorf("Expected max 50 files, got %d", len(ft.files))
	}
}

func TestIsValidFilePath(t *testing.T) {
	tests := []struct {
		path  string
		valid bool
	}{
		{"main.go", true},
		{"src/app.js", true},
		{"test.py", true},
		{"config.yaml", true},
		{"README.md", true},
		{"no_extension", false},
		{"file.unknownext", false},
		{"file.", false},
		{".gitignore", false}, // No base name
	}

	for _, test := range tests {
		result := isValidFilePath(test.path)
		if result != test.valid {
			t.Errorf("isValidFilePath(%s) = %v, expected %v", test.path, result, test.valid)
		}
	}
}

func TestCleanPath(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"  main.go  ", "main.go"},
		{`"quoted.go"`, "quoted.go"},
		{`'single.go'`, "single.go"},
		{"./relative/path.go", "relative/path.go"},
		{"//double//slashes.go", "/double/slashes.go"},
	}

	for _, test := range tests {
		result := cleanPath(test.input)
		if result != test.expected {
			t.Errorf("cleanPath(%s) = %s, expected %s", test.input, result, test.expected)
		}
	}
}

func TestConcurrentAccess(t *testing.T) {
	ft := NewFileTracker()
	done := make(chan bool)

	// Writer goroutine
	go func() {
		for i := 0; i < 100; i++ {
			ft.ExtractFilesFromResponse("Check file.go")
		}
		done <- true
	}()

	// Reader goroutine
	go func() {
		for i := 0; i < 100; i++ {
			ft.GetRecentFiles(5)
			ft.HasSuggestions()
		}
		done <- true
	}()

	// Wait for both
	<-done
	<-done

	// If we get here without deadlock or panic, concurrent access works
}