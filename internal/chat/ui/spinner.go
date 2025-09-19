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
	"time"

	tea "github.com/charmbracelet/bubbletea"
)

// SpinnerStyle defines different spinner animation styles
type SpinnerStyle int

const (
	SpinnerDots SpinnerStyle = iota
	SpinnerLine
	SpinnerBounce
	SpinnerCircle
)

// Spinner provides animated loading indicators for the TUI
type Spinner struct {
	frames     []string
	frameIndex int
	style      SpinnerStyle
	active     bool
	interval   time.Duration
}

// NewSpinner creates a new spinner with the specified style
func NewSpinner(style SpinnerStyle) *Spinner {
	s := &Spinner{
		style:    style,
		active:   false,
		interval: time.Millisecond * 100, // 10 FPS for smooth animation
	}
	s.setFrames()
	return s
}

// setFrames configures the animation frames based on the spinner style
func (s *Spinner) setFrames() {
	switch s.style {
	case SpinnerDots:
		// Braille spinner - smooth and professional
		s.frames = []string{"⠋", "⠙", "⠹", "⠸", "⠼", "⠴", "⠦", "⠧", "⠇", "⠏"}
	case SpinnerLine:
		// Line spinner - simple and clean
		s.frames = []string{"|", "/", "-", "\\"}
	case SpinnerBounce:
		// Bouncing dots - playful but professional
		s.frames = []string{"⠁", "⠂", "⠄", "⡀", "⢀", "⠠", "⠐", "⠈"}
	case SpinnerCircle:
		// Circular progress - clear visual progress
		s.frames = []string{"◐", "◓", "◑", "◒"}
	default:
		// Default to dots
		s.frames = []string{"⠋", "⠙", "⠹", "⠸", "⠼", "⠴", "⠦", "⠧", "⠇", "⠏"}
	}
}

// Start begins the spinner animation
func (s *Spinner) Start() tea.Cmd {
	s.active = true
	s.frameIndex = 0
	return s.tick()
}

// Stop ends the spinner animation
func (s *Spinner) Stop() {
	s.active = false
}

// IsActive returns whether the spinner is currently animating
func (s *Spinner) IsActive() bool {
	return s.active
}

// Frame returns the current animation frame
func (s *Spinner) Frame() string {
	if !s.active || len(s.frames) == 0 {
		return ""
	}
	return s.frames[s.frameIndex]
}

// Update handles Bubbletea update messages for the spinner
func (s *Spinner) Update(msg tea.Msg) tea.Cmd {
	switch msg.(type) {
	case spinnerTickMsg:
		if s.active {
			s.frameIndex = (s.frameIndex + 1) % len(s.frames)
			return s.tick()
		}
	}
	return nil
}

// tick creates a command to advance the spinner animation
func (s *Spinner) tick() tea.Cmd {
	if !s.active {
		return nil
	}
	return tea.Tick(s.interval, func(time.Time) tea.Msg {
		return spinnerTickMsg{}
	})
}

// SetStyle changes the spinner style
func (s *Spinner) SetStyle(style SpinnerStyle) {
	s.style = style
	s.setFrames()
	s.frameIndex = 0
}

// SetInterval changes the animation speed
func (s *Spinner) SetInterval(interval time.Duration) {
	s.interval = interval
}

// spinnerTickMsg is used internally for animation timing
type spinnerTickMsg struct{}

// Helper functions for creating common spinner configurations

// NewDefaultSpinner creates a spinner with the default dots style
func NewDefaultSpinner() *Spinner {
	return NewSpinner(SpinnerDots)
}

// NewFastSpinner creates a faster-animating spinner
func NewFastSpinner() *Spinner {
	s := NewSpinner(SpinnerDots)
	s.SetInterval(time.Millisecond * 80) // Slightly faster
	return s
}

// NewSlowSpinner creates a slower-animating spinner
func NewSlowSpinner() *Spinner {
	s := NewSpinner(SpinnerDots)
	s.SetInterval(time.Millisecond * 150) // Slower for less distraction
	return s
}