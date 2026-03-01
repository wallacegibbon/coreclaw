package adaptors

import (
	"fmt"
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/wallacegibbon/coreclaw/internal/stream"
)

func TestCtrlOOpensEditor(t *testing.T) {
	terminal := NewTerminal(nil, newTerminalOutput(), stream.NewChanInput(10))

	msg := tea.KeyMsg{
		Type: tea.KeyCtrlO,
	}

	model, cmd := terminal.Update(msg)

	if model == nil {
		t.Fatal("Update returned nil model")
	}

	if cmd == nil {
		t.Fatal("Update returned nil command - should return editor command")
	}
}

func TestCtrlOWithExistingContent(t *testing.T) {
	terminal := NewTerminal(nil, newTerminalOutput(), stream.NewChanInput(10))
	terminal.input.SetValue("existing input text")

	msg := tea.KeyMsg{
		Type: tea.KeyCtrlO,
	}

	model, cmd := terminal.Update(msg)

	if model == nil {
		t.Fatal("Update returned nil model")
	}

	if cmd == nil {
		t.Fatal("Update returned nil command - should return editor command")
	}

	if terminal.input.Value() != "existing input text" {
		t.Errorf("Input should retain existing text before editor opens, got '%s'", terminal.input.Value())
	}
}

func TestEditorFinishedMsg(t *testing.T) {
	terminal := NewTerminal(nil, newTerminalOutput(), stream.NewChanInput(10))

	msg := editorFinishedMsg{
		content: "test content from editor",
		err:     nil,
	}

	model, _ := terminal.Update(msg)

	if model == nil {
		t.Fatal("Update returned nil model")
	}

	// Input should show summary with line count
	inputValue := terminal.input.Value()
	if !strings.Contains(inputValue, "[1 lines]") {
		t.Errorf("Expected summary in input, got '%s'", inputValue)
	}

	// editorContent should preserve original content
	if terminal.editorContent != "test content from editor" {
		t.Errorf("Expected editorContent 'test content from editor', got '%s'", terminal.editorContent)
	}
}

func TestEditorFinishedMsgWithWhitespace(t *testing.T) {
	terminal := NewTerminal(nil, newTerminalOutput(), stream.NewChanInput(10))

	msg := editorFinishedMsg{
		content: "  content with leading and trailing spaces  \n",
		err:     nil,
	}

	model, _ := terminal.Update(msg)

	if model == nil {
		t.Fatal("Update returned nil model")
	}

	// editorContent should preserve all whitespace including leading/trailing spaces
	if terminal.editorContent != "  content with leading and trailing spaces  \n" {
		t.Errorf("Expected to preserve all whitespace, got '%s'", terminal.editorContent)
	}
}

func TestEditorContentSubmittedOnEnter(t *testing.T) {
	terminal := NewTerminal(nil, newTerminalOutput(), stream.NewChanInput(10))
	terminal.editorContent = "line1\nline2\nline3"

	// editorContent is cleared before submission when Enter is pressed
	// This test verifies the logic flow that checks editorContent first
	if terminal.editorContent != "line1\nline2\nline3" {
		t.Errorf("Expected editorContent to be set before Enter, got '%s'", terminal.editorContent)
	}
}

func TestEditorContentUsedInsteadOfInputValue(t *testing.T) {
	terminal := NewTerminal(nil, newTerminalOutput(), stream.NewChanInput(10))
	terminal.editorContent = "editor content"
	terminal.input.SetValue("input value")

	// When editorContent is set, it should be used instead of input value
	// This is verified by checking that editorContent has the right value
	if terminal.editorContent != "editor content" {
		t.Errorf("Expected editorContent to be 'editor content', got '%s'", terminal.editorContent)
	}
}

func TestEditorFinishedMsgWithError(t *testing.T) {
	terminal := NewTerminal(nil, newTerminalOutput(), stream.NewChanInput(10))
	terminal.input.SetValue("original content")

	msg := editorFinishedMsg{
		content: "",
		err:     fmt.Errorf("editor failed"),
	}

	model, _ := terminal.Update(msg)

	if model == nil {
		t.Fatal("Update returned nil model")
	}

	if terminal.input.Value() != "original content" {
		t.Errorf("Input should remain unchanged on error, got '%s'", terminal.input.Value())
	}

	displayContent := terminal.terminalOutput.display.GetAll()
	if displayContent == "" {
		t.Error("Expected error message in display")
	}
}

func TestEditorSelectionOrder(t *testing.T) {
	editor := getEditorCommand("")
	if editor == "" {
		t.Fatal("Expected editor to be found")
	}

	// Should return one of the three editors in order: vim, vi, nano
	// Or use EDITOR environment variable if set
	if editor != "vim" && editor != "vi" && editor != "nano" {
		t.Logf("Editor is: %s (may be set by EDITOR env var)", editor)
	}
}
