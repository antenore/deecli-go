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

package config

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"gopkg.in/yaml.v3"
)

type Config struct {
	APIKey           string             `yaml:"api_key"`
	Model            string             `yaml:"model"`
	Temperature      float64            `yaml:"temperature"`
	MaxTokens        int                `yaml:"max_tokens"`
	UserName         string             `yaml:"user_name,omitempty"`              // User display name in chat
	NewlineKey       string             `yaml:"newline_key,omitempty"`
	HistoryBackKey   string             `yaml:"history_back_key,omitempty"`
	HistoryForwardKey string            `yaml:"history_forward_key,omitempty"`
	Profiles         map[string]Profile `yaml:"profiles,omitempty"`
	ActiveProfile    string             `yaml:"active_profile,omitempty"`
	AutoReloadFiles  bool               `yaml:"auto_reload_files,omitempty"`     // Enable file auto-reload
	AutoReloadDebounce int              `yaml:"auto_reload_debounce,omitempty"`  // Debounce time in ms
	ShowReloadNotices  bool             `yaml:"show_reload_notices,omitempty"`   // Show reload notifications
}

type Profile struct {
	APIKey      string  `yaml:"api_key,omitempty"`
	Model       string  `yaml:"model,omitempty"`
	Temperature float64 `yaml:"temperature,omitempty"`
	MaxTokens   int     `yaml:"max_tokens,omitempty"`
}

var (
	defaultConfig = Config{
		Model:            "deepseek-chat",
		Temperature:      0.1,
		MaxTokens:        2048,
		UserName:         "You",
		Profiles:         make(map[string]Profile),
		AutoReloadFiles:  true,
		AutoReloadDebounce: 100,
		ShowReloadNotices: true,
	}
)

type Manager struct {
	globalConfig  *Config
	projectConfig *Config
	mergedConfig  *Config
	globalPath    string
	projectPath   string
}

func NewManager() *Manager {
	home, _ := os.UserHomeDir()
	globalPath := filepath.Join(home, ".deecli", "config.yaml")
	projectPath := filepath.Join(".deecli", "config.yaml")

	return &Manager{
		globalPath:  globalPath,
		projectPath: projectPath,
	}
}

func (m *Manager) Load() error {
	m.globalConfig = &Config{}
	m.projectConfig = &Config{}

	// Load global config
	if err := m.loadConfigFile(m.globalPath, m.globalConfig); err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("failed to load global config: %w", err)
	}

	// Validate global config
	if m.globalConfig != nil && !isEmptyConfig(m.globalConfig) {
		if err := m.globalConfig.Validate(); err != nil {
			return fmt.Errorf("invalid global config: %w", err)
		}
	}

	// Load project config
	if err := m.loadConfigFile(m.projectPath, m.projectConfig); err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("failed to load project config: %w", err)
	}

	// Validate project config
	if m.projectConfig != nil && !isEmptyConfig(m.projectConfig) {
		if err := m.projectConfig.Validate(); err != nil {
			return fmt.Errorf("invalid project config: %w", err)
		}
	}

	// Merge configurations
	m.mergedConfig = m.mergeConfigs()

	// Apply environment variables (highest priority)
	m.applyEnvironmentOverrides()

	// Validate final merged config
	if err := m.mergedConfig.Validate(); err != nil {
		return fmt.Errorf("invalid configuration: %w", err)
	}

	return nil
}

// isEmptyConfig checks if a config struct has all zero values
func isEmptyConfig(c *Config) bool {
	return c.APIKey == "" && c.Model == "" && c.Temperature == 0 && c.MaxTokens == 0
}

func (m *Manager) loadConfigFile(path string, cfg *Config) error {
	data, err := os.ReadFile(path)
	if err != nil {
		return err
	}

	// Start with defaults
	*cfg = defaultConfig

	// Unmarshal YAML, overriding defaults
	if err := yaml.Unmarshal(data, cfg); err != nil {
		return fmt.Errorf("invalid YAML in %s: %w", path, err)
	}

	return nil
}

