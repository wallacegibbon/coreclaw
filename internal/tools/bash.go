package tools

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"strings"

	"charm.land/fantasy"
	"github.com/wallacegibbon/coreclaw/internal/terminal"
)

// BashInput represents the input for the bash tool
type BashInput struct {
	Command string `json:"command" description:"The bash command to execute"`
}

// NewBashTool creates a new bash tool for executing shell commands
func NewBashTool() fantasy.AgentTool {
	return fantasy.NewAgentTool(
		"bash",
		"Execute a bash command in the shell",
		func(ctx context.Context, input BashInput, _ fantasy.ToolCall) (fantasy.ToolResponse, error) {
			cmd := input.Command
			if cmd == "" {
				return fantasy.NewTextErrorResponse("command is required"), nil
			}

			execCmd := exec.CommandContext(ctx, "bash", "-c", cmd)
			output, err := execCmd.CombinedOutput()

			// Get exit status
			exitStatus := 0
			if err != nil {
				if exitErr, ok := err.(*exec.ExitError); ok {
					exitStatus = exitErr.ExitCode()
				}
			}

			// Escape newlines and tabs for display
			displayCmd := strings.ReplaceAll(cmd, "\n", "\\n")
			displayCmd = strings.ReplaceAll(displayCmd, "\t", "\\t")

			// Print command with exit status
			if exitStatus == 0 {
				fmt.Fprint(os.Stderr, terminal.Green(fmt.Sprintf("✓ %s\n", displayCmd)))
			} else {
				fmt.Fprint(os.Stderr, terminal.Red(fmt.Sprintf("● %s [%d]\n", displayCmd, exitStatus)))
			}

			if err != nil {
				return fantasy.NewTextErrorResponse(string(output)), nil
			}

			return fantasy.NewTextResponse(string(output)), nil
		},
	)
}
