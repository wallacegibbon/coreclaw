package tools

import (
	"context"
	"encoding/json"
	"os"

	"github.com/alayacore/alayacore/internal/llm"
	"github.com/alayacore/alayacore/internal/llm/llmcompat"
)

// WriteFileInput represents the input for the write_file tool
type WriteFileInput struct {
	Path    string `json:"path"`
	Content string `json:"content"`
}

// NewWriteFileTool creates a tool for writing files
func NewWriteFileTool() llm.Tool {
	schema := json.RawMessage(`{
		"type": "object",
		"properties": {
			"path": {
				"type": "string",
				"description": "The path of the file to write"
			},
			"content": {
				"type": "string",
				"description": "The content to write to the file"
			}
		},
		"required": ["path", "content"]
	}`)

	return llmcompat.NewTool(
		"write_file",
		"Create a new file or replace the entire content of an existing file.",
	).
		WithSchema(schema).
		WithExecute(func(_ context.Context, input json.RawMessage) (llm.ToolResultOutput, error) {
			var args WriteFileInput
			if err := json.Unmarshal(input, &args); err != nil {
				return llmcompat.NewTextErrorResponse("failed to parse input: " + err.Error()), nil
			}

			if args.Path == "" {
				return llmcompat.NewTextErrorResponse("path is required"), nil
			}

			if err := os.WriteFile(args.Path, []byte(args.Content), 0644); err != nil {
				return llmcompat.NewTextErrorResponse(err.Error()), nil
			}

			return llmcompat.NewTextResponse("File written successfully"), nil
		}).
		Build()
}
