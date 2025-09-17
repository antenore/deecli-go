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
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"math"
	"net/http"
	"strings"
	"sync"
	"time"
)

// DeepSeekClient handles low-level HTTP communication with DeepSeek API
type DeepSeekClient struct {
	apiKey      string
	baseURL     string
	model       string
	temperature float64
	maxTokens   int
	httpClient  *http.Client
	maxRetries  int
	baseDelay   time.Duration

	// Connection management
	lastActivity time.Time
	activityMu   sync.Mutex
	transport    *http.Transport
	ctx          context.Context
	cancel       context.CancelFunc
}

// NewDeepSeekClient creates a new DeepSeek API client
func NewDeepSeekClient(apiKey, model string, temperature float64, maxTokens int) *DeepSeekClient {
	transport := &http.Transport{
		MaxIdleConns:        10,               // Maximum idle connections to keep
		MaxIdleConnsPerHost: 10,               // Maximum idle connections per host
		IdleConnTimeout:     90 * time.Second, // How long to keep idle connections
		TLSHandshakeTimeout: 10 * time.Second,
	}

	ctx, cancel := context.WithCancel(context.Background())

	client := &DeepSeekClient{
		apiKey:      apiKey,
		baseURL:     "https://api.deepseek.com",
		model:       model,
		temperature: temperature,
		maxTokens:   maxTokens,
		httpClient: &http.Client{
			Timeout:   120 * time.Second,
			Transport: transport,
		},
		maxRetries:   3,
		baseDelay:    time.Second,
		lastActivity: time.Now(),
		transport:    transport,
		ctx:          ctx,
		cancel:       cancel,
	}

	// Start connection manager goroutine
	go client.manageConnection()

	return client
}

// SendChatRequest sends a chat completion request
func (client *DeepSeekClient) SendChatRequest(ctx context.Context, messages []Message) (string, error) {
	return client.sendChatRequestWithRetryContext(ctx, messages)
}

// sendChatRequestWithRetryContext sends a chat request with retry logic and context cancellation
func (client *DeepSeekClient) sendChatRequestWithRetryContext(ctx context.Context, messages []Message) (string, error) {
	var lastErr error

	for attempt := 0; attempt <= client.maxRetries; attempt++ {
		if attempt > 0 {
			// Exponential backoff with jitter
			delay := time.Duration(float64(client.baseDelay) * math.Pow(2, float64(attempt-1)))
			if delay > 30*time.Second {
				delay = 30 * time.Second
			}
			time.Sleep(delay)
		}

		// Check if context was cancelled before making request
		if ctx.Err() == context.Canceled {
			return "", APIError{
				StatusCode:  0,
				Message:     "request cancelled by user",
				Retryable:   false,
				UserMessage: "Request cancelled",
			}
		}

		result, err := client.sendSingleRequestWithContext(ctx, messages)
		if err == nil {
			return result, nil
		}

		lastErr = err

		// Check if error is retryable
		if apiErr, ok := err.(APIError); ok && !apiErr.Retryable {
			return "", apiErr
		}

		// Don't retry on the last attempt
		if attempt == client.maxRetries {
			break
		}
	}

	return "", fmt.Errorf("failed after %d attempts: %w", client.maxRetries+1, lastErr)
}

