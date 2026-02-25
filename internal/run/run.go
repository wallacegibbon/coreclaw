package run

import (
	"bufio"
	"context"
	"fmt"
	"os"
	"os/signal"
	"strings"
	"sync"

	"charm.land/fantasy"
	"github.com/wallacegibbon/coreclaw/internal/agent"
	"github.com/wallacegibbon/coreclaw/internal/terminal"
)

// Runner handles running the agent
type Runner struct {
	Processor   *agent.Processor
	Messages    []fantasy.Message
	BaseURL     string
	ModelName   string
	TotalSpent  fantasy.Usage
	ContextSize int64
}

// New creates a new Runner
func New(processor *agent.Processor, baseURL, modelName string) *Runner {
	return &Runner{
		Processor: processor,
		Messages:  nil,
		BaseURL:   baseURL,
		ModelName: modelName,
	}
}

// RunSingle runs a single prompt and exits
func (r *Runner) RunSingle(ctx context.Context, prompt string) error {
	_, _, _, _, err := r.Processor.ProcessPrompt(ctx, prompt, r.Messages)
	return err
}

// RunInteractive starts the interactive REPL
func (r *Runner) RunInteractive(ctx context.Context) error {
	reader := bufio.NewReader(os.Stdin)

	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	requestInProgress := false
	var mu sync.Mutex

	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt)

	go func() {
		for range sigChan {
			mu.Lock()
			if requestInProgress {
				mu.Unlock()
				cancel()
				fmt.Println("\nRequest cancelled.")
			} else {
				mu.Unlock()
			}
		}
	}()

	defer signal.Stop(sigChan)

	for {
		var userPrompt string

		fmt.Fprint(os.Stderr, terminal.GetPrompt(r.BaseURL, r.ModelName))
		input, err := reader.ReadString('\n')
		if err != nil {
			return err
		}
		userPrompt = strings.TrimSpace(input)

		if userPrompt == "" {
			continue
		}

		// Handle /summarize command
		if strings.HasPrefix(userPrompt, "/") {
			command := strings.TrimPrefix(userPrompt, "/")
			switch command {
			case "summarize":
				mu.Lock()
				requestInProgress = true
				mu.Unlock()

				_, summaryMsg, usage, err := r.Processor.Summarize(ctx, r.Messages)

				mu.Lock()
				requestInProgress = false
				mu.Unlock()

				// Accumulate token usage
				r.TotalSpent.InputTokens += usage.InputTokens
				r.TotalSpent.OutputTokens += usage.OutputTokens
				r.TotalSpent.TotalTokens += usage.TotalTokens
				r.TotalSpent.ReasoningTokens += usage.ReasoningTokens

				// Replace messages with summary to reduce token count
				r.Messages = []fantasy.Message{summaryMsg}
				// Context becomes the summarize output
				r.ContextSize = usage.OutputTokens

				// Print context size and total spent
				fmt.Println()
				printUsage(r.ContextSize, r.TotalSpent)

				if err != nil {
					if ctx.Err() == context.Canceled {
						cancel()
						ctx, cancel = context.WithCancel(context.Background())
						defer cancel()
						continue
					}
				}
			default:
				fmt.Printf("Unknown command: %s\n", command)
			}
			continue
		}

		mu.Lock()
		requestInProgress = true
		mu.Unlock()

		// Add user message to history BEFORE processing (so it's preserved on Ctrl-C)
		r.Messages = append(r.Messages, fantasy.NewUserMessage(userPrompt))

		// Create a copy of messages to send to API (without the pending user message)
		// This prevents duplication when Ctrl-C is pressed
		messagesForAPI := make([]fantasy.Message, len(r.Messages)-1)
		copy(messagesForAPI, r.Messages[:len(r.Messages)-1])

		_, responseText, assistantMsg, usage, err := r.Processor.ProcessPrompt(ctx, userPrompt, messagesForAPI)

		mu.Lock()
		requestInProgress = false
		mu.Unlock()

		// Accumulate token usage
		r.TotalSpent.InputTokens += usage.InputTokens
		r.TotalSpent.OutputTokens += usage.OutputTokens
		r.TotalSpent.TotalTokens += usage.TotalTokens
		r.TotalSpent.ReasoningTokens += usage.ReasoningTokens

		if err != nil {
			if ctx.Err() == context.Canceled {
				cancel()
				ctx, cancel = context.WithCancel(context.Background())
				defer cancel()
				continue
			}
			continue
		}

		// Store assistant message with both text and tool calls
		if assistantMsg.Role != "" {
			r.Messages = append(r.Messages, assistantMsg)
		} else if responseText != "" {
			r.Messages = append(r.Messages, fantasy.Message{
				Role:    fantasy.MessageRoleAssistant,
				Content: []fantasy.MessagePart{fantasy.TextPart{Text: responseText}},
			})
		}

		// Accumulate context size
		r.ContextSize += usage.InputTokens

		// Print context size and total spent
		fmt.Println()
		printUsage(r.ContextSize, r.TotalSpent)
	}
}

// printUsage displays context size and total tokens spent
func printUsage(contextSize int64, spent fantasy.Usage) {
	dim := "\x1b[90m"
	reset := "\x1b[0m"
	fmt.Fprintf(os.Stdout, dim+"Tokens: context=%d, spent=%d"+reset+"\n",
		contextSize, spent.TotalTokens)
}
