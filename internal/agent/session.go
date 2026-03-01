package agent

import (
	"context"
	"encoding/json"
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

// SystemInfo contains session system information
type SystemInfo struct {
	ContextTokens int64 `json:"context"`
	TotalTokens   int64 `json:"total"`
}

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
	session := &Session{
		Processor: processor,
		Messages:  nil,
		Agent:     agent,
		BaseURL:   baseURL,
		ModelName: modelName,
		taskQueue: make(chan Task, 10),
	}
	// Start input reader goroutine that reads TLV from input stream
	go session.readFromInput()
	return session
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
	// Send system info with updated token usage
	s.sendSystemInfo()
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

	// Send system info with updated token usage
	s.sendSystemInfo()

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
			s.writeNotify("[Queued] Previous task in progress. Will run after completion.")
		}
		if !s.inProgress {
			go s.runAsync()
		}
	} else {
		s.writeNotify("[Busy] Cannot queue, try again shortly.")
	}
}

// submitPrompt submits a prompt for processing, queueing if necessary
func (s *Session) submitPrompt(prompt string) {
	s.SubmitTask(UserPrompt(prompt))
}

// submitCommand submits a command for async processing via the task queue
func (s *Session) submitCommand(cmd string) error {
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

// handleCommandSync runs the command synchronously within the async loop
func (s *Session) handleCommandSync(ctx context.Context, cmd string) error {
	switch cmd {
	case "summarize":
		return s.Summarize(ctx)
	case "cancel":
		return s.Cancel()
	default:
		return fmt.Errorf("unknown cmd <%s>", cmd)
	}
}


func (s *Session) writeGapped(tag byte, msg string) {
	if s.Processor != nil && s.Processor.Output != nil {
		stream.WriteTLV(s.Processor.Output, stream.TagStreamGap, "")
		stream.WriteTLV(s.Processor.Output, tag, msg)
		stream.WriteTLV(s.Processor.Output, stream.TagStreamGap, "")
		s.Processor.Output.Flush()
	}
}

func (s *Session) signalPromptStart(prompt string) {
	s.writeGapped(stream.TagPromptStart, prompt)
}

func (s *Session) signalCommandStart(cmd string) {
	s.writeGapped(stream.TagPromptStart, "/"+cmd)
}

func (s *Session) writeNotify(msg string) {
	s.writeGapped(stream.TagNotify, msg)
}

func (s *Session) sendSystemInfo() {
	info := SystemInfo{
		ContextTokens: s.ContextTokens,
		TotalTokens:   s.TotalSpent.TotalTokens,
	}
	data, err := json.Marshal(info)
	if err != nil {
		return
	}
	stream.WriteTLV(s.Processor.Output, stream.TagSystem, string(data))
	s.Processor.Output.Flush()
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

// readFromInput reads TLV messages from the input stream and processes them
func (s *Session) readFromInput() {
	for {
		tag, value, err := stream.ReadTLV(s.Processor.Input)
		if err != nil {
			// Input stream closed or error, stop reading
			return
		}

		// Only accept TagUserText messages, emit error for other tags
		if tag == stream.TagUserText {
			// Check if it's a command (starts with "/")
			if len(value) > 0 && value[0] == '/' {
				command := value[1:]
				if err := s.submitCommand(command); err != nil {
					// Emit error for failed command
					if s.Processor != nil && s.Processor.Output != nil {
						stream.WriteTLV(s.Processor.Output, stream.TagError, err.Error())
						stream.WriteTLV(s.Processor.Output, stream.TagStreamGap, "")
						s.Processor.Output.Flush()
					}
				}
			} else {
				// Regular prompt
				s.submitPrompt(value)
			}
		} else {
			// Emit error for invalid tag
			if s.Processor != nil && s.Processor.Output != nil {
				stream.WriteTLV(s.Processor.Output, stream.TagError, fmt.Sprintf("Invalid input tag: %c (only %c is allowed)", tag, stream.TagUserText))
				stream.WriteTLV(s.Processor.Output, stream.TagStreamGap, "")
				s.Processor.Output.Flush()
			}
		}
	}
}
