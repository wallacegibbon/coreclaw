package adaptors

import (
	"fmt"

	"charm.land/fantasy"
	agentpkg "github.com/wallacegibbon/coreclaw/internal/agent"
	"github.com/wallacegibbon/coreclaw/internal/stream"
)

// AgentFactory creates a new agent for each client session
type AgentFactory func() fantasy.Agent

// Adaptor is the interface for terminal adaptors
type Adaptor interface {
	Start()
}

// NewSession creates a processor and session with common setup
func NewSession(
	agent fantasy.Agent,
	baseURL, modelName string,
	input stream.Input,
	output stream.Output,
) *agentpkg.Session {
	processor := agentpkg.NewProcessorWithIO(agent, input, output)
	session := agentpkg.NewSession(agent, baseURL, modelName, processor)
	return session
}

// Dim returns text in dim gray color
func Dim(text string) string {
	return fmt.Sprintf("\x1b[2;38;2;108;112;134m%s\x1b[0m", text)
}

// Bright returns text in bright white color
func Bright(text string) string {
	return fmt.Sprintf("\x1b[1;38;2;205;214;244m%s\x1b[0m", text)
}

// Green returns text in green color
func Green(text string) string {
	return fmt.Sprintf("\x1b[38;2;166;227;161m%s\x1b[0m", text)
}

// Yellow returns text in yellow color
func Yellow(text string) string {
	return fmt.Sprintf("\x1b[38;2;249;226;175m%s\x1b[0m", text)
}
