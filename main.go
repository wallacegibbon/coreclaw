package main

import (
	"bufio"
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"os/exec"
	"strings"

	"charm.land/fantasy"
	"charm.land/fantasy/providers/openai"
	"github.com/chzyer/readline"
)

type BashInput struct {
	Command string `json:"command" description:"The bash command to execute"`
}

type providerConfig struct {
	apiKey    string
	baseURL   string
	modelName string
}

func dim(text string) string {
	return fmt.Sprintf("\x1b[2;38;2;108;112;134m%s\x1b[0m", text)
}

func bright(text string) string {
	return fmt.Sprintf("\x1b[1;38;2;205;214;244m%s\x1b[0m", text)
}

func blue(text string) string {
	return fmt.Sprintf("\x1b[38;2;137;180;250m%s\x1b[0m", text)
}

func yellow(text string) string {
	return fmt.Sprintf("\x1b[38;2;249;226;175m%s\x1b[0m", text)
}

func cyan(text string) string {
	return fmt.Sprintf("\x1b[38;2;137;220;235m%s\x1b[0m", text)
}

func green(text string) string {
	return fmt.Sprintf("\x1b[38;2;166;227;161m%s\x1b[0m", text)
}

func magenta(text string) string {
	return fmt.Sprintf("\x1b[38;2;245;194;231m%s\x1b[0m", text)
}

func getPrompt(_ string) string {
	username := os.Getenv("USER")
	if username == "" {
		username = os.Getenv("USERNAME")
	}
	if username == "" {
		username = "user"
	}

	prompt := fmt.Sprintf("%s@%s%s",
		cyan(username),
		green("coreclaw"),
		bright("⟩ "),
	)
	return prompt
}

func getShortPath(path string) string {
	home := os.Getenv("HOME")
	if home != "" && strings.HasPrefix(path, home) {
		return "~" + path[len(home):]
	}
	return path
}

func isTerminal() bool {
	fileInfo, _ := os.Stdin.Stat()
	return (fileInfo.Mode() & os.ModeCharDevice) != 0
}

