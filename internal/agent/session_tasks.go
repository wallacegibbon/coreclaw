package agent

import (
	"context"

	"charm.land/fantasy"
)

// ============================================================================
// Task Queue
// ============================================================================

func (s *Session) submitTask(task Task) {
	s.mu.Lock()
	queueLen := len(s.taskQueue)
	// Check queue capacity (max 10 tasks)
	if queueLen >= 10 {
		s.mu.Unlock()
		s.writeNotify("Busy. Cannot queue, try again shortly.")
		return
	}
	s.taskQueue = append(s.taskQueue, task)
	inProgress := s.inProgress
	s.signalTaskAvailable()
	s.mu.Unlock()

	if inProgress {
		s.writeNotify("Queued. Previous task in progress. Will run after completion.")
		s.sendSystemInfo()
	}
}

// signalTaskAvailable notifies the task runner that a new task is available
func (s *Session) signalTaskAvailable() {
	select {
	case s.taskAvailable <- struct{}{}:
	default:
	}
}

func (s *Session) taskRunner() {
	for {
		task, ok := s.waitForNextTask()
		if !ok {
			return
		}
		s.setInProgress(true)
		s.runTask(task)
		s.setInProgress(s.hasQueuedTasks())
	}
}

func (s *Session) waitForNextTask() (Task, bool) {
	for {
		s.mu.Lock()
		if len(s.taskQueue) > 0 {
			task := s.taskQueue[0]
			s.taskQueue = s.taskQueue[1:]
			s.mu.Unlock()
			return task, true
		}
		s.mu.Unlock()
		select {
		case <-s.taskAvailable:
			// continue
		case <-s.done:
			return nil, false
		}
	}
}

func (s *Session) hasQueuedTasks() bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	return len(s.taskQueue) > 0
}

func (s *Session) setInProgress(v bool) {
	s.mu.Lock()
	changed := s.inProgress != v
	s.inProgress = v
	s.mu.Unlock()
	if changed {
		s.sendSystemInfo()
	}
}

func (s *Session) runTask(task Task) {
	s.sendSystemInfo()
	ctx, cancel := context.WithCancel(context.Background())
	s.mu.Lock()
	s.cancelCurrent = cancel
	s.mu.Unlock()
	defer func() {
		s.mu.Lock()
		s.cancelCurrent = nil
		s.mu.Unlock()
	}()

	switch t := task.(type) {
	case UserPrompt:
		s.signalPromptStart(string(t))
		s.handleUserPrompt(ctx, string(t))
	case CommandPrompt:
		s.signalCommandStart(t.Command)
		s.handleCommandSync(ctx, t.Command)
	}

	if ctx.Err() == context.Canceled {
		s.appendCancelMessage()
	}
}

func (s *Session) appendCancelMessage() {
	if len(s.Messages) == 0 {
		return
	}
	if s.Messages[len(s.Messages)-1].Role == fantasy.MessageRoleUser {
		s.Messages = append(s.Messages, fantasy.Message{
			Role:    fantasy.MessageRoleAssistant,
			Content: []fantasy.MessagePart{fantasy.TextPart{Text: "The user canceled."}},
		})
	}
}