// sendSingleRequestWithContext makes a single API request with context support for cancellation
func (client *DeepSeekClient) sendSingleRequestWithContext(ctx context.Context, messages []Message) (string, error) {
	// Update activity timestamp
	client.updateActivity()

	// DeepSeek reasoner model doesn't support temperature parameter
	request := ChatRequest{
		Model:     client.model,
		Messages:  messages,
		MaxTokens: client.maxTokens,
	}

	// Only add temperature for non-reasoner models
	if client.model != "deepseek-reasoner" {
		request.Temperature = client.temperature
	}

	jsonData, err := json.Marshal(request)
	if err != nil {
		return "", APIError{
			StatusCode:  0,
			Message:     fmt.Sprintf("failed to marshal request: %v", err),
			Retryable:   false,
			UserMessage: "Request formatting error. Please try again.",
		}
	}

	req, err := http.NewRequestWithContext(ctx, "POST", client.baseURL+"/chat/completions", bytes.NewBuffer(jsonData))
	if err != nil {
		return "", APIError{
			StatusCode:  0,
			Message:     fmt.Sprintf("failed to create request: %v", err),
			Retryable:   false,
			UserMessage: "Request creation error. Please try again.",
		}
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+client.apiKey)

	resp, err := client.httpClient.Do(req)
	if err != nil {
		// Check if context was cancelled
		if ctx.Err() == context.Canceled {
			return "", APIError{
				StatusCode:  0,
				Message:     "request cancelled by user",
				Retryable:   false,
				UserMessage: "Request cancelled",
			}
		}
		// Network errors are generally retryable
		return "", APIError{
			StatusCode:  0,
			Message:     fmt.Sprintf("request failed: %v", err),
			Retryable:   true,
			UserMessage: "Network error. Retrying...",
		}
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", APIError{
			StatusCode:  resp.StatusCode,
			Message:     fmt.Sprintf("failed to read response: %v", err),
			Retryable:   true,
			UserMessage: "Error reading response. Retrying...",
		}
	}

	if resp.StatusCode != http.StatusOK {
		return "", client.handleHTTPError(resp.StatusCode, body)
	}

	var response ChatResponse
	if err := json.Unmarshal(body, &response); err != nil {
		return "", APIError{
			StatusCode:  resp.StatusCode,
			Message:     fmt.Sprintf("failed to unmarshal response: %v", err),
			Retryable:   true,
			UserMessage: "Error parsing response. Retrying...",
		}
	}

	if len(response.Choices) == 0 {
		return "", APIError{
			StatusCode:  resp.StatusCode,
			Message:     "no response choices received",
			Retryable:   true,
			UserMessage: "Empty response received. Retrying...",
		}
	}

	return response.Choices[0].Message.Content, nil
}

// handleHTTPError provides user-friendly error messages for HTTP errors
func (client *DeepSeekClient) handleHTTPError(statusCode int, body []byte) APIError {
	bodyStr := string(body)

	switch statusCode {
	case 400:
		return APIError{
			StatusCode:  statusCode,
			Message:     fmt.Sprintf("bad request: %s", bodyStr),
			Retryable:   false,
			UserMessage: "Invalid request. Please check your input and try again.",
		}
	case 401:
		return APIError{
			StatusCode:  statusCode,
			Message:     fmt.Sprintf("unauthorized: %s", bodyStr),
			Retryable:   false,
			UserMessage: "API key is invalid or missing. Please set DEEPSEEK_API_KEY environment variable.",
		}
	case 403:
		return APIError{
			StatusCode:  statusCode,
			Message:     fmt.Sprintf("forbidden: %s", bodyStr),
			Retryable:   false,
			UserMessage: "Access denied. Please check your API key permissions.",
		}
	case 429:
		return APIError{
			StatusCode:  statusCode,
			Message:     fmt.Sprintf("rate limited: %s", bodyStr),
			Retryable:   true,
			UserMessage: "Rate limit exceeded. Retrying with backoff...",
		}
	case 500, 502, 503, 504:
		return APIError{
			StatusCode:  statusCode,
			Message:     fmt.Sprintf("server error (%d): %s", statusCode, bodyStr),
			Retryable:   true,
			UserMessage: "Server error. Retrying...",
		}
	default:
		return APIError{
			StatusCode:  statusCode,
			Message:     fmt.Sprintf("API error (%d): %s", statusCode, bodyStr),
			Retryable:   statusCode >= 500,
			UserMessage: fmt.Sprintf("API error (status %d). Please try again.", statusCode),
		}
	}
}

// WarmUp performs an initial connection to pre-establish TLS handshake
func (client *DeepSeekClient) WarmUp() error {
	// Send a minimal request to establish connection
	warmupMsg := []Message{
		{Role: "system", Content: "ping"},
		{Role: "user", Content: "pong"},
	}

	// Store original values
	origMaxTokens := client.maxTokens
	client.maxTokens = 1 // Minimal response

	// Perform warm-up request
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	_, err := client.sendSingleRequestWithContext(ctx, warmupMsg)

	// Restore original values
	client.maxTokens = origMaxTokens

	if err != nil {
		// Don't fail if warm-up fails, just log it
		return fmt.Errorf("connection warm-up failed: %w", err)
	}

	return nil
}

// manageConnection monitors activity and closes idle connections
func (client *DeepSeekClient) manageConnection() {
	ticker := time.NewTicker(30 * time.Second) // Check every 30 seconds
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			client.activityMu.Lock()
			inactiveTime := time.Since(client.lastActivity)
			client.activityMu.Unlock()

			// Close connections after 10 minutes of inactivity
			if inactiveTime > 10*time.Minute {
				client.transport.CloseIdleConnections()
			}

		case <-client.ctx.Done():
			// Clean shutdown
			client.transport.CloseIdleConnections()
			return
		}
	}
}

