# CoreClaw

A minimal AI Agent that can handle toolcalling. We only provide one tool: `bash`. All skills are based on this only one tool. And all functionalities are built by skills.

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

```bash
# Single prompt
coreclaw "List all files in the current directory"

# Interactive mode
coreclaw

# Read prompt from file
coreclaw --file prompt.txt

# Quiet mode (no debug output)
coreclaw --quiet "echo 'Hello'"

# Custom system prompt
coreclaw --system "You are a code reviewer" "Review this code"
```

## Environment Variables

CoreClaw requires an API key from one of the following providers:

- `OPENAI_API_KEY` - OpenAI API key (uses GPT-4o)
- `DEEPSEEK_API_KEY` - DeepSeek API key (uses deepseek-chat)
- `ZAI_API_KEY` - ZAI API key (uses GLM-4.7)

Provider selection priority: OPENAI_API_KEY > DEEPSEEK_API_KEY > ZAI_API_KEY

## Features

- Execute bash commands through the AI agent
- Multi-step conversations with tool calls
- Token usage tracking
- Error handling for command execution
- Multi-provider support (OpenAI, DeepSeek, ZAI)
- Single-prompt and interactive modes
- Color-styled output
- Markdown rendering with glamour (can be disabled with --no-markdown)
- Custom system prompts
- Read prompts from files
- Quiet mode for scripting
- Debug mode with verbose output

## Flags

- `-version` - Show version information
- `-help` - Show help information
- `-debug` - Show debug output
- `-quiet` - Suppress debug output
- `-no-markdown` - Disable markdown rendering
- `-file string` - Read prompt from file
- `-system string` - Override system prompt

## Project Status

See [STATE.md](STATE.md) for detailed implementation status and architecture.

## License

MIT
