package tools

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"os"
	"strconv"
	"strings"

	"github.com/alayacore/alayacore/internal/llm"
	"github.com/alayacore/alayacore/internal/llm/llmcompat"
)

// ReadFileInput represents the input for the read_file tool
type ReadFileInput struct {
	Path      string `json:"path"`
	StartLine string `json:"start_line"`
	EndLine   string `json:"end_line"`
}

// NewReadFileTool creates a tool for reading files
func NewReadFileTool() llm.Tool {
	schema := json.RawMessage(`{
		"type": "object",
		"properties": {
			"path": {
				"type": "string",
				"description": "The path of the file to read"
			},
			"start_line": {
				"type": "string",
				"description": "Optional: The starting line number (1-indexed)"
			},
			"end_line": {
				"type": "string",
				"description": "Optional: The ending line number (1-indexed)"
			}
		},
		"required": ["path"]
	}`)

	return llmcompat.NewTool(
		"read_file",
		"Read the contents of a file. Supports optional line range using start_line and end_line parameters (1-indexed).",
	).
		WithSchema(schema).
		WithExecute(func(_ context.Context, input json.RawMessage) (llm.ToolResultOutput, error) {
			var args ReadFileInput
			if err := json.Unmarshal(input, &args); err != nil {
				return llmcompat.NewTextErrorResponse("failed to parse input: " + err.Error()), nil
			}

			if args.Path == "" {
				return llmcompat.NewTextErrorResponse("path is required"), nil
			}

			content, err := os.ReadFile(args.Path)
			if err != nil {
				return llmcompat.NewTextErrorResponse(err.Error()), nil
			}

			if args.StartLine == "" && args.EndLine == "" {
				return llmcompat.NewTextResponse(string(content)), nil
			}

			startLine := 0
			if args.StartLine != "" {
				startLine, err = strconv.Atoi(args.StartLine)
				if err != nil {
					return llmcompat.NewTextErrorResponse("invalid start_line: must be a number"), nil
				}
				if startLine < 1 {
					return llmcompat.NewTextErrorResponse("start_line must be >= 1"), nil
				}
			}

			endLine := 0
			if args.EndLine != "" {
				endLine, err = strconv.Atoi(args.EndLine)
				if err != nil {
					return llmcompat.NewTextErrorResponse("invalid end_line: must be a number"), nil
				}
				if endLine < 1 {
					return llmcompat.NewTextErrorResponse("end_line must be >= 1"), nil
				}
			}

			if startLine > 0 && endLine > 0 && startLine > endLine {
				return llmcompat.NewTextErrorResponse("start_line must be <= end_line"), nil
			}

			lines, err := readLinesRange(bytes.NewReader(content), startLine, endLine)
			if err != nil {
				return llmcompat.NewTextErrorResponse(err.Error()), nil
			}

			return llmcompat.NewTextResponse(strings.Join(lines, "\n")), nil
		}).
		Build()
}

func readLinesRange(r *bytes.Reader, startLine, endLine int) ([]string, error) {
	scanner := bufio.NewScanner(r)
	var lines []string
	currentLine := 1

	for scanner.Scan() {
		if startLine > 0 && currentLine < startLine {
			currentLine++
			continue
		}

		if endLine > 0 && currentLine > endLine {
			break
		}

		lines = append(lines, scanner.Text())
		currentLine++
	}

	if err := scanner.Err(); err != nil {
		return nil, err
	}

	return lines, nil
}
