package agent

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/user"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"time"

	"charm.land/fantasy"
	"github.com/wallacegibbon/coreclaw/internal/stream"
)

// Task represents a unit of work in the task queue
type Task interface{ isTask() }

type UserPrompt string

func (UserPrompt) isTask() {}

type CommandPrompt struct {
	Command string
}

func (CommandPrompt) isTask() {}

// SystemInfo contains session system information
type SystemInfo struct {
	ContextTokens int64 `json:"context"`
	TotalTokens   int64 `json:"total"`
	QueueCount    int   `json:"queue"`
}

// Session manages message history and processes prompts
type Session struct {
	Processor *Processor
	Messages  []fantasy.Message

	// Agent is the fantasy agent instance
	Agent fantasy.Agent

	// BaseURL and ModelName store provider configuration
	BaseURL   string
	ModelName string

	// SessionFile is the file path for saving/loading the session
	SessionFile string

	// TotalSpent tracks total tokens used across all requests
	TotalSpent fantasy.Usage
	// ContextTokens tracks context tokens used (grows with each request, shrinks after summarize)
	ContextTokens int64

	// taskQueue buffers tasks submitted while agent is processing
	taskQueue chan Task

	// inProgress tracks whether a prompt is currently being processed
	inProgress bool

	// cancelCurrent is a function to cancel the current prompt
	cancelCurrent func()

	// mu protects concurrent access to session state
	mu sync.Mutex
}

// cancelTask handles the /cancel command
// Returns error if nothing to cancel
func (s *Session) cancelTask() {
	if s.inProgress {
		if s.cancelCurrent != nil {
			s.cancelCurrent()
			return
		}
	}
	s.writeError("nothing to cancel")
}

// expandPath expands ~ to the user's home directory
func expandPath(path string) string {
	if !strings.HasPrefix(path, "~") {
		return path
	}

	usr, err := user.Current()
	if err != nil {
		return path
	}

	if path == "~" {
		return usr.HomeDir
	}

	return filepath.Join(usr.HomeDir, path[1:])
}

// saveSession handles the /save command
// If args is empty, saves to s.SessionFile
// If args has one element, saves to that file path
func (s *Session) saveSession(args []string) {
	var filepath string
	if len(args) == 0 {
		if s.SessionFile == "" {
			s.writeError("no session file set and no filename provided")
			return
		}
		filepath = s.SessionFile
	} else if len(args) == 1 {
		filepath = expandPath(args[0])
	} else {
		s.writeError("usage: /save [filename]")
		return
	}

	if err := s.SaveSession(filepath); err != nil {
		s.writeError(fmt.Sprintf("failed to save session: %v", err))
	} else {
		s.writeGapped(stream.TagNotify, fmt.Sprintf("Session saved to %s", filepath))
	}
}

// IsInProgress returns true if a prompt is currently being processed
func (s *Session) IsInProgress() bool {
	return s.inProgress
}

// NewSession creates a new session with the given processor
func NewSession(agent fantasy.Agent, baseURL, modelName string, processor *Processor, sessionFile string) *Session {
	session := &Session{
		Processor:   processor,
		Messages:    nil,
		Agent:       agent,
		BaseURL:     baseURL,
		ModelName:   modelName,
		SessionFile: sessionFile,
		taskQueue:   make(chan Task, 10),
	}
	// Start input reader goroutine that reads TLV from input stream
	go session.readFromInput()
	return session
}

// summarize summarizes the conversation history
func (s *Session) summarize(ctx context.Context) {
	summarizePrompt := "Please summarize the conversation above in a concise manner. Return ONLY the summary, no introductions or explanations."

	assistantMsg, usage, err := s.Processor.ProcessPrompt(ctx, summarizePrompt, s.Messages)
	if err != nil {
		s.writeError(err.Error())
		return
	}
	// Replace messages with summary
	s.Messages = []fantasy.Message{assistantMsg}
	s.TotalSpent.InputTokens += usage.InputTokens
	s.TotalSpent.OutputTokens += usage.OutputTokens
	s.TotalSpent.TotalTokens += usage.TotalTokens
	s.TotalSpent.ReasoningTokens += usage.ReasoningTokens
	// After summarize, context shrinks to the summary
	s.ContextTokens = usage.OutputTokens
	// Send system info with updated token usage
	s.sendSystemInfo()
}

