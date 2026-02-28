package agent

import (
	"context"
	"fmt"

	"charm.land/fantasy"
	"github.com/wallacegibbon/coreclaw/internal/stream"
)

// Task represents a unit of work in the task queue
type Task interface{ isTask() }

type UserPrompt string

func (UserPrompt) isTask() {}

type CommandPrompt struct {
	Command string
}

func (CommandPrompt) isTask() {}

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

	// taskQueue buffers tasks submitted while agent is processing
	taskQueue chan Task

	// inProgress tracks whether a prompt is currently being processed
	inProgress bool

	// cancelCurrent is a function to cancel the current prompt
	cancelCurrent func()
}

// CancelCurrent cancels the currently running prompt if any
// Returns true if cancel was initiated, false if cancel is already in progress
func (s *Session) CancelCurrent() bool {
	if s.cancelCurrent != nil {
		s.cancelCurrent()
		return true
	}
	return false
}

// Cancel handles the /cancel command
// Returns error if nothing to cancel
func (s *Session) Cancel() error {
	if !s.inProgress {
		return fmt.Errorf("nothing to cancel")
	}
	if s.CancelCurrent() {
		return nil
	}
	return fmt.Errorf("nothing to cancel")
}

// IsInProgress returns true if a prompt is currently being processed
func (s *Session) IsInProgress() bool {
	return s.inProgress
}

// NewSession creates a new session with the given processor
func NewSession(agent fantasy.Agent, baseURL, modelName string, processor *Processor) *Session {
	return &Session{
		Processor: processor,
		Messages:  nil,
		Agent:     agent,
		BaseURL:   baseURL,
		ModelName: modelName,
		taskQueue: make(chan Task, 10),
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

// SubmitTask submits a task for async processing via the task queue
// Processing runs asynchronously so adaptors can continue receiving input
func (s *Session) SubmitTask(task Task) {
	if s.queueTask(task) {
		if s.inProgress {
			s.writeStatus("[Queued] Previous task in progress. Will run after completion.")
		}
		if !s.inProgress {
			go s.runAsync()
		}
	} else {
		s.writeStatus("[Busy] Cannot queue, try again shortly.")
	}
}

// SubmitPrompt submits a prompt for processing, queueing if necessary
// This is the main entry point for adaptors - handles all queue logic internally
func (s *Session) SubmitPrompt(prompt string) {
	s.SubmitTask(UserPrompt(prompt))
}

// SubmitCommand submits a command for async processing via the task queue
func (s *Session) SubmitCommand(cmd string) error {
	switch cmd {
	case "summarize":
		s.SubmitTask(CommandPrompt{Command: cmd})
		return nil
	default:
		return s.handleCommandSync(context.Background(), cmd)
	}
}

// runAsync processes tasks asynchronously, including any queued tasks
func (s *Session) runAsync() {
	s.inProgress = true
	defer func() {
		s.inProgress = false
	}()

	for {
		queuedTask, ok := s.getQueuedTask()
		if !ok {
			break
		}
		// Create a fresh context for each queued task
		taskCtx, taskCancel := context.WithCancel(context.Background())
		s.cancelCurrent = taskCancel

		// Handle different task types
		switch task := queuedTask.(type) {
		case UserPrompt:
			s.signalPromptStart(string(task))
			s.ProcessPrompt(taskCtx, string(task))
		case CommandPrompt:
			s.signalCommandStart(task.Command)
			s.handleCommandSync(taskCtx, task.Command)
		}

		// Check if cancelled
		if taskCtx.Err() == context.Canceled {
			// Add assistant message to close out the canceled prompt
			// This prevents the next prompt from being concatenated into the canceled one
			s.Messages = append(s.Messages, fantasy.Message{
				Role:    fantasy.MessageRoleAssistant,
				Content: []fantasy.MessagePart{fantasy.TextPart{Text: "The user canceled."}},
			})
			s.cancelCurrent = nil
			continue
		}
		s.cancelCurrent = nil
	}
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

// signalCommandStart signals that a command has started processing
func (s *Session) signalCommandStart(cmd string) {
	if s.Processor != nil && s.Processor.Output != nil {
		stream.WriteTLV(s.Processor.Output, stream.TagStreamGap, "")
		stream.WriteTLV(s.Processor.Output, stream.TagPromptStart, "/"+cmd)
		stream.WriteTLV(s.Processor.Output, stream.TagStreamGap, "")
		s.Processor.Output.Flush()
	}
}

// handleCommandSync runs the command synchronously within the async loop
func (s *Session) handleCommandSync(ctx context.Context, cmd string) error {
	switch cmd {
	case "summarize":
		return s.Summarize(ctx)
	case "cancel":
		return s.Cancel()
	default:
		err := fmt.Errorf("unknown cmd <%s>", cmd)
		s.writeError(err.Error())
		return err
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

// writeError writes an error message to the output
func (s *Session) writeError(msg string) {
	if s.Processor != nil && s.Processor.Output != nil {
		stream.WriteTLV(s.Processor.Output, stream.TagStreamGap, "")
		stream.WriteTLV(s.Processor.Output, stream.TagError, msg)
		stream.WriteTLV(s.Processor.Output, stream.TagStreamGap, "")
		s.Processor.Output.Flush()
	}
}

// queueTask adds a task to the queue (non-blocking)
func (s *Session) queueTask(task Task) bool {
	select {
	case s.taskQueue <- task:
		return true
	default:
		return false
	}
}

// getQueuedTask tries to get a queued task (non-blocking)
func (s *Session) getQueuedTask() (Task, bool) {
	select {
	case task, ok := <-s.taskQueue:
		return task, ok
	default:
		return nil, false
	}
}
