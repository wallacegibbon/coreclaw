package tools

import (
	"context"
	"os"

	"charm.land/fantasy"
)

// EditFileInput represents the input for the edit_file tool
type EditFileInput struct {
	Path    string `json:"path" description:"The path of the file to create or edit"`
	Content string `json:"content" description:"The content to write to the file"`
}

// NewEditFileTool creates a tool for editing/creating files
func NewEditFileTool() fantasy.AgentTool {
	return fantasy.NewAgentTool(
		"edit_file",
		"Create or edit a file with the given content",
		func(ctx context.Context, input EditFileInput, _ fantasy.ToolCall) (fantasy.ToolResponse, error) {
			if input.Path == "" {
				return fantasy.NewTextErrorResponse("path is required"), nil
			}

			err := os.WriteFile(input.Path, []byte(input.Content), 0644)
			if err != nil {
				return fantasy.NewTextErrorResponse(err.Error()), nil
			}

			return fantasy.NewTextResponse("File written successfully"), nil
		},
	)
}
