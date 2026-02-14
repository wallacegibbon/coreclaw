package terminal

import (
	"fmt"
	"os"

	"github.com/chzyer/readline"
)

// GetPrompt returns the shell prompt string
func GetPrompt(model string) string {
	username := os.Getenv("USER")
	if username == "" {
		username = os.Getenv("USERNAME")
	}
	if username == "" {
		username = "user"
	}

	bg := "\x1b[48;2;49;50;68m"
	cyanFg := "\x1b[38;2;137;220;235m"
	greenFg := "\x1b[38;2;166;227;161m"
	reset := "\x1b[0m"

	prompt := fmt.Sprintf("%s%s«%s%s@%s%s%s»%s ", bg, cyanFg, username, greenFg, cyanFg, model, cyanFg, reset)
	return prompt
}

// IsTerminal checks if stdin is a terminal
func IsTerminal() bool {
	fileInfo, _ := os.Stdin.Stat()
	return (fileInfo.Mode() & os.ModeCharDevice) != 0
}

// ReadlineInstance creates and configures a readline instance
func ReadlineInstance(model string) (*readline.Instance, error) {
	return readline.NewEx(&readline.Config{
		Prompt:          GetPrompt(model),
		InterruptPrompt: "^C",
		HistoryFile:     os.Getenv("HOME") + "/.coreclaw_history",
		HistoryLimit:    1000,
	})
}
