package keydetect

import (
	"fmt"

	"github.com/charmbracelet/bubbles/textarea"
	tea "github.com/charmbracelet/bubbletea"
)

// Dependencies defines the dependencies for the key detector
type Dependencies struct {
	ConfigManager  ConfigManager
	MessageLogger  func(msgType, content string)
	RefreshView    func()
	LayoutManager  LayoutManager
	UpdateKeymap   func() // Callback to update textarea keymap after config change
}

// ConfigManager defines methods needed from config
type ConfigManager interface {
	GetNewlineKey() string
	GetHistoryBackKey() string
	GetHistoryForwardKey() string
	SetNewlineKey(key string) error
	SetHistoryBackKey(key string) error
	SetHistoryForwardKey(key string) error
}

// LayoutManager defines methods needed from layout
type LayoutManager interface {
	FormatKeyForDisplay(key string) string
}

// Detector handles key detection and configuration
type Detector struct {
	deps             Dependencies
	detectionMode    bool
	detectionType    string
}

// New creates a new key detector
func New(deps Dependencies) *Detector {
	return &Detector{
		deps: deps,
	}
}

// IsDetecting returns true if in key detection mode
func (d *Detector) IsDetecting() bool {
	return d.detectionMode
}

// GetDetectionType returns the current detection type
func (d *Detector) GetDetectionType() string {
	return d.detectionType
}

// SetDetection sets the detection mode and type
func (d *Detector) SetDetection(enabled bool, keyType string) {
	d.detectionMode = enabled
	d.detectionType = keyType
}

// StartDetection enters key detection mode to capture a specific key type
func (d *Detector) StartDetection(keyType string) {
	d.detectionMode = true
	d.detectionType = keyType

	switch keyType {
	case "newline":
		d.deps.MessageLogger("system", "🎯 Key Detection Mode - Newline")
		d.deps.MessageLogger("system", "Press your preferred key combination for newlines (usually Ctrl+Enter)")
	case "history-back":
		d.deps.MessageLogger("system", "🎯 Key Detection Mode - History Back")
		d.deps.MessageLogger("system", "Press your preferred key for previous history (default: Ctrl+P)")
	case "history-forward":
		d.deps.MessageLogger("system", "🎯 Key Detection Mode - History Forward")
		d.deps.MessageLogger("system", "Press your preferred key for next history (default: Ctrl+N)")
	}
	d.deps.MessageLogger("system", "Press Escape to cancel")
	d.deps.RefreshView()
}

// ShowKeyBindings displays current key bindings
func (d *Detector) ShowKeyBindings() {
	d.deps.MessageLogger("system", "🎹 Current Key Bindings:")

	if d.deps.ConfigManager != nil {
		newlineKey := d.deps.LayoutManager.FormatKeyForDisplay(d.deps.ConfigManager.GetNewlineKey())
		historyBackKey := d.deps.LayoutManager.FormatKeyForDisplay(d.deps.ConfigManager.GetHistoryBackKey())
		historyForwardKey := d.deps.LayoutManager.FormatKeyForDisplay(d.deps.ConfigManager.GetHistoryForwardKey())

		d.deps.MessageLogger("system", fmt.Sprintf("  • Newline:         %s", newlineKey))
		d.deps.MessageLogger("system", fmt.Sprintf("  • History Back:    %s", historyBackKey))
		d.deps.MessageLogger("system", fmt.Sprintf("  • History Forward: %s", historyForwardKey))
	} else {
		d.deps.MessageLogger("system", "  • Newline:         Ctrl+J (default)")
		d.deps.MessageLogger("system", "  • History Back:    Ctrl+P (default)")
		d.deps.MessageLogger("system", "  • History Forward: Ctrl+N (default)")
	}

	d.deps.MessageLogger("system", "")
	d.deps.MessageLogger("system", "To change a key binding:")
	d.deps.MessageLogger("system", "  /keysetup newline        - Configure newline key")
	d.deps.MessageLogger("system", "  /keysetup history-back   - Configure history back key")
	d.deps.MessageLogger("system", "  /keysetup history-forward - Configure history forward key")
	d.deps.RefreshView()
}

