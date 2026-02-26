package adaptors

import (
	"bufio"
	"context"
	"encoding/binary"
	"fmt"
	"io"
	"os"
	"os/signal"
	"strings"

	"github.com/wallacegibbon/coreclaw/internal/stream"
)

// TerminalAdaptor connects stdio to the agent processor
type TerminalAdaptor struct {
	Input        stream.Input
	Output       stream.Output
	AgentFactory AgentFactory
	BaseURL      string
	ModelName    string
}

// NewTerminalAdaptor creates a new terminal adaptor with stdio
func NewTerminalAdaptor(factory AgentFactory, baseURL, modelName string) *TerminalAdaptor {
	return &TerminalAdaptor{
		Input:        &StdinReader{Reader: bufio.NewReader(os.Stdin)},
		Output:       &TLVWriter{Writer: bufio.NewWriter(os.Stdout)},
		AgentFactory: factory,
		BaseURL:      baseURL,
		ModelName:    modelName,
	}
}

// Start runs the terminal adaptor in interactive mode
func (a *TerminalAdaptor) Start() {
	agent := a.AgentFactory()
	session, _ := NewSession(agent, a.BaseURL, a.ModelName, a.Input, a.Output)

	reader := bufio.NewReader(a.Input)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt)
	defer signal.Stop(sigChan)

	go func() {
		for range sigChan {
			if session.IsInProgress() {
				cancel()
			}
		}
	}()

	// Interactive loop
	for {
		fmt.Fprint(os.Stderr, Green("> "))
		input, err := reader.ReadString('\n')
		if err != nil {
			return
		}
		userPrompt := strings.TrimSpace(input)
		if userPrompt == "" {
			continue
		}

		// Handle commands like /summarize
		if strings.HasPrefix(userPrompt, "/") {
			command := strings.TrimPrefix(userPrompt, "/")
			_, err := session.HandleCommand(ctx, command)
			if err != nil && ctx.Err() == context.Canceled {
				ctx, cancel = context.WithCancel(context.Background())
				defer cancel()
			}
			continue
		}

		// Submit prompt - Session handles queue internally
		session.SubmitPrompt(ctx, userPrompt)

		if ctx.Err() == context.Canceled {
			ctx, cancel = context.WithCancel(context.Background())
			defer cancel()
		}
	}
}

// StdinReader wraps os.Stdin as a stream.Input
type StdinReader struct {
	*bufio.Reader
}

// Read implements the stream.Input interface
func (r *StdinReader) Read(p []byte) (n int, err error) {
	return r.Reader.Read(p)
}

// TLVWriter wraps an io.Writer and decodes TLV to apply colors
type TLVWriter struct {
	*bufio.Writer
	buffer []byte
}

// Write implements the stream.Output interface - buffers and processes TLV
func (w *TLVWriter) Write(p []byte) (n int, err error) {
	w.buffer = append(w.buffer, p...)
	w.processBuffer()
	return len(p), nil
}

// WriteString implements the stream.Output interface
func (w *TLVWriter) WriteString(s string) (int, error) {
	return w.Write([]byte(s))
}

// processBuffer extracts and processes complete TLV messages
func (w *TLVWriter) processBuffer() {
	for len(w.buffer) >= 5 {
		tag := w.buffer[0]
		if !isValidTag(tag) {
			// Not a valid tag, write the byte as-is
			w.Writer.WriteByte(w.buffer[0])
			w.buffer = w.buffer[1:]
			continue
		}

		// Read length (big-endian int32)
		length := int32(binary.BigEndian.Uint32(w.buffer[1:5]))

		// Check if we have complete message
		if len(w.buffer) < 5+int(length) {
			break // Wait for more data
		}

		// Extract value
		value := string(w.buffer[5 : 5+length])

		// Apply color based on tag and write
		w.writeColored(tag, value)

		// Remove processed bytes
		w.buffer = w.buffer[5+length:]
	}
}

// writeColored writes the value with appropriate color based on tag
func (w *TLVWriter) writeColored(tag byte, value string) {
	var colored string
	switch tag {
	case stream.TagText:
		colored = Bright(value)
	case stream.TagTool:
		colored = "\n" + w.colorizeTool(value) + "\n"
	case stream.TagReasoning:
		colored = Dim(value)
	case stream.TagError:
		colored = Dim(value)
	case stream.TagUsage:
		colored = "\n" + Dim("Tokens: "+value) + "\n"
	case stream.TagSystem:
		colored = "\n" + Dim(value) + "\n"
	default:
		colored = value
	}
	w.Writer.WriteString(colored)
}

// colorizeTool detects tool call format "toolname: args" and applies colors
func (w *TLVWriter) colorizeTool(value string) string {
	// Find the first ':'
	colonIdx := -1
	for i := 0; i < len(value); i++ {
		if value[i] == ':' {
			colonIdx = i
			break
		}
	}
	if colonIdx > 0 {
		// Extract tool name and rest
		toolName := value[:colonIdx]
		rest := value[colonIdx:]
		return Yellow(toolName) + Green(rest)
	}
	return Yellow(value)
}

// Flush flushes any buffered data
func (w *TLVWriter) Flush() error {
	// Process any remaining non-TLV data
	if len(w.buffer) > 0 {
		w.Writer.Write(w.buffer)
		w.buffer = nil
	}
	return w.Writer.Flush()
}

// Close implements io.Closer
func (w *TLVWriter) Close() error {
	return w.Flush()
}

// isValidTag checks if a byte is a valid TLV tag
func isValidTag(b byte) bool {
	switch b {
	case stream.TagText, stream.TagTool, stream.TagReasoning, stream.TagError, stream.TagUsage, stream.TagSystem, stream.TagStreamGap, stream.TagPromptStart:
		return true
	}
	return false
}

// NewOutputStream creates an OutputStream from an io.Writer
func NewOutputStream(w io.Writer) stream.Output {
	if bw, ok := w.(*bufio.Writer); ok {
		return &TLVWriter{Writer: bw}
	}
	return &stream.GenericWriter{Writer: w}
}

// NewInputStream creates an InputStream from an io.Reader
func NewInputStream(r io.Reader) stream.Input {
	if br, ok := r.(*bufio.Reader); ok {
		return &StdinReader{Reader: br}
	}
	return &stream.GenericReader{Reader: r}
}