// processPrompt processes a user prompt and updates message history
// It handles adding user message, calling API, and storing assistant response
func (s *Session) processPrompt(ctx context.Context, prompt string) {
	// Add user message to history
	s.Messages = append(s.Messages, fantasy.NewUserMessage(prompt))

	// Create a copy of messages WITHOUT the pending user message for API
	// This prevents duplication (API adds user message internally)
	messagesForAPI := make([]fantasy.Message, len(s.Messages)-1)
	copy(messagesForAPI, s.Messages[:len(s.Messages)-1])

	// Process the prompt
	assistantMsg, usage, err := s.Processor.ProcessPrompt(ctx, prompt, messagesForAPI)
	if err != nil {
		s.writeError(err.Error())
		return
	}

	// Track usage
	s.TotalSpent.InputTokens += usage.InputTokens
	s.TotalSpent.OutputTokens += usage.OutputTokens
	s.TotalSpent.TotalTokens += usage.TotalTokens
	s.TotalSpent.ReasoningTokens += usage.ReasoningTokens

	// Context grows with each request
	s.ContextTokens += usage.TotalTokens

	// Send system info with updated token usage
	s.sendSystemInfo()

	// If there is an assistant message, store it.
	if assistantMsg.Role != "" {
		s.Messages = append(s.Messages, assistantMsg)
	}
}

// submitTask submits a task for async processing via the task queue
// Processing runs asynchronously so adaptors can continue receiving input
func (s *Session) submitTask(task Task) {
	if s.queueTask(task) {
		if s.inProgress {
			s.writeNotify("[Queued] Previous task in progress. Will run after completion.")
			s.sendSystemInfo()
		}
		if !s.inProgress {
			go s.runAsync()
		}
	} else {
		s.writeNotify("[Busy] Cannot queue, try again shortly.")
	}
}

// submitPrompt submits a prompt for processing, queueing if necessary
func (s *Session) submitPrompt(prompt string) {
	s.submitTask(UserPrompt(prompt))
}

// submitCommand submits a command for async processing via the task queue
func (s *Session) submitCommand(cmd string) {
	switch cmd {
	case "summarize":
		s.submitTask(CommandPrompt{Command: cmd})
	default:
		s.handleCommandSync(context.Background(), cmd)
	}
}

// runAsync processes tasks asynchronously, including any queued tasks
func (s *Session) runAsync() {
	s.inProgress = true
	defer func() {
		s.inProgress = false
	}()

	for {
		queuedTask, ok := s.getQueuedTask()
		if !ok {
			break
		}
		s.sendSystemInfo()
		// Create a fresh context for each queued task
		taskCtx, taskCancel := context.WithCancel(context.Background())
		s.cancelCurrent = taskCancel

		// Handle different task types
		switch task := queuedTask.(type) {
		case UserPrompt:
			s.signalPromptStart(string(task))
			s.processPrompt(taskCtx, string(task))
		case CommandPrompt:
			s.signalCommandStart(task.Command)
			s.handleCommandSync(taskCtx, task.Command)
		}

		// Check if cancelled
		if taskCtx.Err() == context.Canceled {
			// Add assistant message to close out the canceled prompt
			// This prevents the next prompt from being concatenated into the canceled one
			s.Messages = append(s.Messages, fantasy.Message{
				Role:    fantasy.MessageRoleAssistant,
				Content: []fantasy.MessagePart{fantasy.TextPart{Text: "The user canceled."}},
			})
			s.cancelCurrent = nil
			continue
		}
		s.cancelCurrent = nil
	}
}

// handleCommandSync runs the command synchronously within the async loop
func (s *Session) handleCommandSync(ctx context.Context, cmd string) {
	// Parse command parts
	parts := strings.Fields(cmd)
	if len(parts) == 0 {
		s.writeError("empty command")
		return
	}

	switch parts[0] {
	case "summarize":
		s.summarize(ctx)
	case "cancel":
		s.cancelTask()
	case "save":
		s.saveSession(parts[1:])
	default:
		s.writeError(fmt.Sprintf("unknown cmd <%s>", cmd))
	}
}

func (s *Session) writeGapped(tag byte, msg string) {
	if s.Processor != nil && s.Processor.Output != nil {
		stream.WriteTLV(s.Processor.Output, stream.TagStreamGap, "")
		stream.WriteTLV(s.Processor.Output, tag, msg)
		stream.WriteTLV(s.Processor.Output, stream.TagStreamGap, "")
		s.Processor.Output.Flush()
	}
}

func (s *Session) signalPromptStart(prompt string) {
	s.writeGapped(stream.TagPromptStart, prompt)
}

func (s *Session) signalCommandStart(cmd string) {
	s.writeGapped(stream.TagPromptStart, "/"+cmd)
}

func (s *Session) writeError(msg string) {
	s.writeGapped(stream.TagError, msg)
}

func (s *Session) writeNotify(msg string) {
	s.writeGapped(stream.TagNotify, msg)
}

