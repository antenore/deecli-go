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
	"path/filepath"

	"github.com/sabhiram/go-gitignore"
)

// GitignoreFilter provides gitignore filtering functionality using a proven library
type GitignoreFilter struct {
	ignorer *ignore.GitIgnore
	enabled bool
}

// NewGitignoreFilter creates a new gitignore filter
// If respectGitignore is false, all files will be allowed
func NewGitignoreFilter(respectGitignore bool) *GitignoreFilter {
	gf := &GitignoreFilter{
		enabled: respectGitignore,
	}

	if respectGitignore {
		// Try to load .gitignore file, create empty one if it doesn't exist
		ignorer, err := ignore.CompileIgnoreFile(".gitignore")
		if err != nil {
			// No .gitignore file or error reading it - create minimal ignore patterns
			// Always ignore .git directory
			ignorer = ignore.CompileIgnoreLines(".git/")
		}
		gf.ignorer = ignorer
	}

	return gf
}

// ShouldIgnore returns true if the file path should be ignored according to .gitignore
func (gf *GitignoreFilter) ShouldIgnore(path string) bool {
	if !gf.enabled || gf.ignorer == nil {
		return false
	}

	// Convert to relative path from current directory
	relPath, err := filepath.Rel(".", path)
	if err != nil {
		relPath = path
	}

	// Use the battle-tested gitignore library
	return gf.ignorer.MatchesPath(relPath)
}