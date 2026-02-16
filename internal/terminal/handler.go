package terminal

import (
	"net/url"
	"os"
	"strings"

	"github.com/chzyer/readline"
)

// extractHost extracts hostname and path from base URL for display
func extractHost(baseURL string) string {
	if baseURL == "" {
		return "unknown"
	}

	u, err := url.Parse(baseURL)
	if err != nil {
		// If parsing fails, return the URL as-is
		return baseURL
	}

	host := u.Hostname()
	if u.Port() != "" {
		host += ":" + u.Port()
	}

	if host == "" {
		// Fallback to the base URL if no host found
		return baseURL
	}

	result := host
	if u.Path != "" {
		result += u.Path
	}

	return result
}

// GetBracketedLine returns the bracketed status line with colors
func GetBracketedLine(baseURL, model string) string {
	username := os.Getenv("USER")
	if username == "" {
		username = os.Getenv("USERNAME")
	}
	if username == "" {
		username = "user"
	}

	// Get local hostname
	localHostname, err := os.Hostname()
	if err != nil {
		localHostname = "localhost"
	}

	apiURL := extractHost(baseURL)

	bg := "\x1b[48;2;49;50;68m"
	cyanFg := "\x1b[38;2;137;220;235m"
	greenFg := "\x1b[38;2;166;227;161m"
	reset := "\x1b[0m"

	// Build bracketed part with background
	var bracketed strings.Builder
	bracketed.WriteString(bg)
	bracketed.WriteString(cyanFg)
	bracketed.WriteString("«")
	bracketed.WriteString(greenFg)
	bracketed.WriteString(username)
	bracketed.WriteString(cyanFg)
	bracketed.WriteString("@")
	bracketed.WriteString(localHostname)
	bracketed.WriteString(" — ")
	bracketed.WriteString(greenFg)
	bracketed.WriteString(model)
	bracketed.WriteString(cyanFg)
	bracketed.WriteString("@")
	bracketed.WriteString(apiURL)
	bracketed.WriteString("»")
	bracketed.WriteString(reset)
	bracketed.WriteString("\n")

	return bracketed.String()
}

// GetPrompt returns the shell prompt string (just the input prompt)
func GetPrompt(baseURL, model string) string {
	return Green("❯ ")
}

// IsTerminal checks if stdin is a terminal
func IsTerminal() bool {
	fileInfo, _ := os.Stdin.Stat()
	return (fileInfo.Mode() & os.ModeCharDevice) != 0
}

// ReadlineInstance creates and configures a readline instance
func ReadlineInstance(baseURL, model string) (*readline.Instance, error) {
	return readline.NewEx(&readline.Config{
		Prompt:          GetPrompt(baseURL, model),
		InterruptPrompt: "",
		HistoryFile:     os.Getenv("HOME") + "/.coreclaw_history",
		HistoryLimit:    1000,
	})
}
