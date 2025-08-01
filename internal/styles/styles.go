package styles

import (
	"github.com/charmbracelet/lipgloss"
	"github.com/muesli/termenv"
)

// Theme represents a UI theme
type Theme interface {
	GetStyles() Styles
	GetName() string
	GetColors() ColorScheme
}

// ColorScheme defines the color palette for a theme
type ColorScheme struct {
	// Primary colors
	Primary   lipgloss.Color
	Secondary lipgloss.Color
	Accent    lipgloss.Color

	// Status colors
	Success lipgloss.Color
	Warning lipgloss.Color
	Error   lipgloss.Color
	Info    lipgloss.Color

	// Text colors
	Foreground lipgloss.Color
	Background lipgloss.Color
	Muted      lipgloss.Color

	// UI element colors
	Border    lipgloss.Color
	Highlight lipgloss.Color
	Selection lipgloss.Color
}

// Styles contains all the lipgloss styles for the application
type Styles struct {
	// Color scheme
	Colors ColorScheme

	// Layout styles
	Container lipgloss.Style
	Header    lipgloss.Style
	Footer    lipgloss.Style
	Sidebar   lipgloss.Style

	// Component styles
	ChatMessage   lipgloss.Style
	UserMessage   lipgloss.Style
	AIMessage     lipgloss.Style
	SystemMessage lipgloss.Style
	ErrorMessage  lipgloss.Style

	// Input styles
	UserInput    lipgloss.Style
	InputPrompt  lipgloss.Style
	InputText    lipgloss.Style
	InputFocused lipgloss.Style

	// Status styles
	StatusBar     lipgloss.Style
	StatusActive  lipgloss.Style
	StatusError   lipgloss.Style
	StatusLoading lipgloss.Style

	// Border styles
	Border        lipgloss.Style
	BorderActive  lipgloss.Style
	BorderFocused lipgloss.Style

	// Text styles
	Bold      lipgloss.Style
	Italic    lipgloss.Style
	Code      lipgloss.Style
	Quote     lipgloss.Style
	Link      lipgloss.Style
	Muted     lipgloss.Style
	Primary   lipgloss.Style
	Highlight lipgloss.Style

	// UI element styles
	Button       lipgloss.Style
	ButtonActive lipgloss.Style
	Progress     lipgloss.Style
	Spinner      lipgloss.Style

	// Help styles
	HelpKey    lipgloss.Style
	HelpDesc   lipgloss.Style
	HelpHeader lipgloss.Style
}

// DefaultTheme implements the default theme
type DefaultTheme struct {
	name string
}

// DarkTheme implements a dark theme
type DarkTheme struct {
	name string
}

// LightTheme implements a light theme
type LightTheme struct {
	name string
}

// Theme instances
var (
	defaultTheme = &DefaultTheme{name: "default"}
	darkTheme    = &DarkTheme{name: "dark"}
	lightTheme   = &LightTheme{name: "light"}
)

// GetTheme returns a theme by name
func GetTheme(name string) Theme {
	switch name {
	case "dark":
		return darkTheme
	case "light":
		return lightTheme
	default:
		return defaultTheme
	}
}

// GetAvailableThemes returns all available themes
func GetAvailableThemes() []string {
	return []string{"default", "dark", "light"}
}

// Default theme implementation
func (t *DefaultTheme) GetName() string {
	return t.name
}

func (t *DefaultTheme) GetColors() ColorScheme {
	return ColorScheme{
		Primary:    lipgloss.Color("#007ACC"),
		Secondary:  lipgloss.Color("#6C7B7F"),
		Accent:     lipgloss.Color("#FF6B6B"),
		Success:    lipgloss.Color("#4CAF50"),
		Warning:    lipgloss.Color("#FF9800"),
		Error:      lipgloss.Color("#F44336"),
		Info:       lipgloss.Color("#2196F3"),
		Foreground: lipgloss.Color("#FFFFFF"),
		Background: lipgloss.Color("#1A1A1A"),
		Muted:      lipgloss.Color("#888888"),
		Border:     lipgloss.Color("#444444"),
		Highlight:  lipgloss.Color("#FFD700"),
		Selection:  lipgloss.Color("#264F78"),
	}
}

