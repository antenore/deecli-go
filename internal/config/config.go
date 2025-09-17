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

	"gopkg.in/yaml.v3"
)

type Config struct {
	APIKey      string             `yaml:"api_key"`
	Model       string             `yaml:"model"`
	Temperature float64            `yaml:"temperature"`
	MaxTokens   int                `yaml:"max_tokens"`
	NewlineKey  string             `yaml:"newline_key,omitempty"`
	HistoryBackKey string          `yaml:"history_back_key,omitempty"`
	HistoryForwardKey string       `yaml:"history_forward_key,omitempty"`
	Profiles    map[string]Profile `yaml:"profiles,omitempty"`
	ActiveProfile string           `yaml:"active_profile,omitempty"`
}

type Profile struct {
	APIKey      string  `yaml:"api_key,omitempty"`
	Model       string  `yaml:"model,omitempty"`
	Temperature float64 `yaml:"temperature,omitempty"`
	MaxTokens   int     `yaml:"max_tokens,omitempty"`
}

var (
	defaultConfig = Config{
		Model:       "deepseek-chat",
		Temperature: 0.1,
		MaxTokens:   2048,
		Profiles:    make(map[string]Profile),
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

	// Load project config
	if err := m.loadConfigFile(m.projectPath, m.projectConfig); err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("failed to load project config: %w", err)
	}

	// Merge configurations
	m.mergedConfig = m.mergeConfigs()

	// Apply environment variables (highest priority)
	m.applyEnvironmentOverrides()

	return nil
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
		if len(m.globalConfig.Profiles) > 0 {
			merged.Profiles = m.globalConfig.Profiles
		}
		if m.globalConfig.ActiveProfile != "" {
			merged.ActiveProfile = m.globalConfig.ActiveProfile
		}
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
		if m.projectConfig.ActiveProfile != "" {
			merged.ActiveProfile = m.projectConfig.ActiveProfile
		}
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
		APIKey:      apiKey,
		Model:       defaultConfig.Model,
		Temperature: defaultConfig.Temperature,
		MaxTokens:   defaultConfig.MaxTokens,
		Profiles:    make(map[string]Profile),
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