package provider

import (
	"fmt"
)

// Config holds the provider configuration
type Config struct {
	APIKey    string
	BaseURL   string
	ModelName string
	Provider  string // "anthropic" or "openai"
}

// GetProviderConfig returns the provider configuration based on CLI flags
// Users must specify --type, --base-url, and --api-key explicitly
func GetProviderConfig(apiKey, baseURL, modelName, providerType string) (*Config, error) {
	if providerType == "" {
		return nil, fmt.Errorf("--type is required (anthropic or openai)")
	}
	if baseURL == "" {
		return nil, fmt.Errorf("--base-url is required")
	}
	if apiKey == "" {
		return nil, fmt.Errorf("--api-key is required")
	}

	if providerType != "anthropic" && providerType != "openai" {
		return nil, fmt.Errorf("unknown provider type: %s (supported: anthropic, openai)", providerType)
	}

	return &Config{
		APIKey:    apiKey,
		BaseURL:   baseURL,
		ModelName: modelName,
		Provider:  providerType,
	}, nil
}
