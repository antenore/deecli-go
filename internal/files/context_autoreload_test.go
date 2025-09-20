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
	"io/ioutil"
	"os"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestFileContext_AutoReload_Integration(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	// Create a FileContext with watcher
	fc := NewFileContext()
	watcher, err := NewWatcher(50 * time.Millisecond) // Short delay for testing
	require.NoError(t, err)

	if !watcher.IsSupported() {
		t.Skip("File watching not supported on this platform")
	}

	fc.SetWatcher(watcher)
	defer watcher.Stop()

	// Create a temporary file
	tmpFile, err := ioutil.TempFile("", "test_autoreload_*.txt")
	require.NoError(t, err)
	defer os.Remove(tmpFile.Name())

	// Write initial content
	initialContent := "initial content line 1\ninitial content line 2"
	_, err = tmpFile.WriteString(initialContent)
	require.NoError(t, err)
	tmpFile.Close()

	// Load the file into context
	err = fc.LoadFile(tmpFile.Name())
	require.NoError(t, err)
	assert.Len(t, fc.Files, 1)
	assert.Equal(t, initialContent, fc.Files[0].Content)

	// Set up auto-reload with notification tracking
	reloadNotifications := make(chan []ReloadResult, 10)
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	err = fc.EnableAutoReload(ctx, func(results []ReloadResult) {
		select {
		case reloadNotifications <- results:
		default:
		}
	})
	require.NoError(t, err)
	assert.True(t, fc.IsAutoReloadEnabled())

	// Wait for watcher to set up
	time.Sleep(100 * time.Millisecond)

	// Modify the file
	modifiedContent := "modified content line 1\nmodified content line 2\nnew line 3"
	err = ioutil.WriteFile(tmpFile.Name(), []byte(modifiedContent), 0644)
	require.NoError(t, err)

	// Wait for auto-reload notification
	var reloadResults []ReloadResult
	select {
	case reloadResults = <-reloadNotifications:
		assert.Len(t, reloadResults, 1)
		assert.Equal(t, "changed", reloadResults[0].Status)
	case <-ctx.Done():
		t.Fatal("Timeout waiting for auto-reload notification")
	}

	// Verify file content was updated in context
	assert.Len(t, fc.Files, 1)
	assert.Equal(t, modifiedContent, fc.Files[0].Content)
	assert.Equal(t, int64(len(modifiedContent)), fc.Files[0].Size)
}

func TestFileContext_AutoReload_DuplicatePrevention(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	// Create a FileContext with watcher
	fc := NewFileContext()
	watcher, err := NewWatcher(50 * time.Millisecond)
	require.NoError(t, err)

	if !watcher.IsSupported() {
		t.Skip("File watching not supported on this platform")
	}

	fc.SetWatcher(watcher)
	defer watcher.Stop()

	// Create a temporary file
	tmpFile, err := ioutil.TempFile("", "test_duplicate_*.txt")
	require.NoError(t, err)
	defer os.Remove(tmpFile.Name())

	// Write initial content
	initialContent := "initial content"
	_, err = tmpFile.WriteString(initialContent)
	require.NoError(t, err)
	tmpFile.Close()

	// Load the file into context
	err = fc.LoadFile(tmpFile.Name())
	require.NoError(t, err)

	// Set up auto-reload with proper synchronization
	var reloadCount int32
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	err = fc.EnableAutoReload(ctx, func(results []ReloadResult) {
		atomic.AddInt32(&reloadCount, 1)
	})
	require.NoError(t, err)

	// Wait for watcher setup
	time.Sleep(100 * time.Millisecond)

	// Perform manual reload (this should prevent auto-reload for 500ms)
	modifiedContent := "manually reloaded content"
	err = ioutil.WriteFile(tmpFile.Name(), []byte(modifiedContent), 0644)
	require.NoError(t, err)

	// Immediately perform manual reload
	results, err := fc.ReloadFiles(nil)
	require.NoError(t, err)
	assert.Len(t, results, 1)
	assert.Equal(t, "changed", results[0].Status)

	// Wait 300ms (less than the 500ms cooldown) and check no auto-reload occurred
	time.Sleep(300 * time.Millisecond)
	assert.Equal(t, int32(0), atomic.LoadInt32(&reloadCount), "Auto-reload should not occur within 500ms of manual reload")

	// Wait another 300ms (total 600ms, past the cooldown) and modify again
	time.Sleep(300 * time.Millisecond)

	newContent := "content after cooldown"
	err = ioutil.WriteFile(tmpFile.Name(), []byte(newContent), 0644)
	require.NoError(t, err)

	// Wait for auto-reload
	time.Sleep(200 * time.Millisecond)
	assert.Equal(t, int32(1), atomic.LoadInt32(&reloadCount), "Auto-reload should occur after cooldown period")
}

func TestFileContext_AutoReload_UnsupportedPlatform(t *testing.T) {
	// Create a FileContext without watcher (simulating unsupported platform)
	fc := NewFileContext()

	assert.False(t, fc.IsAutoReloadSupported())
	assert.False(t, fc.IsAutoReloadEnabled())

	// Attempting to enable auto-reload should fail gracefully
	ctx := context.Background()
	err := fc.EnableAutoReload(ctx, func(results []ReloadResult) {})
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "not supported")
}

func TestFileContext_AutoReload_DisableReload(t *testing.T) {
	// Create a FileContext with watcher
	fc := NewFileContext()
	watcher, err := NewWatcher(100 * time.Millisecond)
	require.NoError(t, err)

	if !watcher.IsSupported() {
		t.Skip("File watching not supported on this platform")
	}

	fc.SetWatcher(watcher)
	defer watcher.Stop()

	// Enable auto-reload
	ctx := context.Background()
	err = fc.EnableAutoReload(ctx, func(results []ReloadResult) {})
	require.NoError(t, err)
	assert.True(t, fc.IsAutoReloadEnabled())

	// Disable auto-reload
	fc.DisableAutoReload()
	assert.False(t, fc.IsAutoReloadEnabled())
}

