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
	"github.com/wallacegibbon/coreclaw/internal/adaptors"
)

// Runner handles running the agent
type Runner struct {
	Session   *agent.Session
	Processor *agent.Processor
	BaseURL   string
	ModelName string
	TotalSpent fantasy.Usage
}

// New creates a new Runner with a terminal adaptor
func New(processor *agent.Processor, adaptor *adaptors.Adaptor, baseURL, modelName string) *Runner {
	return &Runner{
		Session:   agent.NewSession(processor),
		Processor: processor,
		BaseURL:   baseURL,
		ModelName: modelName,
	}
}

// RunSingle runs a single prompt and exits
func (r *Runner) RunSingle(ctx context.Context, prompt string) error {
	_, _, err := r.Session.ProcessPrompt(ctx, prompt)
	return err
}

// RunInteractive starts the interactive REPL
func (r *Runner) RunInteractive(ctx context.Context) error {
	reader := bufio.NewReader(r.Processor.Input)

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

		fmt.Fprint(os.Stderr, adaptors.GetPrompt(r.BaseURL, r.ModelName))
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

				_, summaryMsg, usage, err := r.Processor.Summarize(ctx, r.Session.Messages)

				mu.Lock()
				requestInProgress = false
				mu.Unlock()

				// Accumulate token usage
				r.TotalSpent.InputTokens += usage.InputTokens
				r.TotalSpent.OutputTokens += usage.OutputTokens
				r.TotalSpent.TotalTokens += usage.TotalTokens
				r.TotalSpent.ReasoningTokens += usage.ReasoningTokens

				// Replace messages with summary to reduce token count
				r.Session.Messages = []fantasy.Message{summaryMsg}
				// Context becomes the summarize output
				ctxSize := usage.OutputTokens

				// Print context size and total spent
				fmt.Println()
				printUsage(ctxSize, r.TotalSpent)

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

		_, usage, err := r.Session.ProcessPrompt(ctx, userPrompt)

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

		// Print context size and total spent
		fmt.Println()
		printUsage(usage.InputTokens, r.TotalSpent)
	}
}

// printUsage displays context size and total tokens spent
func printUsage(contextSize int64, spent fantasy.Usage) {
	dim := "\x1b[90m"
	reset := "\x1b[0m"
	fmt.Fprintf(os.Stdout, dim+"Tokens: context=%d, spent=%d"+reset+"\n",
		contextSize, spent.TotalTokens)
}
