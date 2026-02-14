package debug

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httputil"
	"os"
	"strings"
	"time"
)

// DebugTransport wraps an http.RoundTripper and logs requests and responses
type DebugTransport struct {
	Transport http.RoundTripper
}

// debugReader wraps an io.Reader to log each chunk of data as it's read
type debugReader struct {
	reader   io.Reader
	buf       []byte
	firstRead bool
}

func newDebugReader(r io.Reader) *debugReader {
	return &debugReader{
		reader:   r,
		buf:       make([]byte, 0, 4096),
		firstRead: true,
	}
}

func (dr *debugReader) Read(p []byte) (n int, err error) {
	n, err = dr.reader.Read(p)

	if n > 0 {
		chunk := p[:n]
		chunkStr := string(chunk)

		// Handle Server-Sent Events (SSE) format: "data: {...}\n"
		lines := strings.Split(chunkStr, "\n")
		for _, line := range lines {
			line = strings.TrimSpace(line)
			if line == "" {
				continue
			}

			// Skip "data: " prefix and try to parse as JSON
			jsonStr := line
			if strings.HasPrefix(line, "data:") {
				jsonStr = strings.TrimSpace(strings.TrimPrefix(line, "data:"))
			}

			// Try to parse as JSON and log it
			var jsonData map[string]any
			if json.Unmarshal([]byte(jsonStr), &jsonData) == nil {
				// Check if finish_reason is null (streaming in progress)
				choices, ok := jsonData["choices"].([]any)
				if ok && len(choices) > 0 {
					choice, ok := choices[0].(map[string]any)
					if ok {
						finishReason, hasFinishReason := choice["finish_reason"]
						if !hasFinishReason || finishReason == nil {
							// Streaming in progress - show condensed format
							if delta, ok := choice["delta"].(map[string]any); ok {
								if content, ok := delta["content"].(string); ok {
									fmt.Fprintf(os.Stderr, "\x1b[38;2;249;226;175m{ \"choices[0].delta.content\": %q }\x1b[0m\n", content)
								} else if toolCalls, ok := delta["tool_calls"].([]any); ok && len(toolCalls) > 0 {
									fmt.Fprintf(os.Stderr, "\x1b[38;2;249;226;175m{ \"choices[0].delta.tool_calls\": [...%d items] }\x1b[0m\n", len(toolCalls))
								}
							}
							continue
						}
					}
				}

				// Full format for final chunks or other cases
				formatted, _ := json.MarshalIndent(jsonData, "", "  ")
				if dr.firstRead {
					fmt.Fprintf(os.Stderr, "\x1b[38;2;203;166;247m<<< Response Stream\x1b[0m\n")
					fmt.Fprintf(os.Stderr, "\x1b[38;2;108;112;134mChunks:\x1b[0m\n")
					dr.firstRead = false
				}
				fmt.Fprintf(os.Stderr, "\x1b[38;2;249;226;175m%s\x1b[0m\n", formatted)
			} else if jsonStr != "[DONE]" {
				// Not JSON and not [DONE], print raw line
				if dr.firstRead {
					fmt.Fprintf(os.Stderr, "\x1b[38;2;203;166;247m<<< Response Stream\x1b[0m\n")
					fmt.Fprintf(os.Stderr, "\x1b[38;2;108;112;134mChunks:\x1b[0m\n")
					dr.firstRead = false
				}
				fmt.Fprintf(os.Stderr, "\x1b[38;2;249;226;175m%s\x1b[0m\n", line)
			}
		}
	}

	return n, err
}

