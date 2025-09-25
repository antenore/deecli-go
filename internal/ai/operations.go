package ai

import (
    "context"
    "fmt"
    "io"
    "os"
    "strings"
    "time"

    "github.com/antenore/deecli/internal/api"
    "github.com/antenore/deecli/internal/config"
    "github.com/antenore/deecli/internal/files"

    tea "github.com/charmbracelet/bubbletea"
)

// EstimateTokens provides a rough estimation of token count from text
// Uses the common approximation of 1 token ≈ 4 characters for most models
func EstimateTokens(text string) int {
	return len(text) / 4
}

// APIResponseMsg for async API calls
type APIResponseMsg struct {
	Response string
	Err      error
}

// ToolCallsResponseMsg for API calls that request tool execution
type ToolCallsResponseMsg struct {
	ToolCalls []api.ToolCall
	Response  *api.ChatResponse
}

// StreamChunkMsg represents a chunk of streaming response
type StreamChunkMsg struct {
	Content string
	IsDone  bool
	Err     error
}

// StreamCompleteMsg signals the end of a streaming response
type StreamCompleteMsg struct {
	TotalContent string
	Err          error
}

// Operations handles AI-related operations
type Operations struct {
	apiClient     *api.Service
	apiMessages   []api.Message
	apiCancel     context.CancelFunc
	fileContext   *files.FileContext
	configManager *config.Manager
	availableTools []api.Tool  // Available function calling tools
}

// NewOperations creates a new Operations instance
func NewOperations(apiClient *api.Service, fileContext *files.FileContext, configManager *config.Manager) *Operations {
	return &Operations{
		apiClient:     apiClient,
		apiMessages:   []api.Message{},
		fileContext:   fileContext,
		configManager: configManager,
	}
}

// GetAPIMessages returns the current API messages
func (o *Operations) GetAPIMessages() []api.Message {
	return o.apiMessages
}

// SetAPIMessages sets the API messages
func (o *Operations) SetAPIMessages(messages []api.Message) {
	o.apiMessages = messages
}

// GetAPICancel returns the current API cancel function
func (o *Operations) GetAPICancel() context.CancelFunc {
	return o.apiCancel
}

// SetAPICancel sets the API cancel function
func (o *Operations) SetAPICancel(cancel context.CancelFunc) {
	o.apiCancel = cancel
}

// SetAvailableTools sets the available tools for function calling
func (o *Operations) SetAvailableTools(tools []api.Tool) {
	o.availableTools = tools
}

// GetAvailableTools returns the available tools
func (o *Operations) GetAvailableTools() []api.Tool {
	return o.availableTools
}

// CallAPI makes an API call with context and user input
func (o *Operations) CallAPI(contextPrompt, userInput string) tea.Cmd {
	// Check context size limit before making API call
	contextSize := len(contextPrompt) + len(userInput)
	contextTokens := EstimateTokens(contextPrompt + userInput)

	cfg := o.configManager.Get()
	maxContextSize := cfg.MaxContextSize
	if maxContextSize == 0 {
		maxContextSize = 100000 // Default 100KB if not configured
	}
	maxContextTokens := EstimateTokens(fmt.Sprintf("%*s", maxContextSize, ""))

    // Optional debug output (enable with DEECLI_DEBUG=1)
    if os.Getenv("DEECLI_DEBUG") == "1" {
        fmt.Printf("Debug: Context size check - chars: %d (limit: %d), tokens: %d (limit: %d)\n",
            contextSize, maxContextSize, contextTokens, maxContextTokens)
    }

	// Check both character and token limits for safety
	if contextSize > maxContextSize || contextTokens > maxContextTokens {
		return func() tea.Msg {
			// Get helpful info about loaded files
			fileInfo := o.fileContext.GetInfo()
			return APIResponseMsg{
				Err: fmt.Errorf("context too large - chars: %d/%d, tokens: %d/%d\n\n%s\n\nTry loading fewer files or unload large files with /clear",
					contextSize, maxContextSize, contextTokens, maxContextTokens, fileInfo),
			}
		}
	}

    // Create a context with model-aware timeout
    timeout := 180 * time.Second
    if o.configManager != nil {
        cfg := o.configManager.Get()
        if cfg != nil && strings.EqualFold(cfg.Model, "deepseek-reasoner") {
            // Reasoner can take longer, allow more time
            timeout = 300 * time.Second
        }
    }
    ctx, cancel := context.WithTimeout(context.Background(), timeout)

	// Store the cancel function so we can use it later
	o.apiCancel = cancel

	return func() tea.Msg {
		// Check if we have tools available
		if len(o.availableTools) > 0 {
			// Use tools-enabled API call
			chatResp, err := o.apiClient.ChatWithHistoryContextAndTools(ctx, o.apiMessages, contextPrompt, userInput, o.availableTools)
			if err != nil {
				return APIResponseMsg{Response: "", Err: err}
			}

			// Check if the response contains tool calls
			if chatResp != nil && len(chatResp.Choices) > 0 && len(chatResp.Choices[0].Message.ToolCalls) > 0 {
				// Return a special message type for tool calls
				return ToolCallsResponseMsg{
					ToolCalls: chatResp.Choices[0].Message.ToolCalls,
					Response:  chatResp,
				}
			}

			// Regular response without tool calls
			if chatResp != nil && len(chatResp.Choices) > 0 {
				return APIResponseMsg{Response: chatResp.Choices[0].Message.Content, Err: nil}
			}
		}

		// Fallback to regular API call without tools
		response, err := o.apiClient.ChatWithHistoryContext(ctx, o.apiMessages, contextPrompt, userInput)
		return APIResponseMsg{Response: response, Err: err}
	}
}

