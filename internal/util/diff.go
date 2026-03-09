package util

import (
	"bytes"
	"fmt"
	"sort"
	"strings"
)

// Edit represents a byte range replacement
type Edit struct {
	Start, End int
	New        string
}

// ApplyEdits applies edits to src and returns the result.
// Edits are applied in order of start offset.
func ApplyEdits(src string, edits []Edit) (string, error) {
	if len(edits) == 0 {
		return src, nil
	}

	sortedEdits := make([]Edit, len(edits))
	copy(sortedEdits, edits)
	sort.Slice(sortedEdits, func(i, j int) bool {
		if sortedEdits[i].Start != sortedEdits[j].Start {
			return sortedEdits[i].Start < sortedEdits[j].Start
		}
		return sortedEdits[i].End < sortedEdits[j].End
	})

	// Validate no overlaps
	lastEnd := 0
	for _, e := range sortedEdits {
		if !(0 <= e.Start && e.Start <= e.End && e.End <= len(src)) {
			return "", fmt.Errorf("out-of-bounds edit")
		}
		if e.Start < lastEnd {
			return "", fmt.Errorf("overlapping edits")
		}
		lastEnd = e.End
	}

	// Apply edits
	result := make([]byte, 0, len(src))
	lastEnd = 0
	for _, e := range sortedEdits {
		result = append(result, src[lastEnd:e.Start]...)
		result = append(result, e.New...)
		lastEnd = e.End
	}
	result = append(result, src[lastEnd:]...)

	return string(result), nil
}

// ApplyEditsReverse applies edits in reverse order (descending start position).
// This prevents earlier edits from shifting byte offsets of later edits.
func ApplyEditsReverse(src string, edits []Edit) (string, error) {
	if len(edits) == 0 {
		return src, nil
	}

	sortedEdits := make([]Edit, len(edits))
	copy(sortedEdits, edits)
	sort.Slice(sortedEdits, func(i, j int) bool {
		if sortedEdits[i].Start != sortedEdits[j].Start {
			return sortedEdits[i].Start > sortedEdits[j].Start
		}
		return sortedEdits[i].End > sortedEdits[j].End
	})

	result := []byte(src)
	for _, e := range sortedEdits {
		if !(0 <= e.Start && e.Start <= e.End && e.End <= len(result)) {
			return "", fmt.Errorf("out-of-bounds edit")
		}
		newResult := make([]byte, 0, len(result)+len(e.New)-(e.End-e.Start))
		newResult = append(newResult, result[:e.Start]...)
		newResult = append(newResult, e.New...)
		newResult = append(newResult, result[e.End:]...)
		result = newResult
	}

	return string(result), nil
}

// Hunk represents a parsed hunk from a unified diff
type Hunk struct {
	OrigStart int        // 1-indexed line number in original file
	OrigCount int        // number of lines in original file
	Lines     []HunkLine // lines in the hunk
}

// HunkLine represents a line in a hunk
type HunkLine struct {
	Op      byte // ' ' context, '+' added, '-' removed
	Content string
}

// ParseUnifiedDiff parses a unified diff string into hunks
func ParseUnifiedDiff(diffStr string) ([]Hunk, error) {
	lines := splitLines(diffStr)
	var hunks []Hunk
	var currentHunk *Hunk

	for i, line := range lines {
		if len(line) == 0 {
			continue
		}

		if line[0] == '@' {
			hunk, err := parseHunkHeader(line)
			if err != nil {
				return nil, err
			}
			hunks = append(hunks, hunk)
			currentHunk = &hunks[len(hunks)-1]
		} else if currentHunk != nil {
			if len(line) >= 1 {
				op := line[0]
				content := string(line[1:])
				if op == ' ' || op == '+' || op == '-' {
					// Context lines (op=' ') and removed lines (op='-') have no newlines here.
					// Added lines (op='+') get newlines based on whether there's a following hunk line.
					if op == '+' {
						// Add newline if next line is another hunk line (not a header)
						if i+1 < len(lines) && len(lines[i+1]) > 0 && lines[i+1][0] != '@' {
							content += "\n"
						}
					}
					currentHunk.Lines = append(currentHunk.Lines, HunkLine{Op: op, Content: content})
				}
			}
		}
	}

	if len(hunks) == 0 {
		return nil, fmt.Errorf("no hunks found in diff")
	}

	return hunks, nil
}

// HunksToEdits converts hunks to edits, applying them to the original content
func HunksToEdits(originalContent string, hunks []Hunk) ([]Edit, error) {
	originalLines := splitContentLines(originalContent)

	lineOffsets := make([]int, len(originalLines)+1)
	offset := 0
	for i, line := range originalLines {
		lineOffsets[i] = offset
		offset += len(line)
	}
	lineOffsets[len(originalLines)] = offset

	var edits []Edit
	for _, hunk := range hunks {
		edit, err := hunkToEdit(originalLines, lineOffsets, hunk)
		if err != nil {
			return nil, err
		}
		edits = append(edits, edit)
	}

	return edits, nil
}

// ApplyUnifiedDiff parses and applies a unified diff to the original content
func ApplyUnifiedDiff(originalContent, diffStr string) (string, error) {
	hunks, err := ParseUnifiedDiff(diffStr)
	if err != nil {
		return "", err
	}

	edits, err := HunksToEdits(originalContent, hunks)
	if err != nil {
		return "", err
	}

	return ApplyEditsReverse(originalContent, edits)
}

