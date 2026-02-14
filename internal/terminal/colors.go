package terminal

import "fmt"

// Dim returns text in dim gray color
func Dim(text string) string {
	return fmt.Sprintf("\x1b[2;38;2;108;112;134m%s\x1b[0m", text)
}

// Bright returns text in bright white color
func Bright(text string) string {
	return fmt.Sprintf("\x1b[1;38;2;205;214;244m%s\x1b[0m", text)
}

// Blue returns text in blue color
func Blue(text string) string {
	return fmt.Sprintf("\x1b[38;2;137;180;250m%s\x1b[0m", text)
}

// Yellow returns text in yellow color
func Yellow(text string) string {
	return fmt.Sprintf("\x1b[38;2;249;226;175m%s\x1b[0m", text)
}

// Cyan returns text in cyan color
func Cyan(text string) string {
	return fmt.Sprintf("\x1b[38;2;137;220;235m%s\x1b[0m", text)
}

// Green returns text in green color
func Green(text string) string {
	return fmt.Sprintf("\x1b[38;2;166;227;161m%s\x1b[0m", text)
}

// BgDark returns text with dark blue background
func BgDark(text string) string {
	return fmt.Sprintf("\x1b[48;2;49;50;68m%s\x1b[0m", text)
}