func (m *Manager) mergeConfigs() *Config {
	merged := defaultConfig

	// Apply global config
	if m.globalConfig != nil {
		if m.globalConfig.APIKey != "" {
			merged.APIKey = m.globalConfig.APIKey
		}
		if m.globalConfig.Model != "" {
			merged.Model = m.globalConfig.Model
		}
		if m.globalConfig.Temperature != 0 {
			merged.Temperature = m.globalConfig.Temperature
		}
		if m.globalConfig.MaxTokens != 0 {
			merged.MaxTokens = m.globalConfig.MaxTokens
		}
		if m.globalConfig.UserName != "" {
			merged.UserName = m.globalConfig.UserName
		}
		if len(m.globalConfig.Profiles) > 0 {
			merged.Profiles = m.globalConfig.Profiles
		}
		if m.globalConfig.ActiveProfile != "" {
			merged.ActiveProfile = m.globalConfig.ActiveProfile
		}
		// Auto-reload settings (use explicit checks for booleans)
		merged.AutoReloadFiles = m.globalConfig.AutoReloadFiles
		if m.globalConfig.AutoReloadDebounce != 0 {
			merged.AutoReloadDebounce = m.globalConfig.AutoReloadDebounce
		}
		merged.ShowReloadNotices = m.globalConfig.ShowReloadNotices
	}

	// Apply project config (higher priority)
	if m.projectConfig != nil {
		if m.projectConfig.APIKey != "" {
			merged.APIKey = m.projectConfig.APIKey
		}
		if m.projectConfig.Model != "" {
			merged.Model = m.projectConfig.Model
		}
		if m.projectConfig.Temperature != 0 {
			merged.Temperature = m.projectConfig.Temperature
		}
		if m.projectConfig.MaxTokens != 0 {
			merged.MaxTokens = m.projectConfig.MaxTokens
		}
		if m.projectConfig.UserName != "" {
			merged.UserName = m.projectConfig.UserName
		}
		if m.projectConfig.ActiveProfile != "" {
			merged.ActiveProfile = m.projectConfig.ActiveProfile
		}
		// Auto-reload settings from project config (higher priority)
		merged.AutoReloadFiles = m.projectConfig.AutoReloadFiles
		if m.projectConfig.AutoReloadDebounce != 0 {
			merged.AutoReloadDebounce = m.projectConfig.AutoReloadDebounce
		}
		merged.ShowReloadNotices = m.projectConfig.ShowReloadNotices
		// Merge profiles
		for name, profile := range m.projectConfig.Profiles {
			merged.Profiles[name] = profile
		}
	}

	// Apply active profile if set
	if merged.ActiveProfile != "" {
		if profile, exists := merged.Profiles[merged.ActiveProfile]; exists {
			if profile.APIKey != "" {
				merged.APIKey = profile.APIKey
			}
			if profile.Model != "" {
				merged.Model = profile.Model
			}
			if profile.Temperature != 0 {
				merged.Temperature = profile.Temperature
			}
			if profile.MaxTokens != 0 {
				merged.MaxTokens = profile.MaxTokens
			}
		}
	}

	return &merged
}

func (m *Manager) applyEnvironmentOverrides() {
	if apiKey := os.Getenv("DEEPSEEK_API_KEY"); apiKey != "" {
		m.mergedConfig.APIKey = apiKey
	}
}

func (m *Manager) Get() *Config {
	if m.mergedConfig == nil {
		return &defaultConfig
	}
	return m.mergedConfig
}

func (m *Manager) GetAPIKey() string {
	return m.Get().APIKey
}

func (m *Manager) GetModel() string {
	return m.Get().Model
}

func (m *Manager) GetTemperature() float64 {
	return m.Get().Temperature
}

func (m *Manager) GetMaxTokens() int {
	return m.Get().MaxTokens
}

func (m *Manager) HasAPIKey() bool {
	return m.Get().APIKey != ""
}

func (m *Manager) SaveGlobal(cfg *Config) error {
	// Ensure directory exists
	dir := filepath.Dir(m.globalPath)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("failed to create config directory: %w", err)
	}

	// Marshal to YAML
	data, err := yaml.Marshal(cfg)
	if err != nil {
		return fmt.Errorf("failed to marshal config: %w", err)
	}

	// Write file
	if err := os.WriteFile(m.globalPath, data, 0600); err != nil {
		return fmt.Errorf("failed to write config file: %w", err)
	}

	return nil
}

func (m *Manager) SaveProject(cfg *Config) error {
	// Ensure directory exists
	dir := filepath.Dir(m.projectPath)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("failed to create config directory: %w", err)
	}

	// Marshal to YAML
	data, err := yaml.Marshal(cfg)
	if err != nil {
		return fmt.Errorf("failed to marshal config: %w", err)
	}

	// Write file
	if err := os.WriteFile(m.projectPath, data, 0600); err != nil {
		return fmt.Errorf("failed to write config file: %w", err)
	}

	return nil
}

func (m *Manager) InitGlobalConfig(apiKey string) error {
	cfg := &Config{
		APIKey:           apiKey,
		Model:            defaultConfig.Model,
		Temperature:      defaultConfig.Temperature,
		MaxTokens:        defaultConfig.MaxTokens,
		UserName:         defaultConfig.UserName,
		Profiles:         make(map[string]Profile),
		AutoReloadFiles:  defaultConfig.AutoReloadFiles,
		AutoReloadDebounce: defaultConfig.AutoReloadDebounce,
		ShowReloadNotices: defaultConfig.ShowReloadNotices,
	}

	return m.SaveGlobal(cfg)
}

