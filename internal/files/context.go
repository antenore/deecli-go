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
	"fmt"
	"strings"
)

type FileContext struct {
	Files      []LoadedFile
	Loader     *FileLoader
	MaxContext int
}

func NewFileContext() *FileContext {
	return &FileContext{
		Files:      []LoadedFile{},
		Loader:     NewFileLoader(),
		MaxContext: 50,
	}
}

func (fc *FileContext) LoadFile(path string) error {
	file, err := fc.Loader.LoadFile(path)
	if err != nil {
		return err
	}

	for i, f := range fc.Files {
		if f.Path == file.Path {
			fc.Files[i] = file
			return nil
		}
	}

	if len(fc.Files) >= fc.MaxContext {
		return fmt.Errorf("context limit reached (%d files)", fc.MaxContext)
	}

	fc.Files = append(fc.Files, file)
	return nil
}

func (fc *FileContext) LoadFiles(patterns []string) error {
	files, err := fc.Loader.LoadFiles(patterns)
	if err != nil {
		return err
	}

	for _, file := range files {
		exists := false
		for i, f := range fc.Files {
			if f.Path == file.Path {
				fc.Files[i] = file
				exists = true
				break
			}
		}

		if !exists {
			if len(fc.Files) >= fc.MaxContext {
				return fmt.Errorf("context limit reached (%d files)", fc.MaxContext)
			}
			fc.Files = append(fc.Files, file)
		}
	}

	return nil
}

func (fc *FileContext) Clear() {
	fc.Files = []LoadedFile{}
}

func (fc *FileContext) RemoveFile(path string) bool {
	absPath := path
	if !strings.HasPrefix(path, "/") {
		for i, f := range fc.Files {
			if f.RelPath == path || strings.HasSuffix(f.Path, path) {
				fc.Files = append(fc.Files[:i], fc.Files[i+1:]...)
				return true
			}
		}
	}

	for i, f := range fc.Files {
		if f.Path == absPath {
			fc.Files = append(fc.Files[:i], fc.Files[i+1:]...)
			return true
		}
	}
	
	return false
}

func (fc *FileContext) GetLoadedPaths() []string {
	paths := make([]string, len(fc.Files))
	for i, f := range fc.Files {
		paths[i] = f.RelPath
	}
	return paths
}

func (fc *FileContext) GetContextSize() int64 {
	var total int64
	for _, f := range fc.Files {
		total += f.Size
	}
	return total
}

func (fc *FileContext) BuildContextPrompt() string {
	if len(fc.Files) == 0 {
		return ""
	}

	var prompt strings.Builder
	prompt.WriteString("I have the following files loaded for context:\n\n")

	for _, file := range fc.Files {
		prompt.WriteString(fmt.Sprintf("=== File: %s (%s) ===\n", file.RelPath, file.Language))
		prompt.WriteString("```")
		if file.Language != "text" {
			prompt.WriteString(file.Language)
		}
		prompt.WriteString("\n")
		prompt.WriteString(file.Content)
		if !strings.HasSuffix(file.Content, "\n") {
			prompt.WriteString("\n")
		}
		prompt.WriteString("```\n\n")
	}

	return prompt.String()
}

func (fc *FileContext) GetInfo() string {
	return fc.Loader.GetFilesInfo(fc.Files)
}

// ReloadFiles reloads files from disk, updating cached content
// If no patterns provided, reloads all currently loaded files
func (fc *FileContext) ReloadFiles(patterns []string) ([]ReloadResult, error) {
	var results []ReloadResult
	var filesToReload []string
	
	if len(patterns) == 0 {
		// Reload all currently loaded files
		for _, f := range fc.Files {
			filesToReload = append(filesToReload, f.Path)
		}
	} else {
		// Expand patterns and filter to only loaded files
		tempFiles, err := fc.Loader.LoadFiles(patterns)
		if err != nil {
			return nil, fmt.Errorf("error expanding patterns: %w", err)
		}
		
		for _, tempFile := range tempFiles {
			// Only include if file is currently loaded
			for _, f := range fc.Files {
				if f.Path == tempFile.Path {
					filesToReload = append(filesToReload, tempFile.Path)
					break
				}
			}
		}
	}
	
	if len(filesToReload) == 0 {
		return results, nil
	}
	
	// Reload each file and track changes
	for _, path := range filesToReload {
		var oldFile *LoadedFile
		var oldIndex int = -1
		
		// Find existing file
		for i, f := range fc.Files {
			if f.Path == path {
				oldFile = &f
				oldIndex = i
				break
			}
		}
		
		if oldFile == nil {
			continue // File not currently loaded
		}
		
		// Load fresh content
		newFile, err := fc.Loader.LoadFile(path)
		if err != nil {
			results = append(results, ReloadResult{
				Path: oldFile.RelPath,
				Error: err.Error(),
				Status: "error",
			})
			continue
		}
		
		// Update in context
		fc.Files[oldIndex] = newFile
		
		// Track result
		status := "unchanged"
		if oldFile.Size != newFile.Size {
			status = "changed"
		} else if oldFile.Content != newFile.Content {
			status = "changed"
		}
		
		results = append(results, ReloadResult{
			Path: newFile.RelPath,
			OldSize: oldFile.Size,
			NewSize: newFile.Size,
			Language: newFile.Language,
			Status: status,
		})
	}
	
	return results, nil
}

// ReloadResult contains information about a file reload operation
type ReloadResult struct {
	Path     string
	OldSize  int64
	NewSize  int64
	Language string
	Status   string // "changed", "unchanged", "error"
	Error    string
}