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
func (p *Processor) ProcessPrompt(ctx context.Context, prompt string, messages []fantasy.Message) (fantasy.Message, fantasy.Usage, error) {
	streamCall := fantasy.AgentStreamCall{
		Prompt: prompt,
	}
	if len(messages) > 0 {
		streamCall.Messages = messages
	}

	streamCall.OnTextStart = func(id string) error {
		return nil
	}

	streamCall.OnTextDelta = func(id, text string) error {
		stream.WriteTLV(p.Output, stream.TagAssistantText, text)
		p.Output.Flush()
		return nil
	}

	streamCall.OnTextEnd = func(id string) error {
		stream.WriteTLV(p.Output, stream.TagStreamGap, "")
		return nil
	}

	streamCall.OnReasoningStart = func(id string, reasoning fantasy.ReasoningContent) error {
		return nil
	}

	// Handle reasoning/thinking content (Anthropic)
	streamCall.OnReasoningDelta = func(id, text string) error {
		stream.WriteTLV(p.Output, stream.TagReasoning, text)
		p.Output.Flush()
		return nil
	}

	streamCall.OnReasoningEnd = func(id string, reasoning fantasy.ReasoningContent) error {
		stream.WriteTLV(p.Output, stream.TagStreamGap, "")
		return nil
	}

	streamCall.OnToolCall = func(tc fantasy.ToolCallContent) error {
		p.handleToolCall(tc)
		stream.WriteTLV(p.Output, stream.TagStreamGap, "")
		p.Output.Flush()
		return nil
	}

	streamCall.OnToolResult = func(tr fantasy.ToolResultContent) error {
		return nil
	}

	agentResult, err := p.Agent.Stream(ctx, streamCall)
	if err != nil {
		return fantasy.Message{}, fantasy.Usage{}, err
	}

	// Flush output buffer
	p.Output.Flush()

	// Extract assistant message from result

	// Some API protocol do not support thinking_text sign, it's better to fetch
	// the last message and treat it as the final answer.
	var assistantMsg fantasy.Message
	if agentResult != nil && len(agentResult.Steps) > 0 {
		lastStep := agentResult.Steps[len(agentResult.Steps)-1]
		for _, msg := range lastStep.Messages {
			if msg.Role == fantasy.MessageRoleAssistant {
				assistantMsg = msg
				break
			}
		}
	}

	return assistantMsg, agentResult.TotalUsage, nil
}

// extractPosixShellCommand extracts the command from posix_shell tool input JSON
func extractPosixShellCommand(input string) string {
	var posixShellInput struct {
		Command string `json:"command"`
	}
	if err := json.Unmarshal([]byte(input), &posixShellInput); err != nil {
		return ""
	}
	return posixShellInput.Command
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
	case "posix_shell":
		cmd := extractPosixShellCommand(tc.Input)
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
