// Package llmcompat provides compatibility with fantasy types
package llmcompat

import (
	"context"
	"encoding/json"

	"github.com/alayacore/alayacore/internal/llm"
)

// MessageFromFantasy converts fantasy types to our types
// This is a temporary compatibility layer during migration

// NewSystemMessage creates a system message
func NewSystemMessage(text string) llm.Message {
	return llm.Message{
		Role: llm.RoleSystem,
		Content: []llm.ContentPart{
			llm.TextPart{Type: "text", Text: text},
		},
	}
}

// NewUserMessage creates a user message
func NewUserMessage(text string) llm.Message {
	return llm.Message{
		Role: llm.RoleUser,
		Content: []llm.ContentPart{
			llm.TextPart{Type: "text", Text: text},
		},
	}
}

// NewAssistantMessage creates an assistant message
func NewAssistantMessage(parts []llm.ContentPart) llm.Message {
	return llm.Message{
		Role:    llm.RoleAssistant,
		Content: parts,
	}
}

// NewToolResultMessage creates a tool result message
func NewToolResultMessage(toolCallID string, output llm.ToolResultOutput) llm.Message {
	return llm.Message{
		Role: llm.RoleTool,
		Content: []llm.ContentPart{
			llm.ToolResultPart{
				Type:       "tool_result",
				ToolCallID: toolCallID,
				Output:     output,
			},
		},
	}
}

// NewTextResponse creates a text tool response
func NewTextResponse(text string) llm.ToolResultOutput {
	return llm.ToolResultOutputText{
		Type: "text",
		Text: text,
	}
}

// NewTextErrorResponse creates an error tool response
func NewTextErrorResponse(errMsg string) llm.ToolResultOutput {
	return llm.ToolResultOutputError{
		Type:  "error",
		Error: errMsg,
	}
}

// ToolBuilder helps build tool definitions
type ToolBuilder struct {
	tool llm.Tool
}

// NewTool creates a new tool builder
func NewTool(name, description string) *ToolBuilder {
	return &ToolBuilder{
		tool: llm.Tool{
			Definition: llm.ToolDefinition{
				Name:        name,
				Description: description,
			},
		},
	}
}

// WithSchema sets the tool schema
func (b *ToolBuilder) WithSchema(schema json.RawMessage) *ToolBuilder {
	b.tool.Definition.Schema = schema
	return b
}

// WithExecute sets the execute function
func (b *ToolBuilder) WithExecute(fn func(ctx context.Context, input json.RawMessage) (llm.ToolResultOutput, error)) *ToolBuilder {
	b.tool.Execute = fn
	return b
}

// Build returns the tool
func (b *ToolBuilder) Build() llm.Tool {
	return b.tool
}
