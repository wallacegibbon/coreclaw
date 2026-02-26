package adaptors

import (
	"encoding/binary"
	"fmt"
	"os"
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

// Display is the display buffer for Terminal
var Display = NewDisplayBuffer()

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

	// Create a custom output that writes to display buffer with TLV support
	terminalOutput := newTerminalOutput()
	processor := agentpkg.NewProcessorWithIO(agent, &stream.NopInput{}, terminalOutput)
	a.processor = processor
	a.session = agentpkg.NewSession(agent, a.BaseURL, a.ModelName, processor)

	t := NewTerminal(a.session, terminalOutput)
	p := tea.NewProgram(t, tea.WithAltScreen(), tea.WithInput(os.Stdin), tea.WithOutput(os.Stdout))
	p.Run()
	return
}

// terminalOutput writes to the Terminal display with TLV support
type terminalOutput struct {
	display *DisplayBuffer
	buffer  []byte

	textStyle        lipgloss.Style
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
		textStyle:        lipgloss.NewStyle().Foreground(lipgloss.Color("#cdd6f4")).Bold(true),
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
	case stream.TagUsage:
		output = w.systemStyle.Render("Tokens: " + value)
	case stream.TagSystem:
		output = w.systemStyle.Render(value)
	case stream.TagPromptStart:
		output = w.promptStyle.Render("> ") + w.textStyle.Render(value)
	case stream.TagStreamGap:
		output = "\n"
	default:
		output = value
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
	session        *agentpkg.Session
	terminalOutput *terminalOutput
	display        viewport.Model
	input          textinput.Model
	status         string
	quitting       bool
	confirmDialog  bool
	focusedWindow  string // "display" or "input"

	inputStyle         lipgloss.Style
	promptStyle        lipgloss.Style
	statusStyle        lipgloss.Style
	borderStyle        lipgloss.Style
	displayBorderStyle lipgloss.Style
}

// NewTerminal creates a new Terminal model
func NewTerminal(session *agentpkg.Session, terminalOutput *terminalOutput) *Terminal {
	input := textinput.New()
	input.Placeholder = "Enter your prompt..."
	input.Focus()
	input.Prompt = "> "

	inputStyle := lipgloss.NewStyle()
	promptStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("#a6e3a1")).
		Bold(true)
	statusStyle := lipgloss.NewStyle().
		Background(lipgloss.Color("#45475a")).
		Foreground(lipgloss.Color("#cdd6f4"))
	borderStyle := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(lipgloss.Color("#45475a"))

	display := viewport.New(80, 20)
	display.SetContent("Welcome to CoreClaw Terminal!\n\nType your prompt below and press Enter to send.\n\n")

	return &Terminal{
		session:        session,
		terminalOutput: terminalOutput,
		display:        display,
		input:          input,
		status:         "Ready",
		inputStyle:     inputStyle,
		promptStyle:    promptStyle,
		statusStyle:    statusStyle,
		borderStyle:    borderStyle,
		displayBorderStyle: lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(lipgloss.Color("#45475a")),
		focusedWindow: "input",
	}
}

// Init initializes the Terminal
func (m *Terminal) Init() tea.Cmd {
	// Tick every 100ms to refresh display during processing
	return tea.Tick(100*time.Millisecond, func(t time.Time) tea.Msg {
		return tickMsg(t)
	})
}

type tickMsg time.Time

