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

	// promptQueue buffers prompts submitted while agent is processing
	promptQueue chan string

	// inProgress tracks whether a prompt is currently being processed
	inProgress bool
	// cancelInProgress tracks whether a cancel operation is ongoing
	cancelInProgress bool

	// cancelCurrent is a function to cancel the current prompt
	cancelCurrent func()
}

// CancelCurrent cancels the currently running prompt if any
// Returns true if cancel was initiated, false if cancel is already in progress
func (s *Session) CancelCurrent() bool {
	if s.cancelInProgress {
		return false // Ignore cancel if already cancelling
	}
	if s.cancelCurrent != nil {
		s.cancelInProgress = true
		s.cancelCurrent()
		return true
	}
	return false
}

// IsInProgress returns true if a prompt is currently being processed
func (s *Session) IsInProgress() bool {
	return s.inProgress
}

// NewSession creates a new session with the given processor
func NewSession(agent fantasy.Agent, baseURL, modelName string, processor *Processor) *Session {
	return &Session{
		Processor:   processor,
		Messages:    nil,
		Agent:       agent,
		BaseURL:     baseURL,
		ModelName:   modelName,
		promptQueue: make(chan string, 10),
	}
}

// HandleCommand processes special commands like /summarize
func (s *Session) HandleCommand(cmd string) error {
	ctx, _ := context.WithCancel(context.Background())
	switch cmd {
	case "summarize":
		return s.Summarize(ctx)
	default:
		return fmt.Errorf("unknown cmd <%s>", cmd)
	}
}

// Summarize summarizes the conversation history
func (s *Session) Summarize(ctx context.Context) error {
	summarizePrompt := "Please summarize the conversation above in a concise manner. Return ONLY the summary, no introductions or explanations."

	assistantMsg, usage, err := s.Processor.ProcessPrompt(ctx, summarizePrompt, s.Messages)
	if err != nil {
		return err
	}
	// Replace messages with summary
	s.Messages = []fantasy.Message{assistantMsg}
	s.TotalSpent.InputTokens += usage.InputTokens
	s.TotalSpent.OutputTokens += usage.OutputTokens
	s.TotalSpent.TotalTokens += usage.TotalTokens
	s.TotalSpent.ReasoningTokens += usage.ReasoningTokens
	// After summarize, context shrinks to the summary
	s.ContextTokens = usage.OutputTokens
	return nil
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
	assistantMsg, usage, err := s.Processor.ProcessPrompt(ctx, prompt, messagesForAPI)

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

	// If there is an assistant message, store it.
	if assistantMsg.Role != "" {
		s.Messages = append(s.Messages, assistantMsg)
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
	stream.WriteTLV(s.Processor.Output, stream.TagStreamGap, "")
	s.Processor.Output.Flush()
}

// SubmitPrompt submits a prompt for processing, queueing if necessary
// This is the main entry point for adaptors - handles all queue logic internally
// Processing runs asynchronously so adaptors can continue receiving input
func (s *Session) SubmitPrompt(prompt string) {
	if s.queuePrompt(prompt) {
		if s.inProgress {
			s.writeStatus("[Queued] Previous task in progress. Will run after completion.")
			return
		}
	} else {
		s.writeStatus("[Busy] Cannot queue, try again shortly.")
	}

	// Start async processing
	s.inProgress = true
	go s.runAsync()
}

// runAsync processes prompts asynchronously, including any queued prompts
func (s *Session) runAsync() {
	// Process any queued prompts
	for {
		if queuedPrompt, ok := s.getQueuedPrompt(); ok {
			// Create a fresh context for each queued prompt
			promptCtx, promptCancel := context.WithCancel(context.Background())
			s.cancelCurrent = promptCancel

			// Signal queued prompt start
			s.signalPromptStart(queuedPrompt)

			s.ProcessPrompt(promptCtx, queuedPrompt)

			// Check if cancelled
			if promptCtx.Err() == context.Canceled && s.cancelInProgress {
				s.cancelInProgress = false
			}

			s.SendUsage()

			// If context was cancelled, stop processing queued prompts
			if promptCtx.Err() == context.Canceled {
				s.cancelCurrent = nil
				break
			}
			s.cancelCurrent = nil
		} else {
			break
		}
	}
	s.inProgress = false
	s.cancelInProgress = false
}

// signalPromptStart signals that a prompt has started processing
func (s *Session) signalPromptStart(prompt string) {
	if s.Processor != nil && s.Processor.Output != nil {
		stream.WriteTLV(s.Processor.Output, stream.TagStreamGap, "")
		stream.WriteTLV(s.Processor.Output, stream.TagPromptStart, prompt)
		stream.WriteTLV(s.Processor.Output, stream.TagStreamGap, "")
		s.Processor.Output.Flush()
	}
}

// writeStatus writes a system status message to the output
func (s *Session) writeStatus(msg string) {
	if s.Processor != nil && s.Processor.Output != nil {
		stream.WriteTLV(s.Processor.Output, stream.TagStreamGap, "")
		stream.WriteTLV(s.Processor.Output, stream.TagSystem, msg)
		stream.WriteTLV(s.Processor.Output, stream.TagStreamGap, "")
		s.Processor.Output.Flush()
	}
}

// queuePrompt adds a prompt to the queue (non-blocking)
func (s *Session) queuePrompt(prompt string) bool {
	select {
	case s.promptQueue <- prompt:
		return true
	default:
		return false
	}
}

// getQueuedPrompt tries to get a queued prompt (non-blocking)
func (s *Session) getQueuedPrompt() (string, bool) {
	select {
	case prompt, ok := <-s.promptQueue:
		return prompt, ok
	default:
		return "", false
	}
}
