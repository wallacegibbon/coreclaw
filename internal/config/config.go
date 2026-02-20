package config

import (
	"flag"
	"fmt"
	"os"

	"github.com/wallacegibbon/coreclaw/internal/provider"
)

const Version = "0.1.0"

// Settings holds all CLI configuration
type Settings struct {
	ShowVersion bool
	ShowHelp     bool
	DebugAPI     bool
	PromptFile   string
	SystemPrompt string
	APIKey       string
	BaseURL      string
	ModelName    string
	Prompt       string
}

// Parse parses CLI flags and returns settings
func Parse() *Settings {
	showVersion := flag.Bool("version", false, "Show version information")
	showHelp := flag.Bool("help", false, "Show help information")
	debugAPI := flag.Bool("debug-api", false, "Show raw API requests and responses")
	promptFile := flag.String("file", "", "Read prompt from file")
	systemPrompt := flag.String("system", "", "Override system prompt")
	apiKey := flag.String("api-key", "", "API key for the provider (required when using --base-url)")
	baseURL := flag.String("base-url", "", "Base URL for the API endpoint (requires --api-key, ignores env vars)")
	modelName := flag.String("model", "", "Model name to use (defaults to provider default)")
	flag.Parse()

	s := &Settings{
		ShowVersion:  *showVersion,
		ShowHelp:     *showHelp,
		DebugAPI:     *debugAPI,
		PromptFile:   *promptFile,
		SystemPrompt: *systemPrompt,
		APIKey:       *apiKey,
		BaseURL:      *baseURL,
		ModelName:    *modelName,
	}

	// Get prompt from file or args
	if s.PromptFile != "" {
		content, err := os.ReadFile(s.PromptFile)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Failed to read prompt file: %v\n", err)
			os.Exit(1)
		}
		s.Prompt = string(content)
	} else if flag.NArg() > 0 {
		s.Prompt = flag.Arg(0)
	}

	return s
}

// GetProviderConfig returns the provider configuration
func (s *Settings) GetProviderConfig() (*provider.Config, error) {
	return provider.GetProviderConfig(s.APIKey, s.BaseURL, s.ModelName)
}
