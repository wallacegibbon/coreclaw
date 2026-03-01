package adaptors

import (
	"fmt"
	"net/http"
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
		input := stream.NewChanInput(100)
		output := &clientOutput{
			conn:    conn,
			closeCh: make(chan struct{}),
		}
		defer close(output.closeCh)

		// Create a new agent, processor, and session for this client
		agent := factory()
		processor := agentpkg.NewProcessorWithIO(agent, input, output)
		session := agentpkg.NewSession(agent, baseURL, modelName, processor)

		// Set session on output and start status updater
		output.session = session
		go output.startStatusUpdater()

		// Read loop - handles client input and blocks until connection closes
		for {
			_, message, err := conn.ReadMessage()
			if err != nil {
				return
			}

			if len(message) == 0 {
				continue
			}
			// Client already encoded as TLV, pass through raw data
			input.EmitRawData(message)
		}
	}
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
			var status string
			var shouldSend bool
			if o.session != nil {
				status = fmt.Sprintf("context=%d|total=%d", o.session.ContextTokens, o.session.TotalSpent.TotalTokens)
				shouldSend = (status != o.lastStatus)
				if shouldSend {
					o.lastStatus = status
				}
			}
			o.mu.Unlock()

			if shouldSend {
				// Encode and send TLV message (tag 'U')
				// Must encode manually and write to conn directly to avoid deadlock
				// (lock is already held by startStatusUpdater loop)
				msg := stream.EncodeTLV('U', status)
				o.mu.Lock()
				o.conn.WriteMessage(websocket.BinaryMessage, msg)
				o.mu.Unlock()
			}
		case <-o.closeCh:
			return
		}
	}
}

//go:embed chat.html
var indexHTML []byte
