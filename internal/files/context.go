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
	"context"
	"fmt"
	"strings"
	"sync"
	"time"
)

type FileContext struct {
	Files             []LoadedFile
	Loader            *FileLoader
	MaxContext        int
	watcher           *FileWatcher
	autoReloadEnabled bool
	reloadMutex       sync.Mutex
	lastManualReload  time.Time // Track manual reloads
	reloadCallback    func([]ReloadResult) // Callback for auto-reload notifications
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

	// Add to watcher if auto-reload is enabled
	if fc.autoReloadEnabled && fc.watcher != nil {
		if err := fc.watcher.Watch(file.Path); err != nil {
			// Log but don't fail
			fmt.Printf("Warning: Could not watch %s: %v\n", file.Path, err)
		}
	}

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

		// Add to watcher if auto-reload is enabled
		if fc.autoReloadEnabled && fc.watcher != nil {
			if err := fc.watcher.Watch(file.Path); err != nil {
				// Log but don't fail
				fmt.Printf("Warning: Could not watch %s: %v\n", file.Path, err)
			}
		}
	}

	return nil
}

func (fc *FileContext) Clear() {
	// Unwatch all files if watcher is active
	if fc.watcher != nil && fc.autoReloadEnabled {
		fc.watcher.UnwatchAll()
	}
	fc.Files = []LoadedFile{}
}

func (fc *FileContext) RemoveFile(path string) bool {
	absPath := path
	removedPath := ""

	if !strings.HasPrefix(path, "/") {
		for i, f := range fc.Files {
			if f.RelPath == path || strings.HasSuffix(f.Path, path) {
				removedPath = f.Path
				fc.Files = append(fc.Files[:i], fc.Files[i+1:]...)
				break
			}
		}
	} else {
		for i, f := range fc.Files {
			if f.Path == absPath {
				removedPath = f.Path
				fc.Files = append(fc.Files[:i], fc.Files[i+1:]...)
				break
			}
		}
	}

	// Unwatch the removed file if watcher is active
	if removedPath != "" {
		if fc.watcher != nil && fc.autoReloadEnabled {
			fc.watcher.Unwatch(removedPath)
		}
		return true
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
	fc.reloadMutex.Lock()
	defer fc.reloadMutex.Unlock()

	// Mark this as a manual reload
	fc.lastManualReload = time.Now()

	// If watcher exists, mark files to skip auto-reload for 500ms
	if fc.watcher != nil {
		var allPaths []string
		for _, file := range fc.Files {
			allPaths = append(allPaths, file.Path)
		}
		fc.watcher.MarkReloadCompleted(allPaths)
	}

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

// SetWatcher sets the file watcher for auto-reload functionality
func (fc *FileContext) SetWatcher(w *FileWatcher) {
	fc.watcher = w
}

// EnableAutoReload enables automatic file reloading on changes
func (fc *FileContext) EnableAutoReload(ctx context.Context, notificationCallback func([]ReloadResult)) error {
	if fc.watcher == nil || !fc.watcher.IsSupported() {
		return fmt.Errorf("file watching not supported on this platform")
	}

	fc.autoReloadEnabled = true
	fc.reloadCallback = notificationCallback

	// Start watcher with reload callback
	fc.watcher.Start(ctx, func(paths []string) error {
		// Perform the reload
		results, err := fc.autoReloadFiles(paths)
		if err != nil {
			return err
		}

		// Notify about reload if callback is set
		if fc.reloadCallback != nil && len(results) > 0 {
			fc.reloadCallback(results)
		}

		return nil
	})

	// Watch all currently loaded files
	for _, file := range fc.Files {
		if err := fc.watcher.Watch(file.Path); err != nil {
			// Log but don't fail
			fmt.Printf("Warning: Could not watch %s: %v\n", file.Path, err)
		}
	}

	return nil
}

// DisableAutoReload disables automatic file reloading
func (fc *FileContext) DisableAutoReload() {
	fc.autoReloadEnabled = false
	if fc.watcher != nil {
		fc.watcher.UnwatchAll()
	}
}

// autoReloadFiles performs automatic reload without duplicate prevention interference
func (fc *FileContext) autoReloadFiles(paths []string) ([]ReloadResult, error) {
	fc.reloadMutex.Lock()
	defer fc.reloadMutex.Unlock()

	var results []ReloadResult

	for _, path := range paths {
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

// IsAutoReloadSupported returns true if auto-reload is supported on this platform
func (fc *FileContext) IsAutoReloadSupported() bool {
	return fc.watcher != nil && fc.watcher.IsSupported()
}

// IsAutoReloadEnabled returns true if auto-reload is currently enabled
func (fc *FileContext) IsAutoReloadEnabled() bool {
	return fc.autoReloadEnabled && fc.IsAutoReloadSupported()
}