// RoundTrip implements the http.RoundTripper interface
func (t *DebugTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	var requestBody []byte
	var isStreaming bool

	// Log request and check if streaming
	if req.Body != nil {
		requestBody, _ = io.ReadAll(req.Body)
		req.Body = io.NopCloser(bytes.NewReader(requestBody))

		var formattedBody any
		if err := json.Unmarshal(requestBody, &formattedBody); err == nil {
			// Check if streaming is enabled
			if reqBody, ok := formattedBody.(map[string]any); ok {
				if stream, ok := reqBody["stream"].(bool); ok && stream {
					isStreaming = true
				}
			}

			formattedBody, _ = json.MarshalIndent(formattedBody, "", "  ")
			fmt.Fprintf(os.Stderr, "\x1b[38;2;166;227;161m>>> Request\x1b[0m\n")
			fmt.Fprintf(os.Stderr, "\x1b[38;2;137;180;250m%s %s %s\x1b[0m\n", req.Method, req.URL.Path, req.URL.RawQuery)
			fmt.Fprintf(os.Stderr, "\x1b[38;2;108;112;134mHeaders:\x1b[0m\n")
			for k, v := range req.Header {
				if k == "Authorization" {
					fmt.Fprintf(os.Stderr, "  %s: ***\n", k)
				} else {
					fmt.Fprintf(os.Stderr, "  %s: %v\n", k, v)
				}
			}
			fmt.Fprintf(os.Stderr, "\x1b[38;2;108;112;134mBody:\x1b[0m\n")
			fmt.Fprintf(os.Stderr, "\x1b[38;2;137;180;250m%s\x1b[0m\n", formattedBody)
		} else {
			fmt.Fprintf(os.Stderr, "\x1b[38;2;166;227;161m>>> Request\x1b[0m\n")
			fmt.Fprintf(os.Stderr, "\x1b[38;2;137;180;250m%s %s\x1b[0m\n", req.Method, req.URL)
			fmt.Fprintf(os.Stderr, "\x1b[38;2;108;112;134mBody:\x1b[0m\n")
			fmt.Fprintf(os.Stderr, "\x1b[38;2;137;180;250m%s\x1b[0m\n", string(requestBody))
		}
		fmt.Fprintf(os.Stderr, "\x1b[38;2;108;112;134m--------------------------------------------------\x1b[0m\n")
	}

	start := time.Now()

	// Perform the request
	resp, err := t.Transport.RoundTrip(req)
	if err != nil {
		fmt.Fprintf(os.Stderr, "\x1b[38;2;203;166;247m<<< Request failed after %v: %v\x1b[0m\n", time.Since(start), err)
		return nil, err
	}

	// Log response
	fmt.Fprintf(os.Stderr, "\x1b[38;2;203;166;247m<<< Response\x1b[0m\n")
	fmt.Fprintf(os.Stderr, "\x1b[38;2;137;180;250m%s %s\x1b[0m\n", resp.Proto, resp.Status)
	fmt.Fprintf(os.Stderr, "\x1b[38;2;108;112;134mHeaders:\x1b[0m\n")
	for k, v := range resp.Header {
		fmt.Fprintf(os.Stderr, "  %s: %v\n", k, v)
	}

	// Check if it's a streaming response by looking at Content-Type
	if req.Body != nil {
		// Read the original request body to check for "stream": true
		bodyBytes, _ := io.ReadAll(req.Body)
		req.Body = io.NopCloser(bytes.NewReader(bodyBytes))
		var reqBody map[string]any
		if json.Unmarshal(bodyBytes, &reqBody) == nil {
			if stream, ok := reqBody["stream"].(bool); ok && stream {
				isStreaming = true
			}
		}
	}

	// Check response content type to confirm streaming
	contentType := resp.Header.Get("Content-Type")
	if isStreaming && strings.Contains(contentType, "text/event-stream") {
		fmt.Fprintf(os.Stderr, "\x1b[38;2;108;112;134mBody:\x1b[0m\n")
		resp.Body = struct {
			io.Reader
			io.Closer
		}{
			Reader: newDebugReader(resp.Body),
			Closer: resp.Body,
		}
	} else {
		responseBody, _ := io.ReadAll(resp.Body)
		resp.Body = io.NopCloser(bytes.NewReader(responseBody))

		var formattedBody any
		if err := json.Unmarshal(responseBody, &formattedBody); err == nil {
			formattedBody, _ = json.MarshalIndent(formattedBody, "", "  ")
			fmt.Fprintf(os.Stderr, "\x1b[38;2;108;112;134mBody:\x1b[0m\n")
			fmt.Fprintf(os.Stderr, "\x1b[38;2;249;226;175m%s\x1b[0m\n", formattedBody)
		} else {
			dump, _ := httputil.DumpResponse(resp, false)
			fmt.Fprintf(os.Stderr, "\x1b[38;2;108;112;134mBody:\x1b[0m\n")
			fmt.Fprintf(os.Stderr, "\x1b[38;2;249;226;175m%s\x1b[0m\n", dump)
		}
		fmt.Fprintf(os.Stderr, "\x1b[38;2;108;112;134m--------------------------------------------------\x1b[0m\n")
		fmt.Fprintf(os.Stderr, "\x1b[38;2;108;112;134mTime: %v\x1b[0m\n", time.Since(start))
	}

	return resp, nil
}

// NewHTTPClient creates a new HTTP client with debug logging enabled
func NewHTTPClient() *http.Client {
	return &http.Client{
		Transport: &DebugTransport{
			Transport: http.DefaultTransport,
		},
	}
}
