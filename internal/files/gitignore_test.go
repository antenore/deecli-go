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
	"strings"
	"testing"

	"github.com/sabhiram/go-gitignore"
)

func TestGitignoreFilter(t *testing.T) {
	// Test with gitignore disabled
	t.Run("disabled filter", func(t *testing.T) {
		filter := NewGitignoreFilter(false)
		if filter.ShouldIgnore("node_modules/test.js") {
			t.Error("disabled filter should not ignore any files")
		}
	})

	// Test with gitignore enabled but no .gitignore file
	t.Run("enabled filter without gitignore file", func(t *testing.T) {
		filter := NewGitignoreFilter(true)

		// Should always ignore .git (our fallback creates this pattern)
		if !filter.ShouldIgnore(".git/config") {
			t.Error("should always ignore .git directory")
		}

		// Should not ignore regular files when no .gitignore exists
		if filter.ShouldIgnore("main.go") {
			t.Error("should not ignore regular files when no .gitignore exists")
		}
	})

	// Test with gitignore patterns using the library
	t.Run("enabled filter with gitignore patterns", func(t *testing.T) {
		// Test with common gitignore patterns using the library directly
		gitignoreContent := `# Test gitignore
*.log
node_modules/
dist
*.tmp
/build/
src/test/
`
		// Create filter using the library's pattern compilation
		ignorer := ignore.CompileIgnoreLines(strings.Split(gitignoreContent, "\n")...)
		filter := &GitignoreFilter{
			enabled: true,
			ignorer: ignorer,
		}

		tests := []struct {
			path   string
			ignore bool
		}{
			{"app.log", true},
			{"debug.log", true},
			{"logs/app.log", true},
			{"main.go", false},
			{"node_modules/package.json", true},
			{"node_modules/lib/index.js", true},
			{"dist", true},
			{"dist/main.js", true},
			{"temp.tmp", true},
			{"src/temp.tmp", true},
			{"build/output", true},
			{"scripts/build/test", false}, // /build/ matches only at root
			{"src/test/helper.go", true},
			{"test/main.go", false}, // src/test/ is specific
		}

		for _, tt := range tests {
			if filter.ShouldIgnore(tt.path) != tt.ignore {
				t.Errorf("ShouldIgnore(%q) = %v, want %v", tt.path, !tt.ignore, tt.ignore)
			}
		}
	})
}

func TestGitignoreLibraryIntegration(t *testing.T) {
	// Test that our integration with the library works correctly
	tests := []struct {
		pattern string
		path    string
		matches bool
	}{
		// Simple wildcard patterns
		{"*.log", "test.log", true},
		{"*.log", "app.log", true},
		{"*.log", "main.go", false},

		// Directory patterns
		{"node_modules/", "node_modules/test.js", true},
		{"node_modules/", "src/node_modules/lib.js", true},
		{"node_modules/", "modules/test.js", false},

		// Exact match
		{"dist", "dist", true},
		{"dist", "dist/main.js", true},
		{"dist", "mydist", false},

		// Complex patterns
		{"*.tmp", "temp.tmp", true},
		{"*.tmp", "src/temp.tmp", true},
		{"*.tmp", "temp.txt", false},
	}

	for _, tt := range tests {
		ignorer := ignore.CompileIgnoreLines(tt.pattern)
		filter := &GitignoreFilter{
			enabled: true,
			ignorer: ignorer,
		}

		if filter.ShouldIgnore(tt.path) != tt.matches {
			t.Errorf("pattern %q with path %q: got %v, want %v", tt.pattern, tt.path, !tt.matches, tt.matches)
		}
	}
}