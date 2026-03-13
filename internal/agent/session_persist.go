package agent

import (
	"fmt"
	"os"
	"time"

	"charm.land/fantasy"
	"github.com/alayacore/alayacore/internal/stream"
)

// ============================================================================
// Persistence
// ============================================================================

func LoadSession(path string) (*SessionData, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("failed to read session file: %w", err)
	}
	return parseSessionMarkdown(data)
}

func (s *Session) saveSessionToFile(path string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	data := SessionData{
		BaseURL:       s.BaseURL,
		ModelName:     s.ModelName,
		Messages:      s.Messages,
		TotalSpent:    s.TotalSpent,
		ContextTokens: s.ContextTokens,
		UpdatedAt:     time.Now(),
	}

	raw, err := formatSessionMarkdown(&data)
	if err != nil {
		return fmt.Errorf("failed to format session data: %w", err)
	}
	if err := os.WriteFile(path, raw, 0644); err != nil {
		return fmt.Errorf("failed to write session file: %w", err)
	}
	return nil
}

// ============================================================================
// Clean Incomplete Tool Calls
// ============================================================================

// cleanIncompleteToolCalls removes incomplete tool calls from the end of messages.
// Incomplete tool calls can only exist at the end (previous messages were successfully processed).
// ToolResult always comes after ToolCall, so if last message is Assistant with ToolCall,
// those calls have no results (incomplete).
func cleanIncompleteToolCalls(messages []fantasy.Message) []fantasy.Message {
	for len(messages) > 0 {
		last := messages[len(messages)-1]

		// Check if message contains ToolResult (complete tool cycle)
		// Note: Anthropic puts ToolResult in user message, OpenAI uses tool role
		if hasToolResult(last) {
			break
		}

		// User message without tool result - stop
		if last.Role == fantasy.MessageRoleUser {
			break
		}

		// Assistant message - filter out ToolCalls (no results since this is last)
		if last.Role == fantasy.MessageRoleAssistant {
			var filteredParts []fantasy.MessagePart
			for _, part := range last.Content {
				// Keep text, reasoning - drop ToolCalls
				if _, ok := part.(fantasy.ToolCallPart); !ok {
					filteredParts = append(filteredParts, part)
				}
			}

			// Has content after filtering - update and stop
			if len(filteredParts) > 0 {
				messages[len(messages)-1].Content = filteredParts
				break
			}

			// No content - remove and continue
			messages = messages[:len(messages)-1]
			continue
		}

		break
	}

	return messages
}

// hasToolResult checks if a message contains any ToolResultPart
func hasToolResult(msg fantasy.Message) bool {
	for _, part := range msg.Content {
		if _, ok := part.(fantasy.ToolResultPart); ok {
			return true
		}
	}
	return false
}

// ============================================================================
// Message Display (for session restore)
// ============================================================================

func (s *Session) displayMessages() {
	if s.Output == nil {
		return
	}
	for _, msg := range s.Messages {
		switch msg.Role {
		case fantasy.MessageRoleUser:
			s.displayUserMessage(msg)
		case fantasy.MessageRoleAssistant:
			s.displayAssistantMessage(msg)
		case fantasy.MessageRoleTool:
			s.displayToolMessage(msg)
		}
	}
}

func (s *Session) displayUserMessage(msg fantasy.Message) {
	var text string
	for _, part := range msg.Content {
		if tp, ok := part.(fantasy.TextPart); ok {
			text += tp.Text
		}
	}
	if text != "" {
		s.signalPromptStart(text)
	}
}

func (s *Session) displayAssistantMessage(msg fantasy.Message) {
	for _, part := range msg.Content {
		switch p := part.(type) {
		case fantasy.TextPart:
			stream.WriteTLV(s.Output, stream.TagAssistantText, p.Text)
			s.Output.Flush()
		case fantasy.ReasoningPart:
			stream.WriteTLV(s.Output, stream.TagReasoning, p.Text)
			s.Output.Flush()
		case fantasy.ToolCallPart:
			if info := formatToolCall(p.ToolName, p.Input); info != "" {
				stream.WriteTLV(s.Output, stream.TagTool, info)
				s.Output.Flush()
			}
		}
	}
}

func (s *Session) displayToolMessage(msg fantasy.Message) {
	for _, part := range msg.Content {
		if tc, ok := part.(fantasy.ToolCallPart); ok {
			if info := formatToolCall(tc.ToolName, tc.Input); info != "" {
				stream.WriteTLV(s.Output, stream.TagTool, info)
				s.Output.Flush()
			}
		}
	}
}
