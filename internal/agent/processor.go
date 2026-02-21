package agent

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"strings"
	"sync"

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
func (p *Processor) ProcessPrompt(ctx context.Context, prompt string, messages []fantasy.Message) (*fantasy.AgentResult, string, fantasy.Message, error) {
	streamCall := fantasy.AgentStreamCall{
		Prompt: prompt,
	}
	if len(messages) > 0 {
		streamCall.Messages = messages
	}

	responseText := &strings.Builder{}
	toolCommands := make(map[string]string)
	var toolCallMutex sync.Mutex

	var (
		suppressNewlines   = true
		lastCharWasNewline = true
	)

	streamCall.OnTextStart = func(id string) error {
		if !lastCharWasNewline {
			fmt.Println()
		}
		return nil
	}

	streamCall.OnTextDelta = func(id, text string) error {
		responseText.WriteString(text)

		// Skip leading newlines to keep output tight
		if suppressNewlines {
			if hasNonNewline(text) {
				suppressNewlines = false
			} else {
				return nil
			}
		}

		fmt.Print(terminal.Bright(text))
		if len(text) > 0 {
			lastCharWasNewline = text[len(text)-1] == '\n'
		}
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
		suppressNewlines = false

		if tc.ToolName == "bash" {
			cmd := extractBashCommand(tc.Input)
			if cmd != "" {
				toolCallMutex.Lock()
				toolCommands[tc.ToolCallID] = cmd
				toolCallMutex.Unlock()

				printCommand(cmd, lastCharWasNewline)
				lastCharWasNewline = false
			}
		}
		return nil
	}

	streamCall.OnToolResult = func(tr fantasy.ToolResultContent) error {
		toolCallMutex.Lock()
		cmd, exists := toolCommands[tr.ToolCallID]
		delete(toolCommands, tr.ToolCallID)
		toolCallMutex.Unlock()

		if exists {
			printResult(cmd, tr)
			lastCharWasNewline = true
			suppressNewlines = true
		}
		return nil
	}

	agentResult, err := p.Agent.Stream(ctx, streamCall)
	if err != nil {
		fmt.Fprintln(os.Stdout, terminal.Dim(fmt.Sprintf("Error: %v", err)))
		return nil, "", fantasy.Message{}, err
	}

	fmt.Println()

	assistantMsg := extractAssistantMessage(agentResult)
	return agentResult, responseText.String(), assistantMsg, nil
}

// hasNonNewline checks if text contains any non-newline characters
func hasNonNewline(text string) bool {
	for _, r := range text {
		if r != '\n' {
			return true
		}
	}
	return false
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
func printCommand(cmd string, lastCharWasNewline bool) {
	displayCmd := formatCommand(cmd)
	prefix := terminal.Dim("→ ")
	if lastCharWasNewline {
		fmt.Printf("%s%s", prefix, terminal.Green(displayCmd))
	} else {
		fmt.Printf("\n%s%s", prefix, terminal.Green(displayCmd))
	}
}

// printResult displays the command result with success/failure status
func printResult(cmd string, tr fantasy.ToolResultContent) {
	exitStatus := 0

	if tr.Result.GetType() == fantasy.ToolResultContentTypeError {
		if errResult, ok := fantasy.AsToolResultOutputType[fantasy.ToolResultOutputContentError](tr.Result); ok && errResult.Error != nil {
			errMsg := errResult.Error.Error()
			if _, err := fmt.Sscanf(errMsg, "[%d]", &exitStatus); err != nil {
				exitStatus = 1
			}
		}
	}

	var statusText string
	if exitStatus == 0 {
		statusText = terminal.Green("✓")
	} else {
		statusText = terminal.Red(fmt.Sprintf("✗ [%d]", exitStatus))
	}
	fmt.Printf(" %s\n", statusText)
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
