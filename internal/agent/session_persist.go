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

// cleanIncompleteToolCalls removes incomplete tool calls from messages.
// Uses two-phase approach:
// 1. Forward pass to identify all unmatched tool calls (tool_call without corresponding tool_result)
// 2. Reverse pass to remove/filter messages containing unmatched tool calls
func cleanIncompleteToolCalls(messages []fantasy.Message) []fantasy.Message {
	// Phase 1: Forward pass to find all unmatched tool calls
	unmatchedCalls := make(map[string]bool)
	for _, msg := range messages {
		for _, part := range msg.Content {
			switch p := part.(type) {
			case fantasy.ToolCallPart:
				unmatchedCalls[p.ToolCallID] = true
			case fantasy.ToolResultPart:
				delete(unmatchedCalls, p.ToolCallID)
			}
		}
	}

	// Early exit if no unmatched calls
	if len(unmatchedCalls) == 0 {
		return messages
	}

	// Phase 2: Reverse pass to remove/filter messages with unmatched tool calls
	// Process from end, stopping at first complete message or user message
	for i := len(messages) - 1; i >= 0; i-- {
		msg := messages[i]

		// Check if message has any unmatched tool calls
		hasUnmatchedCall := false
		for _, part := range msg.Content {
			if tc, ok := part.(fantasy.ToolCallPart); ok && unmatchedCalls[tc.ToolCallID] {
				hasUnmatchedCall = true
				break
			}
		}

		if hasUnmatchedCall {
			// Filter out unmatched tool calls
			filteredParts := make([]fantasy.MessagePart, 0, len(msg.Content))
			for _, part := range msg.Content {
				if tc, ok := part.(fantasy.ToolCallPart); ok && unmatchedCalls[tc.ToolCallID] {
					continue // Skip unmatched tool call
				}
				filteredParts = append(filteredParts, part)
			}

			if len(filteredParts) > 0 {
				// Has other content (text, reasoning, matched calls) - keep and stop
				messages[i].Content = filteredParts
				return messages[:i+1]
			} else {
				// Only had unmatched tool calls - remove and continue
				messages = messages[:i]
				continue
			}
		}

		// No unmatched tool calls - message is complete, stop here
		return messages[:i+1]
	}

	return messages
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
