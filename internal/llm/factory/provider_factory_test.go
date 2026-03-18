package factory

import (
	"testing"

	"github.com/alayacore/alayacore/internal/llm/providers"
)

func TestFactoryPassesPromptCacheToAnthropic(t *testing.T) {
	// Test that ProviderConfig.PromptCache is passed to Anthropic provider
	config := ProviderConfig{
		Type:        "anthropic",
		APIKey:      "test-key",
		Model:       "claude-3-5-sonnet-20241022",
		PromptCache: true,
	}

	provider, err := NewProvider(config)
	if err != nil {
		t.Fatalf("Failed to create provider: %v", err)
	}

	// Type assert to AnthropicProvider to check promptCache field
	anthropicProvider, ok := provider.(*providers.AnthropicProvider)
	if !ok {
		t.Fatalf("Expected AnthropicProvider, got %T", provider)
	}

	// We can't directly access the promptCache field since it's private,
	// but we can verify the provider was created successfully.
	// The actual cache_control behavior is tested in anthropic_system_test.go
	if anthropicProvider == nil {
		t.Fatal("AnthropicProvider is nil")
	}
}

func TestFactoryPromptCacheFalse(t *testing.T) {
	// Test that PromptCache=false works
	config := ProviderConfig{
		Type:        "anthropic",
		APIKey:      "test-key",
		Model:       "claude-3-5-sonnet-20241022",
		PromptCache: false,
	}

	provider, err := NewProvider(config)
	if err != nil {
		t.Fatalf("Failed to create provider: %v", err)
	}

	_, ok := provider.(*providers.AnthropicProvider)
	if !ok {
		t.Fatalf("Expected AnthropicProvider, got %T", provider)
	}
}

func TestFactoryOpenAIIgnoresPromptCache(t *testing.T) {
	// Test that OpenAI provider ignores PromptCache field
	config := ProviderConfig{
		Type:        "openai",
		APIKey:      "test-key",
		Model:       "gpt-4o",
		PromptCache: true, // Should be ignored
	}

	provider, err := NewProvider(config)
	if err != nil {
		t.Fatalf("Failed to create provider: %v", err)
	}

	_, ok := provider.(*providers.OpenAIProvider)
	if !ok {
		t.Fatalf("Expected OpenAIProvider, got %T", provider)
	}
}

func TestFactoryAnthropicWithAllOptions(t *testing.T) {
	// Test that all options work together
	config := ProviderConfig{
		Type:        "anthropic",
		APIKey:      "test-key",
		BaseURL:     "https://custom.anthropic.com",
		Model:       "claude-3-opus-20240229",
		PromptCache: true,
	}

	provider, err := NewProvider(config)
	if err != nil {
		t.Fatalf("Failed to create provider: %v", err)
	}

	_, ok := provider.(*providers.AnthropicProvider)
	if !ok {
		t.Fatalf("Expected AnthropicProvider, got %T", provider)
	}
}
