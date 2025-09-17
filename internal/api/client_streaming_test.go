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
	"context"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

// TestStreamingResponse tests the SSE streaming response parsing
func TestStreamingResponse(t *testing.T) {
	// Create a test server that sends SSE chunks
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/event-stream")
		w.Header().Set("Cache-Control", "no-cache")
		w.Header().Set("Connection", "keep-alive")

		flusher, ok := w.(http.Flusher)
		if !ok {
			t.Fatal("ResponseWriter does not support flushing")
			return
		}

		// Send SSE chunks
		chunks := []string{
			`data: {"id":"chat1","object":"chat.completion.chunk","created":1234567890,"model":"deepseek-chat","choices":[{"index":0,"delta":{"role":"assistant","content":"Hello"},"finish_reason":null}]}`,
			`data: {"id":"chat1","object":"chat.completion.chunk","created":1234567891,"model":"deepseek-chat","choices":[{"index":0,"delta":{"content":" world"},"finish_reason":null}]}`,
			`data: {"id":"chat1","object":"chat.completion.chunk","created":1234567892,"model":"deepseek-chat","choices":[{"index":0,"delta":{"content":"!"},"finish_reason":"stop"}]}`,
			`data: [DONE]`,
		}

		for _, chunk := range chunks {
			fmt.Fprintf(w, "%s\n\n", chunk)
			flusher.Flush()
			time.Sleep(10 * time.Millisecond) // Simulate streaming delay
		}
	}))
	defer server.Close()

	// Create client with test server URL
	client := &DeepSeekClient{
		apiKey:  "test-key",
		baseURL: server.URL,
		model:   "deepseek-chat",
		httpClient: &http.Client{
			Timeout: 5 * time.Second,
		},
		maxTokens: 100,
	}

	// Test streaming request
	ctx := context.Background()
	messages := []Message{
		{Role: "user", Content: "test"},
	}

	reader, err := client.SendChatRequestStream(ctx, messages)
	if err != nil {
		t.Fatalf("Failed to start streaming: %v", err)
	}
	defer reader.Close()

	// Read all chunks
	var content strings.Builder
	for {
		chunk, err := reader.Recv()
		if err == io.EOF {
			break
		}
		if err != nil {
			t.Fatalf("Failed to receive chunk: %v", err)
		}

		if len(chunk.Choices) > 0 {
			content.WriteString(chunk.Choices[0].Delta.Content)
		}
	}

	expected := "Hello world!"
	if content.String() != expected {
		t.Errorf("Expected content %q, got %q", expected, content.String())
	}
}

// TestStreamingCancellation tests that streaming can be cancelled
func TestStreamingCancellation(t *testing.T) {
	// Create a test server that sends chunks slowly
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/event-stream")
		flusher, _ := w.(http.Flusher)

		// Send chunks slowly
		for i := 0; i < 100; i++ {
			select {
			case <-r.Context().Done():
				// Client cancelled
				return
			case <-time.After(100 * time.Millisecond):
				fmt.Fprintf(w, `data: {"id":"chat1","object":"chat.completion.chunk","created":%d,"model":"deepseek-chat","choices":[{"index":0,"delta":{"content":"chunk%d"},"finish_reason":null}]}%s`, time.Now().Unix(), i, "\n\n")
				flusher.Flush()
			}
		}
	}))
	defer server.Close()

	// Create client
	client := &DeepSeekClient{
		apiKey:  "test-key",
		baseURL: server.URL,
		model:   "deepseek-chat",
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
		},
		maxTokens: 100,
	}

	// Create cancellable context
	ctx, cancel := context.WithCancel(context.Background())
	messages := []Message{
		{Role: "user", Content: "test"},
	}

	reader, err := client.SendChatRequestStream(ctx, messages)
	if err != nil {
		t.Fatalf("Failed to start streaming: %v", err)
	}
	defer reader.Close()

	// Read a few chunks then cancel
	chunksReceived := 0
	go func() {
		time.Sleep(250 * time.Millisecond)
		cancel()
	}()

	for {
		_, err := reader.Recv()
		if err != nil {
			if err == context.Canceled {
				// Expected cancellation
				break
			}
			// Other errors are also acceptable (connection closed)
			break
		}
		chunksReceived++
		if chunksReceived > 10 {
			t.Fatal("Should have been cancelled by now")
		}
	}

	if chunksReceived == 0 {
		t.Error("Should have received at least one chunk before cancellation")
	}
}

