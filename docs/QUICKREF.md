# Quick Reference: Fantasy → Custom LLM Migration

## Creating Messages

### Fantasy
```go
import "charm.land/fantasy"

fantasy.NewUserMessage("Hello")
fantasy.NewSystemMessage("You are helpful")
fantasy.Message{
    Role: fantasy.MessageRoleAssistant,
    Content: []fantasy.MessagePart{fantasy.TextPart{Text: "Hi"}},
}
```

### Custom LLM
```go
import "github.com/alayacore/alayacore/internal/llm/llmcompat"

llmcompat.NewUserMessage("Hello")
llmcompat.NewSystemMessage("You are helpful")
llm.Message{
    Role: llm.RoleAssistant,
    Content: []llm.ContentPart{llm.TextPart{Type: "text", Text: "Hi"}},
}
```

## Creating Tools

### Fantasy
```go
fantasy.NewAgentTool(
    "tool_name",
    inputSchema,
    func(ctx context.Context, input ToolInput, tc fantasy.ToolCall) (fantasy.ToolResponse, error) {
        return fantasy.NewTextResponse("result"), nil
    },
)
```

### Custom LLM
```go
llmcompat.NewTool("tool_name", "description").
    WithSchema(inputSchema).
    WithExecute(func(ctx context.Context, input json.RawMessage) (llm.ToolResultOutput, error) {
        return llmcompat.NewTextResponse("result"), nil
    }).
    Build()
```

## Creating Providers

### Fantasy
```go
import "charm.land/fantasy/providers/anthropic"

provider, _ := anthropic.New(
    anthropic.WithAPIKey(key),
    anthropic.WithBaseURL(url),
)
model, _ := provider.LanguageModel(ctx, "claude-3-5-sonnet-20241022")
```

### Custom LLM
```go
import "github.com/alayacore/alayacore/internal/llm"

provider, _ := llm.NewProvider(llm.ProviderConfig{
    Type:    "anthropic",
    APIKey:  key,
    BaseURL: url,
    Model:   "claude-3-5-sonnet-20241022",
})
```

## Streaming

### Fantasy
```go
call := fantasy.AgentStreamCall{Prompt: prompt}
call.OnTextDelta = func(id, text string) error {
    fmt.Print(text)
    return nil
}
call.OnToolCall = func(tc fantasy.ToolCallContent) error {
    // handle tool call
    return nil
}
_, err := agent.Stream(ctx, call)
```

### Custom LLM
```go
result, err := agent.Stream(ctx, messages, llm.StreamCallbacks{
    OnTextDelta: func(delta string) error {
        fmt.Print(delta)
        return nil
    },
    OnToolCall: func(toolCallID, toolName string, input json.RawMessage) error {
        // handle tool call
        return nil
    },
})
```

## Type Checking Content Parts

### Fantasy
```go
if textPart, ok := fantasy.AsMessagePart[fantasy.TextPart](part); ok {
    text := textPart.Text
}
```

### Custom LLM
```go
switch p := part.(type) {
case llm.TextPart:
    text := p.Text
case llm.ToolCallPart:
    toolName := p.ToolName
}
```

## Tool Results

### Fantasy
```go
fantasy.ToolResultPart{
    ToolCallID: id,
    Output: fantasy.ToolResultOutputContentText{Text: "result"},
}
```

### Custom LLM
```go
llm.ToolResultPart{
    Type:       "tool_result",
    ToolCallID: id,
    Output:     llmcompat.NewTextResponse("result"),
}
```

## Provider Types

| Provider Type | Use Case |
|--------------|----------|
| `anthropic` | Anthropic Claude API |
| `openai` | OpenAI GPT API |
| `openaicompat` | Ollama, LM Studio, DeepSeek, etc. |

## Common Patterns

### Check Message Role
```go
if msg.Role == llm.RoleAssistant {
    // handle assistant message
}
```

### Append to Messages
```go
messages = append(messages, llmcompat.NewUserMessage(prompt))
```

### Create Multi-part Message
```go
msg := llm.Message{
    Role: llm.RoleAssistant,
    Content: []llm.ContentPart{
        llm.TextPart{Type: "text", Text: "Here's the answer"},
        llm.ToolCallPart{
            Type:       "tool_use",
            ToolCallID: "call-123",
            ToolName:   "search",
            Input:      json.RawMessage(`{"query":"test"}`),
        },
    },
}
```

## Migration Checklist

For each file that uses fantasy:
1. Replace imports: `charm.land/fantasy` → `github.com/alayacore/alayacore/internal/llm`
2. Update type names: `fantasy.X` → `llm.X`
3. Update constructor calls: Use `llmcompat` helpers
4. Update streaming: Use `StreamCallbacks` instead of `AgentStreamCall`
5. Test thoroughly with real API calls

## Files to Migrate

- [ ] internal/agent/session.go
- [ ] internal/agent/session_prompt.go
- [ ] internal/agent/session_output.go
- [ ] internal/agent/session_commands.go
- [ ] internal/agent/session_markdown.go
- [ ] internal/agent/session_persist.go
- [ ] internal/agent/session_tasks.go
- [ ] internal/tools/*.go
- [ ] internal/app/app.go
- [ ] internal/adaptors/terminal/common.go