// Update handles messages
func (m *Terminal) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tickMsg:
		// Check if still processing
		if m.session.IsInProgress() {
			m.updateDisplayContent()
			m.updateStatus()
			return m, tea.Tick(100*time.Millisecond, func(t time.Time) tea.Msg {
				return tickMsg(t)
			})
		}
		m.updateDisplayContent()
		m.updateStatus()
		return m, nil
	case tea.KeyMsg:
		return m.handleKeyMsg(msg)
	case tea.WindowSizeMsg:
		m.display.Width = msg.Width
		m.display.Height = msg.Height - 6 // Leave room for input, status, and display border
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
			return m, nil
		case "k":
			m.display.ScrollUp(1)
			return m, nil
		case "ctrl+d":
			m.display.ScrollDown(m.display.Height / 2)
			return m, nil
		case "ctrl+u":
			m.display.ScrollUp(m.display.Height / 2)
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
	case tea.KeyEnter:
		prompt := m.input.Value()
		if prompt == "" {
			return m, nil
		}

		// Handle commands
		if strings.HasPrefix(prompt, "/") {
			command := strings.TrimPrefix(prompt, "/")
			if command == "quit" {
				m.confirmDialog = true
			} else {
				m.session.HandleCommand(command)
				m.input.SetValue("")
			}
			//return m, tea.Tick(100*time.Millisecond, func(t time.Time) tea.Msg {
			//return tickMsg(t)

			m.display.GotoBottom()
			return m, nil
		}

		// Submit prompt - session handles queuing
		m.session.SubmitPrompt(prompt)

		m.input.SetValue("")
		m.updateStatus()

		// Start ticking to refresh display
		return m, tea.Tick(100*time.Millisecond, func(t time.Time) tea.Msg {
			return tickMsg(t)
		})
	}

	m.input, _ = m.input.Update(msg)
	return m, nil
}

func (m *Terminal) updateStatus() {
	if m.session != nil && m.session.IsInProgress() {
		m.status = "Processing..."
	} else if m.session != nil {
		m.status = fmt.Sprintf("Ready | Context: %d | Total: %d", m.session.ContextTokens, m.session.TotalSpent.TotalTokens)
	} else {
		m.status = "Ready"
	}
}

func (m *Terminal) updateDisplayContent() {
	newContent := m.terminalOutput.display.GetAll()
	width := m.display.Width - 5
	if width > 0 {
		newContent = wordwrap(newContent, width)
	}
	m.display.SetContent(newContent)
	m.display.GotoBottom()
}

// View renders the Terminal
func (m *Terminal) View() string {
	// Display content is already updated via updateDisplayContent()
	// Just get the current width for styling
	width := m.display.Width

	// Style input and status to match viewport width
	inputStyle := m.inputStyle.Width(width - 6)
	statusStyle := m.statusStyle.Width(width).Padding(0, 1)

	// Determine border colors based on focus
	var displayBorderColor string
	var inputBorderColor string
	if m.focusedWindow == "display" {
		displayBorderColor = "#89d4fa" // Bright cyan for focused
		inputBorderColor = "#45475a"    // Dimmed for unfocused
	} else {
		displayBorderColor = "#45475a" // Dimmed for unfocused
		inputBorderColor = "#89d4fa"   // Bright cyan for focused
	}

	displayBorderStyle := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(lipgloss.Color(displayBorderColor)).
		Width(width - 2).Padding(0, 1)

	inputBorderStyle := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(lipgloss.Color(inputBorderColor)).
		Width(width - 2).Padding(0, 1)

	statusBar := statusStyle.Render(m.status)

	// Build the view
	var sb strings.Builder

	// Display area with border
	sb.WriteString(displayBorderStyle.Render(m.display.View()))

	// Input area with border
	sb.WriteString("\n")
	if m.confirmDialog {
		confirmStyle := lipgloss.NewStyle().
			Foreground(lipgloss.Color("#f9e2af")).
			Bold(true)
		sb.WriteString(inputBorderStyle.Render(confirmStyle.Render("Confirm exit? Press y/n")))
	} else {
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

// wordwrap wraps text to fit the given width using display width
func wordwrap(text string, width int) string {
	if width <= 0 || strings.TrimSpace(text) == "" {
		return text
	}

	var result strings.Builder
	lines := strings.Split(text, "\n")

	for _, line := range lines {
		if lipgloss.Width(line) <= width {
			result.WriteString(line)
			result.WriteString("\n")
			continue
		}

		// Wrap the line
		words := strings.Fields(line)
		currentLen := 0

		for _, word := range words {
			wordLen := lipgloss.Width(word)
			if currentLen == 0 {
				result.WriteString(word)
				currentLen = wordLen
			} else if currentLen+1+wordLen <= width {
				result.WriteString(" ")
				result.WriteString(word)
				currentLen += 1 + wordLen
			} else {
				result.WriteString("\n")
				result.WriteString(word)
				currentLen = wordLen
			}
		}
		result.WriteString("\n")
	}

	return result.String()
}
