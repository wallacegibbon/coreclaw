package main

import (
	"bufio"
	"context"
	"errors"
	"flag"
	"fmt"
	"os"
	"os/signal"
	"strings"
	"sync"

	"charm.land/fantasy"
	"charm.land/fantasy/providers/openai"
	agentpkg "github.com/wallacegibbon/coreclaw/internal/agent"
	debugpkg "github.com/wallacegibbon/coreclaw/internal/debug"
	"github.com/wallacegibbon/coreclaw/internal/provider"
	"github.com/wallacegibbon/coreclaw/internal/terminal"
	"github.com/wallacegibbon/coreclaw/internal/tools"
	"github.com/chzyer/readline"
)

func main() {
	version := "0.1.0"
	showVersion := flag.Bool("version", false, "Show version information")
	showHelp := flag.Bool("help", false, "Show help information")
	debugAPI := flag.Bool("debug-api", false, "Show raw API requests and responses")
	noMarkdown := flag.Bool("no-markdown", false, "Disable markdown rendering")
	promptFile := flag.String("file", "", "Read prompt from file")
	systemPrompt := flag.String("system", "", "Override system prompt")
	apiKey := flag.String("api-key", "", "API key for the provider (required when using --base-url)")
	baseURL := flag.String("base-url", "", "Base URL for the API endpoint (requires --api-key, ignores env vars)")
	modelName := flag.String("model", "", "Model name to use (defaults to provider default)")
	flag.Parse()

	if *showVersion {
		fmt.Printf("coreclaw version %s\n", version)
		os.Exit(0)
	}

	if *showHelp {
		printHelp()
		os.Exit(0)
	}

	config, err := provider.GetProviderConfig(*apiKey, *baseURL, *modelName)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}

	var providerOpts []openai.Option
	providerOpts = append(providerOpts, openai.WithAPIKey(config.APIKey), openai.WithBaseURL(config.BaseURL))
	if *debugAPI {
		providerOpts = append(providerOpts, openai.WithHTTPClient(debugpkg.NewHTTPClient()))
	}

	provider, err := openai.New(providerOpts...)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to create provider: %v\n", err)
		os.Exit(1)
	}

	model, err := provider.LanguageModel(context.Background(), config.ModelName)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to create language model: %v\n", err)
		os.Exit(1)
	}

	defaultSystemPrompt := "You are a helpful AI assistant with access to a bash shell. Use bash tool to execute commands when needed. Be precise and careful with commands."

	// Note: Some models like qwen3-coding may not support OpenAI-compatible tool calling format.
	// For best results, use models that fully support OpenAI's function calling API.

	finalSystemPrompt := defaultSystemPrompt
	if *systemPrompt != "" {
		finalSystemPrompt = *systemPrompt
	}

	bashTool := tools.NewBashTool()
	agent := fantasy.NewAgent(
		model,
		fantasy.WithTools(bashTool),
		fantasy.WithSystemPrompt(finalSystemPrompt),
	)

	processor := agentpkg.NewProcessor(agent)
	processor.NoMarkdown = *noMarkdown

	ctx := context.Background()
	var messages []fantasy.Message

	var userPrompt string
	if *promptFile != "" {
		content, err := os.ReadFile(*promptFile)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Failed to read prompt file: %v\n", err)
			os.Exit(1)
		}
		userPrompt = strings.TrimSpace(string(content))
	} else if flag.NArg() > 0 {
		userPrompt = strings.Join(flag.Args(), " ")
	}

	if userPrompt != "" {
		_, _, err = processor.ProcessPrompt(ctx, userPrompt, messages)
		if err != nil {
			os.Exit(1)
		}
		os.Exit(0)
	}

	runInteractiveMode(processor, messages, config.BaseURL, config.ModelName)
}

func printHelp() {
	fmt.Printf("CoreClaw - A minimal AI Agent with bash tool access\n\n")
	fmt.Printf("Usage:\n  coreclaw [prompt]    Execute a single prompt\n  coreclaw             Run in interactive mode\n\n")
	fmt.Printf("Environment Variables:\n")
	fmt.Printf("  OPENAI_API_KEY      OpenAI API key (uses GPT-4o)\n")
	fmt.Printf("  DEEPSEEK_API_KEY    DeepSeek API key (uses deepseek-chat)\n")
	fmt.Printf("  ZAI_API_KEY         ZAI API key (uses GLM-4.7)\n\n")
	fmt.Printf("Flags (take precedence over environment variables):\n")
	flag.PrintDefaults()
	fmt.Printf("\nExamples:\n")
	fmt.Printf("  coreclaw                        Run in interactive mode\n")
	fmt.Printf("  coreclaw \"list files\"            Execute a single prompt\n")
	fmt.Printf("  coreclaw --api-key sk-xxx --base-url http://localhost:11434/v1 --model llama3 \"hello\"  Specify model\n")
}

func runInteractiveMode(processor *agentpkg.Processor, messages []fantasy.Message, baseURL, model string) {
	isTTY := terminal.IsTerminal()

	var rl interface {
		Readline() (string, error)
	}
	var err error
	if isTTY {
		rl, err = terminal.ReadlineInstance(baseURL, model)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Failed to initialize readline: %v\n", err)
			os.Exit(1)
		}
	}

	ctx, cancel := context.WithCancel(context.Background())
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
			fmt.Print(terminal.GetBracketedLine(baseURL, model))
			userPrompt, err = rl.Readline()
			if err != nil {
				if errors.Is(err, readline.ErrInterrupt) {
					continue
				}
				return
			}
			userPrompt = strings.TrimSpace(userPrompt)
		} else {
			fmt.Fprint(os.Stderr, terminal.GetPrompt(baseURL, model))
			reader := bufio.NewReader(os.Stdin)
			input, _ := reader.ReadString('\n')
			userPrompt = strings.TrimSpace(input)
			if userPrompt == "" {
				return
			}
		}

		if userPrompt == "" {
			continue
		}

		mu.Lock()
		requestInProgress = true
		mu.Unlock()

		_, responseText, err := processor.ProcessPrompt(ctx, userPrompt, messages)

		mu.Lock()
		requestInProgress = false
		mu.Unlock()

		if err != nil {
			if ctx.Err() == context.Canceled {
				cancel()
				ctx, cancel = context.WithCancel(context.Background())
				continue
			}
			continue
		}

		messages = append(messages, fantasy.NewUserMessage(userPrompt))

		if responseText != "" {
			messages = append(messages, fantasy.Message{
				Role:    fantasy.MessageRoleAssistant,
				Content: []fantasy.MessagePart{fantasy.TextPart{Text: responseText}},
			})
		}
	}
}
