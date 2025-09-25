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
	"strings"

	"github.com/antenore/deecli/internal/files"
)

// Service provides high-level AI operations using the underlying client
type Service struct {
	client *DeepSeekClient
}

// NewService creates a new AI service with the provided client
func NewService(client *DeepSeekClient) *Service {
	return &Service{client: client}
}

// ChatAboutCode sends a chat request about code to the AI
func (s *Service) ChatAboutCode(code, userMessage string) (string, error) {
    messages := []Message{
        {
            Role: "system",
            Content: `You are an expert software engineer and code reviewer.
You help developers understand, improve, and debug their code.
Provide clear, actionable advice and explanations.

CRITICAL: If tool results are already present in the conversation history, you MUST use those results to answer. Do not ignore tool outputs or hallucinate different information. Always base your response on the actual tool results provided.`,
        },
    }

	var userContent string
	if code != "" {
		userContent = fmt.Sprintf("%s\n\n%s", code, userMessage)
	} else {
		userContent = userMessage
	}

	messages = append(messages, Message{
		Role:    "user",
		Content: userContent,
	})

	return s.client.SendChatRequest(context.Background(), messages)
}

// ChatWithHistory sends a chat request with conversation history and code context
func (s *Service) ChatWithHistory(conversationHistory []Message, contextPrompt, userMessage string) (string, error) {
	return s.ChatWithHistoryContext(context.Background(), conversationHistory, contextPrompt, userMessage)
}

// ChatWithHistoryContext sends a chat request with conversation history and code context, with cancellation support
func (s *Service) ChatWithHistoryContext(ctx context.Context, conversationHistory []Message, contextPrompt, userMessage string) (string, error) {
	messages := []Message{
		{
			Role: "system",
			Content: `You are an expert software engineer and code reviewer.
You help developers understand, improve, and debug their code.
Provide clear, actionable advice and explanations.`,
		},
	}

	messages = append(messages, conversationHistory...)

	// Add file context as separate system message for better conversation structure
	if contextPrompt != "" {
		messages = append(messages, Message{
			Role:    "system",
			Content: fmt.Sprintf("Files Context:\n%s", contextPrompt),
		})
	}

    // Add user message separately (only if non-empty)
    if strings.TrimSpace(userMessage) != "" {
        messages = append(messages, Message{
            Role:    "user",
            Content: userMessage,
        })
    }

	return s.client.SendChatRequest(ctx, messages)
}

// ChatWithHistoryContextAndTools sends a chat request with tools, conversation history and code context
func (s *Service) ChatWithHistoryContextAndTools(ctx context.Context, conversationHistory []Message, contextPrompt, userMessage string, tools []Tool) (*ChatResponse, error) {
	return s.ChatWithHistoryContextAndToolsWithChoice(ctx, conversationHistory, contextPrompt, userMessage, tools, "auto")
}

// ChatWithHistoryContextAndToolsWithChoice sends a chat request with tools, conversation history, code context and specified tool choice
func (s *Service) ChatWithHistoryContextAndToolsWithChoice(ctx context.Context, conversationHistory []Message, contextPrompt, userMessage string, tools []Tool, toolChoice string) (*ChatResponse, error) {
    messages := []Message{
        {
            Role: "system",
            Content: `You are an expert software engineer and code reviewer.
You help developers understand, improve, and debug their code.
Provide clear, actionable advice and explanations.
You have access to tools to help you gather information about the project.

CRITICAL: If tool results are already present in the conversation history, you MUST use those results to answer. Do not ignore tool outputs or hallucinate different information. Always base your response on the actual tool results provided.`,
        },
    }

	messages = append(messages, conversationHistory...)

	// Add file context as separate system message for better conversation structure
	if contextPrompt != "" {
		messages = append(messages, Message{
			Role:    "system",
			Content: fmt.Sprintf("Files Context:\n%s", contextPrompt),
		})
	}

    // Add user message separately (only if non-empty)
    if strings.TrimSpace(userMessage) != "" {
        messages = append(messages, Message{
            Role:    "user",
            Content: userMessage,
        })
    }

	return s.client.SendChatRequestWithToolsAndChoice(ctx, messages, tools, toolChoice)
}