func (t *DefaultTheme) GetStyles() Styles {
	colors := t.GetColors()

	return Styles{
		Colors: colors,

		// Layout styles
		Container: lipgloss.NewStyle().
			Padding(1).
			Background(colors.Background).
			Foreground(colors.Foreground),

		Header: lipgloss.NewStyle().
			Bold(true).
			Foreground(colors.Primary).
			BorderStyle(lipgloss.NormalBorder()).
			BorderBottom(true).
			BorderForeground(colors.Border).
			Padding(0, 1),

		Footer: lipgloss.NewStyle().
			Foreground(colors.Muted).
			BorderStyle(lipgloss.NormalBorder()).
			BorderTop(true).
			BorderForeground(colors.Border).
			Padding(0, 1),

		// Message styles
		ChatMessage: lipgloss.NewStyle().
			Padding(0, 1).
			Margin(0, 0, 1, 0),

		UserMessage: lipgloss.NewStyle().
			Foreground(colors.Accent).
			Bold(true).
			Padding(0, 1).
			Margin(0, 0, 1, 0),

		AIMessage: lipgloss.NewStyle().
			Foreground(colors.Primary).
			Padding(0, 1).
			Margin(0, 0, 1, 0),

		SystemMessage: lipgloss.NewStyle().
			Foreground(colors.Muted).
			Italic(true).
			Padding(0, 1).
			Margin(0, 0, 1, 0),

		ErrorMessage: lipgloss.NewStyle().
			Foreground(colors.Error).
			Bold(true).
			Padding(0, 1).
			Margin(0, 0, 1, 0),

		// Input styles
		UserInput: lipgloss.NewStyle().
			Foreground(colors.Foreground).
			BorderStyle(lipgloss.RoundedBorder()).
			BorderForeground(colors.Border).
			Padding(0, 1),

		InputPrompt: lipgloss.NewStyle().
			Foreground(colors.Primary).
			Bold(true),

		InputText: lipgloss.NewStyle().
			Foreground(colors.Foreground),

		InputFocused: lipgloss.NewStyle().
			BorderStyle(lipgloss.RoundedBorder()).
			BorderForeground(colors.Primary).
			Padding(0, 1),

		// Status styles
		StatusBar: lipgloss.NewStyle().
			Foreground(colors.Muted).
			Background(colors.Background).
			Padding(0, 1),

		StatusActive: lipgloss.NewStyle().
			Foreground(colors.Success).
			Bold(true),

		StatusError: lipgloss.NewStyle().
			Foreground(colors.Error).
			Bold(true),

		StatusLoading: lipgloss.NewStyle().
			Foreground(colors.Warning),

		// Border styles
		Border: lipgloss.NewStyle().
			BorderStyle(lipgloss.NormalBorder()).
			BorderForeground(colors.Border),

		BorderActive: lipgloss.NewStyle().
			BorderStyle(lipgloss.RoundedBorder()).
			BorderForeground(colors.Primary),

		BorderFocused: lipgloss.NewStyle().
			BorderStyle(lipgloss.ThickBorder()).
			BorderForeground(colors.Accent),

		// Text styles
		Bold: lipgloss.NewStyle().
			Bold(true),

		Italic: lipgloss.NewStyle().
			Italic(true),

		Code: lipgloss.NewStyle().
			Foreground(colors.Info).
			Background(lipgloss.Color("#2A2A2A")).
			Padding(0, 1),

		Quote: lipgloss.NewStyle().
			Italic(true).
			Foreground(colors.Muted).
			BorderLeft(true).
			BorderStyle(lipgloss.NormalBorder()).
			BorderForeground(colors.Border).
			PaddingLeft(1),

		Link: lipgloss.NewStyle().
			Foreground(colors.Info).
			Underline(true),

		Muted: lipgloss.NewStyle().
			Foreground(colors.Muted),

		Primary: lipgloss.NewStyle().
			Foreground(colors.Primary),

		Highlight: lipgloss.NewStyle().
			Background(colors.Highlight).
			Foreground(colors.Background),

		// UI element styles
		Button: lipgloss.NewStyle().
			Foreground(colors.Background).
			Background(colors.Primary).
			Padding(0, 2).
			BorderStyle(lipgloss.RoundedBorder()),

		ButtonActive: lipgloss.NewStyle().
			Foreground(colors.Background).
			Background(colors.Accent).
			Padding(0, 2).
			BorderStyle(lipgloss.RoundedBorder()).
			Bold(true),

		Progress: lipgloss.NewStyle().
			Background(colors.Border).
			Foreground(colors.Primary),

		Spinner: lipgloss.NewStyle().
			Foreground(colors.Primary),

		// Help styles
		HelpKey: lipgloss.NewStyle().
			Foreground(colors.Primary).
			Bold(true),

		HelpDesc: lipgloss.NewStyle().
			Foreground(colors.Foreground),

		HelpHeader: lipgloss.NewStyle().
			Foreground(colors.Accent).
			Bold(true).
			Underline(true),
	}
}

// Dark theme implementation
func (t *DarkTheme) GetName() string {
	return t.name
}

func (t *DarkTheme) GetColors() ColorScheme {
	return ColorScheme{
		Primary:    lipgloss.Color("#61DAFB"),
		Secondary:  lipgloss.Color("#8B949E"),
		Accent:     lipgloss.Color("#F78166"),
		Success:    lipgloss.Color("#56D364"),
		Warning:    lipgloss.Color("#E3B341"),
		Error:      lipgloss.Color("#F85149"),
		Info:       lipgloss.Color("#58A6FF"),
		Foreground: lipgloss.Color("#F0F6FC"),
		Background: lipgloss.Color("#0D1117"),
		Muted:      lipgloss.Color("#8B949E"),
		Border:     lipgloss.Color("#30363D"),
		Highlight:  lipgloss.Color("#FBD834"),
		Selection:  lipgloss.Color("#1F6FEB"),
	}
}

