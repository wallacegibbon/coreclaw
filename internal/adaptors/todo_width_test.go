package adaptors

import (
	"regexp"
	"strings"
	"testing"

	"github.com/wallacegibbon/coreclaw/internal/stream"
	"github.com/wallacegibbon/coreclaw/internal/todo"
)

// stripANSI removes ANSI escape sequences from a string
func stripANSI(s string) string {
	re := regexp.MustCompile(`\x1b\[[0-9;]*[a-zA-Z]`)
	return re.ReplaceAllString(s, "")
}

func TestTodoBoxWidthMatchesInputBox(t *testing.T) {
	// Create terminal with fixed window width
	term := NewTerminal(nil, newTerminalOutput(), stream.NewChanInput(10), "")
	term.windowWidth = 100

	// Create todo manager and add some todos
	mgr := todo.NewManager()
	mgr.SetTodos([]todo.TodoItem{
		{Content: "Test todo 1", Status: "pending", ActiveForm: "Testing todo 1"},
		{Content: "Test todo 2", Status: "in_progress", ActiveForm: "Testing todo 2"},
		{Content: "Test todo 3", Status: "completed", ActiveForm: "Testing todo 3"},
	})
	term.SetTodoMgr(mgr)

	// Render the view
	view := term.View()

	// Split by lines
	lines := strings.Split(view, "\n")

	// Debug: print all lines
	for i, line := range lines {
		t.Logf("Line %d (%d chars): %q", i, len(line), line)
	}

	// Find all border lines (starting with "╭" and ending with "╮") after stripping ANSI codes
	var borderLines []string
	for _, line := range lines {
		stripped := stripANSI(line)
		if strings.HasPrefix(stripped, "╭") && strings.HasSuffix(stripped, "╮") {
			borderLines = append(borderLines, stripped)
		}
	}

	if len(borderLines) < 2 {
		t.Fatalf("Expected at least 2 border lines, found %d", len(borderLines))
	}

	// The last border line should be input box, second last should be todo box
	// (display has no border, todo has border, input has border)
	todoBorderLine := borderLines[len(borderLines)-2]
	inputBorderLine := borderLines[len(borderLines)-1]

	t.Logf("Todo border line (%d chars): %s", len(todoBorderLine), todoBorderLine)
	t.Logf("Input border line (%d chars): %s", len(inputBorderLine), inputBorderLine)

	// Compare lengths
	todoLen := len(todoBorderLine)
	inputLen := len(inputBorderLine)

	if todoLen != inputLen {
		t.Errorf("Todo box width (%d) does not match input box width (%d)", todoLen, inputLen)
	}
}
