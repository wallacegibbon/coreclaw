# CoreClaw

A minimal AI Agent that can handle toolcalling, powered by Large Language Models. It provides multiple tools for file operations and shell execution.

CoreClaw supports all OpenAI-compatible or Anthropic-compatible API servers.

For this project, simplicity is more important than efficiency.

## Installation

```sh
go install github.com/wallacegibbon/coreclaw@latest
go install github.com/wallacegibbon/coreclaw/cmd/coreclaw-web@latest
```

Or build from source:

```sh
git clone https://github.com/wallacegibbon/coreclaw.git
cd coreclaw
go build
go build ./cmd/coreclaw-web/
```

## Usage

All configuration must be specified via command line flags:

```sh
# Local Ollama OpenAI-compatible server
coreclaw --type openai --base-url http://localhost:11434/v1 --api-key xxx --model llama3

# Local Ollama Anthropic-compatible server
coreclaw --type anthropic --base-url http://localhost:11434 --api-key=xxx --model gpt-oss:20b

# MiniMax (Anthropic-compatible)
coreclaw --type anthropic --base-url $MINIMAXI_API_URL --api-key $MINIMAXI_API_KEY --model MiniMax-M2.5

# DeepSeek (OpenAI-compatible)
coreclaw --type openai --base-url $DEEPSEEK_API_URL --api-key $DEEPSEEK_API_KEY --model deepseek-chat

# ZAI (OpenAI-compatible)
coreclaw --type openai --base-url $ZAI_API_URL --api-key $ZAI_API_KEY --model GLM-4.7
```

Running with skills
```sh
coreclaw --type anthropic --base-url http://localhost:11434 --api-key=xxx --model gpt-oss:20b --skill ~/playground/coreclaw/misc/samples/skills/
```

## Web Server

`coreclaw-web` runs a WebSocket server with a built-in chat UI.

```sh
# Start WebSocket server
coreclaw-web --type openai --base-url https://api.openai.com/v1 --api-key $OPENAI_API_KEY --model gpt-4o

# Custom address
coreclaw-web --type anthropic --base-url https://api.anthropic.com --api-key $ANTHROPIC_API_KEY --model claude-sonnet-4-20250514 --addr :9090
```

- **Web UI**: Open `http://localhost:8080` in browser
- **WebSocket**: `ws://localhost:8080/ws`
- Each browser tab gets its own independent agent session

## Flags

- `-type string` - Provider type: `anthropic` or `openai` (required)
- `-base-url string` - API endpoint URL (required)
- `-api-key string` - API key (required)
- `-model string` - Model name to use
- `-version` - Show version information
- `-help` - Show help information
- `-debug-api` - Show raw API requests and responses
- `-system string` - Override system prompt
- `-skill string` - Skills directory path

## Features

- Tools: read_file, write_file, activate_skill, posix_shell
- Multi-step conversations with tool calls
- Token usage tracking
- Error handling for command execution
- Multi-provider support (OpenAI, Anthropic, DeepSeek, ZAI)
- Interactive mode
- Real-time streaming output
- Color-styled output
- Custom system prompts
- Read prompts from files
- API debug mode for HTTP requests and responses
- Skills system (agentskills.io compatible)
- Web server with WebSocket support and chat UI

## Project Status

See [STATE.md](STATE.md) for detailed implementation status and architecture.

## License

MIT
