package tools

import (
	"context"
	"os"

	"charm.land/fantasy"
	"github.com/wallacegibbon/coreclaw/internal/util"
)

// EditFileInput represents the input for the edit_file tool
type EditFileInput struct {
	Path string `json:"path" description:"The path of the file to edit"`
	Diff string `json:"diff" description:"Unified diff content (--- a/... +++ b/... @@ ...)"`
}

// NewEditFileTool creates a tool for editing files using unified diff
func NewEditFileTool() fantasy.AgentTool {
	return fantasy.NewAgentTool(
		"edit_file",
		`Apply changes to a file using unified diff format.

CRITICAL: Read the file first to generate accurate diffs.

Format:
--- a/filepath
+++ b/filepath
@@ -old_start,count +new_start,count @@
 context line
-old line
+new line
 context line

Requirements:
- Line numbers are 1-indexed
- Context lines (with space prefix) must match file exactly
- Include 1-2 lines of context before/after each change
- Each hunk must end with a matching context line
- Use exact indentation/tabs from source file

Example:
--- a/test.go
+++ b/test.go
@@ -2,1 +2,1 @@
-func old()
+func new()
@@ -3,1 +3,1 @@
 }`,
		func(ctx context.Context, input EditFileInput, _ fantasy.ToolCall) (fantasy.ToolResponse, error) {
			if input.Path == "" {
				return fantasy.NewTextErrorResponse("path is required"), nil
			}
			if input.Diff == "" {
				return fantasy.NewTextErrorResponse("diff is required"), nil
			}

			// Read original content
			originalContent, err := os.ReadFile(input.Path)
			if err != nil {
				if !os.IsNotExist(err) {
					return fantasy.NewTextErrorResponse(err.Error()), nil
				}
				originalContent = []byte{}
			}

			// Parse and apply unified diff
			patchedContent, err := util.ApplyUnifiedDiff(string(originalContent), input.Diff)
			if err != nil {
				return fantasy.NewTextErrorResponse(err.Error()), nil
			}

			// Write back
			if err := os.WriteFile(input.Path, []byte(patchedContent), 0644); err != nil {
				return fantasy.NewTextErrorResponse(err.Error()), nil
			}

			return fantasy.NewTextResponse("File patched successfully"), nil
		},
	)
}