// TestSSEParsing tests the SSE parsing logic
func TestSSEParsing(t *testing.T) {
	testCases := []struct {
		name     string
		input    string
		expected []string
	}{
		{
			name: "normal chunks",
			input: `data: {"choices":[{"delta":{"content":"test"}}]}

data: {"choices":[{"delta":{"content":" content"}}]}

data: [DONE]
`,
			expected: []string{"test", " content"},
		},
		{
			name: "with keep-alive comments",
			input: `: keep-alive

data: {"choices":[{"delta":{"content":"hello"}}]}

: keep-alive

data: [DONE]
`,
			expected: []string{"hello"},
		},
		{
			name: "empty lines and whitespace",
			input: `

data: {"choices":[{"delta":{"content":"start"}}]}


data: {"choices":[{"delta":{"content":" end"}}]}

data: [DONE]
`,
			expected: []string{"start", " end"},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// Create a test server that returns the test input
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.Header().Set("Content-Type", "text/event-stream")
				fmt.Fprint(w, tc.input)
			}))
			defer server.Close()

			client := &DeepSeekClient{
				apiKey:  "test-key",
				baseURL: server.URL,
				model:   "deepseek-chat",
				httpClient: &http.Client{
					Timeout: 5 * time.Second,
				},
				maxTokens: 100,
			}

			reader, err := client.SendChatRequestStream(context.Background(), []Message{{Role: "user", Content: "test"}})
			if err != nil {
				t.Fatalf("Failed to start streaming: %v", err)
			}
			defer reader.Close()

			var contents []string
			for {
				chunk, err := reader.Recv()
				if err == io.EOF {
					break
				}
				if err != nil {
					t.Fatalf("Failed to receive chunk: %v", err)
				}

				if len(chunk.Choices) > 0 && chunk.Choices[0].Delta.Content != "" {
					contents = append(contents, chunk.Choices[0].Delta.Content)
				}
			}

			if len(contents) != len(tc.expected) {
				t.Errorf("Expected %d chunks, got %d", len(tc.expected), len(contents))
			}

			for i, content := range contents {
				if i < len(tc.expected) && content != tc.expected[i] {
					t.Errorf("Chunk %d: expected %q, got %q", i, tc.expected[i], content)
				}
			}
		})
	}
}

// TestStreamingErrorHandling tests error handling during streaming
func TestStreamingErrorHandling(t *testing.T) {
	testCases := []struct {
		name           string
		statusCode     int
		responseBody   string
		expectedErrMsg string
	}{
		{
			name:           "unauthorized",
			statusCode:     401,
			responseBody:   `{"error": "Invalid API key"}`,
			expectedErrMsg: "unauthorized",
		},
		{
			name:           "rate limit",
			statusCode:     429,
			responseBody:   `{"error": "Rate limit exceeded"}`,
			expectedErrMsg: "rate limited",
		},
		{
			name:           "server error",
			statusCode:     500,
			responseBody:   `{"error": "Internal server error"}`,
			expectedErrMsg: "server error",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(tc.statusCode)
				fmt.Fprint(w, tc.responseBody)
			}))
			defer server.Close()

			client := &DeepSeekClient{
				apiKey:  "test-key",
				baseURL: server.URL,
				model:   "deepseek-chat",
				httpClient: &http.Client{
					Timeout: 5 * time.Second,
				},
				maxTokens: 100,
			}

			_, err := client.SendChatRequestStream(context.Background(), []Message{{Role: "user", Content: "test"}})
			if err == nil {
				t.Fatal("Expected error but got none")
			}

			if apiErr, ok := err.(APIError); ok {
				if !strings.Contains(apiErr.Message, tc.expectedErrMsg) {
					t.Errorf("Expected error message to contain %q, got %q", tc.expectedErrMsg, apiErr.Message)
				}
			} else {
				t.Errorf("Expected APIError, got %T: %v", err, err)
			}
		})
	}
}