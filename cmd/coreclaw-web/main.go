package main

import (
	"fmt"
	"os"

	"github.com/wallacegibbon/coreclaw/internal/app"
	"github.com/wallacegibbon/coreclaw/internal/config"
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

	port := cfg.Addr
	if port == "" {
		port = ":8080"
	}

	// Create WebSocket adaptor
	wsAdaptor := adaptors.NewWebSocketAdaptor(port, appCfg.AgentFactory())

	// Print startup info
	fmt.Printf("Starting CoreClaw WebSocket server on %s\n", port)
	fmt.Printf("  Provider: %s\n", appCfg.Cfg.ProviderType)
	fmt.Printf("  Model: %s\n", appCfg.Cfg.ModelName)
	fmt.Printf("  Base URL: %s\n", appCfg.Cfg.BaseURL)
	if len(appCfg.Cfg.Skills) > 0 {
		fmt.Printf("  Skills: %v\n", appCfg.Cfg.Skills)
	}
	fmt.Printf("\nWeb UI:   http://localhost%s\n", port)
	fmt.Printf("WebSocket: ws://localhost%s/ws\n", port)

	wsAdaptor.Start()

	// Wait for interrupt
	select {}
}

func printHelp() {
	fmt.Print(`CoreClaw Web - A WebSocket server for CoreClaw

Usage:
  coreclaw-web [flags]

Flags:
  -type string       Provider type: anthropic, openai (required)
  -base-url string   API endpoint URL (required)
  -api-key string    API key for the provider (required)
  -model string      Model name to use
  -addr string       Server address to listen on (default ":8080")
  -debug-api         Show raw API requests and responses
  -system string     Override system prompt
  -skill string      Skills directory path (can be specified multiple times)
  -version           Show version information
  -help              Show help information

Examples:
  coreclaw-web --type openai --base-url https://api.openai.com/v1 --api-key $OPENAI_API_KEY --model gpt-4o
  coreclaw-web --type anthropic --base-url https://api.anthropic.com --api-key $ANTHROPIC_API_KEY --model claude-sonnet-4-20250514
  coreclaw-web --type openai --base-url http://localhost:11434/v1 --api-key xxx --model llama3 --addr :9090
`)
}
