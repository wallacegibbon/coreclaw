# CoreClaw

A minimal AI Agent that can handle toolcalling, powered by Large Language Models. It provides a single tool—`bash`—and all capabilities are built on top of it.

CoreClaw supports multiple providers (OpenAI, Anthropic, DeepSeek, ZAI, and any OpenAI-compatible or Anthropic-compatible server like Ollama) via a simple CLI interface.

For this project, simplicity is more important than efficiency.

## Installation

```bash
go install github.com/wallacegibbon/coreclaw@latest
```

Or build from source:

```bash
git clone https://github.com/wallacegibbon/coreclaw.git
cd coreclaw
go build
```

## Usage

All configuration must be specified via command line flags:

```bash
# Local Ollama OpenAI-compatible server
coreclaw --type openai --base-url http://localhost:11434/v1 --api-key xxx --model llama3 "hello"

# Local Ollama Anthropic-compatible server
coreclaw --type anthropic --base-url http://localhost:11434 --api-key=xxx --model gpt-oss:20b "hello"

# MiniMax (Anthropic-compatible)
coreclaw --type anthropic --base-url $MINIMAXI_API_URL --api-key $MINIMAXI_API_KEY --model MiniMax-M2.5 "hello"

# DeepSeek (OpenAI-compatible)
coreclaw --type openai --base-url $DEEPSEEK_API_URL --api-key $DEEPSEEK_API_KEY --model deepseek-chat "hello"

# ZAI (OpenAI-compatible)
coreclaw --type openai --base-url $ZAI_API_URL --api-key $ZAI_API_KEY --model GLM-4.7 "hello"
```

Running with skills
```bash
coreclaw --type anthropic --base-url http://localhost:11434 --api-key=xxx --model gpt-oss:20b --skill ~/playground/coreclaw/misc/samples/skills/
```

## Flags

- `-type string` - Provider type: `anthropic` or `openai` (required)
- `-base-url string` - API endpoint URL (required)
- `-api-key string` - API key (required)
- `-model string` - Model name to use
- `-version` - Show version information
- `-help` - Show help information
- `-debug-api` - Show raw API requests and responses
- `-file string` - Read prompt from file
- `-system string` - Override system prompt
- `-skill string` - Skills directory path

## Features

- Execute bash commands through the AI agent
- Multi-step conversations with tool calls
- Token usage tracking
- Error handling for command execution
- Multi-provider support (OpenAI, Anthropic, DeepSeek, ZAI)
- Single-prompt and interactive modes
- Real-time streaming output
- Color-styled output
- Custom system prompts
- Read prompts from files
- API debug mode for HTTP requests and responses
- Skills system (agentskills.io compatible)

## Project Status

See [STATE.md](STATE.md) for detailed implementation status and architecture.

## License

MIT