// HandleDetection processes keys during key detection mode
func (d *Detector) HandleDetection(keyStr string) tea.Cmd {
	if keyStr == "esc" {
		d.detectionMode = false
		d.deps.MessageLogger("system", "❌ Key detection cancelled")
		d.deps.RefreshView()
		return nil
	}

	// Detect if user pressed Enter when trying to press Ctrl+Enter
	if keyStr == "enter" {
		d.deps.MessageLogger("system", "🚨 Terminal Limitation Detected!")
		d.deps.MessageLogger("system", "Your terminal sends the same code for Enter and Ctrl+Enter.")
		d.deps.MessageLogger("system", "This is common in many terminals. Let's try a key that actually works:")
		d.deps.MessageLogger("system", "")
		d.deps.MessageLogger("system", "Please try one of these alternatives:")
		d.deps.MessageLogger("system", "  • Press Ctrl+J (this usually works)")
		d.deps.MessageLogger("system", "  • Press Ctrl+M (another alternative)")
		d.deps.MessageLogger("system", "  • Press Alt+Enter (if your terminal supports it)")
		d.deps.MessageLogger("system", "")
		d.deps.MessageLogger("system", "Or press Escape to cancel and use the default (Ctrl+J)")
		d.deps.RefreshView()
		return nil // Stay in detection mode
	}

	// Ignore common keys we don't want to capture
	ignored := []string{"up", "down", "left", "right", "home", "end", "pgup", "pgdown",
					   "f1", "f2", "f3", "f4", "f5", "f6", "f7", "f8", "f9", "f10", "f11", "f12",
					   "tab", "shift+tab", "alt", "ctrl", "shift"}
	for _, ignore := range ignored {
		if keyStr == ignore {
			return nil // Don't capture these keys
		}
	}

	// Filter out problematic key combinations that add unwanted characters
	problematic := []string{"alt+o", "alt+O", "alt+m", "alt+M"}
	for _, prob := range problematic {
		if keyStr == prob {
			d.deps.MessageLogger("system", fmt.Sprintf("⚠️ Key %s detected, but it adds extra characters to input!", keyStr))
			d.deps.MessageLogger("system", "This happens with some Alt+ combinations in certain terminals.")
			d.deps.MessageLogger("system", "")
			d.deps.MessageLogger("system", "Please try a reliable alternative:")
			d.deps.MessageLogger("system", "  • Ctrl+J (most reliable)")
			d.deps.MessageLogger("system", "  • Ctrl+M (alternative)")
			d.deps.MessageLogger("system", "")
			d.deps.MessageLogger("system", "Or press Escape to use default (Ctrl+J)")
			d.deps.RefreshView()
			return nil // Stay in detection mode
		}
	}

	// Different validation based on key type
	var validKeys []string
	var defaultKey string

	switch d.detectionType {
	case "newline":
		validKeys = []string{"ctrl+j", "ctrl+m", "ctrl+k", "ctrl+l", "alt+enter"}
		defaultKey = "Ctrl+J"
	case "history-back", "history-forward":
		// More keys are valid for history navigation
		validKeys = []string{"ctrl+p", "ctrl+n", "alt+up", "alt+down", "ctrl+up", "ctrl+down",
							"ctrl+b", "ctrl+f", "alt+p", "alt+n", "ctrl+r", "ctrl+s"}
		if d.detectionType == "history-back" {
			defaultKey = "Ctrl+P"
		} else {
			defaultKey = "Ctrl+N"
		}
	}

	isValid := false
	for _, valid := range validKeys {
		if keyStr == valid {
			isValid = true
			break
		}
	}

	if !isValid && d.detectionType == "newline" {
		d.deps.MessageLogger("system", fmt.Sprintf("⚠️ Key %s might not work reliably across all terminals.", keyStr))
		d.deps.MessageLogger("system", "For best compatibility, please use one of these tested keys:")
		d.deps.MessageLogger("system", "  • Ctrl+J (most reliable)")
		d.deps.MessageLogger("system", "  • Ctrl+M (alternative)")
		d.deps.MessageLogger("system", "")
		d.deps.MessageLogger("system", fmt.Sprintf("Or press Escape to use default (%s)", defaultKey))
		d.deps.RefreshView()
		return nil // Stay in detection mode
	}

	// For history keys, accept more combinations but warn if unusual
	if !isValid && (d.detectionType == "history-back" || d.detectionType == "history-forward") {
		// Accept it but warn
		d.deps.MessageLogger("system", fmt.Sprintf("⚠️ Key %s is non-standard but will be saved.", keyStr))
	}

	// Capture the validated key
	d.detectionMode = false

	// Save to config based on type
	if d.deps.ConfigManager != nil {
		var err error
		switch d.detectionType {
		case "newline":
			err = d.deps.ConfigManager.SetNewlineKey(keyStr)
		case "history-back":
			err = d.deps.ConfigManager.SetHistoryBackKey(keyStr)
		case "history-forward":
			err = d.deps.ConfigManager.SetHistoryForwardKey(keyStr)
		}

		if err != nil {
			d.deps.MessageLogger("system", fmt.Sprintf("❌ Failed to save key configuration: %v", err))
		} else {
			displayKey := d.deps.LayoutManager.FormatKeyForDisplay(keyStr)
			switch d.detectionType {
			case "newline":
				d.deps.MessageLogger("system", fmt.Sprintf("✅ Newline key set to: %s", displayKey))
				d.deps.MessageLogger("system", "🔄 Updating textarea configuration...")
			case "history-back":
				d.deps.MessageLogger("system", fmt.Sprintf("✅ History back key set to: %s", displayKey))
			case "history-forward":
				d.deps.MessageLogger("system", fmt.Sprintf("✅ History forward key set to: %s", displayKey))
			}

			// Update textarea keymap if it's a newline key change
			if d.detectionType == "newline" && d.deps.UpdateKeymap != nil {
				d.deps.UpdateKeymap()
			}

			d.deps.MessageLogger("system", "✅ Configuration updated! Try your new key combination.")
		}
	} else {
		d.deps.MessageLogger("system", "❌ Config manager not available")
	}

	d.deps.RefreshView()
	return nil
}

// UpdateTextareaKeymap updates the textarea keymap with the configured newline key
func (d *Detector) UpdateTextareaKeymap(textarea *textarea.Model) {
	if d.deps.ConfigManager != nil && textarea != nil {
		newlineKey := d.deps.ConfigManager.GetNewlineKey()
		keyMap := textarea.KeyMap
		keyMap.InsertNewline.SetKeys(newlineKey)
		textarea.KeyMap = keyMap
	}
}