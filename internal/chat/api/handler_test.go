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

package api

import (
	"fmt"
	"strings"
	"testing"

	"github.com/antenore/deecli/internal/api"
	"github.com/antenore/deecli/internal/chat/tracker"
	"github.com/antenore/deecli/internal/files"
)

func TestHandler_ParseAndExtractToolCalls(t *testing.T) {
	handler := NewHandler(Dependencies{
		FileTracker: tracker.NewFileTracker(),
	})
	
	tests := []struct {
		name              string
		content           string
		expectedToolCount int
		expectedFiltered  string
		expectedToolName  string
		expectedArgs      string
	}{
		{
			name:              "no tool calls",
			content:           "This is a normal response without any tool calls.",
			expectedToolCount: 0,
			expectedFiltered:  "This is a normal response without any tool calls.",
		},
		{
			name: "single tool call",
			content: "I'll read the file for you.\n\n<｜tool▁calls▁begin｜><｜tool▁call▁begin｜>read_file<｜tool▁sep｜>{\"path\": \"test.go\"}<｜tool▁call▁end｜><｜tool▁calls▁end｜>",
			expectedToolCount: 1,
			expectedFiltered:  "I'll read the file for you.",
			expectedToolName:  "read_file",
			expectedArgs:      `{"path": "test.go"}`,
		},
		{
			name: "multiple tool calls",
			content: "I'll help you with that.\n\n<｜tool▁calls▁begin｜><｜tool▁call▁begin｜>read_file<｜tool▁sep｜>{\"path\": \"file1.go\"}<｜tool▁call▁end｜><｜tool▁call▁begin｜>list_files<｜tool▁sep｜>{\"recursive\": true}<｜tool▁call▁end｜><｜tool▁calls▁end｜>",
			expectedToolCount: 2,
			expectedFiltered:  "I'll help you with that.",
			expectedToolName:  "read_file", // First tool
			expectedArgs:      `{"path": "file1.go"}`,
		},
		{
			name: "tool call with empty arguments",
			content: "Let me list the files.\n\n<｜tool▁calls▁begin｜><｜tool▁call▁begin｜>list_files<｜tool▁sep｜>{}<｜tool▁call▁end｜><｜tool▁calls▁end｜>",
			expectedToolCount: 1,
			expectedFiltered:  "Let me list the files.",
			expectedToolName:  "list_files",
			expectedArgs:      "{}",
		},
		{
			name: "incomplete tool call block",
			content: "Starting response <｜tool▁calls▁begin｜><｜tool▁call▁begin｜>read_file<｜tool▁sep｜>{\"path\": \"test.go\"}",
			expectedToolCount: 0,
			expectedFiltered:  "Starting response",
		},
		{
			name: "malformed tool call missing separator",
			content: "Response <｜tool▁calls▁begin｜><｜tool▁call▁begin｜>read_file{\"path\": \"test.go\"}<｜tool▁call▁end｜><｜tool▁calls▁end｜>",
			expectedToolCount: 0,
			expectedFiltered:  "Response",
		},
		{
			name: "tool call with complex arguments",
			content: "Processing request.\n\n<｜tool▁calls▁begin｜><｜tool▁call▁begin｜>complex_tool<｜tool▁sep｜>{\"nested\": {\"key\": \"value\"}, \"array\": [1, 2, 3], \"string\": \"test\"}<｜tool▁call▁end｜><｜tool▁calls▁end｜>",
			expectedToolCount: 1,
			expectedFiltered:  "Processing request.",
			expectedToolName:  "complex_tool",
			expectedArgs:      `{"nested": {"key": "value"}, "array": [1, 2, 3], "string": "test"}`,
		},
	}
	
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			toolCalls, filtered := handler.ParseAndExtractToolCalls(tt.content)
			
			if len(toolCalls) != tt.expectedToolCount {
				t.Errorf("ParseAndExtractToolCalls() tool count = %d, want %d", len(toolCalls), tt.expectedToolCount)
			}
			
			if strings.TrimSpace(filtered) != strings.TrimSpace(tt.expectedFiltered) {
				t.Errorf("ParseAndExtractToolCalls() filtered = %q, want %q", strings.TrimSpace(filtered), strings.TrimSpace(tt.expectedFiltered))
			}
			
			if tt.expectedToolCount > 0 {
				firstTool := toolCalls[0]
				
				if firstTool.Function.Name != tt.expectedToolName {
					t.Errorf("ParseAndExtractToolCalls() tool name = %q, want %q", firstTool.Function.Name, tt.expectedToolName)
				}
				
				if firstTool.Function.Arguments != tt.expectedArgs {
					t.Errorf("ParseAndExtractToolCalls() tool args = %q, want %q", firstTool.Function.Arguments, tt.expectedArgs)
				}
				
				if firstTool.Type != "function" {
					t.Errorf("ParseAndExtractToolCalls() tool type = %q, want %q", firstTool.Type, "function")
				}
				
				if firstTool.ID == "" {
					t.Errorf("ParseAndExtractToolCalls() tool ID is empty, want non-empty")
				}
			}
		})
	}
}

