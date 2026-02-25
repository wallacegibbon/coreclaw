package adaptors

import (
	"context"
	"net/http"
	"strings"
	"sync"

	"charm.land/fantasy"
	"github.com/gorilla/websocket"
	agentpkg "github.com/wallacegibbon/coreclaw/internal/agent"
	"github.com/wallacegibbon/coreclaw/internal/stream"

	_ "embed"
)

var upgrader = websocket.Upgrader{
	CheckOrigin: func(r *http.Request) bool {
		return true
	},
}

// AgentFactory creates a new agent for each client session
type AgentFactory func() fantasy.Agent

// WebSocketAdaptor connects WebSocket to the agent processor
type WebSocketAdaptor struct {
	AgentFactory AgentFactory
	Server       *http.Server
}

// NewWebSocketAdaptor creates a new WebSocket adaptor that listens on the given port
// Each client gets its own agent session
func NewWebSocketAdaptor(port string, factory AgentFactory) *WebSocketAdaptor {
	return NewWebSocketAdaptorWithStatic(port, factory, nil)
}

// NewWebSocketAdaptorWithStatic creates a WebSocket adaptor with optional static file server
func NewWebSocketAdaptorWithStatic(port string, factory AgentFactory, staticFS http.FileSystem) *WebSocketAdaptor {
	mux := http.NewServeMux()

	// Handle WebSocket
	mux.HandleFunc("/ws", handleWebSocket(factory))

	// Handle static files or embedded index.html
	if staticFS != nil {
		mux.Handle("/", http.FileServer(staticFS))
	} else {
		mux.HandleFunc("/", serveIndex)
	}

	server := &http.Server{
		Addr:    port,
		Handler: mux,
	}

	return &WebSocketAdaptor{
		AgentFactory: factory,
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
func handleWebSocket(factory AgentFactory) func(http.ResponseWriter, *http.Request) {
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
			conn: conn,
		}

		// Create a new agent, processor, and session for this client
		agent := factory()
		processor := agentpkg.NewProcessorWithIO(agent, input, output)
		session := agentpkg.NewSession(processor)

		// Create cancellable context for this client
		ctx, cancel := context.WithCancel(context.Background())

		// Create runner for tracking request state
		runner := agentpkg.NewSyncRunner(session)
		runner.OnDone = func() {
			stream.WriteTLV(output, 'D', "")
		}

		// Set up command callback to send usage info
		session.OnCommandDone = func() {
			session.SendUsage()
		}

		// Send welcome message
		conn.WriteMessage(websocket.TextMessage, []byte("Connected to CoreClaw\n"))

		// Handle client disconnect and cancel signals
		defer func() {
			cancel()
			conn.Close()
		}()

		// Read loop - handles client input and cancel signals
		go func() {
			for {
				_, message, err := conn.ReadMessage()
				if err != nil {
					cancel()
					return
				}

				// Check for CANCEL signal
				if len(message) >= 6 && string(message[:6]) == "CANCEL" {
					cancel()
					continue
				}

				select {
				case input.clientCh <- message:
				case <-ctx.Done():
					return
				}
			}
		}()

		// Interactive loop - synchronous like terminal
		for {
			// Read prompt from client
			line, err := input.readLine()
			if err != nil {
				return
			}

			if len(line) == 0 {
				continue
			}

			// If context was cancelled, create new one for next request
			if ctx.Err() == context.Canceled {
				ctx, cancel = context.WithCancel(context.Background())
			}

			// Handle commands like /summarize
			if strings.HasPrefix(line, "/") {
				command := strings.TrimPrefix(line, "/")
				_, err := session.HandleCommand(ctx, command)
				if err != nil && ctx.Err() == context.Canceled {
					ctx, cancel = context.WithCancel(context.Background())
				}
				runner.OnDone()
				continue
			}

			runner.SetInProgress(true)
			session.ProcessPrompt(ctx, line)
			runner.SetInProgress(false)

			// Send usage info after prompt
			session.SendUsage()

			runner.OnDone()
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

// Read implements stream.Input (used by processor but we use readLine instead)
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
	conn *websocket.Conn
	mu   sync.Mutex
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

//go:embed chat.html
var indexHTML []byte
