package factory

import (
	"fmt"
	"net/http"
	"strings"

	"github.com/alayacore/alayacore/internal/llm"
	"github.com/alayacore/alayacore/internal/llm/providers"
)

// ProviderConfig configures a provider
type ProviderConfig struct {
	Type       string // "anthropic", "openai", "openaicompat"
	APIKey     string
	BaseURL    string
	Model      string
	HTTPClient *http.Client

	// Provider-specific options
	SupportsReasoning bool // For OpenAI-compatible providers (DeepSeek, etc.)
}

// NewProvider creates a provider based on configuration
func NewProvider(config ProviderConfig) (llm.Provider, error) {
	switch strings.ToLower(config.Type) {
	case "anthropic":
		opts := []providers.AnthropicOption{
			providers.WithAPIKey(config.APIKey),
		}
		if config.BaseURL != "" {
			opts = append(opts, providers.WithBaseURL(config.BaseURL))
		}
		if config.HTTPClient != nil {
			opts = append(opts, providers.WithHTTPClient(config.HTTPClient))
		}
		if config.Model != "" {
			opts = append(opts, providers.WithAnthropicModel(config.Model))
		}
		return providers.NewAnthropic(opts...)

	case "openai":
		opts := []providers.OpenAIOption{
			providers.WithOpenAIAPIKey(config.APIKey),
		}
		if config.BaseURL != "" {
			opts = append(opts, providers.WithOpenAIBaseURL(config.BaseURL))
		}
		if config.HTTPClient != nil {
			opts = append(opts, providers.WithOpenAIHTTPClient(config.HTTPClient))
		}
		if config.Model != "" {
			opts = append(opts, providers.WithOpenAIModel(config.Model))
		}
		return providers.NewOpenAI(opts...)

	case "openaicompat", "openai-compatible":
		opts := []providers.OpenAICompatOption{
			providers.WithOpenAICompatBaseURL(config.BaseURL),
		}
		if config.APIKey != "" {
			opts = append(opts, providers.WithOpenAICompatAPIKey(config.APIKey))
		}
		if config.HTTPClient != nil {
			opts = append(opts, providers.WithOpenAICompatHTTPClient(config.HTTPClient))
		}
		if config.Model != "" {
			opts = append(opts, providers.WithOpenAICompatModel(config.Model))
		}
		opts = append(opts, providers.WithOpenAICompatReasoning(config.SupportsReasoning))
		return providers.NewOpenAICompat(opts...)

	default:
		return nil, fmt.Errorf("unknown provider type: %s", config.Type)
	}
}

// ProviderTypeFromURL infers provider type from base URL
func ProviderTypeFromURL(baseURL string) string {
	if strings.Contains(baseURL, "api.anthropic.com") {
		return "anthropic"
	}
	if strings.Contains(baseURL, "api.openai.com") {
		return "openai"
	}
	// Everything else is treated as OpenAI-compatible
	return "openaicompat"
}
