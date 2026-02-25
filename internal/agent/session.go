package agent

import (
	"context"
	"fmt"

	"charm.land/fantasy"
	"github.com/wallacegibbon/coreclaw/internal/stream"
)

// Session manages message history and processes prompts
type Session struct {
	Processor *Processor
	Messages  []fantasy.Message

	// Agent is the fantasy agent instance
	Agent fantasy.Agent

	// BaseURL and ModelName store provider configuration
	BaseURL   string
	ModelName string

	// TotalSpent tracks total tokens used across all requests
	TotalSpent fantasy.Usage

	// ContextTokens tracks context tokens used (grows with each request, shrinks after summarize)
	ContextTokens int64

	// OnCommandDone is called after a command (like /summarize) completes
	OnCommandDone func()
}

// NewSession creates a new session with the given processor
func NewSession(agent fantasy.Agent, baseURL, modelName string, processor *Processor) *Session {
	return &Session{
		Processor: processor,
		Messages:  nil,
		Agent:     agent,
		BaseURL:   baseURL,
		ModelName: modelName,
	}
}

// HandleCommand processes special commands like /summarize
// Returns (handled bool, error)
func (s *Session) HandleCommand(ctx context.Context, cmd string) (bool, error) {
	switch cmd {
	case "summarize":
		_, usage, err := s.Summarize(ctx)
		s.TotalSpent.InputTokens += usage.InputTokens
		s.TotalSpent.OutputTokens += usage.OutputTokens
		s.TotalSpent.TotalTokens += usage.TotalTokens
		s.TotalSpent.ReasoningTokens += usage.ReasoningTokens
		// After summarize, context shrinks to the summary
		s.ContextTokens = usage.OutputTokens
		if s.OnCommandDone != nil {
			s.OnCommandDone()
		}
		return true, err
	default:
		return false, nil
	}
}

// Summarize summarizes the conversation history
func (s *Session) Summarize(ctx context.Context) (fantasy.Message, fantasy.Usage, error) {
	_, assistantMsg, usage, err := s.Processor.Summarize(ctx, s.Messages)
	if err != nil {
		return fantasy.Message{}, fantasy.Usage{}, err
	}
	// Replace messages with summary
	s.Messages = []fantasy.Message{assistantMsg}
	return assistantMsg, usage, nil
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

	// Track usage
	s.TotalSpent.InputTokens += usage.InputTokens
	s.TotalSpent.OutputTokens += usage.OutputTokens
	s.TotalSpent.TotalTokens += usage.TotalTokens
	s.TotalSpent.ReasoningTokens += usage.ReasoningTokens

	// Context grows with each request
	s.ContextTokens += usage.TotalTokens

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

// SendUsage sends usage info (context size and total tokens spent) via TLV
func (s *Session) SendUsage() {
	if s.Processor == nil || s.Processor.Output == nil {
		return
	}
	msg := fmt.Sprintf("context=%d, spent=%d", s.ContextTokens, s.TotalSpent.TotalTokens)
	stream.WriteTLV(s.Processor.Output, stream.TagUsage, msg)
	s.Processor.Output.Flush()
}