// AnalyzeCode analyzes code and provides suggestions
func (s *Service) AnalyzeCode(code, filename string) (string, error) {
	messages := []Message{
		{
			Role: "system",
			Content: `You are an expert code analyzer. Analyze the provided code and give:
1. Code quality assessment
2. Potential issues or bugs
3. Performance considerations
4. Best practice recommendations
5. Security concerns if any`,
		},
		{
			Role:    "user",
			Content: fmt.Sprintf("Please analyze this code from %s:\n\n```\n%s\n```", filename, code),
		},
	}

	return s.client.SendChatRequest(context.Background(), messages)
}

// ImproveCode suggests improvements for the given code
func (s *Service) ImproveCode(code, filename string) (string, error) {
	messages := []Message{
		{
			Role: "system",
			Content: `You are an expert software engineer. Suggest improvements for the provided code:
1. Code optimization opportunities
2. Better algorithms or data structures
3. Improved readability and maintainability
4. Modern language features that could be used
5. Error handling improvements`,
		},
		{
			Role:    "user",
			Content: fmt.Sprintf("Please suggest improvements for this code from %s:\n\n```\n%s\n```", filename, code),
		},
	}

	return s.client.SendChatRequest(context.Background(), messages)
}

// ExplainCode explains what the code does
func (s *Service) ExplainCode(code, filename string) (string, error) {
	messages := []Message{
		{
			Role: "system",
			Content: `You are an expert code explainer. Explain the provided code clearly:
1. What the code does overall
2. Key functions and their purposes
3. Important algorithms or logic
4. Dependencies and external interactions
5. Use cases and examples`,
		},
		{
			Role:    "user",
			Content: fmt.Sprintf("Please explain this code from %s:\n\n```\n%s\n```", filename, code),
		},
	}

	return s.client.SendChatRequest(context.Background(), messages)
}

// GenerateEditSuggestions analyzes conversation context and suggests which files to edit
func (s *Service) GenerateEditSuggestions(ctx context.Context, conversationHistory []Message, fileContext *files.FileContext) (string, error) {
	var contextBuilder strings.Builder
	contextBuilder.WriteString("=== LOADED FILES ===\n")

	for _, file := range fileContext.Files {
		contextBuilder.WriteString(fmt.Sprintf("File: %s (%s)\n", file.RelPath, file.Language))
		contextBuilder.WriteString(fmt.Sprintf("Size: %d bytes\n", file.Size))
		contextBuilder.WriteString("Content preview:\n")
		contextBuilder.WriteString(file.Content[:min(len(file.Content), 500)])
		if len(file.Content) > 500 {
			contextBuilder.WriteString("...")
		}
		contextBuilder.WriteString("\n\n")
	}

	contextBuilder.WriteString("\n=== CONVERSATION HISTORY ===\n")
	for _, msg := range conversationHistory {
		contextBuilder.WriteString(fmt.Sprintf("%s: %s\n", msg.Role, msg.Content))
	}

	messages := []Message{
		{
			Role: "system",
			Content: `You are an AI assistant helping identify which files need to be edited based on a conversation.

Analyze the conversation history and loaded files, then suggest:
1. Which specific files should be modified
2. What type of changes are needed for each file
3. Priority order for making the changes
4. Brief explanation of why each file needs changes

Format your response as:
## Files to Edit

### High Priority
- **filename.ext**: Brief description of changes needed

### Medium Priority
- **filename.ext**: Brief description of changes needed

### Low Priority
- **filename.ext**: Brief description of changes needed

## Recommendations
Brief explanation of the suggested approach and order.`,
		},
		{
			Role:    "user",
			Content: fmt.Sprintf("Based on this context, suggest which files should be edited:\n\n%s", contextBuilder.String()),
		},
	}

	return s.client.SendChatRequest(ctx, messages)
}

