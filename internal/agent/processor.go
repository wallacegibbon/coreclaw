package agent

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"strings"

	"charm.land/fantasy"
	"github.com/charmbracelet/glamour"
	"github.com/wallacegibbon/coreclaw/internal/terminal"
)

// Processor handles prompt processing with streaming and markdown rendering
type Processor struct {
	Agent     fantasy.Agent
	Debug     bool
	NoMarkdown bool
	Quiet     bool
}

// NewProcessor creates a new prompt processor
func NewProcessor(agent fantasy.Agent) *Processor {
	return &Processor{
		Agent: agent,
	}
}

// ProcessPrompt handles a single prompt with streaming and optional markdown rendering
func (p *Processor) ProcessPrompt(ctx context.Context, prompt string, messages []fantasy.Message) (*fantasy.AgentResult, string, error) {
	if p.Debug && !p.Quiet {
		fmt.Fprintln(os.Stdout, terminal.Dim("\n>>> Sending request to API server"))
		fmt.Fprintln(os.Stdout, terminal.Blue(fmt.Sprintf("User Prompt: %s", prompt)))
		fmt.Fprintln(os.Stdout, terminal.Dim("Available Tools: bash"))
	}

	streamCall := fantasy.AgentStreamCall{
		Prompt: prompt,
	}
	if len(messages) > 0 {
		streamCall.Messages = messages
	}

	var responseText strings.Builder

	if p.Debug && !p.Quiet {
		p.setupDebugCallbacks(&streamCall)
	}

	streamCall.OnStepFinish = func(stepResult fantasy.StepResult) error {
		if p.Debug && !p.Quiet {
			fmt.Println()
			fmt.Fprintln(os.Stdout, terminal.Dim("<<< Step finished"))
		}
		return nil
	}

	streamCall.OnToolInputStart = func(id, toolName string) error {
		if p.Debug && !p.Quiet {
			fmt.Println()
			fmt.Fprintln(os.Stdout, terminal.Dim(fmt.Sprintf(">>> Tool invocation request: %s", toolName)))
		}
		return nil
	}

	streamCall.OnTextDelta = func(id, text string) error {
		if p.NoMarkdown {
			fmt.Print(terminal.Bright(text))
		}
		responseText.WriteString(text)
		return nil
	}

	agentResult, err := p.Agent.Stream(ctx, streamCall)
	if err != nil {
		fmt.Fprintln(os.Stdout, terminal.Dim(fmt.Sprintf("Error: %v", err)))
		return nil, "", err
	}

	if p.Debug && !p.Quiet {
		fmt.Println()
		fmt.Fprintln(os.Stdout, terminal.Dim("<<< Agent finished"))
	}

	if !p.NoMarkdown {
		p.renderMarkdown(responseText.String())
	} else {
		fmt.Println()
	}

	if p.Debug && !p.Quiet {
		fmt.Fprintln(os.Stdout, terminal.Dim(fmt.Sprintf("\nUsage: %d input tokens, %d output tokens, %d total tokens",
			agentResult.TotalUsage.InputTokens,
			agentResult.TotalUsage.OutputTokens,
			agentResult.TotalUsage.TotalTokens,
		)))
	}

	return agentResult, responseText.String(), nil
}

func (p *Processor) setupDebugCallbacks(streamCall *fantasy.AgentStreamCall) {
	streamCall.OnAgentStart = func() {
		fmt.Fprintln(os.Stdout, terminal.Dim(">>> Agent started"))
	}
	streamCall.OnStepStart = func(stepNumber int) error {
		fmt.Fprintln(os.Stdout, terminal.Dim(fmt.Sprintf(">>> Step %d started", stepNumber)))
		return nil
	}
	streamCall.OnToolCall = func(toolCall fantasy.ToolCallContent) error {
		var input map[string]any
		json.Unmarshal([]byte(toolCall.Input), &input)
		fmt.Fprintln(os.Stdout, terminal.Dim(fmt.Sprintf("  Input: %+v", input)))
		return nil
	}
	streamCall.OnToolResult = func(result fantasy.ToolResultContent) error {
		fmt.Fprintln(os.Stdout, terminal.Dim("<<< Tool result received"))
		switch p := result.Result.(type) {
		case fantasy.ToolResultOutputContentText:
			fmt.Fprintln(os.Stdout, terminal.Yellow(fmt.Sprintf("  Output: %s", p.Text)))
		case fantasy.ToolResultOutputContentError:
			fmt.Fprintln(os.Stdout, terminal.Dim(fmt.Sprintf("  Error: %s", p.Error)))
		}
		return nil
	}
}

func (p *Processor) renderMarkdown(text string) {
	renderer, err := glamour.NewTermRenderer(
		glamour.WithAutoStyle(),
		glamour.WithWordWrap(80),
	)
	if err != nil {
		fmt.Fprintln(os.Stdout, terminal.Dim(fmt.Sprintf("Failed to create markdown renderer: %v", err)))
		fmt.Print(terminal.Bright(text))
		fmt.Println()
		return
	}

	rendered, err := renderer.Render(text)
	if err != nil {
		fmt.Fprintln(os.Stdout, terminal.Dim(fmt.Sprintf("Failed to render markdown: %v", err)))
		fmt.Print(terminal.Bright(text))
		fmt.Println()
		return
	}

	fmt.Print(rendered)
}
