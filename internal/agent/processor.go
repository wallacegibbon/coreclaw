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
		return nil
	}

	streamCall.OnTextStart = func(id string) error {
		fmt.Println()
		return nil
	}

	streamCall.OnToolCall = func(tc fantasy.ToolCallContent) error {
		printToolCall(tc)
		return nil
	}

	streamCall.OnToolResult = func(tr fantasy.ToolResultContent) error {
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
		Role:    fantasy.MessageRoleAssistant,
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

// extractSkillName extracts the skill name from activate_skill tool input JSON
func extractSkillName(input string) string {
	var skillInput struct {
		Name string `json:"name"`
	}
	if err := json.Unmarshal([]byte(input), &skillInput); err != nil {
		return ""
	}
	return skillInput.Name
}

// extractReadFilePath extracts the path from read_file tool input JSON
func extractReadFilePath(input string) string {
	var fileInput struct {
		Path string `json:"path"`
	}
	if err := json.Unmarshal([]byte(input), &fileInput); err != nil {
		return ""
	}
	return fileInput.Path
}

// extractWriteFilePath extracts the path from write_file tool input JSON
func extractWriteFilePath(input string) string {
	var fileInput struct {
		Path string `json:"path"`
	}
	if err := json.Unmarshal([]byte(input), &fileInput); err != nil {
		return ""
	}
	return fileInput.Path
}

// formatCommon formats a command for display (escape newlines and tabs)
func formatCommon(cmd string) string {
	cmd = strings.ReplaceAll(cmd, "\n", "\\n")
	cmd = strings.ReplaceAll(cmd, "\t", "\\t")
	return cmd
}

// printToolCall displays tool call info in uniform format
func printToolCall(tc fantasy.ToolCallContent) {
	switch tc.ToolName {
	case "bash":
		cmd := extractBashCommand(tc.Input)
		if cmd != "" {
			displayCmd := formatCommon(cmd)
			fmt.Printf("\n%s %s: %s\n", terminal.Yellow("→"), terminal.Yellow("bash"), terminal.Green(displayCmd))
		}
	case "activate_skill":
		name := extractSkillName(tc.Input)
		if name != "" {
			fmt.Printf("\n%s %s: %s\n", terminal.Yellow("→"), terminal.Yellow("activate_skill"), terminal.Green(name))
		}
	case "read_file":
		path := extractReadFilePath(tc.Input)
		if path != "" {
			fmt.Printf("\n%s %s: %s\n", terminal.Yellow("→"), terminal.Yellow("read_file"), terminal.Green(path))
		}
	case "write_file":
		path := extractWriteFilePath(tc.Input)
		if path != "" {
			fmt.Printf("\n%s %s: %s\n", terminal.Yellow("→"), terminal.Yellow("write_file"), terminal.Green(path))
		}
	}
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