// updateActivity updates the last activity timestamp
func (client *DeepSeekClient) updateActivity() {
	client.activityMu.Lock()
	client.lastActivity = time.Now()
	client.activityMu.Unlock()
}

// Close gracefully shuts down the client and closes connections
func (client *DeepSeekClient) Close() {
	if client.cancel != nil {
		client.cancel()
	}
	if client.transport != nil {
		client.transport.CloseIdleConnections()
	}
}

// deepSeekStreamReader implements StreamReader interface
type deepSeekStreamReader struct {
	reader  *bufio.Reader
	resp    *http.Response
	ctx     context.Context
}

// Recv reads the next chunk from the stream
func (s *deepSeekStreamReader) Recv() (ChatCompletionChunk, error) {
	for {
		// Check context cancellation
		select {
		case <-s.ctx.Done():
			return ChatCompletionChunk{}, s.ctx.Err()
		default:
		}

		line, err := s.reader.ReadBytes('\n')
		if err != nil {
			if err == io.EOF {
				return ChatCompletionChunk{}, io.EOF
			}
			return ChatCompletionChunk{}, err
		}

		lineStr := strings.TrimSpace(string(line))

		// Skip empty lines
		if lineStr == "" {
			continue
		}

		// Skip SSE comments (keep-alive)
		if strings.HasPrefix(lineStr, ":") {
			continue
		}

		// Check for data prefix
		if !strings.HasPrefix(lineStr, "data: ") {
			continue
		}

		// Extract data
		data := strings.TrimPrefix(lineStr, "data: ")

		// Check for stream end
		if data == "[DONE]" {
			return ChatCompletionChunk{}, io.EOF
		}

		// Parse JSON chunk
		var chunk ChatCompletionChunk
		if err := json.Unmarshal([]byte(data), &chunk); err != nil {
			// Skip malformed chunks
			continue
		}

		return chunk, nil
	}
}

// Close closes the stream reader
func (s *deepSeekStreamReader) Close() error {
	if s.resp != nil && s.resp.Body != nil {
		return s.resp.Body.Close()
	}
	return nil
}

// SendChatRequestStream sends a streaming chat completion request
func (client *DeepSeekClient) SendChatRequestStream(ctx context.Context, messages []Message) (StreamReader, error) {
	// Update activity timestamp
	client.updateActivity()

	// Create streaming request
	request := StreamingChatRequest{
		Model:     client.model,
		Messages:  messages,
		MaxTokens: client.maxTokens,
		Stream:    true,
	}

	// Only add temperature for non-reasoner models
	if client.model != "deepseek-reasoner" {
		request.Temperature = client.temperature
	}

	jsonData, err := json.Marshal(request)
	if err != nil {
		return nil, APIError{
			StatusCode:  0,
			Message:     fmt.Sprintf("failed to marshal request: %v", err),
			Retryable:   false,
			UserMessage: "Request formatting error. Please try again.",
		}
	}

	req, err := http.NewRequestWithContext(ctx, "POST", client.baseURL+"/chat/completions", bytes.NewBuffer(jsonData))
	if err != nil {
		return nil, APIError{
			StatusCode:  0,
			Message:     fmt.Sprintf("failed to create request: %v", err),
			Retryable:   false,
			UserMessage: "Request creation error. Please try again.",
		}
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+client.apiKey)
	req.Header.Set("Accept", "text/event-stream")
	req.Header.Set("Cache-Control", "no-cache")
	req.Header.Set("Connection", "keep-alive")

	resp, err := client.httpClient.Do(req)
	if err != nil {
		// Check if context was cancelled
		if ctx.Err() == context.Canceled {
			return nil, APIError{
				StatusCode:  0,
				Message:     "request cancelled by user",
				Retryable:   false,
				UserMessage: "Request cancelled",
			}
		}
		return nil, APIError{
			StatusCode:  0,
			Message:     fmt.Sprintf("request failed: %v", err),
			Retryable:   true,
			UserMessage: "Network error. Please try again.",
		}
	}

	if resp.StatusCode != http.StatusOK {
		defer resp.Body.Close()
		body, _ := io.ReadAll(resp.Body)
		return nil, client.handleHTTPError(resp.StatusCode, body)
	}

	// Create stream reader
	reader := &deepSeekStreamReader{
		reader: bufio.NewReader(resp.Body),
		resp:   resp,
		ctx:    ctx,
	}

	return reader, nil
}
