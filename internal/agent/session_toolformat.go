package agent

import (
	"encoding/json"
	"fmt"
	"strings"
)

//nolint:gocyclo // tool formatting requires handling many tool types
func formatToolCall(toolName, input string) string {
	var fields map[string]interface{}
	if err := json.Unmarshal([]byte(input), &fields); err != nil {
		return ""
	}

	switch toolName {
	case "posix_shell":
		if cmd, ok := fields["command"].(string); ok {
			return fmt.Sprintf("%s: %s", toolName, escapeNewlines(cmd))
		}
	case "activate_skill":
		if name, ok := fields["name"].(string); ok {
			return fmt.Sprintf("%s: %s", toolName, name)
		}
	case "read_file":
		args := []string{}
		if path, ok := fields["path"].(string); ok {
			args = append(args, path)
		}
		if startLine, ok := fields["start_line"].(string); ok && startLine != "" {
			args = append(args, startLine)
		}
		if endLine, ok := fields["end_line"].(string); ok && endLine != "" {
			args = append(args, endLine)
		}
		if len(args) > 0 {
			return fmt.Sprintf("%s: %s", toolName, strings.Join(args, ", "))
		}
	case "write_file":
		path, _ := fields["path"].(string)
		content, _ := fields["content"].(string)
		if path == "" || content == "" {
			return ""
		}
		// Format: tool name and path on first line, content on subsequent lines
		return fmt.Sprintf("%s: %s\n%s", toolName, path, content)
	case "edit_file":
		path, _ := fields["path"].(string)         //nolint:errcheck // type assertion for optional field
		oldStr, _ := fields["old_string"].(string) //nolint:errcheck // type assertion for optional field
		newStr, _ := fields["new_string"].(string) //nolint:errcheck // type assertion for optional field
		if path == "" || oldStr == "" || newStr == "" {
			return ""
		}

		var lines []string
		lines = append(lines, fmt.Sprintf("%s: %s", toolName, path))

		oldLines := strings.Split(oldStr, "\n")
		newLines := strings.Split(newStr, "\n")

		// Use LCS-based diff to properly align old and new lines
		diffPairs := computeDiff(oldLines, newLines)

		// Use null byte as separator for raw data - terminal will format with adaptive width
		for _, pair := range diffPairs {
			oldPart := strings.ReplaceAll(pair.old, "\n", "\\n")
			newPart := strings.ReplaceAll(pair.new, "\n", "\\n")
			// Format: \x00old_content\x00new_content
			lines = append(lines, fmt.Sprintf("\x00%s\x00%s", oldPart, newPart))
		}

		return strings.Join(lines, "\n")
	}
	return ""
}

func truncateString(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen] + "..."
}

func escapeNewlines(s string) string {
	s = strings.ReplaceAll(s, "\n", "\\n")
	s = strings.ReplaceAll(s, "\t", "\\t")
	return s
}

// diffPair represents a pair of old/new lines in a diff
type diffPair struct {
	old string
	new string
}

// computeDiff computes the LCS-based diff between old and new lines
// Returns aligned pairs showing unchanged, added, removed, and changed lines
func computeDiff(oldLines, newLines []string) []diffPair {
	// Compute LCS (Longest Common Subsequence)
	lcs := computeLCS(oldLines, newLines)

	// Build diff by walking through both sequences aligned to LCS
	var result []diffPair
	i, j := 0, 0

	for _, lcsLine := range lcs {
		// Output old lines that were removed (not in LCS)
		for i < len(oldLines) && oldLines[i] != lcsLine {
			result = append(result, diffPair{old: oldLines[i], new: ""})
			i++
		}

		// Output new lines that were added (not in LCS)
		for j < len(newLines) && newLines[j] != lcsLine {
			result = append(result, diffPair{old: "", new: newLines[j]})
			j++
		}

		// Output the matching line from LCS
		if i < len(oldLines) && j < len(newLines) {
			result = append(result, diffPair{old: oldLines[i], new: newLines[j]})
			i++
			j++
		}
	}

	// Output any remaining lines
	for i < len(oldLines) {
		result = append(result, diffPair{old: oldLines[i], new: ""})
		i++
	}
	for j < len(newLines) {
		result = append(result, diffPair{old: "", new: newLines[j]})
		j++
	}

	return result
}

// computeLCS computes the Longest Common Subsequence of two string slices
func computeLCS(a, b []string) []string {
	if len(a) == 0 || len(b) == 0 {
		return nil
	}

	// Build LCS table
	m, n := len(a), len(b)
	dp := make([][]int, m+1)
	for i := range dp {
		dp[i] = make([]int, n+1)
	}

	for i := 1; i <= m; i++ {
		for j := 1; j <= n; j++ {
			if a[i-1] == b[j-1] {
				dp[i][j] = dp[i-1][j-1] + 1
			} else {
				dp[i][j] = max(dp[i-1][j], dp[i][j-1])
			}
		}
	}

	// Backtrack to find LCS
	var lcs []string
	i, j := m, n
	for i > 0 && j > 0 {
		switch {
		case a[i-1] == b[j-1]:
			lcs = append([]string{a[i-1]}, lcs...)
			i--
			j--
		case dp[i-1][j] > dp[i][j-1]:
			i--
		default:
			j--
		}
	}

	return lcs
}
