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
	"log"
	"path/filepath"
	"sync"
	"time"

	"github.com/fsnotify/fsnotify"
)

// FileWatcher monitors files for changes and triggers reload callbacks
type FileWatcher struct {
	watcher          *fsnotify.Watcher    // May be nil on unsupported platforms
	watchedPaths     map[string]time.Time // Track paths and last modification time
	reloadInProgress map[string]bool      // Prevent duplicate reloads
	debounceTimer    *time.Timer
	debounceDelay    time.Duration
	reloadChan       chan []string
	stopChan         chan struct{}
	mu               sync.RWMutex
	supported        bool                 // Platform support flag
	lastReloadTime   map[string]time.Time // Track recent reloads to prevent duplicates
	reloadCallback   func([]string) error // Callback function for reloading files
}

// NewWatcher creates a new file watcher with OS compatibility check
func NewWatcher(debounceDelay time.Duration) (*FileWatcher, error) {
	if debounceDelay == 0 {
		debounceDelay = 100 * time.Millisecond
	}

	fw := &FileWatcher{
		watchedPaths:     make(map[string]time.Time),
		reloadInProgress: make(map[string]bool),
		lastReloadTime:   make(map[string]time.Time),
		debounceDelay:    debounceDelay,
		reloadChan:       make(chan []string, 10),
		stopChan:         make(chan struct{}),
		supported:        true,
	}

	// Try to create fsnotify watcher
	watcher, err := fsnotify.NewWatcher()
	if err != nil {
		// Platform doesn't support file watching
		fw.supported = false
		log.Printf("⚠️ File watching not supported on this platform: %v", err)
		log.Printf("   Manual reload with /reload command will be required")
		return fw, nil // Return degraded watcher, not error
	}

	fw.watcher = watcher
	return fw, nil
}

// IsSupported returns true if file watching is supported on this platform
func (fw *FileWatcher) IsSupported() bool {
	return fw.supported && fw.watcher != nil
}

// Watch adds a file to the watch list
func (fw *FileWatcher) Watch(path string) error {
	if !fw.IsSupported() {
		return nil // Silently ignore on unsupported platforms
	}

	fw.mu.Lock()
	defer fw.mu.Unlock()

	// Get absolute path
	absPath, err := filepath.Abs(path)
	if err != nil {
		return err
	}

	// Check if already watching
	if _, exists := fw.watchedPaths[absPath]; exists {
		return nil
	}

	// Add to watcher
	if err := fw.watcher.Add(absPath); err != nil {
		return err
	}

	fw.watchedPaths[absPath] = time.Now()
	return nil
}

// Unwatch removes a file from the watch list
func (fw *FileWatcher) Unwatch(path string) error {
	if !fw.IsSupported() {
		return nil
	}

	fw.mu.Lock()
	defer fw.mu.Unlock()

	absPath, err := filepath.Abs(path)
	if err != nil {
		return err
	}

	// Remove from watcher
	if _, exists := fw.watchedPaths[absPath]; exists {
		if err := fw.watcher.Remove(absPath); err != nil {
			return err
		}
		delete(fw.watchedPaths, absPath)
		delete(fw.reloadInProgress, absPath)
		delete(fw.lastReloadTime, absPath)
	}

	return nil
}

// UnwatchAll removes all files from the watch list
func (fw *FileWatcher) UnwatchAll() error {
	if !fw.IsSupported() {
		return nil
	}

	fw.mu.Lock()
	defer fw.mu.Unlock()

	for path := range fw.watchedPaths {
		if err := fw.watcher.Remove(path); err != nil {
			log.Printf("Error removing watch for %s: %v", path, err)
		}
	}

	fw.watchedPaths = make(map[string]time.Time)
	fw.reloadInProgress = make(map[string]bool)
	fw.lastReloadTime = make(map[string]time.Time)

	return nil
}

// ShouldReload checks if a file should be reloaded (not recently reloaded)
func (fw *FileWatcher) ShouldReload(path string) bool {
	fw.mu.RLock()
	defer fw.mu.RUnlock()

	absPath, _ := filepath.Abs(path)

	// Check if reload is already in progress
	if fw.reloadInProgress[absPath] {
		return false
	}

	// Check if recently reloaded (within 500ms to prevent duplicates from /edit)
	if lastTime, exists := fw.lastReloadTime[absPath]; exists {
		if time.Since(lastTime) < 500*time.Millisecond {
			return false
		}
	}

	return true
}

