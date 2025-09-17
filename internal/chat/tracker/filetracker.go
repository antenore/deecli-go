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
	"path/filepath"
	"regexp"
	"strings"
	"sync"
	"time"
)

// TrackedFile represents a file mentioned in an AI response
type TrackedFile struct {
	Path        string
	Description string
	Timestamp   time.Time
	Source      string // "ai_response", "user_mention", "edit_suggestion"
}

// FileTracker tracks files mentioned in AI responses
type FileTracker struct {
	mu    sync.RWMutex
	files []TrackedFile
}

// NewFileTracker creates a new file tracker
func NewFileTracker() *FileTracker {
	return &FileTracker{
		files: make([]TrackedFile, 0),
	}
}

// ExtractFilesFromResponse extracts file paths from an AI response
func (ft *FileTracker) ExtractFilesFromResponse(response string) []TrackedFile {
	ft.mu.Lock()
	defer ft.mu.Unlock()

	var extracted []TrackedFile

	// Check if this response contains edit suggestions based on keywords
	isEditSuggestionResponse := strings.Contains(response, "Edit Suggestions") ||
		strings.Contains(response, "edit suggestions") ||
		strings.Contains(response, "📝") ||
		strings.Contains(response, "suggested improvements") ||
		strings.Contains(response, "improvements")

	// Pattern 1: Markdown code blocks with file paths
	// Example: ```go:path/to/file.go
	codeBlockPattern := regexp.MustCompile(`(?m)^\x60{3}[\w]*:([^\s\x60]+)`)
	matches := codeBlockPattern.FindAllStringSubmatch(response, -1)
	for _, match := range matches {
		if len(match) > 1 {
			source := "ai_response"
			if isEditSuggestionResponse {
				source = "edit_suggestion"
			}
			file := TrackedFile{
				Path:        cleanPath(match[1]),
				Description: "Code block reference",
				Timestamp:   time.Now(),
				Source:      source,
			}
			extracted = append(extracted, file)
		}
	}

	// Pattern 2: Bullet point file suggestions (for /edit suggestions) - check first
	// Example: "• **filename.ext** - Description"
	bulletPattern := regexp.MustCompile(`(?m)^[•\-*]\s*\*{0,2}([a-zA-Z0-9_\-/]+\.[a-zA-Z0-9]+)\*{0,2}\s*[-–]\s*(.+)$`)
	bulletMatches := bulletPattern.FindAllStringSubmatch(response, -1)
	for _, match := range bulletMatches {
		if len(match) > 2 {
			file := TrackedFile{
				Path:        cleanPath(match[1]),
				Description: strings.TrimSpace(match[2]),
				Timestamp:   time.Now(),
				Source:      "edit_suggestion",
			}
			if !containsFile(extracted, file.Path) {
				extracted = append(extracted, file)
			}
		}
	}

	// Pattern 3: Explicit file mentions with extensions
	// Example: "Edit the file main.go" or "in src/utils/helper.js"
	// Only mark as edit_suggestion if the response seems to be giving suggestions
	filePattern := regexp.MustCompile(`\b([a-zA-Z0-9_\-/]+\.[a-zA-Z0-9]+)\b`)
	matches = filePattern.FindAllStringSubmatch(response, -1)
	for _, match := range matches {
		if len(match) > 1 && isValidFilePath(match[1]) {
			source := "ai_response"
			description := "File mention"

			// If this looks like an edit suggestion response, mark files as suggestions
			if isEditSuggestionResponse {
				source = "edit_suggestion"
				description = "Suggested file"
			}

			file := TrackedFile{
				Path:        cleanPath(match[1]),
				Description: description,
				Timestamp:   time.Now(),
				Source:      source,
			}
			// Don't add if already extracted from bullet points
			if !containsFile(extracted, file.Path) {
				extracted = append(extracted, file)
			}
		}
	}

	// Add extracted files to the tracker
	ft.files = append(ft.files, extracted...)

	// Keep only the most recent 50 files
	if len(ft.files) > 50 {
		ft.files = ft.files[len(ft.files)-50:]
	}

	return extracted
}

// GetRecentFiles returns the most recent tracked files
func (ft *FileTracker) GetRecentFiles(limit int) []TrackedFile {
	ft.mu.RLock()
	defer ft.mu.RUnlock()

	if limit <= 0 || limit > len(ft.files) {
		limit = len(ft.files)
	}

	// Return files in reverse order (most recent first)
	result := make([]TrackedFile, limit)
	for i := 0; i < limit; i++ {
		result[i] = ft.files[len(ft.files)-1-i]
	}

	return result
}

// GetEditSuggestions returns files marked as edit suggestions
func (ft *FileTracker) GetEditSuggestions() []TrackedFile {
	ft.mu.RLock()
	defer ft.mu.RUnlock()

	var suggestions []TrackedFile
	for i := len(ft.files) - 1; i >= 0; i-- {
		if ft.files[i].Source == "edit_suggestion" {
			suggestions = append(suggestions, ft.files[i])
		}
	}

	return suggestions
}

// Clear removes all tracked files
func (ft *FileTracker) Clear() {
	ft.mu.Lock()
	defer ft.mu.Unlock()
	ft.files = make([]TrackedFile, 0)
}

// HasSuggestions returns true if there are any edit suggestions
func (ft *FileTracker) HasSuggestions() bool {
	ft.mu.RLock()
	defer ft.mu.RUnlock()

	for _, file := range ft.files {
		if file.Source == "edit_suggestion" {
			return true
		}
	}
	return false
}

// Helper functions

func cleanPath(path string) string {
	// Remove leading/trailing whitespace and quotes
	path = strings.TrimSpace(path)
	path = strings.Trim(path, `"'`)

	// Clean the path
	return filepath.Clean(path)
}

func isValidFilePath(path string) bool {
	// Check if it has a valid extension
	ext := filepath.Ext(path)
	if ext == "" || len(ext) > 10 {
		return false
	}

	// Check for common programming file extensions
	validExtensions := []string{
		".go", ".js", ".ts", ".jsx", ".tsx", ".py", ".rb", ".java", ".c", ".cpp",
		".h", ".hpp", ".cs", ".php", ".swift", ".kt", ".rs", ".vue", ".html",
		".css", ".scss", ".sass", ".json", ".xml", ".yaml", ".yml", ".toml",
		".md", ".txt", ".sh", ".bash", ".sql", ".lua", ".vim", ".el",
	}

	for _, validExt := range validExtensions {
		if strings.EqualFold(ext, validExt) {
			return true
		}
	}

	return false
}

func containsFile(files []TrackedFile, path string) bool {
	for _, file := range files {
		if file.Path == path {
			return true
		}
	}
	return false
}