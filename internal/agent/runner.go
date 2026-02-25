package agent

import (
	"sync"
)

// SyncRunner tracks request state for cancel support
type SyncRunner struct {
	Session    *Session
	mu         sync.Mutex
	inProgress bool
}

// NewSyncRunner creates a new thread-safe Runner
func NewSyncRunner(session *Session) *SyncRunner {
	return &SyncRunner{
		Session: session,
	}
}

// SetInProgress safely sets the in-progress flag
func (r *SyncRunner) SetInProgress(inProgress bool) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.inProgress = inProgress
}

// IsInProgress safely checks if a request is in progress
func (r *SyncRunner) IsInProgress() bool {
	r.mu.Lock()
	defer r.mu.Unlock()
	return r.inProgress
}
