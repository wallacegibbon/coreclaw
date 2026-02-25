package agent

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"charm.land/fantasy"
	"github.com/wallacegibbon/coreclaw/internal/stream"
)

// Processor handles prompt processing with streaming
type Processor struct {
	Agent  fantasy.Agent
	Output stream.Output
	Input  stream.Input
}

// NewProcessor creates a new prompt processor
func NewProcessor(agent fantasy.Agent) *Processor {
	return &Processor{
		Agent:  agent,
		Output: &stream.NopOutput{},
		Input:  &stream.NopInput{},
	}
}

// NewProcessorWithIO creates a new prompt processor with custom input/output streams
func NewProcessorWithIO(agent fantasy.Agent, input stream.Input, output stream.Output) *Processor {
	return &Processor{
		Agent:  agent,
		Output: output,
		Input:  input,
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
		stream.WriteTLV(p.Output, stream.TagText, text)
		p.Output.Flush()
		return nil
	}

	// Handle reasoning/thinking content (Anthropic)
	streamCall.OnReasoningDelta = func(id, text string) error {
		stream.WriteTLV(p.Output, stream.TagReasoning, text)
		p.Output.Flush()
		return nil
	}

	streamCall.OnReasoningEnd = func(id string, reasoning fantasy.ReasoningContent) error {
		return nil
	}

	streamCall.OnTextStart = func(id string) error {
		stream.WriteTLV(p.Output, stream.TagText, "\n")
		p.Output.Flush()
		return nil
	}

	streamCall.OnToolCall = func(tc fantasy.ToolCallContent) error {
		p.handleToolCall(tc)
		p.Output.Flush()
		return nil
	}

	streamCall.OnToolResult = func(tr fantasy.ToolResultContent) error {
		return nil
	}

	agentResult, err := p.Agent.Stream(ctx, streamCall)
	if err != nil {
		stream.WriteTLV(p.Output, stream.TagError, fmt.Sprintf("Error: %v\n", err))
		p.Output.Flush()
		return nil, "", fantasy.Message{}, fantasy.Usage{}, err
	}

	// Flush output buffer
	p.Output.Flush()

	assistantMsg := extractAssistantMessage(agentResult)
	return agentResult, responseText.String(), assistantMsg, agentResult.TotalUsage, nil
}

// Summarize handles summarizing the conversation history
func (p *Processor) Summarize(ctx context.Context, messages []fantasy.Message) (string, fantasy.Message, fantasy.Usage, error) {
	summarizePrompt := "Please summarize the conversation above in a concise manner. Return ONLY the summary, no introductions or explanations."

	_, responseText, assistantMsg, usage, err := p.ProcessPrompt(ctx, summarizePrompt, messages)
	return responseText, assistantMsg, usage, err
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

// handleToolCall sends tool call info via TLV as text
func (p *Processor) handleToolCall(tc fantasy.ToolCallContent) {
	var value string

	switch tc.ToolName {
	case "bash":
		cmd := extractBashCommand(tc.Input)
		if cmd != "" {
			displayCmd := formatCommon(cmd)
			value = fmt.Sprintf("%s: %s", tc.ToolName, displayCmd)
		}
	case "activate_skill":
		name := extractSkillName(tc.Input)
		if name != "" {
			value = fmt.Sprintf("%s: %s", tc.ToolName, name)
		}
	case "read_file":
		path := extractReadFilePath(tc.Input)
		if path != "" {
			value = fmt.Sprintf("%s: %s", tc.ToolName, path)
		}
	case "write_file":
		path := extractWriteFilePath(tc.Input)
		if path != "" {
			value = fmt.Sprintf("%s: %s", tc.ToolName, path)
		}
	}

	if value != "" {
		stream.WriteTLV(p.Output, stream.TagTool, value)
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
