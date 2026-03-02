package app

import (
	"context"
	"fmt"
	"strings"

	"charm.land/fantasy"
	"charm.land/fantasy/providers/anthropic"
	"charm.land/fantasy/providers/openai"
	"charm.land/fantasy/providers/openaicompat"
	"github.com/wallacegibbon/coreclaw/internal/config"
	debugpkg "github.com/wallacegibbon/coreclaw/internal/debug"
	"github.com/wallacegibbon/coreclaw/internal/skills"
	"github.com/wallacegibbon/coreclaw/internal/todo"
	"github.com/wallacegibbon/coreclaw/internal/tools"
)

const DefaultSystemPrompt = `You are an AI assistant with POSIX shell and some other tool access.

RULES:
- Never assume - verify with tools
- Check <available_skills> below; activate relevant ones using the activate_skill tool
- When running skill scripts, cd to the skill's directory first (e.g., cd /path/to/skill && ./scripts/script.sh)
- Do NOT use find to locate scripts - use the path from SKILL.md

PLANNING RULES (STRICT):
- For ANY non-trivial task (anything beyond a simple single action):
  1. First, read the current todo list with todo_read
  2. Create or update your plan using todo_write with content, active_form (present continuous), and status (pending/in_progress/completed)
     - IMPORTANT: The "content" field must remain EXACTLY THE SAME when updating status. Only change the "status" field.
     - Example: content="Install dependencies", active_form="Installing dependencies"
     - When updating to in_progress: content="Install dependencies", status="in_progress" (content unchanged)
     - When updating to completed: content="Install dependencies", status="completed" (content unchanged)
  3. Present the plan to the user and STOP - DO NOT execute any tools yet
  4. Wait for explicit user confirmation before proceeding
  5. Only after confirmation, execute tasks while updating todo status as you go
- For trivial tasks (single simple action like "what's in this file?"), you may proceed directly without planning
- ALWAYS STOP and wait for confirmation before executing any multi-step plan`

// Config holds the common app configuration
type Config struct {
	Cfg          *config.Settings
	Model        fantasy.LanguageModel
	SkillsMgr    *skills.Manager
	TodoMgr      *todo.Manager
	AgentTools   []fantasy.AgentTool
	SystemPrompt string
}

// Setup initializes the common app components
func Setup(cfg *config.Settings) (*Config, error) {
	// Compute effective system prompt
	systemPrompt := DefaultSystemPrompt
	if cfg.SystemPrompt != "" {
		systemPrompt = cfg.SystemPrompt
	}

	providerConfig, err := cfg.GetProviderConfig()
	if err != nil {
		return nil, fmt.Errorf("failed to get provider config: %w", err)
	}

	provider, err := CreateProvider(providerConfig.Provider, providerConfig.APIKey, providerConfig.BaseURL, cfg.DebugAPI)
	if err != nil {
		return nil, fmt.Errorf("failed to create provider: %w", err)
	}

	model, err := provider.LanguageModel(context.Background(), providerConfig.ModelName)
	if err != nil {
		return nil, fmt.Errorf("failed to create language model: %w", err)
	}

	skillsManager, err := skills.NewManager(cfg.Skills)
	if err != nil {
		return nil, fmt.Errorf("failed to initialize skills: %w", err)
	}

	// Generate skills fragment for system prompt
	skillsFragment := skillsManager.GenerateSystemPromptFragment()
	if skillsFragment != "" {
		systemPrompt = systemPrompt + "\n\n" + skillsFragment
	}

	// Create todo manager
	todoManager := todo.NewManager()

	readFileTool := tools.NewReadFileTool()
	todoReadTool := tools.NewTodoReadTool(todoManager)
	todoWriteTool := tools.NewTodoWriteTool(todoManager)
	writeFileTool := tools.NewWriteFileTool()
	activateSkillTool := tools.NewActivateSkillTool(skillsManager)
	posixShellTool := tools.NewPosixShellTool()

	return &Config{
		Cfg:          cfg,
		Model:        model,
		SkillsMgr:    skillsManager,
		TodoMgr:      todoManager,
		AgentTools:   []fantasy.AgentTool{readFileTool, todoReadTool, todoWriteTool, writeFileTool, activateSkillTool, posixShellTool},
		SystemPrompt: systemPrompt,
	}, nil
}

// CreateAgent creates a new fantasy agent with the configured tools and system prompt
func (c *Config) CreateAgent() fantasy.Agent {
	return fantasy.NewAgent(c.Model, fantasy.WithTools(c.AgentTools...), fantasy.WithSystemPrompt(c.SystemPrompt))
}

// AgentFactory returns a function that creates new agents (for WebSocket)
func (c *Config) AgentFactory() func() fantasy.Agent {
	return c.CreateAgent
}

// CreateProvider creates a provider based on type
func CreateProvider(provider, apiKey, baseURL string, debugAPI bool) (interface {
	LanguageModel(context.Context, string) (fantasy.LanguageModel, error)
}, error) {
	switch provider {
	case "anthropic":
		return CreateAnthropicProvider(apiKey, baseURL, debugAPI)
	default:
		return CreateOpenAIProvider(apiKey, baseURL, debugAPI)
	}
}

// CreateAnthropicProvider creates an Anthropic provider
func CreateAnthropicProvider(apiKey, baseURL string, debugAPI bool) (interface {
	LanguageModel(context.Context, string) (fantasy.LanguageModel, error)
}, error) {
	var opts []anthropic.Option
	opts = append(opts, anthropic.WithAPIKey(apiKey))
	if baseURL != "" {
		opts = append(opts, anthropic.WithBaseURL(baseURL))
	}
	if debugAPI {
		opts = append(opts, anthropic.WithHTTPClient(debugpkg.NewHTTPClient()))
	}
	return anthropic.New(opts...)
}

// CreateOpenAIProvider creates an OpenAI-compatible provider
func CreateOpenAIProvider(apiKey, baseURL string, debugAPI bool) (interface {
	LanguageModel(context.Context, string) (fantasy.LanguageModel, error)
}, error) {
	// Use openaicompat for non-OpenAI URLs (Ollama, LM Studio, DeepSeek, etc.)
	// This enables reasoning/thinking content support
	if !strings.Contains(baseURL, "api.openai.com") {
		var opts []openaicompat.Option
		opts = append(opts, openaicompat.WithAPIKey(apiKey), openaicompat.WithBaseURL(baseURL))
		if debugAPI {
			opts = append(opts, openaicompat.WithHTTPClient(debugpkg.NewHTTPClient()))
		}
		return openaicompat.New(opts...)
	}

	// Use native OpenAI provider for api.openai.com
	var opts []openai.Option
	opts = append(opts, openai.WithAPIKey(apiKey), openai.WithBaseURL(baseURL))
	if debugAPI {
		opts = append(opts, openai.WithHTTPClient(debugpkg.NewHTTPClient()))
	}
	return openai.New(opts...)
}
