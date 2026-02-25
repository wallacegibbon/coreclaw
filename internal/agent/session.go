package agent

import (
	"context"

	"charm.land/fantasy"
)

// Session manages message history and processes prompts
type Session struct {
	Processor *Processor
	Messages  []fantasy.Message
}

// NewSession creates a new session with the given processor
func NewSession(processor *Processor) *Session {
	return &Session{
		Processor: processor,
		Messages:  nil,
	}
}

// ProcessPrompt processes a user prompt and updates message history
// It handles adding user message, calling API, and storing assistant response
func (s *Session) ProcessPrompt(ctx context.Context, prompt string) (fantasy.Message, fantasy.Usage, error) {
	// Add user message to history
	s.Messages = append(s.Messages, fantasy.NewUserMessage(prompt))

	// Create a copy of messages WITHOUT the pending user message for API
	// This prevents duplication (API adds user message internally)
	messagesForAPI := make([]fantasy.Message, len(s.Messages)-1)
	copy(messagesForAPI, s.Messages[:len(s.Messages)-1])

	// Process the prompt
	_, responseText, assistantMsg, usage, err := s.Processor.ProcessPrompt(ctx, prompt, messagesForAPI)

	if err != nil {
		return fantasy.Message{}, fantasy.Usage{}, err
	}

	// Store assistant message
	if assistantMsg.Role != "" {
		s.Messages = append(s.Messages, assistantMsg)
	} else if responseText != "" {
		s.Messages = append(s.Messages, fantasy.Message{
			Role:    fantasy.MessageRoleAssistant,
			Content: []fantasy.MessagePart{fantasy.TextPart{Text: responseText}},
		})
	}

	return assistantMsg, usage, nil
}
