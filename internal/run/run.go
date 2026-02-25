package run

import (
	"bufio"
	"context"
	"fmt"
	"os"
	"os/signal"
	"strings"

	"github.com/wallacegibbon/coreclaw/internal/adaptors"
	"github.com/wallacegibbon/coreclaw/internal/agent"
)

// Runner handles running the agent
type Runner struct {
	Session   *agent.Session
	Processor *agent.Processor
	BaseURL   string
	ModelName string
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

	// Create runner for tracking request state
	syncRunner := agent.NewSyncRunner(r.Session)

	// Set up command callback to send usage info
	r.Session.OnCommandDone = func() {
		r.Session.SendUsage()
	}

	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt)

	go func() {
		for range sigChan {
			if syncRunner.IsInProgress() {
				cancel()
				fmt.Println("\nRequest cancelled.")
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

		// Handle commands like /summarize
		if strings.HasPrefix(userPrompt, "/") {
			command := strings.TrimPrefix(userPrompt, "/")
			_, err := r.Session.HandleCommand(ctx, command)
			if err != nil && ctx.Err() == context.Canceled {
				cancel()
				ctx, cancel = context.WithCancel(context.Background())
				defer cancel()
			}
			continue
		}

		syncRunner.SetInProgress(true)

		_, _, err = r.Session.ProcessPrompt(ctx, userPrompt)

		syncRunner.SetInProgress(false)

		// Send usage info after each prompt
		r.Session.SendUsage()

		if err != nil && ctx.Err() == context.Canceled {
			cancel()
			ctx, cancel = context.WithCancel(context.Background())
			defer cancel()
		}
	}
}
