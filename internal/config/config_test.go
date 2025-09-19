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
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestValidateModel(t *testing.T) {
	tests := []struct {
		name    string
		model   string
		wantErr bool
		errMsg  string
	}{
		{
			name:    "Valid model deepseek-chat",
			model:   "deepseek-chat",
			wantErr: false,
		},
		{
			name:    "Valid model deepseek-reasoner",
			model:   "deepseek-reasoner",
			wantErr: false,
		},
		{
			name:    "Empty model is valid",
			model:   "",
			wantErr: false,
		},
		{
			name:    "Invalid model gpt-4",
			model:   "gpt-4",
			wantErr: true,
			errMsg:  "invalid model 'gpt-4'. Valid models are: deepseek-chat, deepseek-reasoner",
		},
		{
			name:    "Invalid model claude",
			model:   "claude",
			wantErr: true,
			errMsg:  "invalid model 'claude'",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateModel(tt.model)
			if tt.wantErr {
				assert.Error(t, err)
				if tt.errMsg != "" {
					assert.Contains(t, err.Error(), tt.errMsg)
				}
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestValidateAPIKey(t *testing.T) {
	tests := []struct {
		name    string
		apiKey  string
		wantErr bool
		errMsg  string
	}{
		{
			name:    "Valid API key",
			apiKey:  "sk-abcdef123456789012345678901234567890",
			wantErr: false,
		},
		{
			name:    "Empty API key is valid",
			apiKey:  "",
			wantErr: false,
		},
		{
			name:    "Invalid prefix",
			apiKey:  "api-abcdef123456789012345678901234567890",
			wantErr: true,
			errMsg:  "API key should start with 'sk-'",
		},
		{
			name:    "Too short API key",
			apiKey:  "sk-abc",
			wantErr: true,
			errMsg:  "API key appears too short",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateAPIKey(tt.apiKey)
			if tt.wantErr {
				assert.Error(t, err)
				if tt.errMsg != "" {
					assert.Contains(t, err.Error(), tt.errMsg)
				}
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestValidateKeyBinding(t *testing.T) {
	tests := []struct {
		name    string
		key     string
		wantErr bool
		errMsg  string
	}{
		{
			name:    "Valid ctrl+j",
			key:     "ctrl+j",
			wantErr: false,
		},
		{
			name:    "Valid alt+enter",
			key:     "alt+enter",
			wantErr: false,
		},
		{
			name:    "Valid shift+tab",
			key:     "shift+tab",
			wantErr: false,
		},
		{
			name:    "Valid special key enter",
			key:     "enter",
			wantErr: false,
		},
		{
			name:    "Valid special key escape",
			key:     "escape",
			wantErr: false,
		},
		{
			name:    "Empty key is valid",
			key:     "",
			wantErr: false,
		},
		{
			name:    "Invalid format control-j",
			key:     "control-j",
			wantErr: true,
			errMsg:  "invalid key binding",
		},
		{
			name:    "Invalid key ctrl+xyz",
			key:     "ctrl+xyz",
			wantErr: true,
			errMsg:  "invalid key binding",
		},
		{
			name:    "Invalid modifier xxx+j",
			key:     "xxx+j",
			wantErr: true,
			errMsg:  "invalid key binding",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateKeyBinding(tt.key)
			if tt.wantErr {
				assert.Error(t, err)
				if tt.errMsg != "" {
					assert.Contains(t, err.Error(), tt.errMsg)
				}
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestValidateTemperature(t *testing.T) {
	tests := []struct {
		name    string
		temp    float64
		wantErr bool
	}{
		{
			name:    "Valid temperature 0.0",
			temp:    0.0,
			wantErr: false,
		},
		{
			name:    "Valid temperature 1.0",
			temp:    1.0,
			wantErr: false,
		},
		{
			name:    "Valid temperature 2.0",
			temp:    2.0,
			wantErr: false,
		},
		{
			name:    "Invalid temperature negative",
			temp:    -0.1,
			wantErr: true,
		},
		{
			name:    "Invalid temperature too high",
			temp:    2.1,
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateTemperature(tt.temp)
			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestValidateMaxTokens(t *testing.T) {
	tests := []struct {
		name    string
		tokens  int
		wantErr bool
	}{
		{
			name:    "Valid tokens 1024",
			tokens:  1024,
			wantErr: false,
		},
		{
			name:    "Valid tokens 8192",
			tokens:  8192,
			wantErr: false,
		},
		{
			name:    "Invalid tokens zero",
			tokens:  0,
			wantErr: true,
		},
		{
			name:    "Invalid tokens negative",
			tokens:  -1,
			wantErr: true,
		},
		{
			name:    "Invalid tokens too high",
			tokens:  40000,
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateMaxTokens(tt.tokens)
			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestConfig_Validate(t *testing.T) {
	tests := []struct {
		name    string
		config  Config
		wantErr bool
		errMsg  string
	}{
		{
			name: "Valid config",
			config: Config{
				Model:       "deepseek-chat",
				APIKey:      "sk-abcdef123456789012345678901234567890",
				Temperature: 0.7,
				MaxTokens:   2048,
			},
			wantErr: false,
		},
		{
			name: "Invalid model",
			config: Config{
				Model: "gpt-4",
			},
			wantErr: true,
			errMsg:  "invalid model",
		},
		{
			name: "Key binding conflict",
			config: Config{
				Model:             "deepseek-chat",
				NewlineKey:        "ctrl+j",
				HistoryBackKey:    "ctrl+j",
			},
			wantErr: true,
			errMsg:  "key binding conflict",
		},
		{
			name: "Invalid profile",
			config: Config{
				Model: "deepseek-chat",
				Profiles: map[string]Profile{
					"test": {
						Model: "invalid-model",
					},
				},
			},
			wantErr: true,
			errMsg:  "profile 'test'",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Set defaults for required fields if not specified
			if tt.config.Temperature == 0 {
				tt.config.Temperature = 0.1
			}
			if tt.config.MaxTokens == 0 {
				tt.config.MaxTokens = 2048
			}

			err := tt.config.Validate()
			if tt.wantErr {
				assert.Error(t, err)
				if tt.errMsg != "" {
					assert.Contains(t, err.Error(), tt.errMsg)
				}
			} else {
				assert.NoError(t, err)
			}
		})
	}
}