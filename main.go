package main

import (
	"context"
	"fmt"
	"os"

	agentpkg "github.com/wallacegibbon/coreclaw/internal/agent"
	"github.com/wallacegibbon/coreclaw/internal/app"
	"github.com/wallacegibbon/coreclaw/internal/config"
	"github.com/wallacegibbon/coreclaw/internal/run"
	"github.com/wallacegibbon/coreclaw/internal/adaptors"
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

	appCfg, err := app.Setup(cfg)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}

	agent := appCfg.CreateAgent()

	// Create terminal adaptor for stdio
	adaptor := adaptors.NewAdaptor()

	// Create processor with terminal output stream
	processor := agentpkg.NewProcessorWithIO(agent, adaptor.Input, adaptor.Output)
	runner := run.New(processor, adaptor, appCfg.Cfg.BaseURL, appCfg.Cfg.ModelName)

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
  -api-key string     API key for the provider (required)
  -model string       Model name to use
  -version            Show version information
  -help               Show help information
  -debug-api          Show raw API requests and responses
  -file string        Read prompt from file
  -system string      Override system prompt
  -skill string       Skills directory path (can be specified multiple times)
`)
}
