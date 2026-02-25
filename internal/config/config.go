package config

import (
	"flag"

	"github.com/wallacegibbon/coreclaw/internal/provider"
)

const Version = "0.1.0"

// Settings holds all CLI configuration
type Settings struct {
	ShowVersion  bool
	ShowHelp     bool
	DebugAPI     bool
	SystemPrompt string
	APIKey       string
	BaseURL      string
	ModelName    string
	ProviderType string
	Skills       []string
	Addr         string
}

// Parse parses CLI flags and returns settings
func Parse() *Settings {
	showVersion := flag.Bool("version", false, "Show version information")
	showHelp := flag.Bool("help", false, "Show help information")
	debugAPI := flag.Bool("debug-api", false, "Show raw API requests and responses")
	systemPrompt := flag.String("system", "", "Override system prompt")
	apiKey := flag.String("api-key", "", "API key for the provider (required when using --base-url)")
	baseURL := flag.String("base-url", "", "Base URL for the API endpoint (requires --api-key, ignores env vars)")
	modelName := flag.String("model", "", "Model name to use (defaults to provider default)")
	providerType := flag.String("type", "", "Provider type: anthropic, openai (overrides auto-detection)")
	skill := flag.String("skill", "", "Skill path (can be specified multiple times)")
	addr := flag.String("addr", ":8080", "Server address to listen on (for web server)")
	flag.Parse()

	// Collect skill paths
	var skillPaths []string
	if *skill != "" {
		skillPaths = append(skillPaths, *skill)
	}

	s := &Settings{
		ShowVersion:  *showVersion,
		ShowHelp:     *showHelp,
		DebugAPI:     *debugAPI,
		SystemPrompt: *systemPrompt,
		APIKey:       *apiKey,
		BaseURL:      *baseURL,
		ModelName:    *modelName,
		ProviderType: *providerType,
		Skills:       skillPaths,
		Addr:         *addr,
	}

	return s
}

// GetProviderConfig returns the provider configuration
func (s *Settings) GetProviderConfig() (*provider.Config, error) {
	return provider.GetProviderConfig(s.APIKey, s.BaseURL, s.ModelName, s.ProviderType)
}
