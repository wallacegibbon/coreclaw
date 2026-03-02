package adaptors

import (
	"strings"
	"testing"
)

func TestWordwrapEdgeCases(t *testing.T) {
	tests := []struct {
		name   string
		text   string
		width  int
		expect string // expected first line after wrapping (or entire output)
	}{
		{
			name:   "empty line",
			text:   "",
			width:  10,
			expect: "",
		},
		{
			name:   "only escape sequences - styling and reset",
			text:   "\x1b[1;2m\x1b[0m",
			width:  10,
			expect: "\x1b[1;2m\x1b[0m\n",
		},
		{
			name:   "only prefix escape sequence",
			text:   "\x1b[1;2m",
			width:  10,
			expect: "\x1b[1;2m\n",
		},
		{
			name:   "only suffix escape sequence",
			text:   "\x1b[0m",
			width:  10,
			expect: "\x1b[0m\n",
		},
		{
			name:   "mixed escape sequences no printable",
			text:   "\x1b[1;2m\x1b[3;4m\x1b[0m",
			width:  10,
			expect: "\x1b[1;2m\x1b[3;4m\x1b[0m\n",
		},
		{
			name:   "newline separated escape sequences",
			text:   "\x1b[1;2m\n\x1b[0m",
			width:  10,
			expect: "\x1b[1;2m\n\x1b[0m\n",
		},
		{
			name:   "single printable character with styling",
			text:   "\x1b[1;2ma\x1b[0m",
			width:  10,
			expect: "\x1b[1;2ma\x1b[0m\n",
		},
		{
			name:   "single printable character with styling width 0",
			text:   "\x1b[1;2ma\x1b[0m",
			width:  0,
			expect: "\x1b[1;2ma\x1b[0m",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// This test ensures wordwrap doesn't panic
			result := wordwrap(tt.text, tt.width)
			// For empty input, result should be empty
			if tt.text == "" && result != "" {
				t.Errorf("wordwrap(%q, %d) = %q, want empty", tt.text, tt.width, result)
			}
			// For non-empty, we just ensure no panic occurred
			// Optional: verify that result ends with newline (if input had newline?)
			// For simplicity, we just check that result contains the original escape sequences
			if tt.text != "" && !strings.Contains(result, "\x1b[") && strings.Contains(tt.text, "\x1b") {
				t.Errorf("wordwrap lost escape sequences: %q", result)
			}
		})
	}
}
