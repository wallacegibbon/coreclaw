package tools

import (
	"context"
	"fmt"
	"os/exec"

	"charm.land/fantasy"
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
				// Include exit status in error response
				return fantasy.NewTextErrorResponse(fmt.Sprintf("[%d] %s", exitStatus, string(output))), nil
			}

			return fantasy.NewTextResponse(string(output)), nil
		},
	)
}
