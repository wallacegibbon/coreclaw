package debug

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httputil"
	"os"
	"time"
)

// DebugTransport wraps an http.RoundTripper and logs requests and responses
type DebugTransport struct {
	Transport http.RoundTripper
}

// RoundTrip implements the http.RoundTripper interface
func (t *DebugTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	// Log request
	if req.Body != nil {
		body, _ := io.ReadAll(req.Body)
		req.Body = io.NopCloser(bytes.NewReader(body))

		var formattedBody any
		if err := json.Unmarshal(body, &formattedBody); err == nil {
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
			fmt.Fprintf(os.Stderr, "\x1b[38;2;108;112;134mBody:\x1b[0m\n%s\n", formattedBody)
		} else {
			fmt.Fprintf(os.Stderr, "\x1b[38;2;166;227;161m>>> Request\x1b[0m\n")
			fmt.Fprintf(os.Stderr, "\x1b[38;2;137;180;250m%s %s\x1b[0m\n", req.Method, req.URL)
			fmt.Fprintf(os.Stderr, "\x1b[38;2;108;112;134mBody:\x1b[0m\n%s\n", string(body))
		}
	}

	start := time.Now()

	// Perform the request
	resp, err := t.Transport.RoundTrip(req)
	if err != nil {
		fmt.Fprintf(os.Stderr, "\x1b[38;2;203;166;247m<<< Request failed after %v: %v\x1b[0m\n", time.Since(start), err)
		return nil, err
	}

	// Log response
	responseBody, _ := io.ReadAll(resp.Body)
	resp.Body = io.NopCloser(bytes.NewReader(responseBody))

	var formattedBody any
	if err := json.Unmarshal(responseBody, &formattedBody); err == nil {
		formattedBody, _ = json.MarshalIndent(formattedBody, "", "  ")
		fmt.Fprintf(os.Stderr, "\x1b[38;2;203;166;247m<<< Response\x1b[0m\n")
		fmt.Fprintf(os.Stderr, "\x1b[38;2;137;180;250m%s %s\x1b[0m\n", resp.Proto, resp.Status)
		fmt.Fprintf(os.Stderr, "\x1b[38;2;108;112;134mHeaders:\x1b[0m\n")
		for k, v := range resp.Header {
			fmt.Fprintf(os.Stderr, "  %s: %v\n", k, v)
		}
		fmt.Fprintf(os.Stderr, "\x1b[38;2;108;112;134mBody:\x1b[0m\n%s\n", formattedBody)
	} else {
		dump, _ := httputil.DumpResponse(resp, false)
		fmt.Fprintf(os.Stderr, "\x1b[38;2;203;166;247m<<< Response\x1b[0m\n")
		fmt.Fprintf(os.Stderr, "\x1b[38;2;137;180;250m%s %s\x1b[0m\n", resp.Proto, resp.Status)
		fmt.Fprintf(os.Stderr, "\x1b[38;2;108;112;134mBody:\x1b[0m\n%s\n", dump)
	}

	fmt.Fprintf(os.Stderr, "\x1b[38;2;108;112;134mTime: %v\x1b[0m\n", time.Since(start))

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
