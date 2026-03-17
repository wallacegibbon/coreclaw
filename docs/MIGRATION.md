# Migration Guide: Fantasy → Custom LLM

This guide documents the completed migration from the `charm.land/fantasy` library to our custom LLM implementation.

## Why Migrate?

The fantasy library had become limiting:
- Abstractions hide provider-specific features
- Difficult to debug HTTP requests/responses
- Less control over streaming behavior
- External dependency with its own constraints

## Architecture Overview

### Old (Fantasy)
```
Session → fantasy.Agent → fantasy.Provider → HTTP
                ↓
         fantasy.Message types
```

### New (Custom)
```
Session → llm.Agent → llm.Provider → HTTP
               ↓
        llm.Message types (internal/llm/types.go)
```

## Type Mappings

### Messages

**Fantasy:**
```go
fantasy.NewUserMessage(prompt)
fantasy.Message{Role: fantasy.MessageRoleAssistant, Content: ...}
```

**Custom:**
```go
llmcompat.NewUserMessage(prompt)
llm.Message{Role: llm.RoleAssistant, Content: ...}
```

### Content Parts

**Fantasy:**
```go
fantasy.TextPart{Text: "..."}
fantasy.ToolCallPart{ToolCallID: "...", ToolName: "...", Input: ...}
fantasy.ToolResultPart{ToolCallID: "...", Output: ...}
```

**Custom:**
```go
llm.TextPart{Type: "text", Text: "..."}
llm.ToolCallPart{Type: "tool_use", ToolCallID: "...", ToolName: "...", Input: ...}
llm.ToolResultPart{Type: "tool_result", ToolCallID: "...", Output: ...}
```

### Tools

**Fantasy:**
```go
fantasy.NewAgentTool("name", schema, func(ctx, input, tc) (fantasy.ToolResponse, error) {
    return fantasy.NewTextResponse("result"), nil
})
```

**Custom:**
```go
llmcompat.NewTool("name", "description").
    WithSchema(schema).
    WithExecute(func(ctx, input) (llm.ToolResultOutput, error) {
        return llmcompat.NewTextResponse("result"), nil
    }).
    Build()
```

## Migration Status: COMPLETE ✅

### Phase 1: LLM Implementation ✅
1. ✅ Create new types in `internal/llm/`
2. ✅ Implement Anthropic provider
3. ✅ Implement OpenAI provider
4. ✅ Implement OpenAI-compatible provider (Ollama, etc.)
5. ✅ Add conversion utilities (llmcompat)

### Phase 2: Session Migration ✅
1. ✅ Update Session to use llm.Agent and llm.Provider
2. ✅ Update message types from fantasy to llm
3. ✅ Update streaming callbacks

### Phase 3: Tools Migration ✅
1. ✅ Update tools to return llm.Tool
2. ✅ Update tool execution signatures to use json.RawMessage

### Phase 4: App Migration ✅
1. ✅ Update app.go to use llm types
2. ✅ Remove fantasy provider creation code
3. ✅ Update adaptors

### Phase 5: Cleanup ✅
1. ✅ Remove fantasy from go.mod
2. ✅ All tests passing
2. ⬜ Optimize HTTP client usage
3. ⬜ Add provider-specific features (caching, etc.)

## Code Examples

### Session Integration

**Before:**
```go
import "charm.land/fantasy"

type Session struct {
    Agent    fantasy.Agent
    Messages []fantasy.Message
}

func (s *Session) processPrompt(ctx context.Context, prompt string) {
    call := fantasy.AgentStreamCall{Prompt: prompt}
    call.OnTextDelta = func(_, text string) error {
        // handle text
        return nil
    }
    _, err := s.Agent.Stream(ctx, call)
}
```

**After:**
```go
import "github.com/alayacore/alayacore/internal/llm"

type Session struct {
    Agent    *llm.Agent
    Messages []llm.Message
}

func (s *Session) processPrompt(ctx context.Context, prompt string) {
    s.Messages = append(s.Messages, llmcompat.NewUserMessage(prompt))
    
    result, err := s.Agent.Stream(ctx, s.Messages, llm.StreamCallbacks{
        OnTextDelta: func(delta string) error {
            // handle text
            return nil
        },
    })
}
```

### Provider Creation

**Before:**
```go
import "charm.land/fantasy/providers/anthropic"

provider, err := anthropic.New(
    anthropic.WithAPIKey(key),
    anthropic.WithBaseURL(url),
)
model, err := provider.LanguageModel(ctx, "claude-3-5-sonnet-20241022")
```

**After:**
```go
import "github.com/alayacore/alayacore/internal/llm/providers"

provider, err := providers.NewAnthropic(
    providers.WithAPIKey(key),
    providers.WithBaseURL(url),
)
// Provider is ready to use directly
```

## Provider-Specific Features

### Anthropic Prompt Caching

**Before:**
```go
textPart.ProviderOptions = fantasy.ProviderOptions{
    "anthropic": &anthropic.ProviderCacheControlOptions{
        CacheControl: anthropic.CacheControl{Type: "ephemeral"},
    },
}
```

**After:**
```go
// In anthropic.go, add cache_control to content block:
anthropicContentBlock{
    Type: "text",
    Text: text,
    CacheControl: &anthropicCacheControl{
        Type: "ephemeral",
    },
}
```

### Extended Thinking (Reasoning)

Already supported in our types:
```go
llm.ReasoningPart{Type: "thinking", Text: "..."}
```

## Testing Strategy

1. **Unit tests**: Verify type conversions
2. **Integration tests**: Test with mock HTTP servers
3. **Real API tests**: Compare fantasy vs custom outputs
4. **Load tests**: Verify streaming performance

## Rollback Plan

Keep fantasy until fully migrated:
- Use build tags: `// +build !nofantasy`
- Feature flag: `--use-custom-llm`
- Both systems can coexist temporarily

## Timeline

- **Week 1**: Implement all providers (OpenAI, OpenAI-compat)
- **Week 2**: Update Session code, test with real usage
- **Week 3**: Update tools, comprehensive testing
- **Week 4**: Remove fantasy, cleanup, optimize

## Questions?

- Start with `internal/llm/example_test.go` for usage examples
- Check `internal/llm/types.go` for all available types
- See `internal/llm/providers/anthropic.go` for provider implementation pattern
