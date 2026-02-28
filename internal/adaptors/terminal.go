package adaptors

import (
	_ "embed"
	"encoding/binary"
	"fmt"
	"os"
	"os/exec"
	"strings"
	"sync"
	"time"

	"github.com/charmbracelet/bubbles/textinput"
	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	agentpkg "github.com/wallacegibbon/coreclaw/internal/agent"
	"github.com/wallacegibbon/coreclaw/internal/stream"
)

const (
	tempFilePrefix = "coreclaw-input-*.txt"
)

//go:embed welcome.txt
var welcomeText string

// DisplayBuffer holds text to display in Terminal
type DisplayBuffer struct {
	mu       sync.Mutex
	Messages []string
}

// NewDisplayBuffer creates a new display buffer
func NewDisplayBuffer() *DisplayBuffer {
	return &DisplayBuffer{
		Messages: []string{},
	}
}

// Append adds text to the display buffer
func (d *DisplayBuffer) Append(text string) {
	d.mu.Lock()
	defer d.mu.Unlock()
	d.Messages = append(d.Messages, text)
}

// GetAll returns all messages joined together
func (d *DisplayBuffer) GetAll() string {
	d.mu.Lock()
	defer d.mu.Unlock()
	return strings.Join(d.Messages, "")
}

// TerminalAdaptor is a terminal adaptor with a Terminal interface
type TerminalAdaptor struct {
	AgentFactory AgentFactory
	BaseURL      string
	ModelName    string
	processor    *agentpkg.Processor
	session      *agentpkg.Session
}

// NewTerminalAdaptor creates a new Terminal adaptor
func NewTerminalAdaptor(agentFactory AgentFactory, baseURL, modelName string) *TerminalAdaptor {
	return &TerminalAdaptor{
		AgentFactory: agentFactory,
		BaseURL:      baseURL,
		ModelName:    modelName,
	}
}

// Start runs the Terminal
func (a *TerminalAdaptor) Start() {
	agent := a.AgentFactory()

	// Create first (callback set after Terminal is created)
	terminalOutput := newTerminalOutput()
	processor := agentpkg.NewProcessorWithIO(agent, &stream.NopInput{}, terminalOutput)
	a.processor = processor
	a.session = agentpkg.NewSession(agent, a.BaseURL, a.ModelName, processor)

	t := NewTerminal(a.session, terminalOutput)

	p := tea.NewProgram(t, tea.WithAltScreen(), tea.WithInput(os.Stdin), tea.WithOutput(os.Stdout))
	p.Run()
}

// terminalOutput writes to the Terminal display with TLV support
type terminalOutput struct {
	display    *DisplayBuffer
	buffer     []byte
	updateChan chan struct{}

	textStyle        lipgloss.Style
	userInputStyle   lipgloss.Style
	toolStyle        lipgloss.Style
	toolContentStyle lipgloss.Style
	reasoningStyle   lipgloss.Style
	errorStyle       lipgloss.Style
	systemStyle      lipgloss.Style
	promptStyle      lipgloss.Style
}

func newTerminalOutput() *terminalOutput {
	return &terminalOutput{
		display:          NewDisplayBuffer(),
		updateChan:       make(chan struct{}, 1),
		textStyle:        lipgloss.NewStyle().Foreground(lipgloss.Color("#cdd6f4")).Bold(true),
		userInputStyle:   lipgloss.NewStyle().Foreground(lipgloss.Color("#89d4fa")).Bold(true),
		toolStyle:        lipgloss.NewStyle().Foreground(lipgloss.Color("#f9e2af")),
		toolContentStyle: lipgloss.NewStyle().Foreground(lipgloss.Color("#89d4fa")),
		reasoningStyle:   lipgloss.NewStyle().Foreground(lipgloss.Color("#6c7086")).Italic(true),
		errorStyle:       lipgloss.NewStyle().Foreground(lipgloss.Color("#f38ba8")),
		systemStyle:      lipgloss.NewStyle().Foreground(lipgloss.Color("#6c7086")),
		promptStyle:      lipgloss.NewStyle().Foreground(lipgloss.Color("#a6e3a1")).Bold(true),
	}
}

func (w *terminalOutput) Write(p []byte) (n int, err error) {
	w.buffer = append(w.buffer, p...)
	w.processBuffer()
	return len(p), nil
}

