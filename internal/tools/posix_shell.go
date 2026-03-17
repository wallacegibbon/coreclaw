package tools

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"strings"

	"github.com/alayacore/alayacore/internal/llm"
	"github.com/alayacore/alayacore/internal/llm/llmcompat"
	"mvdan.cc/sh/v3/expand"
	"mvdan.cc/sh/v3/interp"
	"mvdan.cc/sh/v3/syntax"
)

// PosixShellInput represents the input for the posix_shell tool
type PosixShellInput struct {
	Command string `json:"command"`
}

// NewPosixShellTool creates a new posix_shell tool for executing shell commands
func NewPosixShellTool() llm.Tool {
	schema := json.RawMessage(`{
		"type": "object",
		"properties": {
			"command": {
				"type": "string",
				"description": "The shell command to execute"
			}
		},
		"required": ["command"]
	}`)

	return llmcompat.NewTool(
		"posix_shell",
		`Execute a shell command.

Rules:
- Use POSIX-compliant shell syntax only (no bash/zsh-specific features)
- Prefer simple, standard commands over complex pipelines
- Quote filenames with spaces or special characters
- Check command output for errors before proceeding
- Clean up temporary files when done`,
	).
		WithSchema(schema).
		WithExecute(func(ctx context.Context, input json.RawMessage) (llm.ToolResultOutput, error) {
			var args PosixShellInput
			if err := json.Unmarshal(input, &args); err != nil {
				return llmcompat.NewTextErrorResponse("failed to parse input: " + err.Error()), nil
			}

			cmd := args.Command
			if cmd == "" {
				return llmcompat.NewTextErrorResponse("command is required"), nil
			}

			var stdout, stderr bytes.Buffer

			parser := syntax.NewParser()
			prog, err := parser.Parse(strings.NewReader(cmd), "")
			if err != nil {
				return llmcompat.NewTextErrorResponse("parse error: " + err.Error()), nil
			}

			cwd, _ := os.Getwd()
			runner, err := interp.New(
				interp.Dir(cwd),
				interp.Env(expand.ListEnviron(os.Environ()...)),
				interp.StdIO(os.Stdin, &stdout, &stderr),
				interp.ExecHandlers(),
			)
			if err != nil {
				return llmcompat.NewTextErrorResponse("failed to create runner: " + err.Error()), nil
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
					return llmcompat.NewTextErrorResponse(fmt.Sprintf("[%d] %s", exitStatus, output)), nil
				}
				if output != "" {
					return llmcompat.NewTextErrorResponse(fmt.Sprintf("%s\n%s", err.Error(), output)), nil
				}
				return llmcompat.NewTextErrorResponse(err.Error()), nil
			}

			return llmcompat.NewTextResponse(output), nil
		}).
		Build()
}