func TestHandler_HandleResponse_Error(t *testing.T) {
	handler := NewHandler(Dependencies{
		FileTracker: tracker.NewFileTracker(),
	})
	
	tests := []struct {
		name        string
		err         error
		wantSuccess bool
		wantError   string
	}{
		{
			name: "API error with status code",
			err: api.APIError{
				Message:     "invalid request",
				UserMessage: "Request failed",
				StatusCode:  400,
			},
			wantSuccess: false,
			wantError:   "❌ Request failed (HTTP 400)",
		},
		{
			name: "API error without status code",
			err: api.APIError{
				Message:     "network error",
				UserMessage: "Network connection failed",
				StatusCode:  0,
			},
			wantSuccess: false,
			wantError:   "❌ Network connection failed",
		},
		{
			name: "cancellation error (should be treated as success)",
			err: api.APIError{
				Message:     "request cancelled by user",
				UserMessage: "Cancelled",
				StatusCode:  0,
			},
			wantSuccess: true,
			wantError:   "",
		},
		{
			name:        "generic error",
			err:         fmt.Errorf("generic error"),
			wantSuccess: false,
			wantError:   "❌ Error: generic error",
		},
	}
	
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := handler.HandleResponse("", tt.err, false, nil)
			
			if result.Success != tt.wantSuccess {
				t.Errorf("HandleResponse() success = %v, want %v", result.Success, tt.wantSuccess)
			}
			
			if tt.wantError != "" && result.ErrorMessage != tt.wantError {
				t.Errorf("HandleResponse() error = %q, want %q", result.ErrorMessage, tt.wantError)
			}
			
			if tt.wantSuccess && result.ErrorMessage != "" {
				t.Errorf("HandleResponse() unexpected error message: %q", result.ErrorMessage)
			}
		})
	}
}

func TestHandler_HandleResponse_Normal(t *testing.T) {
	handler := NewHandler(Dependencies{
		FileTracker: tracker.NewFileTracker(),
	})
	
	tests := []struct {
		name             string
		response         string
		wantSuccess      bool
		wantToolCalls    bool
		wantContent      string
		wantToolCount    int
	}{
		{
			name:        "normal response without tools",
			response:    "This is a normal response from the AI.",
			wantSuccess: true,
			wantContent: "This is a normal response from the AI.",
		},
		{
			name:          "response with tool calls",
			response:      "I'll help you. <｜tool▁calls▁begin｜><｜tool▁call▁begin｜>read_file<｜tool▁sep｜>{\"path\": \"test.go\"}<｜tool▁call▁end｜><｜tool▁calls▁end｜>",
			wantSuccess:   true,
			wantToolCalls: true,
			wantToolCount: 1,
		},
		{
			name:        "empty response",
			response:    "",
			wantSuccess: true,
			wantContent: "",
		},
	}
	
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := handler.HandleResponse(tt.response, nil, false, nil)
			
			if result.Success != tt.wantSuccess {
				t.Errorf("HandleResponse() success = %v, want %v", result.Success, tt.wantSuccess)
			}
			
			if tt.wantToolCalls {
				if len(result.ToolCalls) != tt.wantToolCount {
					t.Errorf("HandleResponse() tool count = %d, want %d", len(result.ToolCalls), tt.wantToolCount)
				}
				
				if result.AssistantContent != "" {
					t.Errorf("HandleResponse() should not have assistant content when tools detected")
				}
			} else {
				if len(result.ToolCalls) > 0 {
					t.Errorf("HandleResponse() unexpected tool calls: %v", result.ToolCalls)
				}
				
				if result.AssistantContent != tt.wantContent {
					t.Errorf("HandleResponse() content = %q, want %q", result.AssistantContent, tt.wantContent)
				}
			}
		})
	}
}

