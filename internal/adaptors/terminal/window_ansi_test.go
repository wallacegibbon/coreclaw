package terminal

import (
	"testing"

	"github.com/alayacore/alayacore/internal/stream"
)

// TestWindow_WithANSIContent verifies that windows properly handle content
// with ANSI escape sequences from any source (read_file, write_file, posix_shell, etc.)
func TestWindow_WithANSIContent(t *testing.T) {
	styles := DefaultStyles()

	tests := []struct {
		name     string
		tag      string
		content  string
		expected string // Expected text after ANSI stripping (lipgloss ANSI is OK)
	}{
		{
			name:     "read_file result with ANSI",
			tag:      stream.TagFunctionResult,
			content:  "File content with \x1b[31mred text\x1b[0m",
			expected: "File content with red text",
		},
		{
			name:     "posix_shell result with colors",
			tag:      stream.TagFunctionResult,
			content:  "Command output:\n\x1b[32mSuccess\x1b[0m\nDone",
			expected: "Command output:\nSuccess\nDone",
		},
		{
			name:     "write_file result with cursor codes",
			tag:      stream.TagFunctionResult,
			content:  "Writing\x1b[2K\rComplete",
			expected: "Writing\nComplete",
		},
		{
			name:     "tool call with ANSI in command",
			tag:      stream.TagFunctionCall,
			content:  "posix_shell: echo \x1b[31mtest\x1b[0m",
			expected: "· posix_shell: echo test", // Note: includes status indicator
		},
		{
			name:     "text with embedded ANSI",
			tag:      stream.TagTextAssistant,
			content:  "Here is \x1b[1mbold\x1b[0m text",
			expected: "Here is bold text",
		},
		{
			name:     "reasoning with OSC sequence",
			tag:      stream.TagTextReasoning,
			content:  "Thinking\x1b]0;Title\x07...",
			expected: "Thinking...",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			w := &Window{
				ID:      "test-window",
				Tag:     tt.tag,
				Content: tt.content,
			}

			// Render the window content
			result := w.renderGenericContent(80, styles)

			// Strip lipgloss ANSI to check the actual text content
			resultStripped := stripANSI(result)

			if resultStripped != tt.expected {
				t.Errorf("Window render for tag %s:\n  got:  %q\n  want: %q",
					tt.tag, resultStripped, tt.expected)
			}
		})
	}
}

// TestWindow_DiffContentWithANSI verifies that edit_file windows handle ANSI
func TestWindow_DiffContentWithANSI(t *testing.T) {
	styles := DefaultStyles()

	// Use actual escape characters (not literal \x1b)
	oldLine := "\x1b[31mold line\x1b[0m"
	newLine := "\x1b[32mnew line\x1b[0m"
	content := "edit_file: /tmp/test.txt\n- " + oldLine + "\n+ " + newLine + "\n  unchanged"

	result := RenderDiffContent(content, ToolStatusSuccess, styles)

	// Strip lipgloss ANSI to check the actual text
	resultStripped := stripANSI(result)

	// Should contain the text without the embedded ANSI from input
	// (lipgloss will add its own diff colors)
	expected := "• edit_file: /tmp/test.txt\n- old line\n+ new line\n  unchanged"

	if resultStripped != expected {
		t.Errorf("DiffContent:\n  got:  %q\n  want: %q", resultStripped, expected)
	}
}
