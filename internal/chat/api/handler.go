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

	"github.com/antenore/deecli/internal/api"
	"github.com/antenore/deecli/internal/chat/tracker"
	"github.com/antenore/deecli/internal/debug"
	"github.com/antenore/deecli/internal/files"
)

// Handler manages API response processing, including DeepSeek markup parsing
type Handler struct {
	fileTracker *tracker.FileTracker
}

// Dependencies contains the dependencies needed by the API handler
type Dependencies struct {
	FileTracker *tracker.FileTracker
}

// NewHandler creates a new API response handler
func NewHandler(deps Dependencies) *Handler {
	return &Handler{
		fileTracker: deps.FileTracker,
	}
}

// APIResponseResult represents the result of processing an API response
type APIResponseResult struct {
	Success          bool
	ErrorMessage     string
	AssistantContent string
	ToolCalls        []api.ToolCall
	ShouldSuppress   bool // Whether tool parsing was suppressed
}

// HandleResponse processes API responses for both streaming and non-streaming calls
func (h *Handler) HandleResponse(response string, err error, shouldSuppress bool, fileContext *files.FileContext) APIResponseResult {
	if err != nil {
		return h.handleError(err)
	}

	if shouldSuppress {
		return h.handleSuppressedResponse(response, fileContext)
	}

	return h.handleNormalResponse(response, fileContext)
}

// handleError processes API errors
func (h *Handler) handleError(err error) APIResponseResult {
	// Check if it's an enhanced APIError
	if apiErr, ok := err.(api.APIError); ok {
		// Don't show cancellation as error
		if apiErr.Message == "request cancelled by user" {
			return APIResponseResult{Success: true}
		}
		
		errorMsg := fmt.Sprintf("❌ %s", apiErr.UserMessage)
		if apiErr.StatusCode > 0 {
			errorMsg += fmt.Sprintf(" (HTTP %d)", apiErr.StatusCode)
		}
		
		return APIResponseResult{
			Success:      false,
			ErrorMessage: errorMsg,
		}
	}
	
	// Fallback to generic error message
	return APIResponseResult{
		Success:      false,
		ErrorMessage: fmt.Sprintf("❌ Error: %v", err),
	}
}

// handleSuppressedResponse processes responses where tool parsing should be suppressed
func (h *Handler) handleSuppressedResponse(response string, fileContext *files.FileContext) APIResponseResult {
	// Strip markers for display but do not re-run tools
	_, filtered := h.ParseAndExtractToolCalls(response)
	debug.Printf("[DEBUG] Suppressing tool call parsing for this response (tool_choice=none follow-up)\n")

	// Additional filtering for DeepSeek responses that contain JSON-like tool arguments
	// when tool_choice="none" is used but the model still tries to call tools
	filtered = h.cleanUpToolLikeContent(filtered)

	// Track files mentioned in the response
	if h.fileTracker != nil && fileContext != nil {
		h.fileTracker.ExtractFilesFromResponseWithContext(filtered, fileContext.Files)
	}

	return APIResponseResult{
		Success:          true,
		AssistantContent: filtered,
		ShouldSuppress:   true,
	}
}

// handleNormalResponse processes normal API responses with tool call detection
func (h *Handler) handleNormalResponse(response string, fileContext *files.FileContext) APIResponseResult {
	// Check for tool calls in non-streaming response and parse them
	toolCalls, filteredResponse := h.ParseAndExtractToolCalls(response)
	
	if len(toolCalls) > 0 {
		return APIResponseResult{
			Success:   true,
			ToolCalls: toolCalls,
		}
	}
	
	// Track files mentioned in the AI response
	if h.fileTracker != nil && fileContext != nil {
		h.fileTracker.ExtractFilesFromResponseWithContext(filteredResponse, fileContext.Files)
	}
	
	return APIResponseResult{
		Success:          true,
		AssistantContent: filteredResponse,
	}
}

