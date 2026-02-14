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
// Command line flags (--base-url, --model, --api-key) take precedence over environment variables
// When --base-url is specified, environment variables are ignored and --api-key is required
func GetProviderConfig(apiKey, baseURL, modelName string) (*Config, error) {
	providers := embedded.GetAll()

	var selectedAPIKey string

	// If --base-url is specified, ignore environment variables and require --api-key
	if baseURL != "" {
		if apiKey == "" {
			return nil, fmt.Errorf("--api-key is required when --base-url is specified")
		}
		selectedAPIKey = apiKey
		// Default to OpenAI-style configuration for custom base URLs
		config := &Config{
			APIKey:    selectedAPIKey,
			BaseURL:   baseURL,
			ModelName: modelName,
		}
		if config.ModelName == "" {
			config.ModelName = "gpt-4o"
		}
		return config, nil
	}

	// Command line API key takes precedence
	if apiKey != "" {
		selectedAPIKey = apiKey
	} else {
		// Otherwise use environment variables
		openAIKey := os.Getenv("OPENAI_API_KEY")
		deepSeekKey := os.Getenv("DEEPSEEK_API_KEY")
		zaiKey := os.Getenv("ZAI_API_KEY")

		if openAIKey != "" {
			selectedAPIKey = openAIKey
		} else if deepSeekKey != "" {
			selectedAPIKey = deepSeekKey
		} else if zaiKey != "" {
			selectedAPIKey = zaiKey
		} else {
			return nil, fmt.Errorf("one of OPENAI_API_KEY, DEEPSEEK_API_KEY, or ZAI_API_KEY environment variables is required, or use --api-key flag")
		}
	}

	// Determine provider based on where the API key came from
	var config *Config
	if apiKey != "" {
		// Using command line API key - default to OpenAI style
		config = &Config{
			APIKey:    selectedAPIKey,
			BaseURL:   "https://api.openai.com/v1",
			ModelName: "gpt-4o",
		}
	} else if os.Getenv("OPENAI_API_KEY") != "" {
		config = getOpenAIConfig(providers, selectedAPIKey)
	} else if os.Getenv("DEEPSEEK_API_KEY") != "" {
		config = getDeepSeekConfig(providers, selectedAPIKey)
	} else {
		config = getZAIConfig(providers, selectedAPIKey)
	}

	// Override with command line flags if specified
	if modelName != "" {
		config.ModelName = modelName
	}

	return config, nil
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
