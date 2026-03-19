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
