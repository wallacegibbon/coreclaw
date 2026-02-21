package run

import (
	"bufio"
	"context"
	"errors"
	"fmt"
	"os"
	"os/signal"
	"strings"
	"sync"

	"charm.land/fantasy"
	"github.com/chzyer/readline"
	"github.com/wallacegibbon/coreclaw/internal/agent"
	"github.com/wallacegibbon/coreclaw/internal/terminal"
)

// Runner handles running the agent
type Runner struct {
	Processor  *agent.Processor
	Messages  []fantasy.Message
	BaseURL   string
	ModelName string
	VimMode   bool
}

// New creates a new Runner
func New(processor *agent.Processor, baseURL, modelName string, vimMode bool) *Runner {
	return &Runner{
		Processor: processor,
		Messages:  nil,
		BaseURL:   baseURL,
		ModelName: modelName,
		VimMode:   vimMode,
	}
}

// RunSingle runs a single prompt and exits
func (r *Runner) RunSingle(ctx context.Context, prompt string) error {
	_, _, _, err := r.Processor.ProcessPrompt(ctx, prompt, r.Messages)
	return err
}

// RunInteractive starts the interactive REPL
func (r *Runner) RunInteractive(ctx context.Context) error {
	isTTY := terminal.IsTerminal()

	var rl interface {
		Readline() (string, error)
	}
	var err error
	if isTTY {
		rl, err = terminal.ReadlineInstance(r.BaseURL, r.ModelName, r.VimMode)
		if err != nil {
			return fmt.Errorf("failed to initialize readline: %w", err)
		}
	}

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

		if isTTY {
			fmt.Print(terminal.GetBracketedLine(r.BaseURL, r.ModelName))
			userPrompt, err = rl.Readline()
			if err != nil {
				if errors.Is(err, readline.ErrInterrupt) {
					continue
				}
				return err
			}
			userPrompt = strings.TrimSpace(userPrompt)
		} else {
			fmt.Fprint(os.Stderr, terminal.GetPrompt(r.BaseURL, r.ModelName))
			reader := bufio.NewReader(os.Stdin)
			input, _ := reader.ReadString('\n')
			userPrompt = strings.TrimSpace(input)
			if userPrompt == "" {
				return nil
			}
		}

		if userPrompt == "" {
			continue
		}

		mu.Lock()
		requestInProgress = true
		mu.Unlock()

		_, responseText, assistantMsg, err := r.Processor.ProcessPrompt(ctx, userPrompt, r.Messages)

		mu.Lock()
		requestInProgress = false
		mu.Unlock()

		if err != nil {
			if ctx.Err() == context.Canceled {
				cancel()
				ctx, cancel = context.WithCancel(context.Background())
				defer cancel()
				continue
			}
			continue
		}

		r.Messages = append(r.Messages, fantasy.NewUserMessage(userPrompt))

		// Store assistant message with both text and tool calls
		if assistantMsg.Role != "" {
			r.Messages = append(r.Messages, assistantMsg)
		} else if responseText != "" {
			r.Messages = append(r.Messages, fantasy.Message{
				Role:    fantasy.MessageRoleAssistant,
				Content: []fantasy.MessagePart{fantasy.TextPart{Text: responseText}},
			})
		}
	}
}
