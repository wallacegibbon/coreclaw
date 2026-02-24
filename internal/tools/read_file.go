package tools

import (
	"context"
	"os"

	"charm.land/fantasy"
)

// ReadFileInput represents the input for the read_file tool
type ReadFileInput struct {
	Path string `json:"path" description:"The path of the file to read"`
}

// NewReadFileTool creates a tool for reading files
func NewReadFileTool() fantasy.AgentTool {
	return fantasy.NewAgentTool(
		"read_file",
		"Read the contents of a file",
		func(ctx context.Context, input ReadFileInput, _ fantasy.ToolCall) (fantasy.ToolResponse, error) {
			if input.Path == "" {
				return fantasy.NewTextErrorResponse("path is required"), nil
			}

			content, err := os.ReadFile(input.Path)
			if err != nil {
				return fantasy.NewTextErrorResponse(err.Error()), nil
			}

			return fantasy.NewTextResponse(string(content)), nil
		},
	)
}
