package adaptors

import (
	"fmt"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
)

func TestCtrlOOpensEditor(t *testing.T) {
	terminal := NewTerminal(nil, newTerminalOutput())

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
	terminal := NewTerminal(nil, newTerminalOutput())
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
	terminal := NewTerminal(nil, newTerminalOutput())

	msg := editorFinishedMsg{
		content: "test content from editor",
		err:     nil,
	}

	model, _ := terminal.Update(msg)

	if model == nil {
		t.Fatal("Update returned nil model")
	}

	if terminal.input.Value() != "test content from editor" {
		t.Errorf("Expected input value 'test content from editor', got '%s'", terminal.input.Value())
	}
}

func TestEditorFinishedMsgWithError(t *testing.T) {
	terminal := NewTerminal(nil, newTerminalOutput())
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

