package util

import (
	"testing"
)

func TestApplyEdits(t *testing.T) {
	tests := []struct {
		name      string
		src       string
		edits     []Edit
		want      string
		wantError bool
		errorMsg  string
	}{
		{
			name:  "empty edits",
			src:   "hello world",
			edits: []Edit{},
			want:  "hello world",
		},
		{
			name: "single replacement",
			src:  "hello world",
			edits: []Edit{
				{Start: 0, End: 5, New: "goodbye"},
			},
			want: "goodbye world",
		},
		{
			name: "append at end",
			src:  "hello",
			edits: []Edit{
				{Start: 5, End: 5, New: " world"},
			},
			want: "hello world",
		},
		{
			name: "insert in middle",
			src:  "hllo world",
			edits: []Edit{
				{Start: 1, End: 1, New: "e"},
			},
			want: "hello world",
		},
		{
			name: "delete range",
			src:  "hello beautiful world",
			edits: []Edit{
				{Start: 6, End: 16, New: ""},
			},
			want: "hello world",
		},
		{
			name: "multiple edits sorted",
			src:  "abcdefghij",
			edits: []Edit{
				{Start: 0, End: 2, New: "AB"},
				{Start: 5, End: 7, New: "FG"},
			},
			want: "ABcdeFGhij",
		},
		{
			name: "multiple edits unsorted",
			src:  "abcdefghij",
			edits: []Edit{
				{Start: 5, End: 7, New: "FG"},
				{Start: 0, End: 2, New: "AB"},
			},
			want: "ABcdeFGhij",
		},
		{
			name: "replace entire content",
			src:  "old content",
			edits: []Edit{
				{Start: 0, End: 11, New: "new content"},
			},
			want: "new content",
		},
		{
			name: "replace with longer content",
			src:  "hi",
			edits: []Edit{
				{Start: 0, End: 2, New: "hello world"},
			},
			want: "hello world",
		},
		{
			name: "replace with shorter content",
			src:  "hello world",
			edits: []Edit{
				{Start: 0, End: 11, New: "hi"},
			},
			want: "hi",
		},
		{
			name: "out-of-bounds start",
			src:  "hello",
			edits: []Edit{
				{Start: -1, End: 2, New: "hi"},
			},
			wantError: true,
			errorMsg:  "out-of-bounds edit",
		},
		{
			name: "out-of-bounds end",
			src:  "hello",
			edits: []Edit{
				{Start: 0, End: 10, New: "hi"},
			},
			wantError: true,
			errorMsg:  "out-of-bounds edit",
		},
		{
			name: "start after end",
			src:  "hello",
			edits: []Edit{
				{Start: 5, End: 0, New: "hi"},
			},
			wantError: true,
			errorMsg:  "out-of-bounds edit",
		},
		{
			name: "overlapping edits",
			src:  "hello world",
			edits: []Edit{
				{Start: 0, End: 5, New: "hi"},
				{Start: 3, End: 8, New: "there"},
			},
			wantError: true,
			errorMsg:  "overlapping edits",
		},
		{
			name: "touching edits",
			src:  "hello world",
			edits: []Edit{
				{Start: 0, End: 5, New: "hi"},
				{Start: 5, End: 11, New: " there"},
			},
			want: "hi there",
		},
		{
			name: "empty source with insert",
			src:  "",
			edits: []Edit{
				{Start: 0, End: 0, New: "content"},
			},
			want: "content",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := ApplyEdits(tt.src, tt.edits)

			if tt.wantError {
				if err == nil {
					t.Errorf("ApplyEdits() expected error containing %q, got nil", tt.errorMsg)
				} else if err.Error() != tt.errorMsg {
					t.Errorf("ApplyEdits() expected error %q, got %q", tt.errorMsg, err.Error())
				}
				return
			}

			if err != nil {
				t.Errorf("ApplyEdits() unexpected error: %v", err)
				return
			}

			if got != tt.want {
				t.Errorf("ApplyEdits() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestApplyEditsReverse(t *testing.T) {
	tests := []struct {
		name      string
		src       string
		edits     []Edit
		want      string
		wantError bool
		errorMsg  string
	}{
		{
			name:  "empty edits",
			src:   "hello world",
			edits: []Edit{},
			want:  "hello world",
		},
		{
			name: "single replacement",
			src:  "hello world",
			edits: []Edit{
				{Start: 0, End: 5, New: "goodbye"},
			},
			want: "goodbye world",
		},
		{
			name: "multiple edits forward order",
			src:  "abcdefghij",
			edits: []Edit{
				{Start: 0, End: 2, New: "AB"},
				{Start: 5, End: 7, New: "FG"},
			},
			want: "ABcdeFGhij",
		},
		{
			name: "multiple edits reverse order",
			src:  "abcdefghij",
			edits: []Edit{
				{Start: 5, End: 7, New: "FG"},
				{Start: 0, End: 2, New: "AB"},
			},
			want: "ABcdeFGhij",
		},
		{
			name: "edits at beginning and end",
			src:  "middle",
			edits: []Edit{
				{Start: 0, End: 0, New: "start"},
				{Start: 6, End: 6, New: "end"},
			},
			want: "startmiddleend",
		},
		{
			name: "edits shift later positions",
			src:  "123456789",
			edits: []Edit{
				{Start: 0, End: 0, New: "AA"},
				{Start: 3, End: 3, New: "BB"},
				{Start: 6, End: 6, New: "CC"},
			},
			want: "AA123BB456CC789",
		},
		{
			name: "out-of-bounds start",
			src:  "hello",
			edits: []Edit{
				{Start: -1, End: 2, New: "hi"},
			},
			wantError: true,
			errorMsg:  "out-of-bounds edit",
		},
		{
			name: "out-of-bounds end",
			src:  "hello",
			edits: []Edit{
				{Start: 0, End: 10, New: "hi"},
			},
			wantError: true,
			errorMsg:  "out-of-bounds edit",
		},
		{
			name: "start after end",
			src:  "hello",
			edits: []Edit{
				{Start: 5, End: 0, New: "hi"},
			},
			wantError: true,
			errorMsg:  "out-of-bounds edit",
		},
		{
			name: "empty source",
			src:  "",
			edits: []Edit{
				{Start: 0, End: 0, New: "content"},
			},
			want: "content",
		},
		{
			name: "delete all content",
			src:  "hello world",
			edits: []Edit{
				{Start: 0, End: 11, New: ""},
			},
			want: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := ApplyEditsReverse(tt.src, tt.edits)

			if tt.wantError {
				if err == nil {
					t.Errorf("ApplyEditsReverse() expected error containing %q, got nil", tt.errorMsg)
				} else if err.Error() != tt.errorMsg {
					t.Errorf("ApplyEditsReverse() expected error %q, got %q", tt.errorMsg, err.Error())
				}
				return
			}

			if err != nil {
				t.Errorf("ApplyEditsReverse() unexpected error: %v", err)
				return
			}

			if got != tt.want {
				t.Errorf("ApplyEditsReverse() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestParseUnifiedDiff(t *testing.T) {
	tests := []struct {
		name      string
		diffStr   string
		want      []Hunk
		wantError bool
		errorMsg  string
	}{
		{
			name: "simple hunk with context",
			diffStr: `@@ -1,3 +1,3 @@
 context
-old line
+new line
 context`,
			want: []Hunk{
				{
					OrigStart: 1,
					OrigCount: 3,
					Lines: []HunkLine{
						{Op: ' ', Content: "context"},
						{Op: '-', Content: "old line"},
						{Op: '+', Content: "new line\n"},
						{Op: ' ', Content: "context"},
					},
				},
			},
		},
		{
			name: "multiple hunks",
			diffStr: `@@ -1,2 +1,2 @@
 line1
-line2
+new line2
@@ -4,1 +4,1 @@
-old4
+new4
`,
			want: []Hunk{
				{
					OrigStart: 1,
					OrigCount: 2,
					Lines: []HunkLine{
						{Op: ' ', Content: "line1"},
						{Op: '-', Content: "line2"},
						{Op: '+', Content: "new line2"},
					},
				},
				{
					OrigStart: 4,
					OrigCount: 1,
					Lines: []HunkLine{
						{Op: '-', Content: "old4"},
						{Op: '+', Content: "new4"},
					},
				},
			},
		},
		{
			name: "hunk without count",
			diffStr: `@@ -1 +1,2 @@
 line1
+line2`,
			want: []Hunk{
				{
					OrigStart: 1,
					OrigCount: 1,
					Lines: []HunkLine{
						{Op: ' ', Content: "line1"},
						{Op: '+', Content: "line2"},
					},
				},
			},
		},
		{
			name: "empty hunk",
			diffStr: `@@ -1,0 +0,0 @@
`,
			want: []Hunk{
				{
					OrigStart: 1,
					OrigCount: 0,
					Lines:     []HunkLine{},
				},
			},
		},
		{
			name:      "no hunks",
			diffStr:   "some text without hunk headers",
			wantError: true,
			errorMsg:  "no hunks found in diff",
		},
		{
			name:      "empty diff",
			diffStr:   "",
			wantError: true,
			errorMsg:  "no hunks found in diff",
		},
		{
			name:      "invalid hunk header - missing @@",
			diffStr:   `@@ -1,3 +1,3`,
			wantError: true,
			errorMsg:  "missing closing @@",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := ParseUnifiedDiff(tt.diffStr)

			if tt.wantError {
				if err == nil {
					t.Errorf("ParseUnifiedDiff() expected error containing %q, got nil", tt.errorMsg)
				} else if err.Error() != tt.errorMsg {
					t.Errorf("ParseUnifiedDiff() expected error %q, got %q", tt.errorMsg, err.Error())
				}
				return
			}

			if err != nil {
				t.Errorf("ParseUnifiedDiff() unexpected error: %v", err)
				return
			}

			if len(got) != len(tt.want) {
				t.Errorf("ParseUnifiedDiff() got %d hunks, want %d", len(got), len(tt.want))
				return
			}

			for i, hunk := range got {
				if hunk.OrigStart != tt.want[i].OrigStart {
					t.Errorf("ParseUnifiedDiff() hunk %d OrigStart = %d, want %d", i, hunk.OrigStart, tt.want[i].OrigStart)
				}
				if hunk.OrigCount != tt.want[i].OrigCount {
					t.Errorf("ParseUnifiedDiff() hunk %d OrigCount = %d, want %d", i, hunk.OrigCount, tt.want[i].OrigCount)
				}
				if len(hunk.Lines) != len(tt.want[i].Lines) {
					t.Errorf("ParseUnifiedDiff() hunk %d got %d lines, want %d", i, len(hunk.Lines), len(tt.want[i].Lines))
					continue
				}
				for j, line := range hunk.Lines {
					if line.Op != tt.want[i].Lines[j].Op {
						t.Errorf("ParseUnifiedDiff() hunk %d line %d Op = %c, want %c", i, j, line.Op, tt.want[i].Lines[j].Op)
					}
					if line.Content != tt.want[i].Lines[j].Content {
						t.Errorf("ParseUnifiedDiff() hunk %d line %d Content = %q, want %q", i, j, line.Content, tt.want[i].Lines[j].Content)
					}
				}
			}
		})
	}
}

func TestHunksToEdits(t *testing.T) {
	tests := []struct {
		name            string
		originalContent string
		hunks           []Hunk
		want            []Edit
		wantError       bool
		errorMsg        string
	}{
		{
			name:            "single hunk replacement",
			originalContent: "line1\nline2\nline3\n",
			hunks: []Hunk{
				{
					OrigStart: 2,
					OrigCount: 1,
					Lines: []HunkLine{
						{Op: '+', Content: "new line2\n"},
					},
				},
			},
			want: []Edit{
				{Start: 6, End: 12, New: "new line2\n"},
			},
		},
		{
			name:            "single hunk with context",
			originalContent: "line1\nline2\nline3\n",
			hunks: []Hunk{
				{
					OrigStart: 1,
					OrigCount: 3,
					Lines: []HunkLine{
						{Op: ' ', Content: "line1\n"},
						{Op: '-', Content: "line2\n"},
						{Op: '+', Content: "new line2\n"},
						{Op: ' ', Content: "line3\n"},
					},
				},
			},
			want: []Edit{
				{Start: 0, End: 18, New: "line1\nnew line2\nline3\n"},
			},
		},
		{
			name:            "multiple hunks",
			originalContent: "line1\nline2\nline3\nline4\nline5\n",
			hunks: []Hunk{
				{
					OrigStart: 2,
					OrigCount: 1,
					Lines: []HunkLine{
						{Op: '+', Content: "new line2\n"},
					},
				},
				{
					OrigStart: 4,
					OrigCount: 1,
					Lines: []HunkLine{
						{Op: '+', Content: "new line4\n"},
					},
				},
			},
			want: []Edit{
				{Start: 6, End: 12, New: "new line2\n"},
				{Start: 18, End: 24, New: "new line4\n"},
			},
		},
		{
			name:            "delete line",
			originalContent: "line1\nline2\nline3\n",
			hunks: []Hunk{
				{
					OrigStart: 2,
					OrigCount: 1,
					Lines: []HunkLine{
						{Op: '-', Content: "line2\n"},
					},
				},
			},
			want: []Edit{
				{Start: 6, End: 12, New: ""},
			},
		},
		{
			name:            "insert at beginning",
			originalContent: "line1\nline2\n",
			hunks: []Hunk{
				{
					OrigStart: 1,
					OrigCount: 0,
					Lines: []HunkLine{
						{Op: '+', Content: "new line\n"},
					},
				},
			},
			want: []Edit{
				{Start: 0, End: 0, New: "new line\n"},
			},
		},
		{
			name:            "content without trailing newline",
			originalContent: "line1\nline2\nline3",
			hunks: []Hunk{
				{
					OrigStart: 2,
					OrigCount: 1,
					Lines: []HunkLine{
						{Op: '+', Content: "new line2\n"},
					},
				},
			},
			want: []Edit{
				{Start: 6, End: 12, New: "new line2\n"},
			},
		},
		{
			name:            "start line out of range",
			originalContent: "line1\n",
			hunks: []Hunk{
				{
					OrigStart: 10,
					OrigCount: 1,
					Lines: []HunkLine{
						{Op: '+', Content: "new line\n"},
					},
				},
			},
			wantError: true,
			errorMsg:  "start line 10 out of range",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := HunksToEdits(tt.originalContent, tt.hunks)

			if tt.wantError {
				if err == nil {
					t.Errorf("HunksToEdits() expected error containing %q, got nil", tt.errorMsg)
				} else if err.Error() != tt.errorMsg {
					t.Errorf("HunksToEdits() expected error %q, got %q", tt.errorMsg, err.Error())
				}
				return
			}

			if err != nil {
				t.Errorf("HunksToEdits() unexpected error: %v", err)
				return
			}

			if len(got) != len(tt.want) {
				t.Errorf("HunksToEdits() got %d edits, want %d", len(got), len(tt.want))
				return
			}

			for i, edit := range got {
				if edit.Start != tt.want[i].Start {
					t.Errorf("HunksToEdits() edit %d Start = %d, want %d", i, edit.Start, tt.want[i].Start)
				}
				if edit.End != tt.want[i].End {
					t.Errorf("HunksToEdits() edit %d End = %d, want %d", i, edit.End, tt.want[i].End)
				}
				if edit.New != tt.want[i].New {
					t.Errorf("HunksToEdits() edit %d New = %q, want %q", i, edit.New, tt.want[i].New)
				}
			}
		})
	}
}

func TestApplyUnifiedDiff(t *testing.T) {
	tests := []struct {
		name            string
		originalContent string
		diffStr         string
		want            string
		wantError       bool
		errorMsg        string
	}{
		{
			name:            "user issue - original has newline",
			originalContent: "THIS IS LINE 3\n",
			diffStr: `@@ -1,1 +1,1 @@
-THIS IS LINE 3
+this is line 3
`,
			want: "this is line 3\n",
		},
		{
			name:            "user issue - original no newline",
			originalContent: "THIS IS LINE 3",
			diffStr: `@@ -1,1 +1,1 @@
-THIS IS LINE 3
+this is line 3
`,
			want: "this is line 3",
		},
		{
			name:            "simple replacement",
			originalContent: "line1\nline2\nline3\n",
			diffStr: `@@ -2,1 +2,1 @@
-line2
+new line2
`,
			want: "line1\nnew line2\nline3\n",
		},
		{
			name:            "simple replacement - with trailing context",
			originalContent: "line1\nline2\nline3\n",
			diffStr: `@@ -2,1 +2,1 @@
-line2
+new line2
@@ -3,1 +3,1 @@
 line3`,
			want: "line1\nnew line2\nline3\n",
		},
		{
			name:            "add line",
			originalContent: "line1\nline3\n",
			diffStr: `@@ -1,2 +1,3 @@
 line1
+line2
 line3
`,
			want: "line1\nline2\nline3\n",
		},
		{
			name:            "delete line",
			originalContent: "line1\nline2\nline3\n",
			diffStr: `@@ -1,3 +1,2 @@
 line1
-line2
 line3
`,
			want: "line1\nline3\n",
		},
		{
			name:            "multiple hunks",
			originalContent: "line1\nline2\nline3\nline4\nline5\n",
			diffStr: `@@ -2,1 +2,1 @@
-line2
+new line2
@@ -4,1 +4,1 @@
-line4
+new line4
@@ -5,1 +5,1 @@
 line5`,
			want: "line1\nnew line2\nline3\nnew line4\nline5\n",
		},
		{
			name:            "replace entire file",
			originalContent: "old content\nmore old\n",
			diffStr: `@@ -1,2 +1,2 @@
-old content
-more old
+new content
+more new
@@ -3,0 +3,0 @@
`,
			want: "new content\nmore new\n",
		},
		{
			name:            "add at end",
			originalContent: "line1\nline2\n",
			diffStr: `@@ -3,0 +3,1 @@
+line3
`,
			want: "line1\nline2\nline3",
		},
		{
			name:            "content without trailing newline",
			originalContent: "line1\nline2\nline3",
			diffStr: `@@ -2,1 +2,1 @@
-line2
+new line2
@@ -3,1 +3,1 @@
 line3`,
			want: "line1\nnew line2\nline3",
		},
		{
			name:            "empty file with insertion",
			originalContent: "",
			diffStr: `@@ -0,0 +1,1 @@
+new content
`,
			want: "new content",
		},
		{
			name:            "empty file with multiple lines",
			originalContent: "",
			diffStr: `@@ -0,0 +2,1 @@
+line1
+line2
`,
			want: "line1\nline2",
		},
		{
			name:            "insert at very beginning",
			originalContent: "line1\nline2\n",
			diffStr: `@@ -0,0 +1,1 @@
+new first line
@@ -1,2 +2,2 @@
 line1
 line2`,
			want: "new first line\nline1\nline2\n",
		},
		{
			name:      "invalid diff - no hunks",
			diffStr:   "not a valid diff",
			wantError: true,
			errorMsg:  "no hunks found in diff",
		},
		{
			name:            "invalid hunk - out of range",
			originalContent: "line1\n",
			diffStr: `@@ -10,1 +10,1 @@
-line1
+new
`,
			wantError: true,
			errorMsg:  "start line 10 out of range",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := ApplyUnifiedDiff(tt.originalContent, tt.diffStr)

			if tt.wantError {
				if err == nil {
					t.Errorf("ApplyUnifiedDiff() expected error containing %q, got nil", tt.errorMsg)
				} else if err.Error() != tt.errorMsg {
					t.Errorf("ApplyUnifiedDiff() expected error %q, got %q", tt.errorMsg, err.Error())
				}
				return
			}

			if err != nil {
				t.Errorf("ApplyUnifiedDiff() unexpected error: %v", err)
				return
			}

			if got != tt.want {
				t.Errorf("ApplyUnifiedDiff() =\n%q\nwant\n%q", got, tt.want)
			}
		})
	}
}