func (w *terminalOutput) WriteString(s string) (int, error) {
	return w.Write([]byte(s))
}

func (w *terminalOutput) Flush() error {
	return nil
}

func (w *terminalOutput) processBuffer() {
	for len(w.buffer) >= 5 {
		tag := w.buffer[0]

		length := int32(binary.BigEndian.Uint32(w.buffer[1:5]))

		if len(w.buffer) < 5+int(length) {
			break
		}

		value := string(w.buffer[5 : 5+length])
		w.writeColored(tag, value)

		w.buffer = w.buffer[5+length:]
	}
}

func (w *terminalOutput) writeColored(tag byte, value string) {
	switch tag {
	case stream.TagText, stream.TagTool, stream.TagReasoning, stream.TagError, stream.TagSystem, stream.TagPromptStart, stream.TagStreamGap:
		// Notify that content changed (non-blocking)
		select {
		case w.updateChan <- struct{}{}:
		default:
		}
	}

	trimRight := true
	var output string
	switch tag {
	case stream.TagText:
		output = w.textStyle.Render(value)
	case stream.TagTool:
		output = w.colorizeTool(value)
	case stream.TagReasoning:
		output = w.reasoningStyle.Render(value)
	case stream.TagError:
		output = w.errorStyle.Render(value)
	case stream.TagSystem:
		output = w.systemStyle.Render(value)
	case stream.TagPromptStart:
		output = w.promptStyle.Render("> ") + w.userInputStyle.Render(value)
	case stream.TagStreamGap:
		trimRight = false
		output = "\n"
	default:
		trimRight = false
		output = value
	}

	// @WORKAROUND:
	// The `xxStyle.Render` adds many extra spaces on the right (after escape sequences)
	// Remove them to keep the display right.
	if trimRight {
		output = strings.TrimRight(output, " ")
	}

	w.display.Append(output)
}

func (w *terminalOutput) colorizeTool(value string) string {
	colonIdx := strings.Index(value, ":")
	if colonIdx > 0 {
		toolName := value[:colonIdx]
		rest := value[colonIdx:]
		return w.toolStyle.Render(toolName) + w.toolContentStyle.Render(rest)
	}
	return w.toolStyle.Render(value)
}

// Terminal is the main Terminal model
type Terminal struct {
	session          *agentpkg.Session
	terminalOutput   *terminalOutput
	display          viewport.Model
	input            textinput.Model
	status           string
	quitting         bool
	confirmDialog    bool
	focusedWindow    string // "display" or "input"
	userScrolledAway bool   // true after user manually scrolls up
	windowWidth      int    // actual window width
	editorContent    string // content from external editor with newlines preserved

	inputStyle  lipgloss.Style
	statusStyle lipgloss.Style
}

// NewTerminal creates a new Terminal model
func NewTerminal(session *agentpkg.Session, terminalOutput *terminalOutput) *Terminal {
	input := textinput.New()
	input.Placeholder = "Enter your prompt..."
	input.Focus()
	input.Prompt = "> "

	inputStyle := lipgloss.NewStyle()
	statusStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("#45475a")) // Dimmed for status bar

	var display = viewport.New(80, 20)
	display.SetContent(welcomeText)

	return &Terminal{
		session:        session,
		terminalOutput: terminalOutput,
		display:        display,
		input:          input,
		status:         "Context: 0 | Total: 0",
		windowWidth:    80, // Will be updated on first WindowSizeMsg
		inputStyle:     inputStyle,
		statusStyle:    statusStyle,
		focusedWindow:  "input",
	}
}

// Init initializes the Terminal
func (m *Terminal) Init() tea.Cmd {
	return nil
}

type tickMsg struct{}

type editorFinishedMsg struct {
	content string
	err     error
}