// min helper function
func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

// ChatAboutCodeStream sends a streaming chat request about code to the AI
func (s *Service) ChatAboutCodeStream(ctx context.Context, code, userMessage string) (StreamReader, error) {
	messages := []Message{
		{
			Role: "system",
			Content: `You are an expert software engineer and code reviewer.
You help developers understand, improve, and debug their code.
Provide clear, actionable advice and explanations.`,
		},
	}

	var userContent string
	if code != "" {
		userContent = fmt.Sprintf("%s\n\n%s", code, userMessage)
	} else {
		userContent = userMessage
	}

	messages = append(messages, Message{
		Role:    "user",
		Content: userContent,
	})

	return s.client.SendChatRequestStream(ctx, messages)
}

// ChatWithHistoryContextStream sends a streaming chat request with conversation history and code context
func (s *Service) ChatWithHistoryContextStream(ctx context.Context, conversationHistory []Message, contextPrompt, userMessage string) (StreamReader, error) {
    messages := []Message{
        {
            Role: "system",
            Content: `You are an expert software engineer and code reviewer.
You help developers understand, improve, and debug their code.
Provide clear, actionable advice and explanations.

CRITICAL: If tool results are already present in the conversation history, you MUST use those results to answer. Do not ignore tool outputs or hallucinate different information. Always base your response on the actual tool results provided.`,
        },
    }

	messages = append(messages, conversationHistory...)

	// Add file context as separate system message for better conversation structure
	if contextPrompt != "" {
		messages = append(messages, Message{
			Role:    "system",
			Content: fmt.Sprintf("Files Context:\n%s", contextPrompt),
		})
	}

    // Add user message separately (only if non-empty)
    if strings.TrimSpace(userMessage) != "" {
        messages = append(messages, Message{
            Role:    "user",
            Content: userMessage,
        })
    }

	return s.client.SendChatRequestStream(ctx, messages)
}

// ChatWithHistoryContextStreamWithTools sends a streaming chat request with tools, conversation history and code context
func (s *Service) ChatWithHistoryContextStreamWithTools(ctx context.Context, conversationHistory []Message, contextPrompt, userMessage string, tools []Tool) (StreamReader, error) {
	return s.ChatWithHistoryContextStreamWithToolsAndChoice(ctx, conversationHistory, contextPrompt, userMessage, tools, "auto")
}

// ChatWithHistoryContextStreamWithToolsAndChoice sends a streaming chat request with tools, conversation history, code context and specified tool choice
func (s *Service) ChatWithHistoryContextStreamWithToolsAndChoice(ctx context.Context, conversationHistory []Message, contextPrompt, userMessage string, tools []Tool, toolChoice string) (StreamReader, error) {
    messages := []Message{
        {
            Role: "system",
            Content: `You are an expert software engineer and code reviewer.
You help developers understand, improve, and debug their code.
Provide clear, actionable advice and explanations.
You have access to tools to help you gather information about the project.

CRITICAL: If tool results are already present in the conversation history, you MUST use those results to answer. Do not ignore tool outputs or hallucinate different information. Always base your response on the actual tool results provided.`,
        },
    }

	messages = append(messages, conversationHistory...)

	// Add file context as separate system message for better conversation structure
	if contextPrompt != "" {
		messages = append(messages, Message{
			Role:    "system",
			Content: fmt.Sprintf("Files Context:\n%s", contextPrompt),
		})
	}

    // Add user message separately (only if non-empty)
    if strings.TrimSpace(userMessage) != "" {
        messages = append(messages, Message{
            Role:    "user",
            Content: userMessage,
        })
    }

	return s.client.SendChatRequestStreamWithToolsAndChoice(ctx, messages, tools, toolChoice)
}
