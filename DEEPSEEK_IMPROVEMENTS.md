# DeepSeek API Improvements for DeeCLI

## 1. Streaming Responses
**Benefit**: Real-time output as model generates, better UX for long responses

```go
// Example from deepseek-go library
stream, err := client.CreateChatCompletionStream(ctx, ChatCompletionRequest{
    Model: "deepseek-chat",
    Messages: messages,
    Stream: true,
})

defer stream.Close()

for {
    response, err := stream.Recv()
    if errors.Is(err, io.EOF) {
        break
    }
    if err != nil {
        return err
    }
    fmt.Print(response.Choices[0].Delta.Content)
}
```

## 2. FIM (Fill-in-the-Middle) Completions
**Benefit**: Code completion for `/edit` command - provide context, get middle filled

```python
# API endpoint: https://api.deepseek.com/beta
# Use case: User provides start and end of function, AI fills implementation

request = {
    "model": "deepseek-coder",
    "prompt": "def calculate_fibonacci(n):\n    if n <= 1:",
    "suffix": "\n    return calculate_fibonacci(n-1) + calculate_fibonacci(n-2)",
    "max_tokens": 512
}
# AI generates: "        return n"
```

## 3. Token Estimation
**Benefit**: Warn users before hitting limits, optimize API usage

```go
// Estimate tokens before sending
import "github.com/pkoukk/tiktoken-go"

func EstimateTokens(text string) int {
    encoding, _ := tiktoken.GetEncoding("cl100k_base")
    tokens := encoding.Encode(text, nil, nil)
    return len(tokens)
}

// Usage in DeeCLI
if EstimateTokens(userInput) > 4000 {
    fmt.Println("Warning: Input may exceed token limit")
}
```

## 4. JSON Mode for Structured Output
**Benefit**: Parse AI responses reliably for commands like `/analyze`

```go
request := ChatCompletionRequest{
    Model: "deepseek-chat",
    Messages: messages,
    ResponseFormat: &ResponseFormat{
        Type: "json_object",
    },
    // Prompt must mention JSON output
    Messages: []Message{
        {Role: "system", Content: "Output analysis as JSON"},
        {Role: "user", Content: "Analyze this code..."},
    },
}
```

## 5. Stop Sequences
**Benefit**: Control output boundaries, useful for code generation

```go
request := ChatCompletionRequest{
    Model: "deepseek-chat",
    Messages: messages,
    Stop: []string{"```", "// END", "\n\n"},  // Stop at these sequences
    MaxTokens: 2048,
}
```

## 6. Balance Checking
**Benefit**: Show users their API usage/credits

```go
// Check API balance
func (c *Client) GetBalance() (*BalanceResponse, error) {
    req, _ := http.NewRequest("GET", c.baseURL+"/user/balance", nil)
    req.Header.Set("Authorization", "Bearer "+c.apiKey)

    resp, err := c.httpClient.Do(req)
    // Parse response for balance info
}

// Usage in DeeCLI: /balance command
```

## 7. Model-Specific Parameters
**Benefit**: Optimize for each model's capabilities

```go
func CreateRequest(model string, messages []Message) ChatRequest {
    req := ChatRequest{
        Model:     model,
        Messages:  messages,
        MaxTokens: 2048,
    }

    // Reasoner model doesn't support temperature
    if model != "deepseek-reasoner" {
        req.Temperature = 0.7
    }

    // Chat model supports higher context
    if model == "deepseek-chat" {
        req.MaxTokens = 4096
    }

    return req
}
```

## Implementation Priority

1. **High Priority**: Streaming (immediate UX improvement)
2. **Medium Priority**: Token estimation, Stop sequences
3. **Low Priority**: FIM completions, Balance checking
4. **Already Done**: Model-specific temperature handling