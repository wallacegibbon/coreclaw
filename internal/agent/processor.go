package agent

import (
	"context"
	"fmt"
	"os"
	"strings"

	"charm.land/fantasy"
	"github.com/wallacegibbon/coreclaw/internal/terminal"
)

// Processor handles prompt processing with streaming
type Processor struct {
	Agent fantasy.Agent
}

// NewProcessor creates a new prompt processor
func NewProcessor(agent fantasy.Agent) *Processor {
	return &Processor{
		Agent: agent,
	}
}

// ProcessPrompt handles a single prompt with streaming and optional markdown rendering
func (p *Processor) ProcessPrompt(ctx context.Context, prompt string, messages []fantasy.Message) (*fantasy.AgentResult, string, error) {
	streamCall := fantasy.AgentStreamCall{
		Prompt: prompt,
	}
	if len(messages) > 0 {
		streamCall.Messages = messages
	}

	var responseText strings.Builder

	streamCall.OnTextDelta = func(id, text string) error {
		responseText.WriteString(text)
		fmt.Print(terminal.Bright(text))
		return nil
	}

	streamCall.OnToolCall = func(tc fantasy.ToolCallContent) error {
		fmt.Println()
		return nil
	}

	agentResult, err := p.Agent.Stream(ctx, streamCall)
	if err != nil {
		fmt.Fprintln(os.Stdout, terminal.Dim(fmt.Sprintf("Error: %v", err)))
		return nil, "", err
	}

	fmt.Println()

	return agentResult, responseText.String(), nil
}
