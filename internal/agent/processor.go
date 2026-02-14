package agent

import (
	"context"
	"fmt"
	"os"
	"strings"

	"charm.land/fantasy"
	"github.com/charmbracelet/glamour"
	"github.com/wallacegibbon/coreclaw/internal/terminal"
)

// Processor handles prompt processing with streaming and markdown rendering
type Processor struct {
	Agent      fantasy.Agent
	NoMarkdown bool
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

	if !p.NoMarkdown {
		p.renderMarkdown(responseText.String())
	} else {
		fmt.Println()
	}

	return agentResult, responseText.String(), nil
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