func (m *Manager) GlobalConfigExists() bool {
	_, err := os.Stat(m.globalPath)
	return err == nil
}

func (m *Manager) ProjectConfigExists() bool {
	_, err := os.Stat(m.projectPath)
	return err == nil
}

// GetNewlineKey returns the configured newline key with fallback defaults
func (m *Manager) GetNewlineKey() string {
	cfg := m.Get()
	if cfg.NewlineKey != "" {
		return cfg.NewlineKey
	}
	// Default fallback - try common key combinations
	return "ctrl+j"
}

// SetNewlineKey saves the detected newline key to global config
func (m *Manager) SetNewlineKey(key string) error {
	cfg := m.Get()
	cfg.NewlineKey = key
	return m.SaveGlobal(cfg)
}

// GetAutoReloadFiles returns whether file auto-reload is enabled
func (m *Manager) GetAutoReloadFiles() bool {
	cfg := m.Get()
	return cfg.AutoReloadFiles
}

// GetAutoReloadDebounce returns the debounce time for auto-reload in milliseconds
func (m *Manager) GetAutoReloadDebounce() int {
	cfg := m.Get()
	if cfg.AutoReloadDebounce == 0 {
		return 100 // Default to 100ms
	}
	return cfg.AutoReloadDebounce
}

// GetShowReloadNotices returns whether reload notifications should be shown
func (m *Manager) GetShowReloadNotices() bool {
	cfg := m.Get()
	return cfg.ShowReloadNotices
}

// GetUserName returns the configured user name for display in chat
func (m *Manager) GetUserName() string {
	cfg := m.Get()
	if cfg.UserName != "" {
		return cfg.UserName
	}
	return "You"
}

// GetHistoryBackKey returns the configured history back key with fallback defaults
func (m *Manager) GetHistoryBackKey() string {
	cfg := m.Get()
	if cfg.HistoryBackKey != "" {
		return cfg.HistoryBackKey
	}
	// Default fallback - ctrl+p for previous (Unix/Emacs style)
	return "ctrl+p"
}

// SetHistoryBackKey saves the detected history back key to global config
func (m *Manager) SetHistoryBackKey(key string) error {
	cfg := m.Get()
	cfg.HistoryBackKey = key
	return m.SaveGlobal(cfg)
}

// GetHistoryForwardKey returns the configured history forward key with fallback defaults
func (m *Manager) GetHistoryForwardKey() string {
	cfg := m.Get()
	if cfg.HistoryForwardKey != "" {
		return cfg.HistoryForwardKey
	}
	// Default fallback - ctrl+n for next (Unix/Emacs style)
	return "ctrl+n"
}

// SetHistoryForwardKey saves the detected history forward key to global config
func (m *Manager) SetHistoryForwardKey(key string) error {
	cfg := m.Get()
	cfg.HistoryForwardKey = key
	return m.SaveGlobal(cfg)
}

// SetKeyBinding saves a specific key binding to global config
func (m *Manager) SetKeyBinding(keyType, key string) error {
	cfg := m.Get()
	switch keyType {
	case "newline":
		cfg.NewlineKey = key
	case "history-back":
		cfg.HistoryBackKey = key
	case "history-forward":
		cfg.HistoryForwardKey = key
	default:
		return fmt.Errorf("unknown key type: %s", keyType)
	}
	return m.SaveGlobal(cfg)
}

// Validation functions

var (
	// ValidModels contains the list of supported DeepSeek models
	ValidModels = []string{"deepseek-chat", "deepseek-reasoner"}

	// KeyBindingPattern matches valid key binding formats like ctrl+j, alt+enter, shift+tab
	KeyBindingPattern = regexp.MustCompile(`^(ctrl|alt|shift|cmd|meta)(\+(ctrl|alt|shift|cmd|meta))*\+([a-z0-9]|enter|tab|space|escape|esc|up|down|left|right|home|end|pageup|pagedown|f[1-9]|f1[0-2])$|^(enter|tab|space|escape|esc|up|down|left|right|home|end|pageup|pagedown|f[1-9]|f1[0-2])$`)
)

// ValidateModel checks if the model name is valid
func ValidateModel(model string) error {
	if model == "" {
		return nil // Empty is ok, will use default
	}

	for _, valid := range ValidModels {
		if model == valid {
			return nil
		}
	}

	return fmt.Errorf("invalid model '%s'. Valid models are: %s",
		model, strings.Join(ValidModels, ", "))
}

// ValidateAPIKey performs basic validation on the API key
func ValidateAPIKey(apiKey string) error {
	if apiKey == "" {
		return nil // Empty is ok, can be set later or via env
	}

	if !strings.HasPrefix(apiKey, "sk-") {
		return fmt.Errorf("API key should start with 'sk-'. Got: %s...",
			apiKey[:min(4, len(apiKey))])
	}

	if len(apiKey) < 20 {
		return fmt.Errorf("API key appears too short")
	}

	return nil
}

