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

package utils

import (
	"fmt"
	"os"
	"os/exec"
	"strconv"
	"strings"
	"syscall"
	"unsafe"
)

// winsize struct for IOCTL terminal size detection
type winsize struct {
	Row    uint16
	Col    uint16
	Xpixel uint16
	Ypixel uint16
}

// GetTerminalSize returns terminal dimensions using multiple detection methods
func GetTerminalSize() (int, int) {
	// Use this approach for reliable terminal size detection
	if width, height, err := getTerminalSizeIoctl(); err == nil {
		return width, height
	}
	// Fallback to tput
	if width, height := getTerminalSizeTput(); width > 0 && height > 0 {
		return width, height
	}
	// Final fallback to environment
	width, height := getTerminalSizeEnv()
	if width > 0 && height > 0 {
		return width, height
	}
	return 80, 24 // Safe defaults
}

// getTerminalSizeIoctl tries IOCTL system call (most accurate)
func getTerminalSizeIoctl() (int, int, error) {
	ws := &winsize{}
	retCode, _, _ := syscall.Syscall(syscall.SYS_IOCTL,
		uintptr(syscall.Stdin),
		uintptr(syscall.TIOCGWINSZ),
		uintptr(unsafe.Pointer(ws)))
	if int(retCode) != -1 && ws.Col > 0 && ws.Row > 0 {
		return int(ws.Col), int(ws.Row), nil
	}
	
	return 0, 0, fmt.Errorf("ioctl failed")
}

// getTerminalSizeTput tries tput command
func getTerminalSizeTput() (int, int) {
	width, height := 0, 0
	
	if cmd := exec.Command("tput", "cols"); cmd != nil {
		if output, err := cmd.Output(); err == nil {
			if w, err := strconv.Atoi(strings.TrimSpace(string(output))); err == nil && w > 0 {
				width = w
			}
		}
	}
	if cmd := exec.Command("tput", "lines"); cmd != nil {
		if output, err := cmd.Output(); err == nil {
			if h, err := strconv.Atoi(strings.TrimSpace(string(output))); err == nil && h > 0 {
				height = h
			}
		}
	}
	
	return width, height
}

// getTerminalSizeEnv tries environment variables
func getTerminalSizeEnv() (int, int) {
	width, height := 0, 0
	
	if cols := os.Getenv("COLUMNS"); cols != "" {
		if w, err := strconv.Atoi(cols); err == nil && w > 0 {
			width = w
		}
	}
	if lines := os.Getenv("LINES"); lines != "" {
		if h, err := strconv.Atoi(lines); err == nil && h > 0 {
			height = h
		}
	}

	return width, height
}

// GetTextWidth calculates available text width for messages
func GetTextWidth(viewportWidth int, filesVisible bool) int {
	availableWidth := viewportWidth - 10 // Base padding
	if filesVisible {
		// Account for sidebar width and borders
		availableWidth = viewportWidth - 35
	}
	if availableWidth < 20 {
		availableWidth = 20 // Minimum readable width
	}
	return availableWidth
}