// cleanUpToolLikeContent removes JSON-like content that appears to be malformed tool calls
// This helps when DeepSeek returns tool arguments as plain text instead of proper tool calls
func (h *Handler) cleanUpToolLikeContent(content string) string {
	// Look for standalone JSON objects that look like tool arguments
	lines := strings.Split(content, "\n")
	var filteredLines []string

	for _, line := range lines {
		trimmed := strings.TrimSpace(line)

		// Skip empty lines - preserve them
		if trimmed == "" {
			filteredLines = append(filteredLines, line)
			continue
		}

		// Only filter out lines that are purely JSON objects AND look like tool arguments
		if strings.HasPrefix(trimmed, "{") && strings.HasSuffix(trimmed, "}") {
			// Be more specific about what constitutes a tool argument
			if (trimmed == "{}") ||
			   (strings.Contains(trimmed, "\"path\":") && strings.Count(trimmed, ":") <= 2) ||
			   (strings.Contains(trimmed, "\"recursive\":") && strings.Count(trimmed, ":") <= 2) ||
			   (strings.Contains(trimmed, "\"pattern\":") && strings.Count(trimmed, ":") <= 2) {
				debug.Printf("[DEBUG] Filtering out malformed tool call: %s\n", trimmed)
				continue // Skip this line
			}
		}

		filteredLines = append(filteredLines, line)
	}

	result := strings.Join(filteredLines, "\n")

	// If we filtered out too much and left mostly empty content,
	// provide a helpful message instead of showing nothing
	trimmedResult := strings.TrimSpace(result)
	if len(trimmedResult) < 20 {
		return "Tool execution completed. You can continue the conversation."
	}

	return result
}

// ParseAndExtractToolCalls parses DeepSeek's tool call markup and extracts proper tool calls
func (h *Handler) ParseAndExtractToolCalls(content string) ([]api.ToolCall, string) {
	var toolCalls []api.ToolCall

	// Look for DeepSeek's tool call patterns
	if !strings.Contains(content, "<｜tool▁calls▁begin｜>") {
		return toolCalls, content
	}

	debug.Printf("[DEBUG] Parsing tool calls from non-streaming response: %q\n", content)

	filtered := content
	callID := 1

	// Find all tool call blocks
	for {
		start := strings.Index(filtered, "<｜tool▁calls▁begin｜>")
		if start == -1 {
			break
		}

		end := strings.Index(filtered[start:], "<｜tool▁calls▁end｜>")
		if end == -1 {
			// Incomplete tool call block, remove and break
			filtered = filtered[:start]
			break
		}

		end += start + len("<｜tool▁calls▁end｜>")
		toolBlock := filtered[start:end]

		// Extract individual tool calls within the block
		toolStart := strings.Index(toolBlock, "<｜tool▁call▁begin｜>")
		for toolStart != -1 {
			toolEnd := strings.Index(toolBlock[toolStart:], "<｜tool▁call▁end｜>")
			if toolEnd == -1 {
				break
			}

			toolEnd += toolStart
			individualCall := toolBlock[toolStart+len("<｜tool▁call▁begin｜>"):toolEnd]

			// Split by separator to get function name and arguments
			sepIndex := strings.Index(individualCall, "<｜tool▁sep｜>")
			if sepIndex != -1 {
				functionName := strings.TrimSpace(individualCall[:sepIndex])
				argsJSON := strings.TrimSpace(individualCall[sepIndex+len("<｜tool▁sep｜>"):])

				if functionName != "" && argsJSON != "" {
					toolCall := api.ToolCall{
						ID:   fmt.Sprintf("call_%d", callID),
						Type: "function",
					}
					toolCall.Function.Name = functionName
					toolCall.Function.Arguments = argsJSON
					toolCalls = append(toolCalls, toolCall)
					callID++

					debug.Printf("[DEBUG] Extracted tool call: %s with args: %s\n", functionName, argsJSON)
				}
			}

			// Find next tool call in this block
			toolStart = strings.Index(toolBlock[toolEnd+len("<｜tool▁call▁end｜>"):], "<｜tool▁call▁begin｜>")
			if toolStart != -1 {
				toolStart += toolEnd + len("<｜tool▁call▁end｜>")
			}
		}

		// Remove the entire tool call block from the content
		filtered = filtered[:start] + filtered[end:]
	}

	return toolCalls, strings.TrimSpace(filtered)
}

// ToolCallsDetectedMsg represents detected tool calls in API response
type ToolCallsDetectedMsg struct {
	ToolCalls []api.ToolCall
}

// ErrorMsg represents an API error
type ErrorMsg struct {
	Error string
}

// AssistantResponseMsg represents a normal assistant response
type AssistantResponseMsg struct {
	Content string
}