package adaptors

import (
	"context"
	"net/http"
	"sync"

	"charm.land/fantasy"
	"github.com/gorilla/websocket"
	agentpkg "github.com/wallacegibbon/coreclaw/internal/agent"
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
	Server      *http.Server
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

		// Send welcome message
		conn.WriteMessage(websocket.TextMessage, []byte("Connected to CoreClaw\n"))

		// Handle client disconnect
		defer func() {
			cancel()
			conn.Close()
		}()

		// Read loop - forward client input to processor
		go func() {
			for {
				_, message, err := conn.ReadMessage()
				if err != nil {
					cancel()
					return
				}
				select {
				case input.clientCh <- message:
				case <-ctx.Done():
					return
				}
			}
		}()

		// Interactive loop for this client
		for {
			select {
			case <-ctx.Done():
				return
			default:
			}

			// Read prompt from client
			line, err := input.readLine()
			if err != nil {
				return
			}

			if len(line) == 0 {
				continue
			}

			// Process prompt using shared session
			_, _, err = session.ProcessPrompt(ctx, line)

			if err != nil {
				if ctx.Err() == context.Canceled {
					return
				}
				continue
			}
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

// indexHTML is the embedded chat client
var indexHTML = []byte(`<!DOCTYPE html>
<html lang="en">
<head>
    <meta charset="UTF-8">
    <meta name="viewport" content="width=device-width, initial-scale=1.0">
    <title>CoreClaw Chat</title>
    <script src="https://cdn.jsdelivr.net/npm/marked/marked.min.js"></script>
    <style>
        * { box-sizing: border-box; }
        body {
            font-family: -apple-system, BlinkMacSystemFont, 'Segoe UI', Roboto, sans-serif;
            max-width: 800px;
            margin: 0 auto;
            padding: 20px;
            background: #1a1a2e;
            color: #eee;
        }
        h1 { text-align: center; color: #00d4ff; }
        #status {
            text-align: center;
            padding: 10px;
            margin-bottom: 20px;
            border-radius: 5px;
        }
        .connected { background: #28a745; }
        .disconnected { background: #dc3545; }
        .connecting { background: #ffc107; color: #000; }
        #messages {
            height: 60vh;
            overflow-y: auto;
            border: 1px solid #333;
            border-radius: 5px;
            padding: 10px;
            margin-bottom: 20px;
            background: #16213e;
        }
        .message { margin-bottom: 10px; padding: 8px 12px; border-radius: 5px; }
        .user { background: #0f3460; margin-left: 20%; }
        .assistant { background: #1f4068; margin-right: 20%; }
        .tool { background: #2d2d44; color: #ffd700; font-size: 0.9em; margin-right: 10%; }
        .error { background: #721c24; color: #f8d7da; }
        .reasoning { background: #2d2d44; color: #888; font-style: italic; margin-right: 10%; }
        .system { background: #333; color: #aaa; font-size: 0.9em; text-align: center; }
        .message.assistant p { margin: 0 0 8px 0; }
        .message.assistant p:last-child { margin-bottom: 0; }
        .message.assistant code { background: #0a0a15; padding: 2px 6px; border-radius: 3px; font-size: 0.9em; }
        .message.assistant pre { background: #0a0a15; padding: 10px; border-radius: 5px; overflow-x: auto; }
        .message.assistant pre code { background: none; padding: 0; }
        .message.assistant ul, .message.assistant ol { margin: 0 0 8px 0; padding-left: 20px; }
        .message.assistant a { color: #00d4ff; }
        #input-area {
            display: flex;
            gap: 10px;
        }
        #prompt {
            flex: 1;
            padding: 12px;
            border: 1px solid #333;
            border-radius: 5px;
            background: #16213e;
            color: #eee;
            font-size: 16px;
        }
        #prompt:focus { outline: none; border-color: #00d4ff; }
        #send {
            padding: 12px 24px;
            background: #00d4ff;
            border: none;
            border-radius: 5px;
            color: #000;
            font-weight: bold;
            cursor: pointer;
        }
        #send:hover { background: #00b8e6; }
        #send:disabled { background: #555; cursor: not-allowed; }
        pre { white-space: pre-wrap; word-wrap: break-word; }
    </style>
</head>
<body>
    <h1>🤖 CoreClaw Chat</h1>
    <div id="status" class="disconnected">Disconnected</div>
    <div id="messages"></div>
    <div id="input-area">
        <input type="text" id="prompt" placeholder="Type your message..." autocomplete="off">
        <button id="send">Send</button>
    </div>

    <script>
        const messages = document.getElementById('messages');
        const prompt = document.getElementById('prompt');
        const send = document.getElementById('send');
        const status = document.getElementById('status');

        const protocol = location.protocol === 'https:' ? 'wss:' : 'ws:';
        const wsUrl = protocol + '//' + location.host + '/ws';
        let ws = null;

        // Buffer for accumulating incoming binary data
        let buffer = [];
        // Separate accumulators for text and reasoning
        let currentTextValue = '';
        let currentTextElement = null;
        let currentReasoningValue = '';
        let currentReasoningElement = null;

        function connect() {
            status.textContent = 'Connecting...';
            status.className = 'connecting';

            ws = new WebSocket(wsUrl);

            ws.onopen = () => {
                status.textContent = 'Connected';
                status.className = 'connected';
                send.disabled = false;
            };

            ws.onclose = () => {
                status.textContent = 'Disconnected';
                status.className = 'disconnected';
                send.disabled = true;
                setTimeout(connect, 3000);
            };

            ws.onerror = () => {
                status.textContent = 'Error';
                status.className = 'disconnected';
            };

            ws.onmessage = (event) => {
                if (event.data instanceof Blob) {
                    const reader = new FileReader();
                    reader.onload = () => {
                        const bytes = new Uint8Array(reader.result);
                        // Append new bytes to buffer
                        buffer.push(...bytes);
                        processBuffer();
                    };
                    reader.readAsArrayBuffer(event.data);
                } else {
                    addMessage('system', event.data);
                }
            };
        }

        // Process buffer and extract complete TLV messages
        function processBuffer() {
            while (buffer.length >= 5) {
                const tag = String.fromCharCode(buffer[0]);
                const length = new DataView(new Uint8Array(buffer.slice(1, 5)).buffer).getUint32(0, false);

                if (buffer.length < 5 + length) {
                    break; // Wait for more data
                }

                const value = new TextDecoder().decode(new Uint8Array(buffer.slice(5, 5 + length)));
                buffer = buffer.slice(5 + length);

                handleTLV(tag, value);
            }
        }

        // Handle a complete TLV message
        function handleTLV(tag, value) {
            if (tag === 'T') {
                // Text: accumulate and stream to current message
                currentTextValue += value;
                if (currentTextElement) {
                    updateMessageContent(currentTextElement, 'assistant', currentTextValue);
                } else {
                    currentTextElement = addMessageElement('assistant', currentTextValue);
                }
            } else if (tag === 'R') {
                // Reasoning: accumulate in separate element
                currentReasoningValue += value;
                if (currentReasoningElement) {
                    updateMessageContent(currentReasoningElement, 'reasoning', currentReasoningValue);
                } else {
                    currentReasoningElement = addMessageElement('reasoning', currentReasoningValue);
                }
            } else if (tag === 't') {
                // Tool call: flush current text/reasoning, show tool
                flushCurrentText();
                addMessage('tool', value);
            } else if (tag === 'E') {
                // Error: flush current text/reasoning, show error
                flushCurrentText();
                addMessage('error', value);
            }
        }

        // Flush accumulated text/reasoning to finalize the message
        function flushCurrentText() {
            if (currentTextElement && currentTextValue) {
                updateMessageContent(currentTextElement, 'assistant', currentTextValue);
            }
            if (currentReasoningElement && currentReasoningValue) {
                updateMessageContent(currentReasoningElement, 'reasoning', currentReasoningValue);
            }
            currentTextValue = '';
            currentTextElement = null;
            currentReasoningValue = '';
            currentReasoningElement = null;
        }

        // Create message element without content (for streaming)
        function addMessageElement(type, text) {
            const div = document.createElement('div');
            div.className = 'message ' + type;
            if (type === 'tool') {
                div.innerHTML = '<pre>' + escapeHtml(text) + '</pre>';
            } else if (type === 'assistant' || type === 'reasoning') {
                div.innerHTML = marked.parse(text);
            } else {
                div.textContent = text;
            }
            messages.appendChild(div);
            messages.scrollTop = messages.scrollHeight;
            return div;
        }

        // Update message content (for streaming)
        function updateMessageContent(element, type, text) {
            if (type === 'tool') {
                element.innerHTML = '<pre>' + escapeHtml(text) + '</pre>';
            } else if (type === 'assistant' || type === 'reasoning') {
                element.innerHTML = marked.parse(text);
            } else {
                element.textContent = text;
            }
            messages.scrollTop = messages.scrollHeight;
        }

        // Add complete message
        function addMessage(type, text) {
            flushCurrentText();
            addMessageElement(type, text);
        }

        function escapeHtml(text) {
            const div = document.createElement('div');
            div.textContent = text;
            return div.innerHTML;
        }

        function sendMessage() {
            const text = prompt.value.trim();
            if (!text || !ws || ws.readyState !== WebSocket.OPEN) return;

            ws.send(text + '\n');
            prompt.value = '';
        }

        send.addEventListener('click', sendMessage);
        prompt.addEventListener('keypress', (e) => {
            if (e.key === 'Enter') sendMessage();
        });

        connect();
    </script>
</body>
</html>`)