func (s *Session) sendSystemInfo() {
	info := SystemInfo{
		ContextTokens: s.ContextTokens,
		TotalTokens:   s.TotalSpent.TotalTokens,
		QueueCount:    len(s.taskQueue),
	}
	data, err := json.Marshal(info)
	if err != nil {
		return
	}
	stream.WriteTLV(s.Processor.Output, stream.TagSystem, string(data))
	s.Processor.Output.Flush()
}

// queueTask adds a task to the queue (non-blocking)
func (s *Session) queueTask(task Task) bool {
	select {
	case s.taskQueue <- task:
		return true
	default:
		return false
	}
}

// getQueuedTask tries to get a queued task (non-blocking)
func (s *Session) getQueuedTask() (Task, bool) {
	select {
	case task, ok := <-s.taskQueue:
		return task, ok
	default:
		return nil, false
	}
}

// readFromInput reads TLV messages from the input stream and processes them
func (s *Session) readFromInput() {
	for {
		tag, value, err := stream.ReadTLV(s.Processor.Input)
		if err != nil {
			// Input stream closed or error, stop reading
			return
		}

		// Only accept TagUserText messages, emit error for other tags
		if tag == stream.TagUserText {
			// Check if it's a command (starts with "/")
			if len(value) > 0 && value[0] == '/' {
				command := value[1:]
				s.submitCommand(command)
			} else {
				// Regular prompt
				s.submitPrompt(value)
			}
		} else {
			s.writeError(fmt.Sprintf("Invalid input tag: %c (only %c is allowed)", tag, stream.TagUserText))
		}
	}
}

// SessionData represents the persistent data saved for a session
type SessionData struct {
	BaseURL       string            `json:"base_url"`
	ModelName     string            `json:"model_name"`
	Messages      []fantasy.Message `json:"messages"`
	TotalSpent    fantasy.Usage     `json:"total_spent"`
	ContextTokens int64             `json:"context_tokens"`
	CreatedAt     time.Time         `json:"created_at"`
	UpdatedAt     time.Time         `json:"updated_at"`
}

// formatToolCall formats a tool call for display, matching processor's formatting
func formatToolCall(toolName, input string) string {
	var value string
	switch toolName {
	case "posix_shell":
		cmd := extractPosixShellCommand(input)
		if cmd != "" {
			displayCmd := formatCommon(cmd)
			value = fmt.Sprintf("%s: %s", toolName, displayCmd)
		}
	case "activate_skill":
		name := extractSkillName(input)
		if name != "" {
			value = fmt.Sprintf("%s: %s", toolName, name)
		}
	case "read_file":
		path := extractReadFilePath(input)
		if path != "" {
			value = fmt.Sprintf("%s: %s", toolName, path)
		}
	case "write_file":
		path := extractWriteFilePath(input)
		if path != "" {
			value = fmt.Sprintf("%s: %s", toolName, path)
		}
	}
	return value
}

// GetSessionsDir returns the sessions directory path
func GetSessionsDir() (string, error) {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	sessionsDir := filepath.Join(homeDir, ".coreclaw", "sessions")
	if err := os.MkdirAll(sessionsDir, 0755); err != nil {
		return "", err
	}
	return sessionsDir, nil
}

// GenerateSessionFilename generates a session filename based on current time
func GenerateSessionFilename() string {
	now := time.Now()
	return now.Format("2006-01-02-150405-1.json")
}

// LoadLatestSession finds and loads the most recent session file
func LoadLatestSession() (*SessionData, string, error) {
	sessionsDir, err := GetSessionsDir()
	if err != nil {
		return nil, "", fmt.Errorf("failed to get sessions directory: %w", err)
	}

	entries, err := os.ReadDir(sessionsDir)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, "", nil // No sessions yet
		}
		return nil, "", fmt.Errorf("failed to read sessions directory: %w", err)
	}

	if len(entries) == 0 {
		return nil, "", nil // No sessions yet
	}

	var sessionFiles []os.FileInfo
	for _, entry := range entries {
		if !entry.IsDir() && filepath.Ext(entry.Name()) == ".json" {
			info, err := entry.Info()
			if err != nil {
				continue
			}
			sessionFiles = append(sessionFiles, info)
		}
	}

	if len(sessionFiles) == 0 {
		return nil, "", nil // No session files
	}

	// Sort by modification time, most recent first
	sort.Slice(sessionFiles, func(i, j int) bool {
		return sessionFiles[i].ModTime().After(sessionFiles[j].ModTime())
	})

	latestFile := sessionFiles[0]
	latestPath := filepath.Join(sessionsDir, latestFile.Name())

	data, err := os.ReadFile(latestPath)
	if err != nil {
		return nil, "", fmt.Errorf("failed to read session file: %w", err)
	}

	var sessionData SessionData
	if err := json.Unmarshal(data, &sessionData); err != nil {
		return nil, "", fmt.Errorf("failed to parse session data: %w", err)
	}

	return &sessionData, latestPath, nil
}

