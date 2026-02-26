package main

import (
	"fmt"
	"os"

	"github.com/wallacegibbon/coreclaw/internal/adaptors"
	"github.com/wallacegibbon/coreclaw/internal/app"
	"github.com/wallacegibbon/coreclaw/internal/config"
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

	adaptor := adaptors.NewTUIAdaptor(appCfg.AgentFactory(), appCfg.Cfg.BaseURL, appCfg.Cfg.ModelName)
	adaptor.Start()
}

func printHelp() {
	fmt.Print(`CoreClaw - A minimal AI Agent

Usage:
  coreclaw [flags]

Flags:
  -type string        Provider type: anthropic, openai (required)
  -base-url string    API endpoint URL (required)
  -api-key string     API key for the provider (required)
  -model string       Model name to use
  -system string      Override system prompt
  -skill string       Skills directory path (can be specified multiple times)
  -debug-api          Show raw API requests and responses
  -version            Show version information
  -help               Show help information

Examples:
  coreclaw --type openai --base-url https://api.openai.com/v1 --api-key $OPENAI_API_KEY --model gpt-4o
  coreclaw --type anthropic --base-url https://api.anthropic.com --api-key $ANTHROPIC_API_KEY --model claude-sonnet-4-20250514
  coreclaw --type openai --base-url http://localhost:11434/v1 --api-key xxx --model llama3

`)
}