// MarkReloadStarted marks files as having reload in progress
func (fw *FileWatcher) MarkReloadStarted(paths []string) {
	fw.mu.Lock()
	defer fw.mu.Unlock()

	for _, path := range paths {
		absPath, _ := filepath.Abs(path)
		fw.reloadInProgress[absPath] = true
	}
}

// MarkReloadCompleted marks files as having completed reload
func (fw *FileWatcher) MarkReloadCompleted(paths []string) {
	fw.mu.Lock()
	defer fw.mu.Unlock()

	now := time.Now()
	for _, path := range paths {
		absPath, _ := filepath.Abs(path)
		delete(fw.reloadInProgress, absPath)
		fw.lastReloadTime[absPath] = now
	}
}

// Start begins watching files and processing events
func (fw *FileWatcher) Start(ctx context.Context, reloadCallback func([]string) error) {
	if !fw.IsSupported() {
		return // Don't start on unsupported platforms
	}

	fw.reloadCallback = reloadCallback

	go fw.processEvents(ctx)
}

// processEvents handles file system events with debouncing
func (fw *FileWatcher) processEvents(ctx context.Context) {
	pendingReloads := make(map[string]bool)
	var mu sync.Mutex

	for {
		select {
		case <-ctx.Done():
			return

		case <-fw.stopChan:
			return

		case event, ok := <-fw.watcher.Events:
			if !ok {
				return
			}

			// Re-add the file to watcher if it was renamed (common with editor saves)
			if event.Op&fsnotify.Rename == fsnotify.Rename {
				absPath, _ := filepath.Abs(event.Name)
				// Check if this file should be watched
				fw.mu.RLock()
				_, shouldWatch := fw.watchedPaths[absPath]
				fw.mu.RUnlock()

				if shouldWatch {
					// Re-add the file to the watcher after a small delay
					go func() {
						time.Sleep(50 * time.Millisecond) // Wait for rename to complete
						fw.watcher.Add(absPath)
					}()
				}
			}

			// Handle write, create, and rename events (many editors use rename when saving)
			if event.Op&fsnotify.Write == fsnotify.Write ||
			   event.Op&fsnotify.Create == fsnotify.Create ||
			   event.Op&fsnotify.Rename == fsnotify.Rename {
				mu.Lock()
				absPath, _ := filepath.Abs(event.Name)

				// Check if we should reload this file
				if fw.ShouldReload(absPath) {
					pendingReloads[absPath] = true

					// Reset or start debounce timer
					if fw.debounceTimer != nil {
						fw.debounceTimer.Stop()
					}

					fw.debounceTimer = time.AfterFunc(fw.debounceDelay, func() {
						mu.Lock()
						paths := make([]string, 0, len(pendingReloads))
						for path := range pendingReloads {
							paths = append(paths, path)
						}
						pendingReloads = make(map[string]bool)
						mu.Unlock()

						if len(paths) > 0 {
							fw.triggerReload(paths)
						}
					})
				}
				mu.Unlock()
			}

		case err, ok := <-fw.watcher.Errors:
			if !ok {
				return
			}
			log.Printf("File watcher error: %v", err)
		}
	}
}

// triggerReload calls the reload callback with proper duplicate prevention
func (fw *FileWatcher) triggerReload(paths []string) {
	// Filter paths that should actually be reloaded
	var pathsToReload []string
	for _, path := range paths {
		if fw.ShouldReload(path) {
			pathsToReload = append(pathsToReload, path)
		}
	}

	if len(pathsToReload) == 0 {
		return
	}

	// Mark as reloading
	fw.MarkReloadStarted(pathsToReload)

	// Perform reload
	if fw.reloadCallback != nil {
		if err := fw.reloadCallback(pathsToReload); err != nil {
			log.Printf("Error reloading files: %v", err)
		}
	}

	// Mark as completed
	fw.MarkReloadCompleted(pathsToReload)
}

// Stop stops the file watcher
func (fw *FileWatcher) Stop() error {
	if !fw.IsSupported() {
		return nil
	}

	close(fw.stopChan)

	if fw.debounceTimer != nil {
		fw.debounceTimer.Stop()
	}

	if fw.watcher != nil {
		return fw.watcher.Close()
	}

	return nil
}