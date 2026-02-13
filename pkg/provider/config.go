package provider

import (
	"fmt"
	"os"

	"charm.land/catwalk/pkg/catwalk"
	"charm.land/catwalk/pkg/embedded"
)

// Config holds the provider configuration
type Config struct {
	APIKey    string
	BaseURL   string
	ModelName string
}

// GetProviderConfig returns the provider configuration based on available API keys
// Provider selection priority: OPENAI_API_KEY > DEEPSEEK_API_KEY > ZAI_API_KEY
func GetProviderConfig() (*Config, error) {
	providers := embedded.GetAll()

	openAIKey := os.Getenv("OPENAI_API_KEY")
	deepSeekKey := os.Getenv("DEEPSEEK_API_KEY")
	zaiKey := os.Getenv("ZAI_API_KEY")

	if openAIKey != "" {
		return getOpenAIConfig(providers, openAIKey), nil
	}

	if deepSeekKey != "" {
		return getDeepSeekConfig(providers, deepSeekKey), nil
	}

	if zaiKey != "" {
		return getZAIConfig(providers, zaiKey), nil
	}

	return nil, fmt.Errorf("one of OPENAI_API_KEY, DEEPSEEK_API_KEY, or ZAI_API_KEY environment variables is required")
}

func getOpenAIConfig(providers []catwalk.Provider, apiKey string) *Config {
	config := &Config{
		APIKey:    apiKey,
		BaseURL:   os.Getenv("OPENAI_API_ENDPOINT"),
		ModelName: "gpt-4o",
	}

	for _, p := range providers {
		if p.ID == "openai" {
			if p.DefaultLargeModelID != "" {
				config.ModelName = p.DefaultLargeModelID
			}
			if p.DefaultSmallModelID != "" {
				config.ModelName = p.DefaultSmallModelID
			}
			if p.APIEndpoint != "" && p.APIEndpoint[0] != '$' {
				config.BaseURL = p.APIEndpoint
			}
			break
		}
	}

	return config
}

func getDeepSeekConfig(providers []catwalk.Provider, apiKey string) *Config {
	config := &Config{
		APIKey:    apiKey,
		BaseURL:   "https://api.deepseek.com/v1",
		ModelName: "deepseek-chat",
	}

	for _, p := range providers {
		if p.ID == "deepseek" {
			config.BaseURL = p.APIEndpoint
			// Use small model as default since reasoning models
			// require special handling for tool calls
			if p.DefaultSmallModelID != "" {
				config.ModelName = p.DefaultSmallModelID
			}
			break
		}
	}

	return config
}

func getZAIConfig(providers []catwalk.Provider, apiKey string) *Config {
	config := &Config{
		APIKey:    apiKey,
		BaseURL:   "https://api.z.ai/api/coding/paas/v4",
		ModelName: "glm-4.7",
	}

	for _, p := range providers {
		if p.ID == "zai" {
			if p.DefaultLargeModelID != "" {
				config.ModelName = p.DefaultLargeModelID
			} else if p.DefaultSmallModelID != "" {
				config.ModelName = p.DefaultSmallModelID
			}
			config.BaseURL = p.APIEndpoint
			break
		}
	}

	return config
}
