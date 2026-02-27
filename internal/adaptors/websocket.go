package adaptors

import (
	"fmt"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/gorilla/websocket"

	_ "embed"

	agentpkg "github.com/wallacegibbon/coreclaw/internal/agent"
	"github.com/wallacegibbon/coreclaw/internal/stream"
)

var upgrader = websocket.Upgrader{
	CheckOrigin: func(r *http.Request) bool {
		return true
	},
}

// WebSocketAdaptor connects WebSocket to the agent processor
type WebSocketAdaptor struct {
	AgentFactory AgentFactory
	BaseURL      string
	ModelName    string
	Server       *http.Server
}

// NewWebSocketAdaptor creates a new WebSocket adaptor that listens on the given port
// Each client gets its own agent session
func NewWebSocketAdaptor(port string, factory AgentFactory, baseURL, modelName string) *WebSocketAdaptor {
	mux := http.NewServeMux()

	// Handle WebSocket
	mux.HandleFunc("/ws", handleWebSocket(factory, baseURL, modelName))

	// Serve embedded index.html
	mux.HandleFunc("/", serveIndex)

	server := &http.Server{
		Addr:    port,
		Handler: mux,
	}

	return &WebSocketAdaptor{
		AgentFactory: factory,
		BaseURL:      baseURL,
		ModelName:    modelName,
		Server:       server,
	}
}

// serveIndex serves the embedded index.html
func serveIndex(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.Write(indexHTML)
}

// Start starts the WebSocket server in a goroutine
func (a *WebSocketAdaptor) Start() {
	go func() {
		a.Server.ListenAndServe()
	}()
}

// handleWebSocket handles WebSocket connections with per-client sessions
func handleWebSocket(factory AgentFactory, baseURL, modelName string) func(http.ResponseWriter, *http.Request) {
	return func(w http.ResponseWriter, r *http.Request) {
		conn, err := upgrader.Upgrade(w, r, nil)
		if err != nil {
			return
		}
		defer conn.Close()

		// Create per-client streams
		input := &clientInput{
			clientCh: make(chan []byte, 10),
		}
		output := &clientOutput{
			conn:    conn,
			closeCh: make(chan struct{}),
		}
		defer close(output.closeCh)

		// Create a new agent, processor, and session for this client
		agent := factory()
		session := NewSession(agent, baseURL, modelName, input, output)

		// Set session on output and start status updater
		output.session = session
		go output.startStatusUpdater()

		// Read loop - handles client input
		go func() {
			for {
				_, message, err := conn.ReadMessage()
				if err != nil {
					return
				}

				msgStr := strings.TrimSpace(string(message))
				if msgStr == "" {
					continue
				}
				input.clientCh <- message
			}
		}()

		// Interactive loop - synchronous like terminal
		for {
			line, err := input.readLine()
			if err != nil {
				return
			}
			userPrompt := strings.TrimSpace(line)
			if userPrompt == "" {
				continue
			}

			// Handle commands like /summarize
			if strings.HasPrefix(userPrompt, "/") {
				command := strings.TrimPrefix(userPrompt, "/")
				if err := session.SubmitCommand(command); err != nil {
					stream.WriteTLV(session.Processor.Output, stream.TagError, err.Error())
				}
				continue
			}

			// Submit prompt - Session handles queue internally
			session.SubmitPrompt(userPrompt)
		}
	}
}

// clientInput implements stream.Input for a single WebSocket client
type clientInput struct {
	clientCh chan []byte
	buf      []byte
}

// readLine reads a newline-terminated line from the client
func (i *clientInput) readLine() (string, error) {
	var line []byte

	for {
		// If we have buffered data, check for newline
		if len(i.buf) > 0 {
			for idx, b := range i.buf {
				if b == '\n' {
					line = append(line, i.buf[:idx]...)
					i.buf = i.buf[idx+1:]
					return string(line), nil
				}
			}
			// No newline found, append all buffer and continue
			line = append(line, i.buf...)
			i.buf = nil
		}

		// Wait for more data
		msg, ok := <-i.clientCh
		if !ok {
			return string(line), nil
		}
		i.buf = msg
	}
}

// Read implements stream.Input interface (used by processor)
func (i *clientInput) Read(p []byte) (n int, err error) {
	if len(i.buf) > 0 {
		n = copy(p, i.buf)
		i.buf = i.buf[n:]
		return n, nil
	}

	msg, ok := <-i.clientCh
	if !ok {
		return 0, nil
	}

	i.buf = msg
	n = copy(p, i.buf)
	i.buf = i.buf[n:]
	return n, nil
}

// clientOutput implements stream.Output for a single WebSocket client
type clientOutput struct {
	conn       *websocket.Conn
	session    *agentpkg.Session
	mu         sync.Mutex
	closeCh    chan struct{}
	lastStatus string
}

// Write implements stream.Output
func (o *clientOutput) Write(p []byte) (n int, err error) {
	o.mu.Lock()
	defer o.mu.Unlock()
	err = o.conn.WriteMessage(websocket.BinaryMessage, p)
	if err != nil {
		return 0, err
	}
	return len(p), nil
}

// WriteString implements stream.Output
func (o *clientOutput) WriteString(s string) (int, error) {
	return o.Write([]byte(s))
}

// Flush implements stream.Output
func (o *clientOutput) Flush() error {
	return nil
}

// startStatusUpdater periodically sends status updates to the client
// NOTE: Just a quick and dirty workaround since websocket client can not get session data directly like terminal client.
func (o *clientOutput) startStatusUpdater() {
	ticker := time.NewTicker(500 * time.Millisecond)
	defer ticker.Stop()
	for {
		select {
		case <-ticker.C:
			o.mu.Lock()
			if o.session != nil {
				status := fmt.Sprintf("context=%d|total=%d", o.session.ContextTokens, o.session.TotalSpent.TotalTokens)
				if status != o.lastStatus {
					o.lastStatus = status
					// Send as binary TLV message (tag 'U' + length + value)
					msg := []byte{'U', byte(len(status) >> 24), byte(len(status) >> 16), byte(len(status) >> 8), byte(len(status))}
					msg = append(msg, status...)
					o.conn.WriteMessage(websocket.BinaryMessage, msg)
				}
			}
			o.mu.Unlock()
		case <-o.closeCh:
			return
		}
	}
}

//go:embed chat.html
var indexHTML []byte
