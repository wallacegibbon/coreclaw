package tools

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"strings"

	"github.com/alayacore/alayacore/internal/llm"
	"github.com/alayacore/alayacore/internal/llm/llmcompat"
)

// EditFileInput represents the input for the edit_file tool
type EditFileInput struct {
	Path      string `json:"path"`
	OldString string `json:"old_string"`
	NewString string `json:"new_string"`
}

// NewEditFileTool creates a tool for editing files using search/replace
func NewEditFileTool() llm.Tool {
	schema := json.RawMessage(`{
		"type": "object",
		"properties": {
			"path": {
				"type": "string",
				"description": "The path of the file to edit"
			},
			"old_string": {
				"type": "string",
				"description": "The exact text to find and replace (must match exactly)"
			},
			"new_string": {
				"type": "string",
				"description": "The replacement text"
			}
		},
		"required": ["path", "old_string", "new_string"]
	}`)

	return llmcompat.NewTool(
		"edit_file",
		`Apply a search/replace edit to a file.

CRITICAL: Read the file first to get the exact text including whitespace.

Parameters:
- path: The file path to edit
- old_string: The exact text to find (must match exactly including all whitespace, indentation, newlines)
- new_string: The replacement text

Requirements:
- old_string must match EXACTLY (every space, tab, newline, character)
- Include 3-5 lines of context to make old_string unique
- If old_string appears multiple times, the edit fails
- To replace multiple occurrences, make separate calls with unique context

Example:
{
  "path": "test.go",
  "old_string": "func old() {\n    doSomething()\n}",
  "new_string": "func new() {\n    doSomethingElse()\n}"
}`,
	).
		WithSchema(schema).
		WithExecute(func(_ context.Context, input json.RawMessage) (llm.ToolResultOutput, error) {
			var args EditFileInput
			if err := json.Unmarshal(input, &args); err != nil {
				return llmcompat.NewTextErrorResponse("failed to parse input: " + err.Error()), nil
			}

			if args.Path == "" {
				return llmcompat.NewTextErrorResponse("path is required"), nil
			}
			if args.OldString == "" {
				return llmcompat.NewTextErrorResponse("old_string is required"), nil
			}

			// Read original content
			originalContent, err := os.ReadFile(args.Path)
			if err != nil {
				if os.IsNotExist(err) {
					return llmcompat.NewTextErrorResponse(fmt.Sprintf("file not found: %s", args.Path)), nil
				}
				return llmcompat.NewTextErrorResponse(err.Error()), nil
			}

			originalStr := string(originalContent)

			// Count occurrences
			count := strings.Count(originalStr, args.OldString)
			if count == 0 {
				return llmcompat.NewTextErrorResponse(fmt.Sprintf("old_string not found in file. Make sure to copy the exact text including all whitespace and indentation.\n\nSearched for:\n%q", args.OldString)), nil
			}
			if count > 1 {
				return llmcompat.NewTextErrorResponse(fmt.Sprintf("old_string found %d times in file. Include more surrounding context to make it unique, or use a different portion of text.", count)), nil
			}

			// Apply the replacement
			newContent := strings.Replace(originalStr, args.OldString, args.NewString, 1)

			// Write back
			if err := os.WriteFile(args.Path, []byte(newContent), 0644); err != nil {
				return llmcompat.NewTextErrorResponse(err.Error()), nil
			}

			return llmcompat.NewTextResponse("File edited successfully"), nil
		}).
		Build()
}
