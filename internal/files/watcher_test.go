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
	"io/ioutil"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewWatcher(t *testing.T) {
	tests := []struct {
		name           string
		debounceDelay  time.Duration
		expectedDelay  time.Duration
	}{
		{
			name:          "Default debounce delay",
			debounceDelay: 0,
			expectedDelay: 100 * time.Millisecond,
		},
		{
			name:          "Custom debounce delay",
			debounceDelay: 200 * time.Millisecond,
			expectedDelay: 200 * time.Millisecond,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			watcher, err := NewWatcher(tt.debounceDelay)
			require.NoError(t, err)
			assert.NotNil(t, watcher)
			assert.Equal(t, tt.expectedDelay, watcher.debounceDelay)
			assert.NotNil(t, watcher.watchedPaths)
			assert.NotNil(t, watcher.reloadInProgress)
			assert.NotNil(t, watcher.lastReloadTime)

			// Clean up
			if watcher.IsSupported() {
				watcher.Stop()
			}
		})
	}
}

func TestFileWatcher_IsSupported(t *testing.T) {
	watcher, err := NewWatcher(100 * time.Millisecond)
	require.NoError(t, err)

	// On most platforms, file watching should be supported
	// If not supported, it should gracefully degrade
	supported := watcher.IsSupported()
	assert.IsType(t, true, supported) // Should be boolean

	// Clean up
	if supported {
		watcher.Stop()
	}
}

func TestFileWatcher_WatchUnwatch(t *testing.T) {
	watcher, err := NewWatcher(100 * time.Millisecond)
	require.NoError(t, err)

	if !watcher.IsSupported() {
		t.Skip("File watching not supported on this platform")
	}
	defer watcher.Stop()

	// Create a temporary file
	tmpFile, err := ioutil.TempFile("", "test_watcher_*.txt")
	require.NoError(t, err)
	defer os.Remove(tmpFile.Name())
	tmpFile.Close()

	// Test watching a file
	err = watcher.Watch(tmpFile.Name())
	assert.NoError(t, err)

	// Verify file is in watched paths
	absPath, _ := filepath.Abs(tmpFile.Name())
	watcher.mu.RLock()
	_, exists := watcher.watchedPaths[absPath]
	watcher.mu.RUnlock()
	assert.True(t, exists)

	// Test unwatching the file
	err = watcher.Unwatch(tmpFile.Name())
	assert.NoError(t, err)

	// Verify file is no longer in watched paths
	watcher.mu.RLock()
	_, exists = watcher.watchedPaths[absPath]
	watcher.mu.RUnlock()
	assert.False(t, exists)
}

func TestFileWatcher_ShouldReload(t *testing.T) {
	watcher, err := NewWatcher(100 * time.Millisecond)
	require.NoError(t, err)
	defer func() {
		if watcher.IsSupported() {
			watcher.Stop()
		}
	}()

	testPath := "/tmp/test_file.txt"

	// Initially should reload
	assert.True(t, watcher.ShouldReload(testPath))

	// Mark as in progress
	watcher.MarkReloadStarted([]string{testPath})
	assert.False(t, watcher.ShouldReload(testPath))

	// Complete the reload
	watcher.MarkReloadCompleted([]string{testPath})
	// Should still be false due to 500ms cooldown
	assert.False(t, watcher.ShouldReload(testPath))

	// Wait for cooldown and test again
	time.Sleep(600 * time.Millisecond)
	assert.True(t, watcher.ShouldReload(testPath))
}

func TestFileWatcher_MarkReloadStartedCompleted(t *testing.T) {
	watcher, err := NewWatcher(100 * time.Millisecond)
	require.NoError(t, err)
	defer func() {
		if watcher.IsSupported() {
			watcher.Stop()
		}
	}()

	testPaths := []string{"/tmp/test1.txt", "/tmp/test2.txt"}

	// Mark as started
	watcher.MarkReloadStarted(testPaths)

	watcher.mu.RLock()
	for _, path := range testPaths {
		absPath, _ := filepath.Abs(path)
		assert.True(t, watcher.reloadInProgress[absPath])
	}
	watcher.mu.RUnlock()

	// Mark as completed
	beforeTime := time.Now()
	watcher.MarkReloadCompleted(testPaths)

	watcher.mu.RLock()
	for _, path := range testPaths {
		absPath, _ := filepath.Abs(path)
		assert.False(t, watcher.reloadInProgress[absPath])
		reloadTime, exists := watcher.lastReloadTime[absPath]
		assert.True(t, exists)
		assert.True(t, reloadTime.After(beforeTime) || reloadTime.Equal(beforeTime))
	}
	watcher.mu.RUnlock()
}

