package providers

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"

	"github.com/alayacore/alayacore/internal/llm"
)

// OpenAICompatProvider implements a generic OpenAI-compatible API
// Works with Ollama, LM Studio, DeepSeek, and other compatible providers
type OpenAICompatProvider struct {
	apiKey            string
	baseURL           string
	client            *http.Client
	model             string
	supportsReasoning bool // Whether provider supports reasoning/thinking tokens
}

// OpenAICompatOption configures the provider
type OpenAICompatOption func(*OpenAICompatProvider)

// NewOpenAICompat creates a new OpenAI-compatible provider
func NewOpenAICompat(opts ...OpenAICompatOption) (*OpenAICompatProvider, error) {
	p := &OpenAICompatProvider{
		client: &http.Client{},
		model:  "local-model",
	}
	for _, opt := range opts {
		opt(p)
	}
	if p.baseURL == "" {
		return nil, fmt.Errorf("base URL is required")
	}
	return p, nil
}

// WithOpenAICompatAPIKey sets the API key
func WithOpenAICompatAPIKey(key string) OpenAICompatOption {
	return func(p *OpenAICompatProvider) {
		p.apiKey = key
	}
}

// WithOpenAICompatBaseURL sets the base URL
func WithOpenAICompatBaseURL(url string) OpenAICompatOption {
	return func(p *OpenAICompatProvider) {
		p.baseURL = strings.TrimSuffix(url, "/")
	}
}

// WithOpenAICompatHTTPClient sets the HTTP client
func WithOpenAICompatHTTPClient(client *http.Client) OpenAICompatOption {
	return func(p *OpenAICompatProvider) {
		p.client = client
	}
}

// WithOpenAICompatModel sets the model
func WithOpenAICompatModel(model string) OpenAICompatOption {
	return func(p *OpenAICompatProvider) {
		p.model = model
	}
}

// WithOpenAICompatReasoning enables reasoning support
func WithOpenAICompatReasoning(supports bool) OpenAICompatOption {
	return func(p *OpenAICompatProvider) {
		p.supportsReasoning = supports
	}
}

// StreamMessages streams messages from OpenAI-compatible API
func (p *OpenAICompatProvider) StreamMessages(
	ctx context.Context,
	messages []llm.Message,
	tools []llm.ToolDefinition,
	systemPrompt string,
) (<-chan llm.StreamEvent, error) {

	// Build request - same structure as OpenAI
	reqBody := map[string]interface{}{
		"model":    p.model,
		"messages": p.buildMessages(messages, systemPrompt),
		"stream":   true,
	}

	if len(tools) > 0 {
		reqBody["tools"] = p.buildTools(tools)
	}

	bodyBytes, err := json.Marshal(reqBody)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, "POST", p.baseURL+"/chat/completions", bytes.NewReader(bodyBytes))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	if p.apiKey != "" {
		req.Header.Set("Authorization", "Bearer "+p.apiKey)
	}

	resp, err := p.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to send request: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		resp.Body.Close()
		return nil, fmt.Errorf("API error (status %d): %s", resp.StatusCode, string(body))
	}

	// Create event channel
	eventChan := make(chan llm.StreamEvent, 100)

	// Start streaming goroutine
	go p.parseStream(resp.Body, eventChan)

	return eventChan, nil
}

// buildMessages converts messages to OpenAI-compatible format
func (p *OpenAICompatProvider) buildMessages(messages []llm.Message, systemPrompt string) []interface{} {
	apiMessages := make([]interface{}, 0, len(messages)+1)

	// Add system message
	if systemPrompt != "" {
		apiMessages = append(apiMessages, map[string]interface{}{
			"role":    "system",
			"content": systemPrompt,
		})
	}

	// Convert messages
	for _, msg := range messages {
		apiMsg := p.convertMessage(msg)
		apiMessages = append(apiMessages, apiMsg)
	}

	return apiMessages
}

