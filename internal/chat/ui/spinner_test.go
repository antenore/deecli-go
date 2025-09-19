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

package ui

import (
	"testing"
	"time"

	tea "github.com/charmbracelet/bubbletea"
)

func TestNewSpinner(t *testing.T) {
	tests := []struct {
		name  string
		style SpinnerStyle
	}{
		{"Dots spinner", SpinnerDots},
		{"Line spinner", SpinnerLine},
		{"Bounce spinner", SpinnerBounce},
		{"Circle spinner", SpinnerCircle},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			spinner := NewSpinner(tt.style)

			if spinner == nil {
				t.Fatal("NewSpinner returned nil")
			}

			if spinner.style != tt.style {
				t.Errorf("Expected style %v, got %v", tt.style, spinner.style)
			}

			if spinner.active {
				t.Error("Spinner should not be active initially")
			}

			if len(spinner.frames) == 0 {
				t.Error("Spinner should have frames")
			}

			if spinner.interval <= 0 {
				t.Error("Spinner should have positive interval")
			}
		})
	}
}

func TestDefaultSpinner(t *testing.T) {
	spinner := NewDefaultSpinner()

	if spinner == nil {
		t.Fatal("NewDefaultSpinner returned nil")
	}

	if spinner.style != SpinnerDots {
		t.Errorf("Expected SpinnerDots style, got %v", spinner.style)
	}
}

func TestSpinnerFrames(t *testing.T) {
	tests := []struct {
		name          string
		style         SpinnerStyle
		expectedCount int
		minFrames     int
	}{
		{"Dots frames", SpinnerDots, 10, 8},
		{"Line frames", SpinnerLine, 4, 4},
		{"Bounce frames", SpinnerBounce, 8, 6},
		{"Circle frames", SpinnerCircle, 4, 4},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			spinner := NewSpinner(tt.style)

			if len(spinner.frames) < tt.minFrames {
				t.Errorf("Expected at least %d frames, got %d", tt.minFrames, len(spinner.frames))
			}

			// Verify all frames are non-empty
			for i, frame := range spinner.frames {
				if frame == "" {
					t.Errorf("Frame %d is empty", i)
				}
			}
		})
	}
}

func TestSpinnerStartStop(t *testing.T) {
	spinner := NewSpinner(SpinnerDots)

	// Initially not active
	if spinner.IsActive() {
		t.Error("Spinner should not be active initially")
	}

	if spinner.Frame() != "" {
		t.Error("Inactive spinner should return empty frame")
	}

	// Start spinner
	cmd := spinner.Start()
	if cmd == nil {
		t.Error("Start should return a command")
	}

	if !spinner.IsActive() {
		t.Error("Spinner should be active after Start")
	}

	frame := spinner.Frame()
	if frame == "" {
		t.Error("Active spinner should return non-empty frame")
	}

	// Stop spinner
	spinner.Stop()
	if spinner.IsActive() {
		t.Error("Spinner should not be active after Stop")
	}
}

func TestSpinnerUpdate(t *testing.T) {
	spinner := NewSpinner(SpinnerDots)

	// Start spinner
	cmd := spinner.Start()
	if cmd == nil {
		t.Fatal("Start should return a command")
	}

	initialFrame := spinner.Frame()

	// Simulate tick message
	tickCmd := spinner.Update(spinnerTickMsg{})

	// Frame should advance
	newFrame := spinner.Frame()
	if newFrame == initialFrame && len(spinner.frames) > 1 {
		t.Error("Frame should advance on tick")
	}

	// Should return another tick command if active
	if tickCmd == nil {
		t.Error("Update should return tick command when active")
	}

	// Stop and verify no more commands
	spinner.Stop()
	tickCmd = spinner.Update(spinnerTickMsg{})
	if tickCmd != nil {
		t.Error("Update should not return command when inactive")
	}
}

func TestSpinnerFrameCycling(t *testing.T) {
	spinner := NewSpinner(SpinnerDots)
	spinner.Start()

	frameCount := len(spinner.frames)
	if frameCount == 0 {
		t.Fatal("Spinner should have frames")
	}

	// Record frames as we cycle through
	var seenFrames []string

	for i := 0; i < frameCount*2; i++ {
		frame := spinner.Frame()
		seenFrames = append(seenFrames, frame)
		spinner.Update(spinnerTickMsg{})
	}

	// Should see each frame at least once
	framesSeen := make(map[string]bool)
	for _, frame := range seenFrames {
		framesSeen[frame] = true
	}

	if len(framesSeen) != frameCount {
		t.Errorf("Expected to see %d unique frames, got %d", frameCount, len(framesSeen))
	}

	// Should cycle back to beginning
	firstFrame := seenFrames[0]
	frameAfterCycle := seenFrames[frameCount]
	if firstFrame != frameAfterCycle {
		t.Error("Frames should cycle back to beginning")
	}
}

func TestSpinnerSetStyle(t *testing.T) {
	spinner := NewSpinner(SpinnerDots)
	originalFrameCount := len(spinner.frames)

	// Change style
	spinner.SetStyle(SpinnerLine)

	if spinner.style != SpinnerLine {
		t.Error("SetStyle should change spinner style")
	}

	newFrameCount := len(spinner.frames)
	if newFrameCount == originalFrameCount && SpinnerDots != SpinnerLine {
		t.Error("SetStyle should change frame count for different styles")
	}

	if spinner.frameIndex != 0 {
		t.Error("SetStyle should reset frame index")
	}
}

func TestSpinnerSetInterval(t *testing.T) {
	spinner := NewSpinner(SpinnerDots)
	originalInterval := spinner.interval

	newInterval := time.Millisecond * 50
	spinner.SetInterval(newInterval)

	if spinner.interval != newInterval {
		t.Errorf("Expected interval %v, got %v", newInterval, spinner.interval)
	}

	if spinner.interval == originalInterval {
		t.Error("SetInterval should change the interval")
	}
}

func TestFastAndSlowSpinners(t *testing.T) {
	fast := NewFastSpinner()
	slow := NewSlowSpinner()
	normal := NewDefaultSpinner()

	if fast.interval >= normal.interval {
		t.Error("Fast spinner should have shorter interval than normal")
	}

	if slow.interval <= normal.interval {
		t.Error("Slow spinner should have longer interval than normal")
	}

	if fast.interval >= slow.interval {
		t.Error("Fast spinner should have shorter interval than slow")
	}
}

func TestSpinnerTickMessage(t *testing.T) {
	// Test that spinnerTickMsg is properly typed
	var msg tea.Msg = spinnerTickMsg{}

	switch msg.(type) {
	case spinnerTickMsg:
		// This is expected
	default:
		t.Error("spinnerTickMsg should be a valid tea.Msg")
	}
}

func TestSpinnerWithZeroFrames(t *testing.T) {
	spinner := &Spinner{
		frames:    []string{}, // Empty frames
		active:    true,
		frameIndex: 0,
	}

	frame := spinner.Frame()
	if frame != "" {
		t.Error("Spinner with no frames should return empty string")
	}
}

func TestSpinnerFrameIndexBounds(t *testing.T) {
	spinner := NewSpinner(SpinnerDots)
	spinner.Start()

	// Manually set frame index beyond bounds
	spinner.frameIndex = len(spinner.frames) + 5

	// Update should handle out of bounds gracefully
	spinner.Update(spinnerTickMsg{})

	if spinner.frameIndex >= len(spinner.frames) {
		t.Error("Frame index should be within bounds after update")
	}
}