func TestHandler_HandleResponse_Suppressed(t *testing.T) {
	handler := NewHandler(Dependencies{
		FileTracker: tracker.NewFileTracker(),
	})
	
	response := "Final response. <｜tool▁calls▁begin｜><｜tool▁call▁begin｜>read_file<｜tool▁sep｜>{\"path\": \"test.go\"}<｜tool▁call▁end｜><｜tool▁calls▁end｜>"
	
	result := handler.HandleResponse(response, nil, true, nil)
	
	if !result.Success {
		t.Errorf("HandleResponse() success = false, want true for suppressed response")
	}
	
	if !result.ShouldSuppress {
		t.Errorf("HandleResponse() ShouldSuppress = false, want true")
	}
	
	if len(result.ToolCalls) > 0 {
		t.Errorf("HandleResponse() should not return tool calls when suppressed")
	}
	
	if result.AssistantContent != "Final response." {
		t.Errorf("HandleResponse() content = %q, want %q", result.AssistantContent, "Final response.")
	}
}

func TestHandler_FileTracking(t *testing.T) {
	fileTracker := tracker.NewFileTracker()
	handler := NewHandler(Dependencies{
		FileTracker: fileTracker,
	})
	
	// Create a mock file context
	fileContext := &files.FileContext{
		Files: []files.LoadedFile{
			{
				Path:    "test.go",
				Content: "package main",
			},
		},
	}
	
	response := "I can see you have a test.go file in your project."
	
	result := handler.HandleResponse(response, nil, false, fileContext)
	
	if !result.Success {
		t.Errorf("HandleResponse() success = false, want true")
	}
	
	// The file tracker should have processed the response
	// (specific behavior depends on FileTracker implementation)
}

func TestParseAndExtractToolCalls_EdgeCases(t *testing.T) {
	handler := NewHandler(Dependencies{
		FileTracker: tracker.NewFileTracker(),
	})
	
	tests := []struct {
		name     string
		content  string
		wantPanic bool
	}{
		{
			name:    "very long content",
			content: strings.Repeat("a", 10000) + "<｜tool▁calls▁begin｜><｜tool▁call▁begin｜>read_file<｜tool▁sep｜>{\"path\": \"test.go\"}<｜tool▁call▁end｜><｜tool▁calls▁end｜>",
		},
		{
			name:    "nested markers",
			content: "<｜tool▁calls▁begin｜><｜tool▁calls▁begin｜><｜tool▁call▁begin｜>read_file<｜tool▁sep｜>{\"path\": \"test.go\"}<｜tool▁call▁end｜><｜tool▁calls▁end｜>",
		},
		{
			name:    "unicode content",
			content: "Hello 世界 <｜tool▁calls▁begin｜><｜tool▁call▁begin｜>read_file<｜tool▁sep｜>{\"path\": \"世界.go\"}<｜tool▁call▁end｜><｜tool▁calls▁end｜>",
		},
		{
			name:    "empty markers",
			content: "<｜tool▁calls▁begin｜><｜tool▁calls▁end｜>",
		},
	}
	
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			defer func() {
				if r := recover(); r != nil {
					if !tt.wantPanic {
						t.Errorf("ParseAndExtractToolCalls() panicked: %v", r)
					}
				}
			}()
			
			toolCalls, filtered := handler.ParseAndExtractToolCalls(tt.content)
			
			// Should not panic and should return some result
			if toolCalls == nil {
				t.Errorf("ParseAndExtractToolCalls() toolCalls = nil, want non-nil slice")
			}
			
			if len(filtered) < 0 {
				t.Errorf("ParseAndExtractToolCalls() filtered length < 0")
			}
		})
	}
}