func parseHunkHeader(line []byte) (Hunk, error) {
	start := bytes.Index(line, []byte("@@ "))
	if start == -1 {
		return Hunk{}, fmt.Errorf("missing @@")
	}
	end := bytes.Index(line[start+3:], []byte(" @@"))
	if end == -1 {
		return Hunk{}, fmt.Errorf("missing closing @@")
	}

	rangeStr := string(line[start+3 : start+3+end])
	parts := fields([]byte(rangeStr))
	if len(parts) != 2 {
		return Hunk{}, fmt.Errorf("expected 2 range specifications")
	}

	oldRange := string(parts[0])
	if len(oldRange) == 0 || oldRange[0] != '-' {
		return Hunk{}, fmt.Errorf("old range should start with '-'")
	}
	oldValues := split(oldRange[1:], ',')
	oldStart, err := atoi(oldValues[0])
	if err != nil {
		return Hunk{}, fmt.Errorf("invalid oldStart: %w", err)
	}
	oldCount := 1
	if len(oldValues) > 1 {
		oldCount, err = atoi(oldValues[1])
		if err != nil {
			return Hunk{}, fmt.Errorf("invalid oldCount: %w", err)
		}
	}

	return Hunk{OrigStart: oldStart, OrigCount: oldCount}, nil
}

func hunkToEdit(originalLines []string, lineOffsets []int, hunk Hunk) (Edit, error) {
	startLine := max(hunk.OrigStart-1, 0)

	if startLine >= len(lineOffsets) {
		return Edit{}, fmt.Errorf("start line %d out of range", hunk.OrigStart)
	}
	startByte := lineOffsets[startLine]

	endLine := max(startLine+hunk.OrigCount, 0)
	if endLine > len(lineOffsets) {
		return Edit{}, fmt.Errorf("end line %d out of range", endLine+1)
	}
	endByte := lineOffsets[endLine]

	var newContent bytes.Buffer
	origLineIndex := startLine
	for i, hl := range hunk.Lines {
		switch hl.Op {
		case ' ':
			if origLineIndex >= len(originalLines) {
				return Edit{}, fmt.Errorf("context line %d out of range", origLineIndex+1)
			}
			// Validate that context line matches original file
			origLine := originalLines[origLineIndex]
			origContent := strings.TrimSuffix(origLine, "\n")
			hunkContent := strings.TrimSuffix(hl.Content, "\n")
			if hunkContent != origContent {
				return Edit{}, fmt.Errorf("context line mismatch at line %d: diff has %q but file has %q", origLineIndex+1, hunkContent, origContent)
			}
			newContent.WriteString(origLine)
			origLineIndex++
		case '-':
			if origLineIndex >= len(originalLines) {
				return Edit{}, fmt.Errorf("remove line %d out of range", origLineIndex+1)
			}
			// Validate that removed line matches original file
			origLine := originalLines[origLineIndex]
			origContent := strings.TrimSuffix(origLine, "\n")
			hunkContent := strings.TrimSuffix(hl.Content, "\n")
			if hunkContent != origContent {
				return Edit{}, fmt.Errorf("remove line mismatch at line %d: diff has %q but file has %q", origLineIndex+1, hunkContent, origContent)
			}
			origLineIndex++
		case '+':
			newContent.WriteString(hl.Content)
			// If this is the last hunk line and we need to preserve a newline:
			// 1. For insertions (OrigCount=0): check if there's a newline at insertion point
			// 2. For replacements (OrigCount>0): check if the replaced line ended with newline
			if i == len(hunk.Lines)-1 {
				shouldAddNewline := false
				if hunk.OrigCount == 0 && startLine < len(originalLines) && len(originalLines[startLine]) > 0 {
					// Insertion: check if line at startLine has a newline
					lastChar := originalLines[startLine][len(originalLines[startLine])-1]
					shouldAddNewline = (lastChar == '\n')
				} else if hunk.OrigCount > 0 && endByte > startByte {
					// Replacement: check if line at endLine-1 has a newline
					if endLine > 0 && endLine <= len(originalLines) && len(originalLines[endLine-1]) > 0 {
						lastChar := originalLines[endLine-1][len(originalLines[endLine-1])-1]
						shouldAddNewline = (lastChar == '\n')
					}
				}
				if shouldAddNewline && (len(hl.Content) == 0 || hl.Content[len(hl.Content)-1] != '\n') {
					newContent.WriteString("\n")
				}
			}
		}
	}

	return Edit{Start: startByte, End: endByte, New: newContent.String()}, nil
}

func splitContentLines(content string) []string {
	if len(content) == 0 {
		return []string{}
	}

	var lines []string
	start := 0
	for i, r := range content {
		if r == '\n' {
			lines = append(lines, content[start:i+1])
			start = i + 1
		}
	}
	if start < len(content) {
		lines = append(lines, content[start:])
	}
	return lines
}

func splitLines(s string) [][]byte {
	var result [][]byte
	start := 0
	for i := 0; i < len(s); i++ {
		if s[i] == '\n' {
			result = append(result, []byte(s[start:i]))
			start = i + 1
		}
	}
	result = append(result, []byte(s[start:]))
	return result
}

func split(s string, sep byte) []string {
	var result []string
	start := 0
	for i := 0; i < len(s); i++ {
		if s[i] == sep {
			result = append(result, s[start:i])
			start = i + 1
		}
	}
	result = append(result, s[start:])
	return result
}

func fields(b []byte) [][]byte {
	var result [][]byte
	start := -1
	for i := range len(b) {
		if b[i] == ' ' || b[i] == '\t' {
			if start >= 0 {
				result = append(result, b[start:i])
				start = -1
			}
		} else if start < 0 {
			start = i
		}
	}
	if start >= 0 {
		result = append(result, b[start:])
	}
	return result
}

func atoi(b string) (int, error) {
	result := 0
	for i := range len(b) {
		if b[i] < '0' || b[i] > '9' {
			return 0, fmt.Errorf("invalid number")
		}
		result = result*10 + int(b[i]-'0')
	}
	return result, nil
}
