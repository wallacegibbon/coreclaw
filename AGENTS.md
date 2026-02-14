# CoreClaw

A minimal AI Agent that can handle toolcalling. We only provide one tool: `bash`.
All skills are based on this only one tool. And all functionalities are built by skills.

For this project, simplicity is more important than efficiency.


## Project
- Module: `github.com/wallacegibbon/coreclaw`
- Binary: `coreclaw`
- Dependencies:
  - `charm.land/catwalk` - Provider database
  - `charm.land/fantasy` - Agent framework
  - `github.com/charmbracelet/glamour` - Markdown rendering
  - `github.com/chzyer/readline` - Terminal input handling


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

# Custom system prompt
coreclaw --system "You are a code reviewer" "Review this code"

# Disable markdown rendering (streaming output)
coreclaw --no-markdown "List files"

# Debug mode - show agent internals
coreclaw --debug "list files"

# Debug API mode - show raw API requests and responses
coreclaw --debug-api "list files"

# Use local/openai-compatible server (requires --api-key)
coreclaw --api-key sk-xxx --base-url http://localhost:11434/v1 "hello"

# Specify model with custom server
coreclaw --api-key sk-xxx --base-url http://localhost:11434/v1 --model llama3 "hello"
```


## Environment Variables

CoreClaw requires an API key from one of the following providers:

- `OPENAI_API_KEY` - OpenAI API key (uses GPT-4o or configured model)
- `DEEPSEEK_API_KEY` - DeepSeek API key (uses deepseek-chat)
- `ZAI_API_KEY` - ZAI API key (uses GLM-4.7)

Provider selection priority: OPENAI_API_KEY > DEEPSEEK_API_KEY > ZAI_API_KEY

Provider configurations are loaded from the embedded catwalk database.

**Note**: When using `--base-url` for custom or local servers, environment variables are ignored. You must specify `--api-key` along with `--base-url`.


## CLI Flags

- `-version` - Show version information
- `-help` - Show help information
- `-debug` - Show debug output
- `-debug-api` - Show raw API requests and responses (to stderr)
- `-no-markdown` - Disable markdown rendering
- `-file string` - Read prompt from file
- `-system string` - Override system prompt
- `-api-key string` - API key for the provider (required when using --base-url)
- `-base-url string` - Base URL for the API endpoint (requires --api-key, ignores env vars)
- `-model string` - Model name to use (defaults to provider default)


## Agent Instructions
- **Read STATE.md** at the start of every conversation
- **Update STATE.md** after completing any meaningful work (features, bug fixes, etc.)
- Keep STATE.md as the single source of truth for project status