func TestFileContext_AutoReload_FileOperations(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	// Create a FileContext with watcher
	fc := NewFileContext()
	watcher, err := NewWatcher(50 * time.Millisecond)
	require.NoError(t, err)

	if !watcher.IsSupported() {
		t.Skip("File watching not supported on this platform")
	}

	fc.SetWatcher(watcher)
	defer watcher.Stop()

	// Create temporary files
	tmpFile1, err := ioutil.TempFile("", "test_ops1_*.txt")
	require.NoError(t, err)
	defer os.Remove(tmpFile1.Name())
	tmpFile1.WriteString("content 1")
	tmpFile1.Close()

	tmpFile2, err := ioutil.TempFile("", "test_ops2_*.txt")
	require.NoError(t, err)
	defer os.Remove(tmpFile2.Name())
	tmpFile2.WriteString("content 2")
	tmpFile2.Close()

	// Enable auto-reload with proper synchronization
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	var reloadCount int32
	err = fc.EnableAutoReload(ctx, func(results []ReloadResult) {
		atomic.AddInt32(&reloadCount, 1)
	})
	require.NoError(t, err)

	// Load files
	err = fc.LoadFile(tmpFile1.Name())
	require.NoError(t, err)
	err = fc.LoadFile(tmpFile2.Name())
	require.NoError(t, err)
	assert.Len(t, fc.Files, 2)

	// Wait for watcher setup
	time.Sleep(100 * time.Millisecond)

	// Test removing a file stops watching it
	removed := fc.RemoveFile(tmpFile1.Name())
	assert.True(t, removed)
	assert.Len(t, fc.Files, 1)

	// Modify the removed file - should not trigger reload
	err = ioutil.WriteFile(tmpFile1.Name(), []byte("modified removed file"), 0644)
	require.NoError(t, err)
	time.Sleep(150 * time.Millisecond)
	assert.Equal(t, int32(0), atomic.LoadInt32(&reloadCount), "Removed file should not trigger auto-reload")

	// Modify the remaining file - should trigger reload
	err = ioutil.WriteFile(tmpFile2.Name(), []byte("modified remaining file"), 0644)
	require.NoError(t, err)
	time.Sleep(150 * time.Millisecond)
	assert.Equal(t, int32(1), atomic.LoadInt32(&reloadCount), "Remaining file should trigger auto-reload")

	// Test Clear() stops watching all files
	fc.Clear()
	assert.Len(t, fc.Files, 0)

	// Re-add a file and modify - should not trigger reload since Clear() unwatched everything
	err = fc.LoadFile(tmpFile2.Name())
	require.NoError(t, err)

	// We need to re-enable watching since Clear() cleared everything
	time.Sleep(100 * time.Millisecond) // Wait for watcher to process

	err = ioutil.WriteFile(tmpFile2.Name(), []byte("content after clear"), 0644)
	require.NoError(t, err)
	time.Sleep(150 * time.Millisecond)

	// The count should be 2 now because the file was automatically watched when loaded
	assert.Equal(t, int32(2), atomic.LoadInt32(&reloadCount), "File should be auto-watched when loaded after clear")
}

func TestFileContext_AutoReload_MultipleFiles(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	// Create a FileContext with watcher
	fc := NewFileContext()
	watcher, err := NewWatcher(50 * time.Millisecond)
	require.NoError(t, err)

	if !watcher.IsSupported() {
		t.Skip("File watching not supported on this platform")
	}

	fc.SetWatcher(watcher)
	defer watcher.Stop()

	// Create multiple temporary files
	var tmpFiles []*os.File
	var tmpFilePaths []string
	for i := 0; i < 3; i++ {
		tmpFile, err := ioutil.TempFile("", "test_multi_*.txt")
		require.NoError(t, err)
		defer os.Remove(tmpFile.Name())

		tmpFile.WriteString("initial content")
		tmpFile.Close()

		tmpFiles = append(tmpFiles, tmpFile)
		tmpFilePaths = append(tmpFilePaths, tmpFile.Name())

		// Load file into context
		err = fc.LoadFile(tmpFile.Name())
		require.NoError(t, err)
	}

	assert.Len(t, fc.Files, 3)

	// Enable auto-reload with proper synchronization
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	var allReloadResults []ReloadResult
	var resultsMutex sync.Mutex
	err = fc.EnableAutoReload(ctx, func(results []ReloadResult) {
		resultsMutex.Lock()
		allReloadResults = append(allReloadResults, results...)
		resultsMutex.Unlock()
	})
	require.NoError(t, err)

	// Wait for watcher setup
	time.Sleep(100 * time.Millisecond)

	// Modify all files simultaneously
	for i, tmpPath := range tmpFilePaths {
		content := fmt.Sprintf("modified content %d", i+1)
		err = ioutil.WriteFile(tmpPath, []byte(content), 0644)
		require.NoError(t, err)
	}

	// Wait for auto-reload to process all files
	time.Sleep(200 * time.Millisecond)

	// Verify all files were reloaded (with synchronization)
	resultsMutex.Lock()
	resultCount := len(allReloadResults)
	resultsMutex.Unlock()
	assert.GreaterOrEqual(t, resultCount, 3, "All files should be detected as changed")

	// Verify file contents were updated
	for i, file := range fc.Files {
		expectedContent := fmt.Sprintf("modified content %d", i+1)
		assert.Equal(t, expectedContent, file.Content)
	}
}