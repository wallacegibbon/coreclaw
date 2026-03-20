package agent

import (
	"strings"
	"testing"
)

func TestComputeDiff(t *testing.T) {
	tests := []struct {
		name     string
		oldLines []string
		newLines []string
		want     []diffPair
	}{
		{
			name:     "identical lines",
			oldLines: []string{"a", "b", "c"},
			newLines: []string{"a", "b", "c"},
			want: []diffPair{
				{old: "a", new: "a"},
				{old: "b", new: "b"},
				{old: "c", new: "c"},
			},
		},
		{
			name:     "add line at end",
			oldLines: []string{"a", "b"},
			newLines: []string{"a", "b", "c"},
			want: []diffPair{
				{old: "a", new: "a"},
				{old: "b", new: "b"},
				{old: "", new: "c"},
			},
		},
		{
			name:     "remove line from end",
			oldLines: []string{"a", "b", "c"},
			newLines: []string{"a", "b"},
			want: []diffPair{
				{old: "a", new: "a"},
				{old: "b", new: "b"},
				{old: "c", new: ""},
			},
		},
		{
			name:     "change line in middle",
			oldLines: []string{"a", "b", "c"},
			newLines: []string{"a", "B", "c"},
			want: []diffPair{
				{old: "a", new: "a"},
				{old: "b", new: ""},
				{old: "", new: "B"},
				{old: "c", new: "c"},
			},
		},
		{
			name:     "add and remove",
			oldLines: []string{"a", "b", "c"},
			newLines: []string{"a", "B", "C", "d"},
			want: []diffPair{
				{old: "a", new: "a"},
				{old: "b", new: ""},
				{old: "c", new: ""},
				{old: "", new: "B"},
				{old: "", new: "C"},
				{old: "", new: "d"},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := computeDiff(tt.oldLines, tt.newLines)
			if len(got) != len(tt.want) {
				t.Errorf("computeDiff() returned %d pairs, want %d", len(got), len(tt.want))
				t.Errorf("got: %v", got)
				t.Errorf("want: %v", tt.want)
				return
			}
			for i := range got {
				if got[i] != tt.want[i] {
					t.Errorf("computeDiff()[%d] = %v, want %v", i, got[i], tt.want[i])
				}
			}
		})
	}
}

func TestFormatToolCallEditFile(t *testing.T) {
	input := `{"path": "/tmp/test.txt", "old_string": "line1\nline2\nline3", "new_string": "line1\nLINE2\nline3"}`
	output := formatToolCall("edit_file", input)

	// Should contain the path
	if !strings.Contains(output, "edit_file: /tmp/test.txt") {
		t.Errorf("expected output to contain 'edit_file: /tmp/test.txt', got %q", output)
	}

	// Should contain diff markers
	if !strings.Contains(output, "\x00") {
		t.Errorf("expected output to contain null byte separators, got %q", output)
	}
}

func TestFormatToolCallWriteFile(t *testing.T) {
	// Test that write_file shows full content on separate lines
	input := `{"path": "/tmp/test.txt", "content": "line1\nline2\nline3\nline4\nline5"}`
	output := formatToolCall("write_file", input)

	// Should contain the path on first line
	if !strings.Contains(output, "write_file: /tmp/test.txt") {
		t.Errorf("expected output to contain 'write_file: /tmp/test.txt', got %q", output)
	}

	// Should contain the full content with actual newlines (not escaped)
	if !strings.Contains(output, "line1\nline2\nline3\nline4\nline5") {
		t.Errorf("expected output to contain full content with newlines, got %q", output)
	}

	// Should NOT contain "..." (truncation indicator)
	if strings.Contains(output, "...") {
		t.Errorf("expected output NOT to contain truncation indicator '...', got %q", output)
	}

	// Content should be on a separate line from the path
	lines := strings.Split(output, "\n")
	if len(lines) < 2 {
		t.Errorf("expected output to have path on first line and content on subsequent lines, got %q", output)
	}
	if lines[0] != "write_file: /tmp/test.txt" {
		t.Errorf("expected first line to be 'write_file: /tmp/test.txt', got %q", lines[0])
	}
}
