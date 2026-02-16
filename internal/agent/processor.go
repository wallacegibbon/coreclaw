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
func (p *Processor) ProcessPrompt(ctx context.Context, prompt string, messages []fantasy.Message) (*fantasy.AgentResult, string, error) {
	streamCall := fantasy.AgentStreamCall{
		Prompt: prompt,
	}
	if len(messages) > 0 {
		streamCall.Messages = messages
	}

	var responseText strings.Builder
	var lastCharWasNewline bool

	streamCall.OnTextDelta = func(id, text string) error {
		responseText.WriteString(text)
		// Skip leading newlines if we're already at start of line (to avoid blank lines)
		printText := text
		if lastCharWasNewline && len(text) > 0 {
			// Find first non-newline character
			firstNonNewline := 0
			for firstNonNewline < len(text) && text[firstNonNewline] == '\n' {
				firstNonNewline++
			}
			if firstNonNewline > 0 {
				// Skip leading newlines, but if the entire text is newlines, return early
				if firstNonNewline == len(text) {
					return nil
				}
				printText = text[firstNonNewline:]
			}
		}
		fmt.Print(terminal.Bright(printText))
		if len(printText) > 0 {
			lastCharWasNewline = (printText[len(printText)-1] == '\n')
		}
		return nil
	}

	// Track tool calls to update status on the same line
	toolCommands := make(map[string]string)
	var toolCallMutex sync.Mutex

	streamCall.OnToolCall = func(tc fantasy.ToolCallContent) error {
		// Extract command from tool input (Input is JSON string)
		if tc.ToolName == "bash" {
			var bashInput struct {
				Command string `json:"command"`
			}
			if err := json.Unmarshal([]byte(tc.Input), &bashInput); err == nil {
				toolCallMutex.Lock()
				toolCommands[tc.ToolCallID] = bashInput.Command
				toolCallMutex.Unlock()
				// Print command with arrow prefix (status will be appended when it finishes)
				displayCmd := strings.ReplaceAll(bashInput.Command, "\n", "\\n")
				displayCmd = strings.ReplaceAll(displayCmd, "\t", "\\t")
				// Only add newline if last text didn't end with one
				if lastCharWasNewline {
					fmt.Printf("  %s%s", terminal.Dim("→ "), terminal.Green(displayCmd))
				} else {
					fmt.Printf("\n  %s%s", terminal.Dim("→ "), terminal.Green(displayCmd))
				}
			}
		}
		return nil
	}

	streamCall.OnToolResult = func(tr fantasy.ToolResultContent) error {
		toolCallMutex.Lock()
		cmdStr, exists := toolCommands[tr.ToolCallID]
		delete(toolCommands, tr.ToolCallID)
		toolCallMutex.Unlock()

		if exists {
			displayCmd := strings.ReplaceAll(cmdStr, "\n", "\\n")
			displayCmd = strings.ReplaceAll(displayCmd, "\t", "\\t")

			// Determine status based on result type
			exitStatus := 0
			if tr.Result.GetType() == fantasy.ToolResultContentTypeError {
				if errResult, ok := fantasy.AsToolResultOutputType[fantasy.ToolResultOutputContentError](tr.Result); ok && errResult.Error != nil {
					errMsg := errResult.Error.Error()
					if parsed, err := fmt.Sscanf(errMsg, "[%d]", &exitStatus); err != nil || parsed != 1 {
						exitStatus = 1
					}
				}
			}

			// Update the line with final status
			// Just append the status after the command (no carriage return needed)
			var statusText string
			if exitStatus == 0 {
				statusText = terminal.Green("✓")
			} else {
				statusText = terminal.Red(fmt.Sprintf("● [%d]", exitStatus))
			}
			fmt.Printf(" %s\n", statusText)
		}
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