func main() {
	version := "0.1.0"
	showVersion := flag.Bool("version", false, "Show version information")
	showHelp := flag.Bool("help", false, "Show help information")
	debug := flag.Bool("debug", false, "Show debug output")
	quiet := flag.Bool("quiet", false, "Suppress debug output")
	promptFile := flag.String("file", "", "Read prompt from file")
	systemPrompt := flag.String("system", "", "Override system prompt")
	flag.Parse()

	if *showVersion {
		fmt.Printf("coreclaw version %s\n", version)
		os.Exit(0)
	}

	if *showHelp {
		fmt.Printf("CoreClaw - A minimal AI Agent with bash tool access\n\n")
		fmt.Printf("Usage:\n  coreclaw [prompt]    Execute a single prompt\n  coreclaw             Run in interactive mode\n\n")
		fmt.Printf("Environment Variables:\n")
		fmt.Printf("  OPENAI_API_KEY      OpenAI API key (uses GPT-4o)\n")
		fmt.Printf("  DEEPSEEK_API_KEY    DeepSeek API key (uses deepseek-chat)\n")
		fmt.Printf("  ZAI_API_KEY         ZAI API key (uses GPT-4o)\n\n")
		fmt.Printf("Flags:\n")
		flag.PrintDefaults()
		fmt.Printf("\nExamples:\n")
		fmt.Printf("  coreclaw                    Run in interactive mode\n")
		fmt.Printf("  coreclaw \"list files\"        Execute a single prompt\n")
		fmt.Printf("  coreclaw --debug \"list files\" Execute with debug output\n")
		fmt.Printf("  coreclaw --quiet \"list files\" Execute without debug output\n")
		os.Exit(0)
	}

	// Determine the final system prompt
	finalSystemPrompt := "You are a helpful AI assistant with access to a bash shell. Use bash tool to execute commands when needed. Be precise and careful with commands."
	if *systemPrompt != "" {
		finalSystemPrompt = *systemPrompt
	}
	openAIKey := os.Getenv("OPENAI_API_KEY")
	deepSeekKey := os.Getenv("DEEPSEEK_API_KEY")
	zaiKey := os.Getenv("ZAI_API_KEY")

	var config providerConfig

	if openAIKey != "" {
		config = providerConfig{
			apiKey:    openAIKey,
			baseURL:   "",
			modelName: "gpt-4o",
		}
	} else if deepSeekKey != "" {
		config = providerConfig{
			apiKey:    deepSeekKey,
			baseURL:   "https://api.deepseek.com/v1",
			modelName: "deepseek-chat",
		}
	} else if zaiKey != "" {
		config = providerConfig{
			apiKey:    zaiKey,
			baseURL:   "https://api.zai.ai/v1",
			modelName: "gpt-4o",
		}
	} else {
		fmt.Fprintln(os.Stderr, "One of OPENAI_API_KEY, DEEPSEEK_API_KEY, or ZAI_API_KEY environment variables is required")
		os.Exit(1)
	}

	opts := []openai.Option{openai.WithAPIKey(config.apiKey)}
	if config.baseURL != "" {
		opts = append(opts, openai.WithBaseURL(config.baseURL))
	}

	provider, err := openai.New(opts...)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to create provider: %v\n", err)
		os.Exit(1)
	}

	model, err := provider.LanguageModel(context.Background(), config.modelName)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to create language model: %v\n", err)
		os.Exit(1)
	}

	if *debug && !*quiet {
		fmt.Fprintln(os.Stderr, dim(fmt.Sprintf("Using model: %s", config.modelName)))
	}

	bashTool := fantasy.NewAgentTool(
		"bash",
		"Execute a bash command in the shell",
		func(ctx context.Context, input BashInput, _ fantasy.ToolCall) (fantasy.ToolResponse, error) {
			cmd := input.Command
			if cmd == "" {
				return fantasy.NewTextErrorResponse("command is required"), nil
			}

			execCmd := exec.CommandContext(ctx, "bash", "-c", cmd)
			output, err := execCmd.CombinedOutput()
			if err != nil {
				return fantasy.NewTextErrorResponse(string(output)), nil
			}

			return fantasy.NewTextResponse(string(output)), nil
		},
	)

	agent := fantasy.NewAgent(
		model,
		fantasy.WithTools(bashTool),
		fantasy.WithSystemPrompt(finalSystemPrompt),
	)

	ctx := context.Background()

	var messages []fantasy.Message

	processPrompt := func(prompt string, includeMessages bool) (*fantasy.AgentResult, string) {
		if *debug && !*quiet {
			fmt.Fprintln(os.Stderr, dim("\n>>> Sending request to API server"))
			fmt.Fprintln(os.Stderr, dim(fmt.Sprintf("System Prompt: %s", finalSystemPrompt)))
			fmt.Fprintln(os.Stderr, blue(fmt.Sprintf("User Prompt: %s", prompt)))
			fmt.Fprintln(os.Stderr, dim("Available Tools: bash"))
		}

		streamCall := fantasy.AgentStreamCall{
			Prompt: prompt,
		}
		if includeMessages {
			streamCall.Messages = messages
		}

		var responseText strings.Builder

		streamCall.OnStepFinish = func(stepResult fantasy.StepResult) error {
			fmt.Println()
			if *debug && !*quiet {
				fmt.Fprintln(os.Stderr, dim("<<< Step finished"))
			}
			return nil
		}
		streamCall.OnToolInputStart = func(id, toolName string) error {
			fmt.Println()
			if *debug && !*quiet {
				fmt.Fprintln(os.Stderr, dim(fmt.Sprintf(">>> Tool invocation request: %s", toolName)))
			}
			return nil
		}

		if *debug && !*quiet {
			streamCall.OnAgentStart = func() {
				fmt.Fprintln(os.Stderr, dim(">>> Agent started"))
			}
			streamCall.OnStepStart = func(stepNumber int) error {
				fmt.Fprintln(os.Stderr, dim(fmt.Sprintf(">>> Step %d started", stepNumber)))
				return nil
			}
			streamCall.OnToolCall = func(toolCall fantasy.ToolCallContent) error {
				var input map[string]any
				json.Unmarshal([]byte(toolCall.Input), &input)
				fmt.Fprintln(os.Stderr, dim(fmt.Sprintf("  Input: %+v", input)))
				return nil
			}
			streamCall.OnToolResult = func(result fantasy.ToolResultContent) error {
				fmt.Fprintln(os.Stderr, dim("<<< Tool result received"))
				switch p := result.Result.(type) {
				case fantasy.ToolResultOutputContentText:
					fmt.Fprintln(os.Stderr, yellow(fmt.Sprintf("  Output: %s", p.Text)))
				case fantasy.ToolResultOutputContentError:
					fmt.Fprintln(os.Stderr, dim(fmt.Sprintf("  Error: %s", p.Error)))
				}
				return nil
			}
		}

		streamCall.OnTextDelta = func(id, text string) error {
			fmt.Print(bright(text))
			responseText.WriteString(text)
			return nil
		}

		agentResult, err := agent.Stream(ctx, streamCall)
		if err != nil {
			fmt.Fprintln(os.Stderr, dim(fmt.Sprintf("Error: %v", err)))
			return nil, ""
		}

		if *debug && !*quiet {
			fmt.Println()
			fmt.Fprintln(os.Stderr, dim("<<< Agent finished"))
		}

		fmt.Println()
		if *debug && !*quiet {
			fmt.Fprintln(os.Stderr, dim(fmt.Sprintf("\nUsage: %d input tokens, %d output tokens, %d total tokens",
				agentResult.TotalUsage.InputTokens,
				agentResult.TotalUsage.OutputTokens,
				agentResult.TotalUsage.TotalTokens,
			)))
		}

		return agentResult, responseText.String()
	}

	var userPrompt string
	if *promptFile != "" {
		content, err := os.ReadFile(*promptFile)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Failed to read prompt file: %v\n", err)
			os.Exit(1)
		}
		userPrompt = strings.TrimSpace(string(content))
	} else if flag.NArg() > 0 {
		userPrompt = strings.Join(flag.Args(), " ")
	}

	if userPrompt != "" {
		result, _ := processPrompt(userPrompt, false)
		if result == nil {
			os.Exit(1)
		}
		os.Exit(0)
	}

	isTTY := isTerminal()

	var rl *readline.Instance
	if isTTY {
		var err error
		rl, err = readline.NewEx(&readline.Config{
			Prompt:          getPrompt(""),
			InterruptPrompt: "^C",
			HistoryFile:     os.Getenv("HOME") + "/.coreclaw_history",
			HistoryLimit:    1000,
		})
		if err != nil {
			fmt.Fprintf(os.Stderr, "Failed to initialize readline: %v\n", err)
			os.Exit(1)
		}
		defer rl.Close()
	}

	for {
		var userPrompt string
		var err error

		if isTTY {
			rl.SetPrompt(getPrompt(""))
			userPrompt, err = rl.Readline()
			if err != nil {
				if err == readline.ErrInterrupt {
					continue
				}
				return
			}
			userPrompt = strings.TrimSpace(userPrompt)
		} else {
			fmt.Fprint(os.Stderr, getPrompt(""))
			reader := bufio.NewReader(os.Stdin)
			input, _ := reader.ReadString('\n')
			userPrompt = strings.TrimSpace(input)
			if userPrompt == "" {
				return
			}
		}

		if userPrompt == "" {
			continue
		}

		result, responseText := processPrompt(userPrompt, true)
		if result == nil {
			continue
		}

		messages = append(messages, fantasy.NewUserMessage(userPrompt))

		if responseText != "" {
			messages = append(messages, fantasy.Message{
				Role:    fantasy.MessageRoleAssistant,
				Content: []fantasy.MessagePart{fantasy.TextPart{Text: responseText}},
			})
		}
	}
}