// CallAPIStream makes a streaming API call with context and user input
// It returns a command that starts the streaming process
func (o *Operations) CallAPIStream(contextPrompt, userInput string) tea.Cmd {
	// Check context size limit before making API call
	contextSize := len(contextPrompt) + len(userInput)
	contextTokens := EstimateTokens(contextPrompt + userInput)

	cfg := o.configManager.Get()
	maxContextSize := cfg.MaxContextSize
	if maxContextSize == 0 {
		maxContextSize = 100000 // Default 100KB if not configured
	}
	maxContextTokens := EstimateTokens(fmt.Sprintf("%*s", maxContextSize, ""))

    // Optional debug output (enable with DEECLI_DEBUG=1)
    if os.Getenv("DEECLI_DEBUG") == "1" {
        fmt.Printf("Debug: Streaming context size check - chars: %d (limit: %d), tokens: %d (limit: %d)\n",
            contextSize, maxContextSize, contextTokens, maxContextTokens)
    }

	// Check both character and token limits for safety
	if contextSize > maxContextSize || contextTokens > maxContextTokens {
		return func() tea.Msg {
			// Get helpful info about loaded files
			fileInfo := o.fileContext.GetInfo()
			return StreamCompleteMsg{
				Err: fmt.Errorf("context too large - chars: %d/%d, tokens: %d/%d\n\n%s\n\nTry loading fewer files or unload large files with /clear",
					contextSize, maxContextSize, contextTokens, maxContextTokens, fileInfo),
			}
		}
	}

    // Set model-aware timeout
    timeout := 180 * time.Second
    if o.configManager != nil {
        cfg := o.configManager.Get()
        if cfg != nil && strings.EqualFold(cfg.Model, "deepseek-reasoner") {
            timeout = 300 * time.Second
        }
    }

	// Create a context with timeout
    ctx, cancel := context.WithTimeout(context.Background(), timeout)

	// Store the cancel function so we can use it later
	o.apiCancel = cancel

	return func() tea.Msg {
		stream, err := o.apiClient.ChatWithHistoryContextStream(ctx, o.apiMessages, contextPrompt, userInput)
		if err != nil {
			return StreamCompleteMsg{Err: err}
		}

		// Return a StreamReader wrapper that the model can use
		return StreamStartedMsg{
			Stream: stream,
			Ctx:    ctx,
		}
	}
}

// StreamStartedMsg indicates that streaming has started
type StreamStartedMsg struct {
	Stream api.StreamReader
	Ctx    context.Context
}

// ReadNextChunk returns a command to read the next chunk from a stream
func ReadNextChunk(stream api.StreamReader, accumulated string) tea.Cmd {
	return func() tea.Msg {
		chunk, err := stream.Recv()
		if err != nil {
			if err == io.EOF {
				// Stream completed successfully
				stream.Close()
				return StreamCompleteMsg{TotalContent: accumulated}
			}
			// Stream error
			stream.Close()
			return StreamCompleteMsg{TotalContent: accumulated, Err: err}
		}

		// Extract content from chunk
		content := ""
		if len(chunk.Choices) > 0 {
			content = chunk.Choices[0].Delta.Content
		}

		return StreamChunkMsg{
			Content: content,
			IsDone:  false,
		}
	}
}

// AnalyzeFiles analyzes loaded files
func (o *Operations) AnalyzeFiles() tea.Cmd {
	return func() tea.Msg {
		if len(o.fileContext.Files) == 0 {
			return APIResponseMsg{Err: fmt.Errorf("no files loaded")}
		}

		var allAnalysis strings.Builder
		for _, file := range o.fileContext.Files {
			analysis, err := o.apiClient.AnalyzeCode(file.Content, file.RelPath)
			if err != nil {
				return APIResponseMsg{Err: fmt.Errorf("error analyzing %s: %w", file.RelPath, err)}
			}
			allAnalysis.WriteString(fmt.Sprintf("Analysis of %s:\n\n%s\n\n", file.RelPath, analysis))
		}

		return APIResponseMsg{Response: allAnalysis.String()}
	}
}

