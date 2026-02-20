# CoreClaw Project Status

## Overview
CoreClaw is a minimal AI Agent that can handle toolcalling. We only provide one tool: `bash`.
All skills are based on this only one tool. And all functionalities are built by skills.

For this project, simplicity is more important than efficiency.

## Implementation Status

### Completed
- ✅ Go module initialized (github.com/wallacegibbon/coreclaw)
- ✅ fantasy dependency added (v0.8.0)
- ✅ catwalk dependency added (v0.19.0) for provider database
- ✅ readline dependency added for terminal input handling
- ✅ Basic agent structure with OpenAI provider
- ✅ Bash tool implementation with `fantasy.NewAgentTool`
- ✅ Command-line interface with prompt input
- ✅ Tool calling support with bash command execution
- ✅ Tool result display
- ✅ Usage statistics (input/output/total tokens)
- ✅ Multi-provider support (OpenAI, DeepSeek, ZAI)
- ✅ Color styling with ANSI escape codes
  - AI responses: bold white (#cdd6f4)
  - User prompts: blue (#89b4fa)
  - Debug/info messages: dim gray (#6c7086)
  - Applied to both single-prompt and interactive modes
  - Uses raw ANSI codes for lightweight output without padding
- ✅ CLI flags for version and help information
- ✅ File-based prompts (--file) to read prompts from files
- ✅ Custom system prompts (--system) to override default behavior
- ✅ README.md with comprehensive documentation
- ✅ Readline library integration for proper terminal input handling
  - Automatic TTY detection
  - Command history support (~/.coreclaw_history, max 1000 entries)
  - Proper backspace/delete for all character encodings
  - Ctrl-C interruption support
- ✅ Real-time streaming output
  - All text (including thinking) displays immediately
  - Interleaved with bash command outputs
- ✅ API debug mode (--debug-api)
  - Logs raw API requests and responses to stderr
  - Shows HTTP method, URL, headers (with sensitive data masked), and body
  - Colors request messages in green and response messages in purple
  - Useful for troubleshooting API communication issues
- ✅ Prompt formatting with cyan brackets, green usernames, and consistent background
  - Format: «USER@LOCALHOSTNAME — MODEL@API_URL»
  - Background color applied to bracketed section only
  - Newline before green ❯ prompt character with single space
- ✅ Fixed prompt duplication bug
  - Separated bracketed status line from input prompt
  - Bracketed line prints once at session start
  - Input prompt shows green ❯ with reset for clean input
- ✅ Bash command visibility
  - Commands printed when they start with arrow prefix (→)
  - Command text in green, arrow in dim color
  - Status appended at the end when command finishes
  - Green ✓ for success, red ✗ [exit_code] for errors
  - Newlines and tabs escaped for clean single-line display
  - No carriage return tricks - simple and reliable formatting
- ✅ Debug API filtering of empty SSE content chunks
  - Filters out empty delta content messages during streaming
  - Prevents log spam while maintaining normal SSE behavior
- ✅ Ctrl-C handling in interactive mode
  - Cancels unfinished requests when Ctrl-C is pressed
  - Does nothing (continues waiting) if no request is in progress (at the prompt)
  - Uses context cancellation to stop ongoing API calls
  - Displays "Request cancelled." message when a request is interrupted
  - Properly handles readline.ErrInterrupt to prevent process termination
- ✅ Refactored codebase for better maintainability
  - Extracted CLI flag parsing to `internal/config` package
  - Extracted run logic to `internal/run` package
  - Simplified main.go to ~80 lines of minimal glue code

### Architecture
- **Language Models**:
  - OpenAI GPT-4o (or model from catwalk database)
  - DeepSeek deepseek-chat (reasoning models require special tool call handling)
  - ZAI GLM-4.7
- **Providers**: OpenAI, DeepSeek, ZAI (all using OpenAI-compatible API)
- **Tool**: bash (executes shell commands)
- **Framework**: charm.land/fantasy
- **Provider Database**: charm.land/catwalk (embedded)
- **UI Styling**: Raw ANSI escape codes (lightweight, no padding)

### Code Structure
```
internal/
  agent/       - Agent processor for streaming responses
  config/      - CLI flags and settings parsing
  debug/       - Debug HTTP transport for API debugging
  provider/    - Provider configuration (API keys, endpoints)
  run/         - Runner for single prompt and interactive modes
  terminal/    - Terminal utilities (colors, prompts, readline)
  tools/       - Tool implementations (bash)
main.go       - Entry point, minimal glue code
```

### Features
- Execute bash commands through the AI agent
- Multi-step conversations with tool calls
- Token usage tracking
- Error handling for command execution
- Multi-provider support with automatic provider detection
- Provider selection priority: OPENAI_API_KEY > DEEPSEEK_API_KEY > ZAI_API_KEY
- CLI flags for version and help information
- Read prompts from files for batch processing
- Custom system prompts for specialized behaviors
- Color-coded output for better readability
- Command history for interactive sessions
- Robust terminal input handling with readline (backspace, delete, Ctrl-C)
- Proper conversation history management for multi-turn tool calls

### Usage
```bash
# Set API key (any one of the three)
export OPENAI_API_KEY=your-openai-key
# or
export DEEPSEEK_API_KEY=your-deepseek-key
# or
export ZAI_API_KEY=your-zai-key

# Run with prompt (streaming output enabled by default)
./coreclaw "List files in current directory"

# Run with API debug (shows raw HTTP requests/responses)
./coreclaw --debug-api "List files"

# Run with prompt from file
./coreclaw --file prompt.txt

# Run with custom system prompt
./coreclaw --system "You are a code reviewer" "Review this code"

# Run interactively (no prompt provided)
./coreclaw

# Show version
./coreclaw --version

# Show help
./coreclaw --help
```

### Supported Providers
- **OpenAI** (OPENAI_API_KEY): Uses GPT-4o at https://api.openai.com/v1 (or custom endpoint via OPENAI_API_ENDPOINT, or model from catwalk database)
- **DeepSeek** (DEEPSEEK_API_KEY): Uses deepseek-chat at https://api.deepseek.com/v1 (reasoning models require special tool call handling)
- **ZAI** (ZAI_API_KEY): Uses GLM-4.7 at https://api.z.ai/api/coding/paas/v4

Provider selection priority: OPENAI_API_KEY > DEEPSEEK_API_KEY > ZAI_API_KEY

**Important**: When using `--base-url` for custom or local servers, environment variables are ignored. You must specify `--api-key` along with `--base-url`.

## Next Steps
- Add more sophisticated skills built on bash tool
- Add config file support for persistent settings
- Consider adding conversation history save/restore
