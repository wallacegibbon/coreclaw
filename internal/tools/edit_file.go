package tools

import (
	"context"
	"os"
	"os/exec"
	"strings"

	"charm.land/fantasy"
)

// EditFileInput represents the input for the edit_file tool
type EditFileInput struct {
	Path string `json:"path" description:"The path of the file to create or edit"`
	Diff string `json:"diff" description:"Unified diff to apply to the file"`
}

// NewEditFileTool creates a tool for editing/creating files using diffs
func NewEditFileTool() fantasy.AgentTool {
	return fantasy.NewAgentTool(
		"edit_file",
		"Apply a unified diff to create or edit a file",
		func(ctx context.Context, input EditFileInput, _ fantasy.ToolCall) (fantasy.ToolResponse, error) {
			if input.Path == "" {
				return fantasy.NewTextErrorResponse("path is required"), nil
			}

			tmpFile := input.Path + ".tmp"

			// If original file exists, copy it to temp
			if original, err := os.ReadFile(input.Path); err == nil {
				if err := os.WriteFile(tmpFile, original, 0644); err != nil {
					return fantasy.NewTextErrorResponse(err.Error()), nil
				}
			} else if !os.IsNotExist(err) {
				return fantasy.NewTextErrorResponse(err.Error()), nil
			}

			// Apply diff using patch command
			cmd := exec.CommandContext(ctx, "bash", "-c", "patch -u - "+tmpFile+" < /dev/stdin")
			cmd.Stdin = strings.NewReader(input.Diff)
			output, err := cmd.CombinedOutput()

			// Clean up temp file
			os.Remove(tmpFile)

			if err != nil {
				return fantasy.NewTextErrorResponse(string(output)), nil
			}

			// Read patched content
			var result []byte
			if _, err := os.Stat(tmpFile); err == nil {
				result, err = os.ReadFile(tmpFile)
				os.Remove(tmpFile)
			} else {
				result, err = os.ReadFile(input.Path)
			}

			if err != nil {
				return fantasy.NewTextErrorResponse(err.Error()), nil
			}

			// Write final result
			if err := os.WriteFile(input.Path, result, 0644); err != nil {
				return fantasy.NewTextErrorResponse(err.Error()), nil
			}

			return fantasy.NewTextResponse("File updated successfully"), nil
		},
	)
}
