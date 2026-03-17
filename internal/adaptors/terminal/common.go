package terminal

import (
	"github.com/alayacore/alayacore/internal/llm"
)

// AgentFactory creates a new agent for each client session
type AgentFactory func() *llm.Agent

// Adaptor is the interface for terminal adaptors
type Adaptor interface {
	Start()
}