// ExplainFiles explains loaded files
func (o *Operations) ExplainFiles() tea.Cmd {
	return func() tea.Msg {
		if len(o.fileContext.Files) == 0 {
			return APIResponseMsg{Err: fmt.Errorf("no files loaded")}
		}

		var allExplanations strings.Builder
		for _, file := range o.fileContext.Files {
			explanation, err := o.apiClient.ExplainCode(file.Content, file.RelPath)
			if err != nil {
				return APIResponseMsg{Err: fmt.Errorf("error explaining %s: %w", file.RelPath, err)}
			}
			allExplanations.WriteString(fmt.Sprintf("Explanation of %s:\n\n%s\n\n", file.RelPath, explanation))
		}

		return APIResponseMsg{Response: allExplanations.String()}
	}
}

// ImproveFiles suggests improvements for loaded files
func (o *Operations) ImproveFiles() tea.Cmd {
	return func() tea.Msg {
		if len(o.fileContext.Files) == 0 {
			return APIResponseMsg{Err: fmt.Errorf("no files loaded")}
		}

		var allImprovements strings.Builder
		for _, file := range o.fileContext.Files {
			improvements, err := o.apiClient.ImproveCode(file.Content, file.RelPath)
			if err != nil {
				return APIResponseMsg{Err: fmt.Errorf("error improving %s: %w", file.RelPath, err)}
			}
			allImprovements.WriteString(fmt.Sprintf("Improvement suggestions for %s:\n\n%s\n\n", file.RelPath, improvements))
		}

		return APIResponseMsg{Response: allImprovements.String()}
	}
}

// GenerateEditSuggestions suggests edits based on conversation history
func (o *Operations) GenerateEditSuggestions() tea.Cmd {
	// Create a context that can be cancelled
	ctx, cancel := context.WithCancel(context.Background())

	// Store the cancel function
	o.apiCancel = cancel

	return func() tea.Msg {
		// Build prompt for AI to analyze conversation and suggest file edits
		var promptBuilder strings.Builder
		promptBuilder.WriteString("You are an expert software engineer reviewing a conversation with a developer. ")
		promptBuilder.WriteString("Based on the conversation history and the loaded files, suggest specific files that should be edited and what changes should be made.\n\n")

		// Add loaded files context
		promptBuilder.WriteString("## Loaded Files:\n")
		for _, file := range o.fileContext.Files {
			promptBuilder.WriteString(fmt.Sprintf("**%s** (%s, %d bytes)\n", file.RelPath, file.Language, file.Size))
		}
		promptBuilder.WriteString("\n")

		// Add conversation context (last 10 messages for relevance)
		promptBuilder.WriteString("## Recent Conversation:\n")
		startIdx := 0
		if len(o.apiMessages) > 10 {
			startIdx = len(o.apiMessages) - 10
		}
		for i := startIdx; i < len(o.apiMessages); i++ {
			msg := o.apiMessages[i]
			promptBuilder.WriteString(fmt.Sprintf("**%s**: %s\n", strings.Title(msg.Role), msg.Content))
		}

		promptBuilder.WriteString("\n## Your Task:\n")
		promptBuilder.WriteString("Analyze the conversation and suggest specific files that need editing based on:\n")
		promptBuilder.WriteString("1. Issues or bugs mentioned\n")
		promptBuilder.WriteString("2. Feature requests or improvements discussed\n")
		promptBuilder.WriteString("3. Code quality concerns raised\n")
		promptBuilder.WriteString("4. Missing functionality identified\n\n")
		promptBuilder.WriteString("Format your response as:\n")
		promptBuilder.WriteString("**📝 Edit Suggestions:**\n")
		promptBuilder.WriteString("• **filename.ext** - Brief description of what changes are needed\n")
		promptBuilder.WriteString("• **another.file** - Another change description\n\n")
		promptBuilder.WriteString("If no specific changes are needed, suggest general improvements or say 'No specific edits needed based on current conversation'.")

		// Create messages for API call
		messages := []api.Message{
			{
				Role:    "system",
				Content: "You are an expert software engineer and code reviewer who analyzes conversations to suggest targeted file modifications.",
			},
			{
				Role:    "user",
				Content: promptBuilder.String(),
			},
		}

		// Call API with context for cancellation
		response, err := o.apiClient.ChatWithHistoryContext(ctx, messages, "", "")
		return APIResponseMsg{Response: response, Err: err}
	}
}
