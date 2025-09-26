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

package state

import (
	"testing"

	"github.com/antenore/deecli/internal/chat/input"
	"github.com/antenore/deecli/internal/chat/ui"
	"github.com/antenore/deecli/internal/config"
	"github.com/antenore/deecli/internal/history"
)

func createTestStateManager(t *testing.T) *Manager {
	// Create minimal dependencies
	layoutManager := ui.NewLayout(nil)
	
	// Create a minimal input manager for testing
	historyMgr, err := history.NewManager()
	if err != nil {
		// Use nil if history manager fails in tests
		historyMgr = nil
	}
	
	inputManager := input.NewManager(
		[]string{},
		historyMgr,
		nil, // completion engine - not needed for state tests
		func(role, content string) {}, // addMessage callback
		func() {}, // refreshViewport callback
	)
	
	return NewManager(Dependencies{
		LayoutManager: layoutManager,
		InputManager:  inputManager,
	})
}

func TestManager_InitializeViewports(t *testing.T) {
	manager := createTestStateManager(t)
	
	if manager.IsReady() {
		t.Errorf("Manager should not be ready before initialization")
	}
	
	manager.InitializeViewports(80, 24)
	
	if !manager.IsReady() {
		t.Errorf("Manager should be ready after initialization")
	}
	
	width, height := manager.GetDimensions()
	if width != 80 || height != 24 {
		t.Errorf("Dimensions = (%d, %d), want (80, 24)", width, height)
	}
	
	viewport := manager.GetViewport()
	if viewport == nil {
		t.Errorf("GetViewport() returned nil after initialization")
	}
	
	sidebarViewport := manager.GetSidebarViewport()
	if sidebarViewport == nil {
		t.Errorf("GetSidebarViewport() returned nil after initialization")
	}
}

func TestManager_CreateTextarea(t *testing.T) {
	manager := createTestStateManager(t)
	
	configManager := config.NewManager()
	textarea := manager.CreateTextarea(80, configManager)
	
	if textarea.Value() != "" {
		t.Errorf("Initial textarea value should be empty")
	}
	
	if !textarea.Focused() {
		t.Errorf("Textarea should be focused after creation")
	}
	
	retrievedTextarea := manager.GetTextarea()
	if retrievedTextarea == nil {
		t.Errorf("GetTextarea() returned nil after CreateTextarea")
	}
}

func TestManager_FocusMode(t *testing.T) {
	manager := createTestStateManager(t)
	
	// Initial focus should be input
	if manager.GetFocusMode() != "input" {
		t.Errorf("Initial focus mode = %s, want input", manager.GetFocusMode())
	}
	
	// Test setting focus mode
	manager.SetFocusMode("viewport")
	if manager.GetFocusMode() != "viewport" {
		t.Errorf("Focus mode after set = %s, want viewport", manager.GetFocusMode())
	}
	
	manager.SetFocusMode("sidebar")
	if manager.GetFocusMode() != "sidebar" {
		t.Errorf("Focus mode after set = %s, want sidebar", manager.GetFocusMode())
	}
	
	// Test cycling focus
	manager.SetFocusMode("input")
	nextMode := manager.CycleFocus()
	if nextMode != "viewport" {
		t.Errorf("CycleFocus from input = %s, want viewport", nextMode)
	}
	
	// Cycle from viewport (without files widget visible)
	nextMode = manager.CycleFocus()
	if nextMode != "input" {
		t.Errorf("CycleFocus from viewport without files = %s, want input", nextMode)
	}
	
	// Enable files widget and test cycling
	manager.SetFilesWidgetVisible(true)
	manager.SetFocusMode("viewport")
	nextMode = manager.CycleFocus()
	if nextMode != "sidebar" {
		t.Errorf("CycleFocus from viewport with files = %s, want sidebar", nextMode)
	}
	
	nextMode = manager.CycleFocus()
	if nextMode != "input" {
		t.Errorf("CycleFocus from sidebar = %s, want input", nextMode)
	}
}

func TestManager_HelpVisible(t *testing.T) {
	manager := createTestStateManager(t)
	
	// Initial state should be not visible
	if manager.IsHelpVisible() {
		t.Errorf("Initial help visible = true, want false")
	}
	
	manager.SetHelpVisible(true)
	if !manager.IsHelpVisible() {
		t.Errorf("Help visible after set true = false, want true")
	}
	
	manager.SetHelpVisible(false)
	if manager.IsHelpVisible() {
		t.Errorf("Help visible after set false = true, want false")
	}
}

