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

func main() {
	version := "0.1.0"
	showVersion := flag.Bool("version", false, "Show version information")
	showHelp := flag.Bool("help", false, "Show help information")
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

	scanner := bufio.NewScanner(os.Stdin)

	var messages []fantasy.Message


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
		if !*quiet {
			fmt.Fprintln(os.Stderr, dim("\n=== Sending to API ==="))
			fmt.Fprintln(os.Stderr, dim(fmt.Sprintf("System Prompt: %s", finalSystemPrompt)))
			fmt.Fprintln(os.Stderr, blue(fmt.Sprintf("User Prompt: %s", userPrompt)))
			fmt.Fprintln(os.Stderr, dim("Available Tools: bash"))
			fmt.Fprintln(os.Stderr, dim("======================"))
		}

		result, err := agent.Generate(ctx, fantasy.AgentCall{
			Prompt: userPrompt,
		})
		if err != nil {
			fmt.Fprintln(os.Stderr, dim(fmt.Sprintf("Error: %v", err)))
			os.Exit(1)
		}

		for _, step := range result.Steps {
			for _, content := range step.Content {
				switch c := content.(type) {
				case fantasy.TextContent:
					fmt.Print(bright(c.Text))
				case fantasy.ToolCallContent:
					if !*quiet {
						fmt.Fprintln(os.Stderr, dim(fmt.Sprintf("\n[Tool call: %s]", c.ToolName)))
						var input map[string]any
						json.Unmarshal([]byte(c.Input), &input)
						fmt.Fprintln(os.Stderr, dim(fmt.Sprintf("  Input: %+v", input)))
					}
				case fantasy.ToolResultContent:
					if !*quiet {
						fmt.Fprintln(os.Stderr, dim("[Tool result]"))
						switch p := c.Result.(type) {
						case fantasy.ToolResultOutputContentText:
							fmt.Fprintln(os.Stderr, yellow(fmt.Sprintf("  Output: %s", p.Text)))
						case fantasy.ToolResultOutputContentError:
							fmt.Fprintln(os.Stderr, dim(fmt.Sprintf("  Error: %s", p.Error)))
						}
					}
				}
			}
		}

		fmt.Println()
		if !*quiet {
			fmt.Fprintln(os.Stderr, dim(fmt.Sprintf("\nUsage: %d input tokens, %d output tokens, %d total tokens",
				result.TotalUsage.InputTokens,
				result.TotalUsage.OutputTokens,
				result.TotalUsage.TotalTokens,
			)))
		}
		os.Exit(0)
	}

	for {
		fmt.Fprint(os.Stderr, dim("Enter your prompt (Ctrl-C to exit): "))
		if !scanner.Scan() {
			return
		}

		userPrompt := scanner.Text()
		if userPrompt == "" {
			continue
		}

		if !*quiet {
			fmt.Fprintln(os.Stderr, dim("\n=== Sending to API ==="))
			fmt.Fprintln(os.Stderr, dim(fmt.Sprintf("System Prompt: %s", finalSystemPrompt)))
			fmt.Fprintln(os.Stderr, blue(fmt.Sprintf("User Prompt: %s", userPrompt)))
			fmt.Fprintln(os.Stderr, dim("Available Tools: bash"))
			fmt.Fprintln(os.Stderr, dim("======================"))
		}

		result, err := agent.Generate(ctx, fantasy.AgentCall{
			Prompt:   userPrompt,
			Messages: messages,
		})
		if err != nil {
			fmt.Fprintln(os.Stderr, dim(fmt.Sprintf("Error: %v", err)))
			continue
		}

		for _, step := range result.Steps {
			for _, content := range step.Content {
				switch c := content.(type) {
				case fantasy.TextContent:
					fmt.Print(bright(c.Text))
				case fantasy.ToolCallContent:
					if !*quiet {
						fmt.Fprintln(os.Stderr, dim(fmt.Sprintf("\n[Tool call: %s]", c.ToolName)))
						var input map[string]any
						json.Unmarshal([]byte(c.Input), &input)
						fmt.Fprintln(os.Stderr, dim(fmt.Sprintf("  Input: %+v", input)))
					}
				case fantasy.ToolResultContent:
					if !*quiet {
						fmt.Fprintln(os.Stderr, dim("[Tool result]"))
						switch p := c.Result.(type) {
						case fantasy.ToolResultOutputContentText:
							fmt.Fprintln(os.Stderr, yellow(fmt.Sprintf("  Output: %s", p.Text)))
						case fantasy.ToolResultOutputContentError:
							fmt.Fprintln(os.Stderr, dim(fmt.Sprintf("  Error: %s", p.Error)))
						}
					}
				}
			}
		}

		messages = append(messages, fantasy.NewUserMessage(userPrompt))

		var assistantParts []fantasy.MessagePart
		var toolParts []fantasy.MessagePart

		for _, step := range result.Steps {
			for _, content := range step.Content {
				// Convert content to MessagePart by marshaling and unmarshaling
				data, err := json.Marshal(content)
				if err != nil {
					continue
				}
				if part, err := fantasy.UnmarshalMessagePart(data); err == nil {
					switch part.GetType() {
				case fantasy.ContentTypeToolResult:
					toolParts = append(toolParts, part)
				default:
					assistantParts = append(assistantParts, part)
				}
				}
			}
		}

		if len(assistantParts) > 0 {
			messages = append(messages, fantasy.Message{
				Role:    fantasy.MessageRoleAssistant,
				Content: assistantParts,
			})
		}

		if len(toolParts) > 0 {
			messages = append(messages, fantasy.Message{
				Role:    fantasy.MessageRoleTool,
				Content: toolParts,
			})
		}

		fmt.Println()
		if !*quiet {
			fmt.Fprintln(os.Stderr, dim(fmt.Sprintf("\nUsage: %d input tokens, %d output tokens, %d total tokens",
				result.TotalUsage.InputTokens,
				result.TotalUsage.OutputTokens,
				result.TotalUsage.TotalTokens,
			)))
		}
	}
}
