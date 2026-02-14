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

			// Escape newlines and tabs for display
			displayCmd := strings.ReplaceAll(cmd, "\n", "\\n")
			displayCmd = strings.ReplaceAll(displayCmd, "\t", "\\t")
			fmt.Fprint(os.Stderr, terminal.Green("→ "+displayCmd+"\n"))

			execCmd := exec.CommandContext(ctx, "bash", "-c", cmd)
			output, err := execCmd.CombinedOutput()
			if err != nil {
				return fantasy.NewTextErrorResponse(string(output)), nil
			}

			return fantasy.NewTextResponse(string(output)), nil
		},
	)
}
