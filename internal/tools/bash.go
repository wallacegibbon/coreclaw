package tools

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"os"
	"strings"

	"charm.land/fantasy"
	"mvdan.cc/sh/v3/expand"
	"mvdan.cc/sh/v3/interp"
	"mvdan.cc/sh/v3/syntax"
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

			var stdout, stderr bytes.Buffer

			parser := syntax.NewParser()
			prog, err := parser.Parse(strings.NewReader(cmd), "")
			if err != nil {
				return fantasy.NewTextErrorResponse("parse error: " + err.Error()), nil
			}

			runner, err := interp.New(
				interp.Dir("/"),
				interp.Env(expand.ListEnviron(os.Environ()...)),
				interp.StdIO(os.Stdin, &stdout, &stderr),
			)
			if err != nil {
				return fantasy.NewTextErrorResponse("failed to create runner: " + err.Error()), nil
			}

			err = runner.Run(ctx, prog)
			output := stdout.String()
			if stderr.Len() > 0 {
				if output != "" {
					output += "\n"
				}
				output += stderr.String()
			}

			if err != nil {
				var exitStatus interp.ExitStatus
				if errors.As(err, &exitStatus) {
					return fantasy.NewTextErrorResponse(fmt.Sprintf("[%d] %s", exitStatus, output)), nil
				}
				if output != "" {
					return fantasy.NewTextErrorResponse(fmt.Sprintf("%s\n%s", err.Error(), output)), nil
				}
				return fantasy.NewTextErrorResponse(err.Error()), nil
			}

			return fantasy.NewTextResponse(output), nil
		},
	)
}
