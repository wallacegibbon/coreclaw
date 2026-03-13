package agent

import (
	"context"
	"fmt"
	"strconv"
	"sync/atomic"

	"charm.land/fantasy"
	"github.com/alayacore/alayacore/internal/stream"
)

// ============================================================================
// Prompt Handling
// ============================================================================

func (s *Session) handleUserPrompt(ctx context.Context, prompt string) {
	// Auto-summarize when context usage reaches 80% of the limit
	if s.shouldAutoSummarize() {
		s.autoSummarize(ctx)
	}

	s.Messages = append(s.Messages, fantasy.NewUserMessage(prompt))
	history := s.Messages[:len(s.Messages)-1]

	result, err := s.processPromptWithResult(ctx, prompt, history)
	if err != nil {
		s.writeError(err.Error())
		return
	}
	// Append all messages from the agent result (assistant + tool messages)
	for _, step := range result.Steps {
		if len(step.Messages) > 0 {
			s.Messages = append(s.Messages, step.Messages...)
		}
	}
}

// shouldAutoSummarize checks if context usage exceeds 80% threshold
func (s *Session) shouldAutoSummarize() bool {
	return s.ContextLimit > 0 && s.ContextTokens > 0 &&
		s.ContextTokens >= s.ContextLimit*80/100
}

// autoSummarize performs synchronous summarization to reduce context
func (s *Session) autoSummarize(ctx context.Context) {
	usage := float64(s.ContextTokens) * 100 / float64(s.ContextLimit)
	s.writeNotify(fmt.Sprintf("Context usage at %d/%d tokens (%.0f%%). Auto-summarizing...",
		s.ContextTokens, s.ContextLimit, usage))
	s.summarize(ctx)
}

// processPromptWithResult processes a prompt and returns the full agent result
func (s *Session) processPromptWithResult(ctx context.Context, prompt string, history []fantasy.Message) (*fantasy.AgentResult, error) {
	call := fantasy.AgentStreamCall{Prompt: prompt}
	promptId := atomic.AddUint64(&s.nextPromptID, 1) - 1

	var stepCount int

	if len(history) > 0 {
		call.Messages = history
	}

	/// the final ID is [:promptId-stepCount-id:]
	assembleId := func(id string) string {
		return "[:" + strconv.FormatUint(promptId, 10) + "-" + strconv.FormatInt(int64(stepCount), 10) + "-" + id + ":]"
	}

	call.OnStepStart = func(step int) error {
		stepCount = step
		return nil
	}
	call.OnStepFinish = func(stepResult fantasy.StepResult) error {
		s.trackUsage(stepResult.Usage)
		return nil
	}

	// The `id` in the callback is not reliable, it does not work for some providers.
	// Here we only need to distinguish the delta type, so we give numbers directly.
	call.OnTextDelta = func(_, text string) error {
		stream.WriteTLV(s.Output, stream.TagAssistantText, assembleId("t")+text)
		s.Output.Flush()
		return nil
	}
	call.OnReasoningDelta = func(_, text string) error {
		stream.WriteTLV(s.Output, stream.TagReasoning, assembleId("r")+text)
		s.Output.Flush()
		return nil
	}
	call.OnToolCall = func(tc fantasy.ToolCallContent) error {
		s.writeToolCall(tc.ToolName, tc.Input, tc.ToolCallID)
		s.Output.Flush()
		return nil
	}

	result, err := s.Agent.Stream(ctx, call)
	if err != nil {
		return nil, err
	}
	s.Output.Flush()

	return result, nil
}

func (s *Session) trackUsage(usage fantasy.Usage) {
	s.mu.Lock()
	s.TotalSpent.InputTokens += usage.InputTokens
	s.TotalSpent.OutputTokens += usage.OutputTokens
	s.TotalSpent.TotalTokens += usage.TotalTokens
	s.TotalSpent.ReasoningTokens += usage.ReasoningTokens
	// ContextTokens tracks the total context size (input tokens sent to API)
	// This is what counts toward provider context limits
	s.ContextTokens = usage.InputTokens
	s.mu.Unlock()
	s.sendSystemInfo()
}