func TestFileWatcher_UnwatchAll(t *testing.T) {
	watcher, err := NewWatcher(100 * time.Millisecond)
	require.NoError(t, err)

	if !watcher.IsSupported() {
		t.Skip("File watching not supported on this platform")
	}
	defer watcher.Stop()

	// Create temporary files
	tmpFile1, err := ioutil.TempFile("", "test_watcher1_*.txt")
	require.NoError(t, err)
	defer os.Remove(tmpFile1.Name())
	tmpFile1.Close()

	tmpFile2, err := ioutil.TempFile("", "test_watcher2_*.txt")
	require.NoError(t, err)
	defer os.Remove(tmpFile2.Name())
	tmpFile2.Close()

	// Watch both files
	err = watcher.Watch(tmpFile1.Name())
	assert.NoError(t, err)
	err = watcher.Watch(tmpFile2.Name())
	assert.NoError(t, err)

	// Verify files are watched
	watcher.mu.RLock()
	initialCount := len(watcher.watchedPaths)
	watcher.mu.RUnlock()
	assert.Equal(t, 2, initialCount)

	// Unwatch all
	err = watcher.UnwatchAll()
	assert.NoError(t, err)

	// Verify all files are unwatched
	watcher.mu.RLock()
	assert.Equal(t, 0, len(watcher.watchedPaths))
	assert.Equal(t, 0, len(watcher.reloadInProgress))
	assert.Equal(t, 0, len(watcher.lastReloadTime))
	watcher.mu.RUnlock()
}

func TestFileWatcher_UnsupportedPlatform(t *testing.T) {
	// Create a watcher that simulates unsupported platform
	watcher := &FileWatcher{
		watchedPaths:     make(map[string]time.Time),
		reloadInProgress: make(map[string]bool),
		lastReloadTime:   make(map[string]time.Time),
		debounceDelay:    100 * time.Millisecond,
		supported:        false, // Simulate unsupported
		watcher:          nil,
	}

	// Test that operations don't fail on unsupported platforms
	assert.False(t, watcher.IsSupported())

	err := watcher.Watch("/tmp/test.txt")
	assert.NoError(t, err) // Should not error, just silently ignore

	err = watcher.Unwatch("/tmp/test.txt")
	assert.NoError(t, err)

	err = watcher.UnwatchAll()
	assert.NoError(t, err)

	err = watcher.Stop()
	assert.NoError(t, err)

	// Start should not panic
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	watcher.Start(ctx, func(paths []string) error {
		return nil
	})
}

func TestFileWatcher_Integration(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	watcher, err := NewWatcher(50 * time.Millisecond) // Shorter delay for testing
	require.NoError(t, err)

	if !watcher.IsSupported() {
		t.Skip("File watching not supported on this platform")
	}
	defer watcher.Stop()

	// Create a temporary file
	tmpFile, err := ioutil.TempFile("", "test_integration_*.txt")
	require.NoError(t, err)
	defer os.Remove(tmpFile.Name())

	// Write initial content
	_, err = tmpFile.WriteString("initial content")
	require.NoError(t, err)
	tmpFile.Close()

	// Set up callback to capture reload events
	reloadCalled := make(chan []string, 1)
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	// Start watcher
	watcher.Start(ctx, func(paths []string) error {
		select {
		case reloadCalled <- paths:
		default:
		}
		return nil
	})

	// Watch the file
	err = watcher.Watch(tmpFile.Name())
	require.NoError(t, err)

	// Give the watcher time to set up
	time.Sleep(100 * time.Millisecond)

	// Modify the file
	err = ioutil.WriteFile(tmpFile.Name(), []byte("modified content"), 0644)
	require.NoError(t, err)

	// Wait for reload callback
	select {
	case paths := <-reloadCalled:
		absPath, _ := filepath.Abs(tmpFile.Name())
		assert.Contains(t, paths, absPath)
	case <-ctx.Done():
		t.Fatal("Timeout waiting for file change detection")
	}
}