// Update handles messages
func (m *Terminal) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	// Non-blocking check for display updates
	select {
	case <-m.terminalOutput.updateChan:
		m.updateDisplayContent()
		m.updateStatus()
	default:
	}

	switch msg := msg.(type) {
	case tea.KeyMsg:
		return m.handleKeyMsg(msg)
	case tea.WindowSizeMsg:
		m.windowWidth = msg.Width
		m.display.Width = max(0, msg.Width-8)   // Leave room for padding (4 on each side)
		m.display.Height = max(0, msg.Height-4) // Leave room for input box (3) and status bar (1)
		return m, nil
	case tickMsg:
		m.updateDisplayContent()
		m.updateStatus()
		if m.session != nil && m.session.IsInProgress() {
			return m, tea.Tick(50*time.Millisecond, func(t time.Time) tea.Msg {
				return tickMsg{}
			})
		}
		return m, nil
	case editorFinishedMsg:
		if msg.err != nil {
			m.terminalOutput.display.Append(m.terminalOutput.errorStyle.Render(fmt.Sprintf("Editor error: %v", msg.err)))
		} else if msg.content != "" {
			m.editorContent = msg.content
			lineCount := strings.Count(msg.content, "\n") + 1
			preview := strings.Fields(msg.content)
			var previewText string
			if len(preview) > 0 && len(preview[0]) > 20 {
				previewText = preview[0][:20] + "..."
			} else if len(preview) > 0 {
				previewText = preview[0]
			} else {
				previewText = "(empty)"
			}
			m.input.SetValue(fmt.Sprintf("[%d lines] %s (press Enter to send)", lineCount, previewText))
			m.input.CursorEnd()
			m.input.Focus()
		}
		return m, nil
	}

	// Update input
	m.input, _ = m.input.Update(msg)
	return m, nil
}

func (m *Terminal) handleKeyMsg(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	// Handle confirm dialog
	if m.confirmDialog {
		switch msg.String() {
		case "y", "Y":
			m.quitting = true
			return m, tea.Quit
		case "n", "N", "esc", "ctrl+c":
			m.confirmDialog = false
			return m, nil
		}
		return m, nil
	}

	// Handle Tab to switch focus
	if msg.Type == tea.KeyTab {
		if m.focusedWindow == "display" {
			m.focusedWindow = "input"
			m.input.Focus()
		} else {
			m.focusedWindow = "display"
			m.input.Blur()
		}
		return m, nil
	}

	// Handle j/k for display window scrolling
	if m.focusedWindow == "display" {
		switch msg.String() {
		case "j":
			m.display.ScrollDown(1)
			// Check if now at bottom
			if m.display.AtBottom() {
				m.userScrolledAway = false
			} else {
				m.userScrolledAway = true
			}
			return m, nil
		case "k":
			m.display.ScrollUp(1)
			m.userScrolledAway = true
			return m, nil
		case "G":
			m.display.GotoBottom()
			m.userScrolledAway = false
			return m, nil
		case "g":
			m.display.GotoTop()
			m.userScrolledAway = true
			return m, nil
		case "ctrl+d":
			m.display.ScrollDown(m.display.Height / 2)
			if m.display.AtBottom() {
				m.userScrolledAway = false
			} else {
				m.userScrolledAway = true
			}
			return m, nil
		case "ctrl+u":
			m.display.ScrollUp(m.display.Height / 2)
			m.userScrolledAway = true
			return m, nil
		}
	}

	switch msg.Type {
	case tea.KeyCtrlC, tea.KeyEsc:
		m.confirmDialog = true
		return m, nil
	case tea.KeyCtrlG:
		// Cancel the current request if one is in progress
		if m.session.IsInProgress() {
			if m.session.CancelCurrent() {
				// Cancel initiated successfully - message will be shown via TLV
			}
		}
		return m, nil
	case tea.KeyCtrlO:
		// Open external editor for multi-line input
		return m, m.openEditor()
	case tea.KeyEnter:
		var prompt string

		// Check if we have editor content to submit
		if m.editorContent != "" {
			prompt = m.editorContent
			m.editorContent = ""
		} else {
			prompt = m.input.Value()
		}

		if prompt == "" {
			return m, nil
		}

		// Handle commands
		if command, found := strings.CutPrefix(prompt, "/"); found {
			if command == "quit" {
				m.confirmDialog = true
			} else {
				if err := m.session.SubmitCommand(command); err != nil {
					m.terminalOutput.display.Append(m.terminalOutput.errorStyle.Render(err.Error()))
				}
				m.input.SetValue("")
				// Start ticking to check for updates during command processing
				return m, tea.Tick(50*time.Millisecond, func(t time.Time) tea.Msg {
					return tickMsg{}
				})
			}

			m.display.GotoBottom()
			return m, nil
		}

		// Submit prompt - session handles queuing
		m.session.SubmitPrompt(prompt)

		m.input.SetValue("")
		m.updateStatus()

		// Start ticking to check for updates during processing
		return m, tea.Tick(50*time.Millisecond, func(t time.Time) tea.Msg {
			return tickMsg{}
		})
	}

	oldValue := m.input.Value()
	m.input, _ = m.input.Update(msg)
	newValue := m.input.Value()

	// If user modified input and we have editorContent, clear it
	if m.editorContent != "" && oldValue != newValue && !strings.HasPrefix(oldValue, "[") {
		m.editorContent = ""
	}

	return m, nil
}

