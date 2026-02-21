package main

import (
	"context"
	"fmt"
	"os"
	"strings"

	"charm.land/fantasy"
	"charm.land/fantasy/providers/anthropic"
	"charm.land/fantasy/providers/openai"
	"charm.land/fantasy/providers/openaicompat"
	agentpkg "github.com/wallacegibbon/coreclaw/internal/agent"
	"github.com/wallacegibbon/coreclaw/internal/config"
	debugpkg "github.com/wallacegibbon/coreclaw/internal/debug"
	"github.com/wallacegibbon/coreclaw/internal/run"
	"github.com/wallacegibbon/coreclaw/internal/tools"
)

const defaultSystemPrompt = `You are an AI assistant with access to a bash shell. Use the bash tool to interact with the system.

CRITICAL RULES:
- The bash tool is your ONLY way to interact with the system
- ALWAYS use bash for: listing files, reading content, running commands, installing packages, checking system info
- NEVER assume file contents or command outputs - use bash to verify
- Be precise and careful with commands - double-check before executing
- When uncertain about system state, use bash to investigate
- For network operations (HTTP requests, downloading files, API calls), ALWAYS use curl

GENERAL WORKFLOW:
1. Use bash to gather information before making assumptions
2. Execute commands to verify your understanding
3. Run appropriate commands based on user requests
4. Provide accurate, helpful responses based on actual command outputs

You can help with any task that can be accomplished through shell commands: file operations, system administration, development tasks, network operations (using curl), package management, etc.`

func main() {
	cfg := config.Parse()

	// Compute effective system prompt
	systemPrompt := defaultSystemPrompt
	if cfg.SystemPrompt != "" {
		systemPrompt = cfg.SystemPrompt
	}

	if cfg.ShowVersion {
		fmt.Printf("coreclaw version %s\n", config.Version)
		os.Exit(0)
	}

	if cfg.ShowHelp {
		printHelp()
		os.Exit(0)
	}

	providerConfig, err := cfg.GetProviderConfig()
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}

	provider, err := createProvider(providerConfig.Provider, providerConfig.APIKey, providerConfig.BaseURL, cfg.DebugAPI)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to create provider: %v\n", err)
		os.Exit(1)
	}

	model, err := provider.LanguageModel(context.Background(), providerConfig.ModelName)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to create language model: %v\n", err)
		os.Exit(1)
	}

	bashTool := tools.NewBashTool()
	agent := fantasy.NewAgent(
		model,
		fantasy.WithTools(bashTool),
		fantasy.WithSystemPrompt(systemPrompt),
	)

	processor := agentpkg.NewProcessor(agent)
	runner := run.New(processor, providerConfig.BaseURL, providerConfig.ModelName)

	ctx := context.Background()

	if cfg.Prompt != "" {
		err = runner.RunSingle(ctx, cfg.Prompt)
		if err != nil {
			os.Exit(1)
		}
		os.Exit(0)
	}

	err = runner.RunInteractive(ctx)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func createProvider(provider, apiKey, baseURL string, debugAPI bool) (interface{ LanguageModel(context.Context, string) (fantasy.LanguageModel, error) }, error) {
	switch provider {
	case "anthropic":
		return createAnthropicProvider(apiKey, baseURL, debugAPI)
	default:
		return createOpenAIProvider(apiKey, baseURL, debugAPI)
	}
}

func createAnthropicProvider(apiKey, baseURL string, debugAPI bool) (interface{ LanguageModel(context.Context, string) (fantasy.LanguageModel, error) }, error) {
	var opts []anthropic.Option
	opts = append(opts, anthropic.WithAPIKey(apiKey))
	if baseURL != "" {
		opts = append(opts, anthropic.WithBaseURL(baseURL))
	}
	if debugAPI {
		opts = append(opts, anthropic.WithHTTPClient(debugpkg.NewHTTPClient()))
	}
	return anthropic.New(opts...)
}

func createOpenAIProvider(apiKey, baseURL string, debugAPI bool) (interface{ LanguageModel(context.Context, string) (fantasy.LanguageModel, error) }, error) {
	// Use openaicompat for non-OpenAI URLs (Ollama, LM Studio, DeepSeek, etc.)
	// This enables reasoning/thinking content support
	if !strings.Contains(baseURL, "api.openai.com") {
		var opts []openaicompat.Option
		opts = append(opts, openaicompat.WithAPIKey(apiKey), openaicompat.WithBaseURL(baseURL))
		if debugAPI {
			opts = append(opts, openaicompat.WithHTTPClient(debugpkg.NewHTTPClient()))
		}
		return openaicompat.New(opts...)
	}

	// Use native OpenAI provider for api.openai.com
	var opts []openai.Option
	opts = append(opts, openai.WithAPIKey(apiKey), openai.WithBaseURL(baseURL))
	if debugAPI {
		opts = append(opts, openai.WithHTTPClient(debugpkg.NewHTTPClient()))
	}
	return openai.New(opts...)
}

func printHelp() {
	fmt.Print(`CoreClaw - A minimal AI Agent with bash tool access

Usage:
  coreclaw [prompt]    Execute a single prompt
  coreclaw             Run in interactive mode

Examples:
  coreclaw --type openai --base-url https://api.openai.com/v1 --api-key $OPENAI_API_KEY --model gpt-4o "hello"
  coreclaw --type anthropic --base-url https://api.anthropic.com --api-key $ANTHROPIC_API_KEY --model claude-sonnet-4-20250514 "hello"
  coreclaw --type openai --base-url http://localhost:11434/v1 --api-key xxx --model llama3 "hello"

Flags:
  -type string        Provider type: anthropic, openai (required)
  -base-url string    API endpoint URL (required)
  -api-key string     API key (required)
  -model string       Model name to use
  -version            Show version information
  -help               Show help information
  -debug-api          Show raw API requests and responses
  -file string        Read prompt from file
  -system string      Override system prompt
`)
}