// convertMessage converts our message to OpenAI-compatible format
func (p *OpenAICompatProvider) convertMessage(msg llm.Message) map[string]interface{} {
	apiMsg := map[string]interface{}{
		"role": string(msg.Role),
	}

	// Handle tool results
	if msg.Role == llm.RoleTool {
		if len(msg.Content) > 0 {
			if tr, ok := msg.Content[0].(llm.ToolResultPart); ok {
				apiMsg["tool_call_id"] = tr.ToolCallID
				switch out := tr.Output.(type) {
				case llm.ToolResultOutputText:
					apiMsg["content"] = out.Text
				case llm.ToolResultOutputError:
					apiMsg["content"] = out.Error
				}
			}
		}
		return apiMsg
	}

	// Handle assistant messages with tool calls
	if msg.Role == llm.RoleAssistant {
		var toolCalls []interface{}
		for _, part := range msg.Content {
			if tc, ok := part.(llm.ToolCallPart); ok {
				toolCalls = append(toolCalls, map[string]interface{}{
					"id":   tc.ToolCallID,
					"type": "function",
					"function": map[string]interface{}{
						"name":      tc.ToolName,
						"arguments": string(tc.Input),
					},
				})
			}
		}

		if len(toolCalls) > 0 {
			apiMsg["tool_calls"] = toolCalls
		}
	}

	// Build content
	var contentBuilder strings.Builder
	var hasContent bool

	for _, part := range msg.Content {
		switch v := part.(type) {
		case llm.TextPart:
			contentBuilder.WriteString(v.Text)
			hasContent = true
		case llm.ReasoningPart:
			// Some providers (DeepSeek) support reasoning_content field
			// For others, we'll add it as marked text
			if p.supportsReasoning {
				apiMsg["reasoning_content"] = v.Text
			} else {
				contentBuilder.WriteString("[Thinking] ")
				contentBuilder.WriteString(v.Text)
				hasContent = true
			}
		}
	}

	if hasContent {
		apiMsg["content"] = contentBuilder.String()
	}

	return apiMsg
}

// buildTools converts tools to OpenAI-compatible format
func (p *OpenAICompatProvider) buildTools(tools []llm.ToolDefinition) []interface{} {
	apiTools := make([]interface{}, 0, len(tools))
	for _, tool := range tools {
		apiTools = append(apiTools, map[string]interface{}{
			"type": "function",
			"function": map[string]interface{}{
				"name":        tool.Name,
				"description": tool.Description,
				"parameters":  tool.Schema,
			},
		})
	}
	return apiTools
}

// parseStream parses the SSE stream
func (p *OpenAICompatProvider) parseStream(reader io.Reader, eventChan chan<- llm.StreamEvent) {
	defer close(eventChan)

	scanner := bufio.NewScanner(reader)

	for scanner.Scan() {
		line := scanner.Text()

		if !strings.HasPrefix(line, "data: ") {
			continue
		}

		data := strings.TrimPrefix(line, "data: ")
		if data == "[DONE]" {
			break
		}

		if err := p.handleEvent(data, eventChan); err != nil {
			eventChan <- llm.StreamErrorEvent{Error: err}
			return
		}
	}

	if err := scanner.Err(); err != nil {
		eventChan <- llm.StreamErrorEvent{Error: err}
	}
}

// handleEvent handles a single SSE event
func (p *OpenAICompatProvider) handleEvent(data string, eventChan chan<- llm.StreamEvent) error {
	// Parse generic JSON response
	var streamResp map[string]interface{}
	if err := json.Unmarshal([]byte(data), &streamResp); err != nil {
		return fmt.Errorf("failed to parse event: %w", err)
	}

	choices, ok := streamResp["choices"].([]interface{})
	if !ok || len(choices) == 0 {
		return nil
	}

	choice, ok := choices[0].(map[string]interface{})
	if !ok {
		return nil
	}

	delta, ok := choice["delta"].(map[string]interface{})
	if !ok {
		return nil
	}

	// Handle text content
	if content, ok := delta["content"].(string); ok && content != "" {
		eventChan <- llm.TextDeltaEvent{Delta: content}
	}

	// Handle reasoning content (DeepSeek, etc.)
	if p.supportsReasoning {
		if reasoning, ok := delta["reasoning_content"].(string); ok && reasoning != "" {
			eventChan <- llm.ReasoningDeltaEvent{Delta: reasoning}
		}
	}

	// Handle tool calls
	if toolCalls, ok := delta["tool_calls"].([]interface{}); ok {
		for _, tc := range toolCalls {
			tcMap, ok := tc.(map[string]interface{})
			if !ok {
				continue
			}

			id, _ := tcMap["id"].(string)
			function, ok := tcMap["function"].(map[string]interface{})
			if !ok {
				continue
			}

			name, _ := function["name"].(string)
			args, _ := function["arguments"].(string)

			if name != "" {
				eventChan <- llm.ToolCallEvent{
					ToolCallID: id,
					ToolName:   name,
					Input:      json.RawMessage(args),
				}
			}
		}
	}

	// Handle completion
	if finishReason, ok := choice["finish_reason"].(string); ok && finishReason != "" {
		usage, ok := streamResp["usage"].(map[string]interface{})
		if ok {
			promptTokens, _ := usage["prompt_tokens"].(float64)
			completionTokens, _ := usage["completion_tokens"].(float64)

			if promptTokens > 0 || completionTokens > 0 {
				eventChan <- llm.StepCompleteEvent{
					Messages: []llm.Message{},
					Usage: llm.Usage{
						InputTokens:  int64(promptTokens),
						OutputTokens: int64(completionTokens),
					},
				}
			}
		}
	}

	return nil
}