func (m *Terminal) updateStatus() {
	if m.session != nil {
		m.status = fmt.Sprintf("Context: %d | Total: %d", m.session.ContextTokens, m.session.TotalSpent.TotalTokens)
	} else {
		m.status = ""
	}
}

func (m *Terminal) openEditor() tea.Cmd {
	editorCmd := getEditorCommand(os.Getenv("EDITOR"))

	if editorCmd == "" {
		return func() tea.Msg {
			return editorFinishedMsg{content: "", err: fmt.Errorf("no editor found (tried: vim, vi, nano)")}
		}
	}

	tmpFile, err := os.CreateTemp("", tempFilePrefix)
	if err != nil {
		return func() tea.Msg {
			return editorFinishedMsg{content: "", err: err}
		}
	}

	tmpFileName := tmpFile.Name()

	existingContent := m.getInputForEditor()

	if existingContent != "" {
		if _, err := tmpFile.WriteString(existingContent); err != nil {
			tmpFile.Close()
			os.Remove(tmpFileName)
			return func() tea.Msg {
				return editorFinishedMsg{content: "", err: err}
			}
		}
	}
	tmpFile.Close()

	cmd := exec.Command(editorCmd, tmpFileName)

	return tea.ExecProcess(cmd, func(err error) tea.Msg {
		defer os.Remove(tmpFileName)

		if err != nil {
			return editorFinishedMsg{content: "", err: err}
		}

		content, readErr := os.ReadFile(tmpFileName)
		if readErr != nil {
			return editorFinishedMsg{content: "", err: readErr}
		}

		return editorFinishedMsg{content: string(content), err: nil}
	})
}

// getInputForEditor returns the content to pre-populate in the editor
// If editorContent is set (from a previous Ctrl+O), use that.
// Otherwise, use the current input value.
func (m *Terminal) getInputForEditor() string {
	if m.editorContent != "" {
		return m.editorContent
	}
	return m.input.Value()
}

func (m *Terminal) updateDisplayContent() {
	newContent := m.terminalOutput.display.GetAll()

	// Wordwrap to viewport width for proper word boundary wrapping
	width := m.display.Width

	if width > 0 {
		newContent = wordwrap(newContent, width)
	}
	m.display.SetContent(newContent)
	// Auto-scroll by default, unless user has manually scrolled away
	if !m.userScrolledAway {
		m.display.GotoBottom()
	}
}

// View renders the Terminal
func (m *Terminal) View() string {
	// Display content is already updated via updateDisplayContent()
	// Use window width for input and status, viewport width for display
	windowWidth := m.windowWidth

	// Style display, input, and status (accounting for padding)
	displayStyle := lipgloss.NewStyle().Padding(0, 4)
	inputStyle := m.inputStyle.Width(max(0, windowWidth-4))
	statusStyle := m.statusStyle.Width(max(0, windowWidth-8)).Padding(0, 4)

	// Input border color based on focus
	var inputBorderColor string
	if m.focusedWindow == "input" {
		inputBorderColor = "#89d4fa" // Bright cyan for focused
	} else {
		inputBorderColor = "#45475a" // Dimmed for unfocused
	}

	inputBorderStyle := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(lipgloss.Color(inputBorderColor)).
		Padding(0, 1)

	statusBar := statusStyle.Render(m.status)

	// Build the view
	var sb strings.Builder

	// Display area with padding but no border
	sb.WriteString(displayStyle.Render(m.display.View()))

	// Input area with border
	sb.WriteString("\n")
	if m.confirmDialog {
		confirmStyle := lipgloss.NewStyle().
			Width(max(0, windowWidth-4)).
			Foreground(lipgloss.Color("#f9e2af")).
			Bold(true)
		sb.WriteString(inputBorderStyle.Render(confirmStyle.Render("Confirm exit? Press y/n")))
	} else {
		// Set prompt color to match border color
		m.input.PromptStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color(inputBorderColor)).
			Bold(true)

		// Apply dimming to text content when unfocused
		if m.focusedWindow == "input" {
			m.input.TextStyle = lipgloss.NewStyle()
		} else {
			m.input.TextStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("#45475a"))
		}

		inputContent := inputStyle.Render(m.input.View())
		sb.WriteString(inputBorderStyle.Render(inputContent))
	}

	// Status bar
	sb.WriteString("\n")
	sb.WriteString(statusBar)

	return sb.String()
}

