package terminal

import "time"

// ============================================================================
// Color Palette (Catppuccin Mocha)
// ============================================================================

// Color constants for consistent theming across the UI.
// These follow the Catppuccin Mocha color palette.
const (
	// ColorBase is the background color - used for invisible borders
	ColorBase = "#1e1e2e"
	// ColorSurface1 is the surface color - used for subtle backgrounds
	ColorSurface1 = "#585b70"
	// ColorAccent is the primary accent color (blue) - used for focused borders, prompts
	ColorAccent = "#89d4fa"
	// ColorDim is the dimmed color - used for unfocused borders, blurred text
	ColorDim = "#45475a"
	// ColorMuted is the muted color - used for placeholder text, system messages
	ColorMuted = "#6c7086"
	// ColorText is the primary text color (white)
	ColorText = "#cdd6f4"
	// ColorWarning is the warning/accent color (yellow)
	ColorWarning = "#f9e2af"
	// ColorError is the error color (red)
	ColorError = "#f38ba8"
	// ColorSuccess is the success color (green)
	ColorSuccess = "#a6e3a1"
	// ColorPeach is the peach color - used for cursor border highlight
	ColorPeach = "#fab387"
)

// ============================================================================
// Layout Constants
// ============================================================================

const (
	DefaultWidth  = 80
	DefaultHeight = 20

	// Row allocation: input box, status bar, newlines
	InputRows  = 3
	StatusRows = 1
	LayoutGap  = 4 // input + status + newlines between sections

	// Component sizing
	InputPaddingH     = 8  // horizontal padding for input fields (border + padding both sides)
	SelectorMaxHeight = 30 // maximum height for model selector and similar overlays
)

// ============================================================================
// Timing Constants
// ============================================================================

const (
	UpdateThrottleInterval = 100 * time.Millisecond // batch rapid display updates (lower = sooner signal)
	TickInterval           = 250 * time.Millisecond // polling during streaming (lower = smoother refresh)
	FlusherInterval        = 50 * time.Millisecond  // update flusher tick
	SubmitTickDelay        = 50 * time.Millisecond  // delay before first tick after submit
)