// ValidateKeyBinding checks if a key binding string is valid
func ValidateKeyBinding(key string) error {
	if key == "" {
		return nil // Empty is ok, will use default
	}

	key = strings.ToLower(key)
	if !KeyBindingPattern.MatchString(key) {
		return fmt.Errorf("invalid key binding '%s'. Format should be like: ctrl+j, alt+enter, shift+tab, or special keys like: enter, tab, escape", key)
	}

	return nil
}

// ValidateTemperature checks if temperature is in valid range
func ValidateTemperature(temp float64) error {
	if temp < 0.0 || temp > 2.0 {
		return fmt.Errorf("temperature must be between 0.0 and 2.0, got: %.2f", temp)
	}
	return nil
}

// ValidateMaxTokens checks if max tokens is valid
func ValidateMaxTokens(tokens int) error {
	if tokens <= 0 {
		return fmt.Errorf("max_tokens must be positive, got: %d", tokens)
	}
	if tokens > 32768 {
		return fmt.Errorf("max_tokens exceeds maximum (32768), got: %d", tokens)
	}
	return nil
}

// ValidateAutoReloadDebounce checks if debounce time is valid
func ValidateAutoReloadDebounce(debounce int) error {
	if debounce < 0 {
		return fmt.Errorf("auto_reload_debounce cannot be negative, got: %d", debounce)
	}
	if debounce > 5000 {
		return fmt.Errorf("auto_reload_debounce too high (max 5000ms), got: %d", debounce)
	}
	return nil
}

// ValidateUserName checks if user name is valid
func ValidateUserName(name string) error {
	if name == "" {
		return nil // Empty is ok, will use default
	}
	if len(name) > 50 {
		return fmt.Errorf("user name too long (max 50 characters), got: %d", len(name))
	}
	// Basic sanitization check - no control characters
	for _, r := range name {
		if r < 32 || r == 127 {
			return fmt.Errorf("user name contains invalid characters")
		}
	}
	return nil
}

// Validate performs validation on the entire config
func (c *Config) Validate() error {
	// Validate model
	if err := ValidateModel(c.Model); err != nil {
		return err
	}

	// Validate API key
	if err := ValidateAPIKey(c.APIKey); err != nil {
		return err
	}

	// Validate temperature
	if err := ValidateTemperature(c.Temperature); err != nil {
		return err
	}

	// Validate max tokens
	if err := ValidateMaxTokens(c.MaxTokens); err != nil {
		return err
	}

	// Validate user name
	if err := ValidateUserName(c.UserName); err != nil {
		return err
	}

	// Validate key bindings
	if err := ValidateKeyBinding(c.NewlineKey); err != nil {
		return fmt.Errorf("newline_key: %w", err)
	}

	if err := ValidateKeyBinding(c.HistoryBackKey); err != nil {
		return fmt.Errorf("history_back_key: %w", err)
	}

	if err := ValidateKeyBinding(c.HistoryForwardKey); err != nil {
		return fmt.Errorf("history_forward_key: %w", err)
	}

	// Check for key binding conflicts
	keys := make(map[string]string)
	if c.NewlineKey != "" {
		keys[strings.ToLower(c.NewlineKey)] = "newline"
	}
	if c.HistoryBackKey != "" {
		if existing, ok := keys[strings.ToLower(c.HistoryBackKey)]; ok {
			return fmt.Errorf("key binding conflict: %s is used for both %s and history_back",
				c.HistoryBackKey, existing)
		}
		keys[strings.ToLower(c.HistoryBackKey)] = "history_back"
	}
	if c.HistoryForwardKey != "" {
		if existing, ok := keys[strings.ToLower(c.HistoryForwardKey)]; ok {
			return fmt.Errorf("key binding conflict: %s is used for both %s and history_forward",
				c.HistoryForwardKey, existing)
		}
	}

	// Validate auto-reload debounce
	if err := ValidateAutoReloadDebounce(c.AutoReloadDebounce); err != nil {
		return err
	}

	// Validate profiles
	for name, profile := range c.Profiles {
		if err := ValidateModel(profile.Model); err != nil {
			return fmt.Errorf("profile '%s': %w", name, err)
		}
		if err := ValidateAPIKey(profile.APIKey); err != nil {
			return fmt.Errorf("profile '%s': %w", name, err)
		}
		if profile.Temperature != 0 {
			if err := ValidateTemperature(profile.Temperature); err != nil {
				return fmt.Errorf("profile '%s': %w", name, err)
			}
		}
		if profile.MaxTokens != 0 {
			if err := ValidateMaxTokens(profile.MaxTokens); err != nil {
				return fmt.Errorf("profile '%s': %w", name, err)
			}
		}
	}

	return nil
}

// min returns the minimum of two integers
func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}