func (t *DarkTheme) GetStyles() Styles {
	// Use similar structure to default theme but with dark colors
	colors := t.GetColors()
	styles := defaultTheme.GetStyles()
	styles.Colors = colors

	// Update styles with new colors
	styles = updateStylesWithColors(styles, colors)

	return styles
}

// Light theme implementation
func (t *LightTheme) GetName() string {
	return t.name
}

func (t *LightTheme) GetColors() ColorScheme {
	return ColorScheme{
		Primary:    lipgloss.Color("#0969DA"),
		Secondary:  lipgloss.Color("#656D76"),
		Accent:     lipgloss.Color("#CF222E"),
		Success:    lipgloss.Color("#1A7F37"),
		Warning:    lipgloss.Color("#9A6700"),
		Error:      lipgloss.Color("#D1242F"),
		Info:       lipgloss.Color("#0969DA"),
		Foreground: lipgloss.Color("#24292F"),
		Background: lipgloss.Color("#FFFFFF"),
		Muted:      lipgloss.Color("#656D76"),
		Border:     lipgloss.Color("#D0D7DE"),
		Highlight:  lipgloss.Color("#FFF8C5"),
		Selection:  lipgloss.Color("#B6E3FF"),
	}
}

func (t *LightTheme) GetStyles() Styles {
	colors := t.GetColors()
	styles := defaultTheme.GetStyles()
	styles.Colors = colors

	// Update styles with new colors
	styles = updateStylesWithColors(styles, colors)

	return styles
}

// updateStylesWithColors updates all styles with new colors
func updateStylesWithColors(styles Styles, colors ColorScheme) Styles {
	// Update all styles that reference colors
	styles.Container = styles.Container.
		Background(colors.Background).
		Foreground(colors.Foreground)

	styles.Header = styles.Header.
		Foreground(colors.Primary).
		BorderForeground(colors.Border)

	styles.Footer = styles.Footer.
		Foreground(colors.Muted).
		BorderForeground(colors.Border)

	styles.UserMessage = styles.UserMessage.
		Foreground(colors.Accent)

	styles.AIMessage = styles.AIMessage.
		Foreground(colors.Primary)

	styles.SystemMessage = styles.SystemMessage.
		Foreground(colors.Muted)

	styles.ErrorMessage = styles.ErrorMessage.
		Foreground(colors.Error)

	// Continue updating other styles...

	return styles
}

// GetResponsiveStyles returns styles adjusted for screen size
func GetResponsiveStyles(theme Theme, width, height int) Styles {
	styles := theme.GetStyles()

	// Adjust padding based on screen size
	if width < 80 {
		// Small screen adjustments
		styles.Container = styles.Container.Padding(0)
		styles.Header = styles.Header.Padding(0)
		styles.Footer = styles.Footer.Padding(0)
	} else if width > 120 {
		// Large screen adjustments
		styles.Container = styles.Container.Padding(2)
		styles.Header = styles.Header.Padding(1, 2)
		styles.Footer = styles.Footer.Padding(1, 2)
	}

	return styles
}

// DetectColorProfile detects the terminal's color capabilities
func DetectColorProfile() termenv.Profile {
	return termenv.ColorProfile()
}

// AdaptColorsToProfile adapts colors to the terminal's capabilities
func AdaptColorsToProfile(colors ColorScheme, profile termenv.Profile) ColorScheme {
	// Convert colors based on terminal capabilities
	switch profile {
	case termenv.Ascii:
		// No colors supported, use basic styling
		return ColorScheme{
			Primary:    lipgloss.Color(""),
			Secondary:  lipgloss.Color(""),
			Accent:     lipgloss.Color(""),
			Success:    lipgloss.Color(""),
			Warning:    lipgloss.Color(""),
			Error:      lipgloss.Color(""),
			Info:       lipgloss.Color(""),
			Foreground: lipgloss.Color(""),
			Background: lipgloss.Color(""),
			Muted:      lipgloss.Color(""),
			Border:     lipgloss.Color(""),
			Highlight:  lipgloss.Color(""),
			Selection:  lipgloss.Color(""),
		}
	case termenv.ANSI:
		// Basic 16 colors
		return ColorScheme{
			Primary:    lipgloss.Color("12"), // Bright Blue
			Secondary:  lipgloss.Color("8"),  // Gray
			Accent:     lipgloss.Color("9"),  // Bright Red
			Success:    lipgloss.Color("10"), // Bright Green
			Warning:    lipgloss.Color("11"), // Bright Yellow
			Error:      lipgloss.Color("1"),  // Red
			Info:       lipgloss.Color("14"), // Bright Cyan
			Foreground: lipgloss.Color("15"), // White
			Background: lipgloss.Color("0"),  // Black
			Muted:      lipgloss.Color("8"),  // Gray
			Border:     lipgloss.Color("8"),  // Gray
			Highlight:  lipgloss.Color("11"), // Bright Yellow
			Selection:  lipgloss.Color("4"),  // Blue
		}
	default:
		// Full color support, return as-is
		return colors
	}
}