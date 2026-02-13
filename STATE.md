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
  - Tool outputs: yellow (#f9e2af)
  - Debug/info messages: dim gray (#6c7086)
  - Applied to both single-prompt and interactive modes
  - Uses raw ANSI codes for lightweight output without padding
- ✅ CLI flags for version and help information
- ✅ Quiet mode (--quiet) to suppress debug output
- ✅ File-based prompts (--file) to read prompts from files
- ✅ Custom system prompts (--system) to override default behavior
- ✅ README.md with comprehensive documentation
- ✅ Readline library integration for proper terminal input handling
  - Automatic TTY detection
  - Command history support (~/.coreclaw_history, max 1000 entries)
  - Proper backspace/delete for all character encodings
  - Ctrl-C interruption support
- ✅ Markdown rendering with glamour
  - Renders AI responses with syntax highlighting and formatting
  - Can be disabled with --no-markdown flag for streaming output

### Architecture
- **Language Models**:
  - OpenAI GPT-4o (or model from catwalk database)
  - DeepSeek deepseek-chat (reasoning models require special tool call handling)
  - ZAI GLM-4.7
- **Providers**: OpenAI, DeepSeek, ZAI (all using OpenAI-compatible API)
- **Tool**: bash (executes shell commands)
- **Framework**: charm.land/fantasy
- **Provider Database**: charm.land/catwalk (embedded)
- **Markdown Rendering**: github.com/charmbracelet/glamour
- **UI Styling**: Raw ANSI escape codes (lightweight, no padding)

### Features
- Execute bash commands through the AI agent
- Multi-step conversations with tool calls
- Token usage tracking
- Error handling for command execution
- Multi-provider support with automatic provider detection
- Provider selection priority: OPENAI_API_KEY > DEEPSEEK_API_KEY > ZAI_API_KEY
- CLI flags for version and help information
- Quiet mode to suppress debug output for scripting
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

# Run with prompt (markdown rendering enabled by default)
./coreclaw "List files in current directory"

# Run with quiet mode
./coreclaw --quiet "List files"

# Run without markdown rendering (streaming output)
./coreclaw --no-markdown "List files"

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

## Next Steps
- Explore streaming support for better UX
- Add more sophisticated skills built on bash tool
- Add config file support for persistent settings
- Consider adding conversation history save/restore
