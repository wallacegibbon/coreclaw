package adaptors

// GetPrompt returns the shell prompt string (just the input prompt)
func GetPrompt(baseURL, model string) string {
	return Green("> ")
}
