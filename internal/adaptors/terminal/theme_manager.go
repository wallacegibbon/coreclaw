package terminal

// ThemeManager manages theme loading from a themes folder.
// It loads theme files (*.conf) from a specified directory and provides
// theme switching functionality.

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

// ThemeInfo represents a theme's metadata for display in the selector.
type ThemeInfo struct {
	Name string // Theme name (filename without .conf extension)
	Path string // Full path to the theme file
}

// ThemeManager handles theme loading and management.
type ThemeManager struct {
	themesFolder string
	themes       []ThemeInfo
}

// NewThemeManager creates a new theme manager.
// If themesFolder is empty, it defaults to ~/.alayacore/themes.
// If the themes folder doesn't exist, it creates it with default themes.
func NewThemeManager(themesFolder string) *ThemeManager {
	tm := &ThemeManager{
		themesFolder: themesFolder,
	}

	// Set default folder if not provided
	if tm.themesFolder == "" {
		home, err := os.UserHomeDir()
		if err == nil {
			tm.themesFolder = filepath.Join(home, ".alayacore", "themes")
		}
	}

	// Initialize themes folder with default themes if needed
	tm.initializeThemesFolder()

	// Load theme list
	tm.ReloadThemes()

	return tm
}

// initializeThemesFolder creates the themes folder and populates it with default themes
func (tm *ThemeManager) initializeThemesFolder() {
	if tm.themesFolder == "" {
		return
	}

	// Check if folder exists
	if _, err := os.Stat(tm.themesFolder); os.IsNotExist(err) {
		// Create the folder
		if err := os.MkdirAll(tm.themesFolder, 0755); err != nil {
			fmt.Fprintf(os.Stderr, "Warning: failed to create themes folder: %v\n", err)
			return
		}

		// Create default themes
		tm.createDefaultThemes()
	}
}

// createDefaultThemes creates the default theme-dark.conf and theme-light.conf files
func (tm *ThemeManager) createDefaultThemes() {
	// theme-dark.conf - using default Catppuccin Mocha colors
	darkTheme := `# AlayaCore Dark Theme
# Based on Catppuccin Mocha color palette

# Background color - used for invisible borders, separator backgrounds
base: #1e1e2e

# Surface color - used for subtle backgrounds
surface1: #585b70

# Primary accent color (blue) - used for focused borders, prompts, highlights
accent: #89d4fa

# Dimmed color - used for unfocused borders, blurred text
dim: #45475a

# Muted color - used for placeholder text, system messages, tool content
muted: #6c7086

# Primary text color (white)
text: #cdd6f4

# Warning/accent color (yellow) - used for warnings, tool names
warning: #f9e2af

# Error color (red) - used for errors, diff removals
error: #f38ba8

# Success color (green) - used for success indicators, diff additions
success: #a6e3a1

# Peach color - used for window cursor border highlight (when navigating windows with j/k)
peach: #fab387

# Cursor color - text input cursor in the input box
cursor: #cdd6f4

# Diff colors
# Added lines in diff output (green)
diff_add: #a6e3a1

# Removed lines in diff output (red)
diff_remove: #f38ba8
`
	darkPath := filepath.Join(tm.themesFolder, "theme-dark.conf")
	if err := os.WriteFile(darkPath, []byte(darkTheme), 0644); err != nil {
		fmt.Fprintf(os.Stderr, "Warning: failed to create default dark theme: %v\n", err)
	}

	// theme-light.conf - using Catppuccin Latte colors
	lightTheme := `# AlayaCore Light Theme
# Based on Catppuccin Latte color palette
# Optimized for white/light terminal backgrounds

# Background color - use a light gray that blends with light terminal background
base: #e6e6e6

# Surface color - slightly darker for subtle backgrounds
surface1: #ccd0da

# Primary accent color - deep blue for visibility on light backgrounds
accent: #1e66f5

# Dimmed color - medium gray for unfocused borders, blurred text
dim: #acb0be

# Muted color - for placeholder text, system messages
muted: #6c6f85

# Primary text color - dark for readability on light backgrounds
text: #4c4f69

# Warning/accent color - orange for visibility
warning: #df8e1d

# Error color - deep red for errors
error: #d20f39

# Success color - deep green for success indicators
success: #40a02b

# Peach color - cursor border highlight (when navigating windows with j/k)
# Dark maroon/burgundy for maximum visibility on light backgrounds
peach: #881337

# Cursor color - text input cursor in the input box
# Dark color for visibility on light backgrounds
cursor: #1e1e2e

# Diff colors
# Added lines in diff output (deep green)
diff_add: #40a02b

# Removed lines in diff output (deep red)
diff_remove: #d20f39
`
	lightPath := filepath.Join(tm.themesFolder, "theme-light.conf")
	if err := os.WriteFile(lightPath, []byte(lightTheme), 0644); err != nil {
		fmt.Fprintf(os.Stderr, "Warning: failed to create default light theme: %v\n", err)
	}
}

// ReloadThemes reloads the list of available themes from the themes folder.
func (tm *ThemeManager) ReloadThemes() {
	tm.themes = nil

	if tm.themesFolder == "" {
		return
	}

	// Read directory
	entries, err := os.ReadDir(tm.themesFolder)
	if err != nil {
		// Folder doesn't exist or can't be read - that's OK
		return
	}

	// Find all .conf files
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}

		name := entry.Name()
		if !strings.HasSuffix(name, ".conf") {
			continue
		}

		// Strip .conf extension to get theme name
		themeName := strings.TrimSuffix(name, ".conf")

		tm.themes = append(tm.themes, ThemeInfo{
			Name: themeName,
			Path: filepath.Join(tm.themesFolder, name),
		})
	}

	// Sort themes alphabetically
	sort.Slice(tm.themes, func(i, j int) bool {
		return tm.themes[i].Name < tm.themes[j].Name
	})
}

// GetThemes returns the list of available themes.
func (tm *ThemeManager) GetThemes() []ThemeInfo {
	return tm.themes
}

// GetThemesFolder returns the themes folder path.
func (tm *ThemeManager) GetThemesFolder() string {
	return tm.themesFolder
}

// LoadTheme loads a theme by name.
// If the theme doesn't exist or name is empty, returns the default theme.
func (tm *ThemeManager) LoadTheme(name string) *Theme {
	if name == "" {
		return DefaultTheme()
	}

	// Find the theme
	for _, theme := range tm.themes {
		if theme.Name == name {
			loaded, err := LoadTheme(theme.Path)
			if err != nil {
				fmt.Fprintf(os.Stderr, "Warning: failed to load theme %s: %v\n", name, err)
				return DefaultTheme()
			}
			return loaded
		}
	}

	// Theme not found
	fmt.Fprintf(os.Stderr, "Warning: theme %s not found, using default\n", name)
	return DefaultTheme()
}

// ThemeExists checks if a theme with the given name exists.
func (tm *ThemeManager) ThemeExists(name string) bool {
	for _, theme := range tm.themes {
		if theme.Name == name {
			return true
		}
	}
	return false
}