func TestManager_FilesWidget(t *testing.T) {
	manager := createTestStateManager(t)
	
	// Initial state should be not visible
	if manager.IsFilesWidgetVisible() {
		t.Errorf("Initial files widget visible = true, want false")
	}
	
	manager.SetFilesWidgetVisible(true)
	if !manager.IsFilesWidgetVisible() {
		t.Errorf("Files widget visible after set true = false, want true")
	}
	
	// Test toggle
	result := manager.ToggleFilesWidget()
	if result != false {
		t.Errorf("ToggleFilesWidget result = true, want false")
	}
	
	if manager.IsFilesWidgetVisible() {
		t.Errorf("Files widget should be hidden after toggle")
	}
	
	result = manager.ToggleFilesWidget()
	if result != true {
		t.Errorf("ToggleFilesWidget result = false, want true")
	}
}

func TestManager_LoadingState(t *testing.T) {
	manager := createTestStateManager(t)
	
	// Initial state should be not loading
	if manager.IsLoading() {
		t.Errorf("Initial loading state = true, want false")
	}
	
	if manager.GetLoadingMessage() != "" {
		t.Errorf("Initial loading message = %q, want empty", manager.GetLoadingMessage())
	}
	
	manager.SetLoading(true, "Test loading...")
	if !manager.IsLoading() {
		t.Errorf("Loading state after set = false, want true")
	}
	
	if manager.GetLoadingMessage() != "Test loading..." {
		t.Errorf("Loading message = %q, want %q", manager.GetLoadingMessage(), "Test loading...")
	}
	
	manager.SetLoading(false, "")
	if manager.IsLoading() {
		t.Errorf("Loading state after clear = true, want false")
	}
}

func TestManager_APICancelFunction(t *testing.T) {
	manager := createTestStateManager(t)
	
	// Initial state should be nil
	if manager.GetAPICancel() != nil {
		t.Errorf("Initial API cancel function should be nil")
	}
	
	// Create a mock cancel function
	cancelled := false
	cancelFunc := func() {
		cancelled = true
	}
	
	manager.SetAPICancel(cancelFunc)
	retrievedCancel := manager.GetAPICancel()
	if retrievedCancel == nil {
		t.Errorf("GetAPICancel() returned nil after set")
	}
	
	// Test calling the cancel function
	retrievedCancel()
	if !cancelled {
		t.Errorf("Cancel function was not called properly")
	}
	
	manager.ClearAPICancel()
	if manager.GetAPICancel() != nil {
		t.Errorf("API cancel function should be nil after clear")
	}
}

func TestManager_ViewportContent(t *testing.T) {
	manager := createTestStateManager(t)
	manager.InitializeViewports(80, 24)
	
	testContent := "Test viewport content"
	manager.SetViewportContent(testContent)
	
	// Note: The actual content returned by GetViewportContent() is the rendered view,
	// not the raw content, so we just test that it's not empty
	content := manager.GetViewportContent()
	if content == "" {
		t.Errorf("GetViewportContent() returned empty after setting content")
	}
}

func TestManager_SidebarContent(t *testing.T) {
	manager := createTestStateManager(t)
	manager.InitializeViewports(80, 24)
	
	testContent := "Test sidebar content"
	manager.SetSidebarContent(testContent)
	
	// Note: Similar to viewport, we test that content is not empty
	content := manager.GetSidebarContent()
	if content == "" {
		t.Errorf("GetSidebarContent() returned empty after setting content")
	}
}

func TestManager_HandleResize(t *testing.T) {
	manager := createTestStateManager(t)
	manager.InitializeViewports(80, 24)
	
	manager.HandleResize(100, 30)
	
	width, height := manager.GetDimensions()
	if width != 100 || height != 30 {
		t.Errorf("Dimensions after resize = (%d, %d), want (100, 30)", width, height)
	}
	
	viewport := manager.GetViewport()
	if viewport.Width != 100 {
		t.Errorf("Viewport width after resize = %d, want 100", viewport.Width)
	}
}

func TestManager_ScrollHandling(t *testing.T) {
	manager := createTestStateManager(t)
	manager.InitializeViewports(80, 24)
	
	// Test viewport scrolling when not in viewport mode
	manager.SetFocusMode("input")
	handled := manager.HandleViewportScroll("down")
	if handled {
		t.Errorf("HandleViewportScroll should not handle keys when not in viewport mode")
	}
	
	// Test viewport scrolling when in viewport mode
	manager.SetFocusMode("viewport")
	handled = manager.HandleViewportScroll("down")
	if !handled {
		t.Errorf("HandleViewportScroll should handle keys when in viewport mode")
	}
	
	// Test sidebar scrolling when not in sidebar mode
	manager.SetFocusMode("input")
	handled = manager.HandleSidebarScroll("down")
	if handled {
		t.Errorf("HandleSidebarScroll should not handle keys when not in sidebar mode")
	}
	
	// Test sidebar scrolling when in sidebar mode
	manager.SetFocusMode("sidebar")
	handled = manager.HandleSidebarScroll("down")
	if !handled {
		t.Errorf("HandleSidebarScroll should handle keys when in sidebar mode")
	}
}