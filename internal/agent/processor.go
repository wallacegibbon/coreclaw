package agent

import (
	"context"
	"encoding/json"
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

// ProcessPrompt handles a single prompt with streaming
func (p *Processor) ProcessPrompt(ctx context.Context, prompt string, messages []fantasy.Message) (*fantasy.AgentResult, string, fantasy.Message, fantasy.Usage, error) {
	streamCall := fantasy.AgentStreamCall{
		Prompt: prompt,
	}
	if len(messages) > 0 {
		streamCall.Messages = messages
	}

	responseText := &strings.Builder{}

	streamCall.OnTextStart = func(id string) error {
		fmt.Println()
		return nil
	}

	streamCall.OnTextDelta = func(id, text string) error {
		responseText.WriteString(text)
		fmt.Print(terminal.Bright(text))
		return nil
	}

	// Handle reasoning/thinking content (Anthropic)
	streamCall.OnReasoningDelta = func(id, text string) error {
		fmt.Print(terminal.Dim(text))
		return nil
	}

	streamCall.OnReasoningEnd = func(id string, reasoning fantasy.ReasoningContent) error {
		fmt.Println()
		return nil
	}

	streamCall.OnToolCall = func(tc fantasy.ToolCallContent) error {
		if tc.ToolName == "bash" {
			cmd := extractBashCommand(tc.Input)
			if cmd != "" {
				printCommand(cmd)
			}
		}
		return nil
	}

	streamCall.OnToolResult = func(tr fantasy.ToolResultContent) error {
		// Print tool output from beginning of line
		fmt.Println()

		// Extract text from tool result
		var text string
		switch result := tr.Result.(type) {
		case fantasy.ToolResultOutputContentText:
			text = result.Text
		case fantasy.ToolResultOutputContentError:
			text = result.Error.Error()
		default:
			text = fmt.Sprintf("%v", tr.Result)
		}

		fmt.Print(terminal.Dim(text))
		return nil
	}

	agentResult, err := p.Agent.Stream(ctx, streamCall)
	if err != nil {
		fmt.Fprintln(os.Stdout, terminal.Dim(fmt.Sprintf("Error: %v", err)))
		return nil, "", fantasy.Message{}, fantasy.Usage{}, err
	}

	assistantMsg := extractAssistantMessage(agentResult)
	return agentResult, responseText.String(), assistantMsg, agentResult.TotalUsage, nil
}

// Summarize handles summarizing the conversation history
func (p *Processor) Summarize(ctx context.Context, messages []fantasy.Message) (string, fantasy.Message, fantasy.Usage, error) {
	summarizePrompt := "Please summarize the conversation above in a concise manner. Return ONLY the summary, no introductions or explanations."

	streamCall := fantasy.AgentStreamCall{
		Prompt: summarizePrompt,
	}
	if len(messages) > 0 {
		streamCall.Messages = messages
	}

	responseText := &strings.Builder{}

	streamCall.OnTextStart = func(id string) error {
		return nil
	}

	streamCall.OnTextDelta = func(id, text string) error {
		responseText.WriteString(text)
		fmt.Print(terminal.Bright(text))
		return nil
	}

	streamCall.OnReasoningDelta = func(id, text string) error {
		fmt.Print(terminal.Dim(text))
		return nil
	}

	streamCall.OnReasoningEnd = func(id string, reasoning fantasy.ReasoningContent) error {
		fmt.Println()
		return nil
	}

	streamCall.OnToolCall = func(tc fantasy.ToolCallContent) error {
		return nil
	}

	streamCall.OnToolResult = func(tr fantasy.ToolResultContent) error {
		return nil
	}

	agentResult, err := p.Agent.Stream(ctx, streamCall)
	if err != nil {
		return "", fantasy.Message{}, fantasy.Usage{}, err
	}

	var usage fantasy.Usage
	if agentResult != nil {
		usage = agentResult.TotalUsage
	}

	// Create a summary message that replaces the conversation
	summaryMsg := fantasy.Message{
		Role:    fantasy.MessageRoleUser,
		Content: []fantasy.MessagePart{fantasy.TextPart{Text: responseText.String()}},
	}

	return responseText.String(), summaryMsg, usage, nil
}

// extractBashCommand extracts the command from bash tool input JSON
func extractBashCommand(input string) string {
	var bashInput struct {
		Command string `json:"command"`
	}
	if err := json.Unmarshal([]byte(input), &bashInput); err != nil {
		return ""
	}
	return bashInput.Command
}

// formatCommand formats a command for display (escape newlines and tabs)
func formatCommand(cmd string) string {
	cmd = strings.ReplaceAll(cmd, "\n", "\\n")
	cmd = strings.ReplaceAll(cmd, "\t", "\\t")
	return cmd
}

// printCommand displays a command with arrow prefix
func printCommand(cmd string) {
	displayCmd := formatCommand(cmd)
	prefix := terminal.Dim("→ ")
	fmt.Printf("%s%s\n", prefix, terminal.Green(displayCmd))
}

// extractAssistantMessage extracts the assistant message from agent result
func extractAssistantMessage(agentResult *fantasy.AgentResult) fantasy.Message {
	if agentResult == nil || len(agentResult.Steps) == 0 {
		return fantasy.Message{}
	}

	lastStep := agentResult.Steps[len(agentResult.Steps)-1]
	for _, msg := range lastStep.Messages {
		if msg.Role == fantasy.MessageRoleAssistant {
			return msg
		}
	}
	return fantasy.Message{}
}