// LoadSession loads a session from a specific file path
func LoadSession(path string) (*SessionData, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("failed to read session file: %w", err)
	}

	var sessionData SessionData
	if err := json.Unmarshal(data, &sessionData); err != nil {
		return nil, fmt.Errorf("failed to parse session data: %w", err)
	}

	return &sessionData, nil
}

// SaveSession saves session data to a file
func (s *Session) SaveSession(path string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	sessionData := SessionData{
		BaseURL:       s.BaseURL,
		ModelName:     s.ModelName,
		Messages:      s.Messages,
		TotalSpent:    s.TotalSpent,
		ContextTokens: s.ContextTokens,
		UpdatedAt:     time.Now(),
	}

	data, err := json.MarshalIndent(sessionData, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal session data: %w", err)
	}

	if err := os.WriteFile(path, data, 0644); err != nil {
		return fmt.Errorf("failed to write session file: %w", err)
	}

	return nil
}

// RestoreFromSession restores a session from saved session data
func RestoreFromSession(agent fantasy.Agent, baseURL, modelName string, processor *Processor, sessionData *SessionData, sessionFile string) *Session {
	session := &Session{
		Processor:     processor,
		Messages:      sessionData.Messages,
		Agent:         agent,
		BaseURL:       baseURL,
		ModelName:     modelName,
		SessionFile:   sessionFile,
		TotalSpent:    sessionData.TotalSpent,
		ContextTokens: sessionData.ContextTokens,
		taskQueue:     make(chan Task, 10),
	}
	// Start input reader goroutine
	go session.readFromInput()
	return session
}

// DisplayMessages outputs all session messages to the output stream
// This is used to display conversation history when loading a session
func (s *Session) DisplayMessages() {
	if s.Processor == nil || s.Processor.Output == nil {
		return
	}

	for _, msg := range s.Messages {
		switch msg.Role {
		case fantasy.MessageRoleUser:
			// Find the text content from user message
			var userText string
			for _, part := range msg.Content {
				if textPart, ok := part.(fantasy.TextPart); ok {
					userText += textPart.Text
				}
			}
			if userText != "" {
				s.signalPromptStart(userText)
			}

		case fantasy.MessageRoleAssistant:
			// Output all parts from assistant message
			for _, part := range msg.Content {
				if textPart, ok := part.(fantasy.TextPart); ok {
					stream.WriteTLV(s.Processor.Output, stream.TagAssistantText, textPart.Text)
					stream.WriteTLV(s.Processor.Output, stream.TagStreamGap, "")
					s.Processor.Output.Flush()
				} else if reasoningPart, ok := part.(fantasy.ReasoningPart); ok {
					stream.WriteTLV(s.Processor.Output, stream.TagReasoning, reasoningPart.Text)
					stream.WriteTLV(s.Processor.Output, stream.TagStreamGap, "")
					s.Processor.Output.Flush()
				}
			}

		case fantasy.MessageRoleTool:
			// Output tool call information
			for _, part := range msg.Content {
				if toolCall, ok := part.(fantasy.ToolCallPart); ok {
					// Format tool call similar to how processor does it
					toolInfo := formatToolCall(toolCall.ToolName, toolCall.Input)
					if toolInfo != "" {
						stream.WriteTLV(s.Processor.Output, stream.TagTool, toolInfo)
						stream.WriteTLV(s.Processor.Output, stream.TagStreamGap, "")
						s.Processor.Output.Flush()
					}
				}
			}
		}
	}
}

// LoadOrNewSession loads an existing session or creates a new one
// If sessionFile is specified, it loads that specific session
// Otherwise, it always creates a new session
// Returns the session and the session file path
func LoadOrNewSession(agent fantasy.Agent, baseURL, modelName string, processor *Processor, sessionFile string) (*Session, string) {
	var sessionData *SessionData

	// Expand ~ to home directory if present
	sessionFile = expandPath(sessionFile)

	if sessionFile != "" {
		// Load specific session
		data, err := LoadSession(sessionFile)
		if err != nil {
			// Failed to load specific session, will create new
			sessionData = nil
		} else {
			sessionData = data
		}
	}

	// Create or restore session
	if sessionData != nil {
		return RestoreFromSession(agent, baseURL, modelName, processor, sessionData, sessionFile), sessionFile
	}

	// Create new session (without auto-saving)
	session := NewSession(agent, baseURL, modelName, processor, sessionFile)

	return session, sessionFile
}
