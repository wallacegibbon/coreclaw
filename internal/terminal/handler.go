package terminal

import (
	"fmt"
	"os"

	"github.com/chzyer/readline"
)

// GetPrompt returns the shell prompt string
func GetPrompt(_ string) string {
	username := os.Getenv("USER")
	if username == "" {
		username = os.Getenv("USERNAME")
	}
	if username == "" {
		username = "user"
	}

	prompt := fmt.Sprintf("%s@%s%s",
		Cyan(username),
		Green("coreclaw"),
		Bright("⟩ "),
	)
	return prompt
}

// IsTerminal checks if stdin is a terminal
func IsTerminal() bool {
	fileInfo, _ := os.Stdin.Stat()
	return (fileInfo.Mode() & os.ModeCharDevice) != 0
}

// ReadlineInstance creates and configures a readline instance
func ReadlineInstance() (*readline.Instance, error) {
	return readline.NewEx(&readline.Config{
		Prompt:          GetPrompt(""),
		InterruptPrompt: "^C",
		HistoryFile:     os.Getenv("HOME") + "/.coreclaw_history",
		HistoryLimit:    1000,
	})
}