var (
	_ tea.Model = (*Terminal)(nil)
)

// wordwrap breaks text to fit the given width
func wordwrap(text string, width int) string {
	if width <= 0 || text == "" {
		return text
	}

	var result strings.Builder

	for line := range strings.SplitSeq(text, "\n") {
		if lipgloss.Width(line) <= width {
			result.WriteString(line)
			result.WriteString("\n")
			continue
		}

		// Break line at width limit, handling ANSI escape sequences
		for len(line) > 0 {
			breakAt := 0
			currentWidth := 0

			for breakAt < len(line) {
				skip := skipEscapeSequence(line[breakAt:])
				if skip > 0 {
					breakAt += skip
					continue
				}

				r := rune(line[breakAt])
				charWidth := lipgloss.Width(string(r))

				if currentWidth+charWidth > width {
					break
				}
				currentWidth += charWidth
				breakAt++
			}

			// Try to break at last space for word boundary
			lastSpace := -1
			for i := breakAt - 1; i >= 0; i-- {
				if line[i] == ' ' {
					lastSpace = i
					break
				}
			}

			if lastSpace > 0 {
				breakAt = lastSpace + 1
			}

			if breakAt == 0 {
				breakAt = 1
			}

			result.WriteString(line[:breakAt])
			result.WriteString("\n")
			line = line[breakAt:]
		}
	}

	return result.String()
}

// getEditorCommand returns the editor command to use
// First checks EDITOR env var, then tries vim, vi, nano in order
func getEditorCommand(editorCmd string) string {
	if editorCmd != "" {
		return editorCmd
	}

	for _, editor := range []string{"vim", "vi", "nano"} {
		path, err := exec.LookPath(editor)
		if err == nil {
			return path
		}
	}

	return ""
}

// skipEscapeSequence returns the length of an ANSI escape sequence at the start of s,
// or 0 if there is no escape sequence.
func skipEscapeSequence(s string) int {
	if len(s) == 0 || s[0] != '\x1b' {
		return 0
	}
	if len(s) < 2 {
		return 0
	}

	switch s[1] {
	case '[':
		return skipCSI(s)
	case ']':
		return skipOSC(s)
	default:
		return 2
	}
}

// skipCSI skips a CSI (Control Sequence Introducer) sequence: ESC [ ... <final byte>
// Final byte is in range 0x40-0x7E (@A-Z[\]^_`a-z{|}~)
func skipCSI(s string) int {
	if len(s) < 3 {
		return len(s)
	}

	pos := 2
	for pos < len(s) {
		c := s[pos]

		if c >= 0x40 && c <= 0x7E {
			return pos + 1
		}

		if c >= 0x20 && c <= 0x3F {
			pos++
		} else {
			break
		}
	}

	return pos
}

// skipOSC skips an OSC (Operating System Command) sequence: ESC ] ... ST
// ST (String Terminator) is either BEL (\x07) or ESC \ (\x1b\\)
func skipOSC(s string) int {
	if len(s) < 3 {
		return len(s)
	}

	pos := 2
	for pos < len(s) {
		c := s[pos]

		if c == '\x07' {
			return pos + 1
		}

		if c == '\x1b' && pos+1 < len(s) && s[pos+1] == '\\' {
			return pos + 2
		}

		pos++
	}

	return pos
}
