package adaptors

import (
	"context"
	"encoding/binary"
	"fmt"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/charmbracelet/bubbles/textinput"
	"github.com/charmbracelet/bubbles/viewport"
	"github.com/charmbracelet/lipgloss"
	tea "github.com/charmbracelet/bubbletea"

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

	ctx    context.Context
	cancel context.CancelFunc
}

// NewTerminalAdaptor creates a new Terminal adaptor
func NewTerminalAdaptor(agentFactory AgentFactory, baseURL, modelName string) *TerminalAdaptor {
	ctx, cancel := context.WithCancel(context.Background())
	return &TerminalAdaptor{
		AgentFactory: agentFactory,
		BaseURL:      baseURL,
		ModelName:    modelName,
		ctx:          ctx,
		cancel:       cancel,
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

	tui := NewTerminal(a.session, a.ctx, a.cancel, terminalOutput)
	p := tea.NewProgram(tui, tea.WithAltScreen(), tea.WithInput(os.Stdin), tea.WithOutput(os.Stdout))
	p.Run()
	return
}

// terminalOutput writes to the Terminal display with TLV support
type terminalOutput struct {
	display *DisplayBuffer
	buffer  []byte

	textStyle      lipgloss.Style
	toolStyle      lipgloss.Style
	reasoningStyle lipgloss.Style
	errorStyle     lipgloss.Style
	systemStyle    lipgloss.Style
	promptStyle    lipgloss.Style
}

func newTerminalOutput() *terminalOutput {
	return &terminalOutput{
		display:       NewDisplayBuffer(),
		textStyle:      lipgloss.NewStyle().Foreground(lipgloss.Color("#cdd6f4")),
		toolStyle:      lipgloss.NewStyle().Foreground(lipgloss.Color("#f9e2af")),
		reasoningStyle: lipgloss.NewStyle().Foreground(lipgloss.Color("#6c7086")),
		errorStyle:     lipgloss.NewStyle().Foreground(lipgloss.Color("#f38ba8")),
		systemStyle:    lipgloss.NewStyle().Foreground(lipgloss.Color("#6c7086")),
		promptStyle:    lipgloss.NewStyle().Foreground(lipgloss.Color("#a6e3a1")).Bold(true),
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
		if !isValidTag(tag) {
			w.display.Append(string(w.buffer[0]))
			w.buffer = w.buffer[1:]
			continue
		}

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
		output = w.systemStyle.Render("Tokens: "+value)
	case stream.TagSystem:
		output = w.systemStyle.Render(value)
	case stream.TagStreamGap:
		output = "\n"
	case stream.TagPromptStart:
		// Prompt started - display as user prompt with styled ">" prefix
		output = "\n" + w.promptStyle.Render("> ") + w.textStyle.Render(value) + "\n"
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
		return w.toolStyle.Render(toolName) + w.textStyle.Render(rest)
	}
	return w.toolStyle.Render(value)
}

// Terminal is the main Terminal model
type Terminal struct {
	session   *agentpkg.Session
	ctx       context.Context
	cancel    context.CancelFunc
	terminalOutput *terminalOutput
	display   viewport.Model
	input     textinput.Model
	status    string
	quitting  bool

	inputStyle  lipgloss.Style
	promptStyle lipgloss.Style
	statusStyle lipgloss.Style
}

// NewTerminal creates a new Terminal model
func NewTerminal(session *agentpkg.Session, ctx context.Context, cancel context.CancelFunc, terminalOutput *terminalOutput) *Terminal {
	input := textinput.New()
	input.Placeholder = "Enter your prompt..."
	input.Focus()
	input.Prompt = "> "

	inputStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("#89b4fa"))
	promptStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("#a6e3a1")).
		Bold(true)
	statusStyle := lipgloss.NewStyle().
		Background(lipgloss.Color("#45475a")).
		Foreground(lipgloss.Color("#cdd6f4"))

	display := viewport.New(80, 20)
	display.SetContent("Welcome to CoreClaw Terminal!\n\nType your prompt below and press Enter to send.\n\n")

	return &Terminal{
		session:     session,
		ctx:         ctx,
		cancel:      cancel,
		terminalOutput:   terminalOutput,
		display:     display,
		input:       input,
		status:      "Ready",
		inputStyle:  inputStyle,
		promptStyle:    promptStyle,
		statusStyle:    statusStyle,
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
		// Done processing - check if context was cancelled and recreate if needed
		if m.ctx.Err() == context.Canceled {
			m.ctx, m.cancel = context.WithCancel(context.Background())
		}
		m.updateDisplayContent()
		m.updateStatus()
		return m, nil
	case tea.KeyMsg:
		return m.handleKeyMsg(msg)
	case tea.WindowSizeMsg:
		m.display.Width = msg.Width
		m.display.Height = msg.Height - 2 // Leave room for input and status
		return m, nil
	}

	// Update input
	m.input, _ = m.input.Update(msg)
	return m, nil
}

func (m *Terminal) handleKeyMsg(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.Type {
	case tea.KeyCtrlC, tea.KeyEsc:
		m.quitting = true
		return m, tea.Quit
	case tea.KeyCtrlG:
		// Cancel the current request if one is in progress
		if m.session.IsInProgress() {
			m.session.CancelCurrent()
			m.terminalOutput.display.Append("\n" + m.statusStyle.Render("Request cancelled.") + "\n")
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
			m.session.HandleCommandStr(command)
			m.input.SetValue("")
			m.display.GotoBottom()
			return m, nil
		}

		// Submit prompt - session handles queuing
		m.session.SubmitPrompt(m.ctx, prompt)

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
	width := m.display.Width
	if width > 0 {
		newContent = wordwrap(newContent, width)
	}
	m.display.SetContent(newContent)
	m.display.GotoBottom()
}

// View renders the Terminal
func (m *Terminal) View() string {
	// Update display content from buffer with word wrapping
	newContent := m.terminalOutput.display.GetAll()
	width := m.display.Width
	if width > 0 {
		newContent = wordwrap(newContent, width)
	}
	m.display.SetContent(newContent)
	m.display.GotoBottom()

	// Style input and status to match viewport width
	inputStyle := m.inputStyle.Width(width)
	statusStyle := m.statusStyle.Width(width)

	statusBar := statusStyle.Render(m.status)

	// Build the view
	var sb strings.Builder

	// Display area
	sb.WriteString(m.display.View())

	// Input area
	sb.WriteString("\n")
	sb.WriteString(inputStyle.Render(m.input.View()))

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
