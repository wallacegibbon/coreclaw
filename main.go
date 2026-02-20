package main

import (
	"context"
	"fmt"
	"os"

	"charm.land/fantasy"
	"charm.land/fantasy/providers/openai"
	agentpkg "github.com/wallacegibbon/coreclaw/internal/agent"
	"github.com/wallacegibbon/coreclaw/internal/config"
	debugpkg "github.com/wallacegibbon/coreclaw/internal/debug"
	"github.com/wallacegibbon/coreclaw/internal/run"
	"github.com/wallacegibbon/coreclaw/internal/tools"
)

func main() {
	cfg := config.Parse()

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

	provider, err := createProvider(providerConfig.APIKey, providerConfig.BaseURL, cfg.DebugAPI)
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
	)

	processor := agentpkg.NewProcessor(agent)
	runner := run.New(processor, providerConfig.BaseURL, providerConfig.ModelName, cfg.SystemPrompt)

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

func createProvider(apiKey, baseURL string, debugAPI bool) (interface{ LanguageModel(context.Context, string) (fantasy.LanguageModel, error) }, error) {
	var opts []openai.Option
	opts = append(opts, openai.WithAPIKey(apiKey), openai.WithBaseURL(baseURL))
	if debugAPI {
		opts = append(opts, openai.WithHTTPClient(debugpkg.NewHTTPClient()))
	}
	return openai.New(opts...)
}

func printHelp() {
	fmt.Printf("CoreClaw - A minimal AI Agent with bash tool access\n\n")
	fmt.Printf("Usage:\n")
	fmt.Printf("  coreclaw [prompt]    Execute a single prompt\n")
	fmt.Printf("  coreclaw             Run in interactive mode\n\n")
	fmt.Printf("Environment Variables:\n")
	fmt.Printf("  OPENAI_API_KEY      OpenAI API key (uses GPT-4o)\n")
	fmt.Printf("  DEEPSEEK_API_KEY    DeepSeek API key (uses deepseek-chat)\n")
	fmt.Printf("  ZAI_API_KEY         ZAI API key (uses GLM-4.7)\n\n")
	fmt.Printf("Flags:\n")
	fmt.Printf("  -version            Show version information\n")
	fmt.Printf("  -help               Show help information\n")
	fmt.Printf("  -debug-api          Show raw API requests and responses\n")
	fmt.Printf("  -file string        Read prompt from file\n")
	fmt.Printf("  -system string      Override system prompt\n")
	fmt.Printf("  -api-key string     API key (required with --base-url)\n")
	fmt.Printf("  -base-url string    Custom API endpoint\n")
	fmt.Printf("  -model string       Model name to use\n")
}
