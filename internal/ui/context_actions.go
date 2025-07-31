package ui

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"runtime"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

// ContextAction represents an action that can be performed on selected text or elements
type ContextAction struct {
	Name        string
	Description string
	Icon        string
	Pattern     *regexp.Regexp
	Action      func(content string, line int, col int) tea.Cmd
	Priority    int
}

// ContextActionManager manages context-sensitive actions
type ContextActionManager struct {
	actions []ContextAction
	styles  ContextActionStyles
}

// ContextActionStyles holds styling for context actions
type ContextActionStyles struct {
	Menu       lipgloss.Style
	MenuItem   lipgloss.Style
	MenuSelect lipgloss.Style
	Icon       lipgloss.Style
	Shortcut   lipgloss.Style
}

// DefaultContextActionStyles returns default styling for context actions
func DefaultContextActionStyles() ContextActionStyles {
	return ContextActionStyles{
		Menu: lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(lipgloss.Color("62")).
			Background(lipgloss.Color("235")).
			Padding(1),
		MenuItem: lipgloss.NewStyle().
			Padding(0, 1),
		MenuSelect: lipgloss.NewStyle().
			Background(lipgloss.Color("62")).
			Foreground(lipgloss.Color("230")).
			Padding(0, 1),
		Icon: lipgloss.NewStyle().
			Foreground(lipgloss.Color("39")).
			Bold(true),
		Shortcut: lipgloss.NewStyle().
			Foreground(lipgloss.Color("246")),
	}
}

// NewContextActionManager creates a new context action manager
func NewContextActionManager() *ContextActionManager {
	cam := &ContextActionManager{
		actions: make([]ContextAction, 0),
		styles:  DefaultContextActionStyles(),
	}

	cam.registerBuiltinActions()
	return cam
}

// registerBuiltinActions registers built-in context actions
func (cam *ContextActionManager) registerBuiltinActions() {
	actions := []ContextAction{
		{
			Name:        "open_file",
			Description: "Open file",
			Icon:        "📁",
			Pattern:     regexp.MustCompile(`(?:[./])?[\w\-_/]+\.[\w]+`),
			Priority:    10,
			Action: func(content string, line int, col int) tea.Cmd {
				return cam.openFile(content)
			},
		},
		{
			Name:        "open_url",
			Description: "Open URL in browser",
			Icon:        "🌐",
			Pattern:     regexp.MustCompile(`https?://[^\s]+`),
			Priority:    10,
			Action: func(content string, line int, col int) tea.Cmd {
				return cam.openURL(content)
			},
		},
		{
			Name:        "copy_code",
			Description: "Copy code block",
			Icon:        "📋",
			Pattern:     regexp.MustCompile("```[\\s\\S]*?```"),
			Priority:    8,
			Action: func(content string, line int, col int) tea.Cmd {
				return cam.copyCodeBlock(content)
			},
		},
		{
			Name:        "run_code",
			Description: "Run code snippet",
			Icon:        "▶️",
			Pattern:     regexp.MustCompile("```(?:bash|sh|python|py|go|js|javascript)[\\s\\S]*?```"),
			Priority:    9,
			Action: func(content string, line int, col int) tea.Cmd {
				return cam.runCodeBlock(content)
			},
		},
		{
			Name:        "show_error_details",
			Description: "Show error details",
			Icon:        "🔍",
			Pattern:     regexp.MustCompile(`(?i)error|exception|failed|panic|fatal`),
			Priority:    7,
			Action: func(content string, line int, col int) tea.Cmd {
				return cam.showErrorDetails(content, line)
			},
		},
		{
			Name:        "search_docs",
			Description: "Search in documentation",
			Icon:        "📚",
			Pattern:     regexp.MustCompile(`\b[A-Z][a-zA-Z]*[A-Z][a-zA-Z]*\b`), // CamelCase words
			Priority:    5,
			Action: func(content string, line int, col int) tea.Cmd {
				return cam.searchDocumentation(content)
			},
		},
		{
			Name:        "explain_function",
			Description: "Explain function",
			Icon:        "💡",
			Pattern:     regexp.MustCompile(`\b\w+\([^)]*\)`), // Function calls
			Priority:    6,
			Action: func(content string, line int, col int) tea.Cmd {
				return cam.explainFunction(content)
			},
		},
		{
			Name:        "format_json",
			Description: "Format JSON",
			Icon:        "✨",
			Pattern:     regexp.MustCompile(`^\s*[{\[].*[}\]]\s*$`),
			Priority:    4,
			Action: func(content string, line int, col int) tea.Cmd {
				return cam.formatJSON(content)
			},
		},
		{
			Name:        "create_file",
			Description: "Create file from path",
			Icon:        "📄",
			Pattern:     regexp.MustCompile(`(?:[./])?[\w\-_/]+\.[\w]+`),
			Priority:    3,
			Action: func(content string, line int, col int) tea.Cmd {
				return cam.createFile(content)
			},
		},
		{
			Name:        "copy_selection",
			Description: "Copy to clipboard",
			Icon:        "📋",
			Pattern:     regexp.MustCompile(`.+`), // Matches any text
			Priority:    1,
			Action: func(content string, line int, col int) tea.Cmd {
				return cam.copyToClipboard(content)
			},
		},
	}

	for _, action := range actions {
		cam.actions = append(cam.actions, action)
	}
}

// RegisterAction registers a new context action
func (cam *ContextActionManager) RegisterAction(action ContextAction) {
	cam.actions = append(cam.actions, action)
}

// GetActionsForContent returns applicable actions for the given content
func (cam *ContextActionManager) GetActionsForContent(content string, line int, col int) []ContextAction {
	var applicableActions []ContextAction

	for _, action := range cam.actions {
		if action.Pattern.MatchString(content) {
			applicableActions = append(applicableActions, action)
		}
	}

	// Sort by priority (higher priority first)
	for i := 0; i < len(applicableActions)-1; i++ {
		for j := i + 1; j < len(applicableActions); j++ {
			if applicableActions[i].Priority < applicableActions[j].Priority {
				applicableActions[i], applicableActions[j] = applicableActions[j], applicableActions[i]
			}
		}
	}

	return applicableActions
}

// ExecuteAction executes a context action
func (cam *ContextActionManager) ExecuteAction(action ContextAction, content string, line int, col int) tea.Cmd {
	return action.Action(content, line, col)
}

// openFile opens a file in the default editor or file manager
func (cam *ContextActionManager) openFile(filePath string) tea.Cmd {
	return func() tea.Msg {
		// Clean up the file path
		cleanPath := strings.TrimSpace(filePath)

		// Check if file exists
		if _, err := os.Stat(cleanPath); os.IsNotExist(err) {
			return ContextActionResultMsg{
				Success: false,
				Message: fmt.Sprintf("File does not exist: %s", cleanPath),
			}
		}

		// Get absolute path
		absPath, err := filepath.Abs(cleanPath)
		if err != nil {
			return ContextActionResultMsg{
				Success: false,
				Message: fmt.Sprintf("Failed to get absolute path: %v", err),
			}
		}

		// Open file based on OS
		var cmd *exec.Cmd
		switch runtime.GOOS {
		case "darwin":
			cmd = exec.Command("open", absPath)
		case "linux":
			cmd = exec.Command("xdg-open", absPath)
		case "windows":
			cmd = exec.Command("cmd", "/c", "start", absPath)
		default:
			return ContextActionResultMsg{
				Success: false,
				Message: "Unsupported operating system",
			}
		}

		err = cmd.Start()
		if err != nil {
			return ContextActionResultMsg{
				Success: false,
				Message: fmt.Sprintf("Failed to open file: %v", err),
			}
		}

		return ContextActionResultMsg{
			Success: true,
			Message: fmt.Sprintf("Opened file: %s", absPath),
		}
	}
}

// openURL opens a URL in the default browser
func (cam *ContextActionManager) openURL(url string) tea.Cmd {
	return func() tea.Msg {
		var cmd *exec.Cmd
		switch runtime.GOOS {
		case "darwin":
			cmd = exec.Command("open", url)
		case "linux":
			cmd = exec.Command("xdg-open", url)
		case "windows":
			cmd = exec.Command("cmd", "/c", "start", url)
		default:
			return ContextActionResultMsg{
				Success: false,
				Message: "Unsupported operating system",
			}
		}

		err := cmd.Start()
		if err != nil {
			return ContextActionResultMsg{
				Success: false,
				Message: fmt.Sprintf("Failed to open URL: %v", err),
			}
		}

		return ContextActionResultMsg{
			Success: true,
			Message: fmt.Sprintf("Opened URL: %s", url),
		}
	}
}

// copyCodeBlock copies a code block to clipboard
func (cam *ContextActionManager) copyCodeBlock(content string) tea.Cmd {
	return func() tea.Msg {
		// Extract code from markdown code block
		code := content
		if strings.HasPrefix(content, "```") {
			lines := strings.Split(content, "\n")
			if len(lines) > 2 {
				// Remove first and last line (``` markers)
				code = strings.Join(lines[1:len(lines)-1], "\n")
			}
		}

		return cam.copyToClipboardSync(code)
	}
}

// runCodeBlock executes a code block
func (cam *ContextActionManager) runCodeBlock(content string) tea.Cmd {
	return func() tea.Msg {
		// Extract language and code from markdown code block
		lines := strings.Split(content, "\n")
		if len(lines) < 2 {
			return ContextActionResultMsg{
				Success: false,
				Message: "Invalid code block format",
			}
		}

		langLine := strings.TrimPrefix(lines[0], "```")
		lang := strings.TrimSpace(strings.ToLower(langLine))
		code := strings.Join(lines[1:len(lines)-1], "\n")

		// Execute based on language
		var cmd *exec.Cmd
		switch lang {
		case "bash", "sh":
			cmd = exec.Command("bash", "-c", code)
		case "python", "py":
			cmd = exec.Command("python", "-c", code)
		case "go":
			// For Go, we'd need to create a temporary file
			return ContextActionResultMsg{
				Success: false,
				Message: "Go code execution requires file creation (not implemented)",
			}
		case "javascript", "js":
			cmd = exec.Command("node", "-e", code)
		default:
			return ContextActionResultMsg{
				Success: false,
				Message: fmt.Sprintf("Unsupported language: %s", lang),
			}
		}

		output, err := cmd.CombinedOutput()
		if err != nil {
			return ContextActionResultMsg{
				Success: false,
				Message: fmt.Sprintf("Execution failed: %v\nOutput: %s", err, string(output)),
			}
		}

		return ContextActionResultMsg{
			Success: true,
			Message: fmt.Sprintf("Code executed successfully.\nOutput:\n%s", string(output)),
		}
	}
}

// showErrorDetails shows detailed error information
func (cam *ContextActionManager) showErrorDetails(content string, line int) tea.Cmd {
	return func() tea.Msg {
		details := fmt.Sprintf("Error Details:\nLine: %d\nContent: %s\n\nSuggestions:\n", line, content)

		// Add some basic error analysis
		lowerContent := strings.ToLower(content)
		if strings.Contains(lowerContent, "permission denied") {
			details += "- Check file permissions\n- Try running with elevated privileges\n"
		} else if strings.Contains(lowerContent, "not found") {
			details += "- Verify the file or command exists\n- Check PATH environment variable\n"
		} else if strings.Contains(lowerContent, "syntax error") {
			details += "- Review code syntax\n- Check for missing brackets or semicolons\n"
		} else {
			details += "- Check logs for more details\n- Verify input parameters\n"
		}

		return ShowErrorDetailsMsg{
			Line:    line,
			Content: content,
			Details: details,
		}
	}
}

// searchDocumentation searches for documentation
func (cam *ContextActionManager) searchDocumentation(term string) tea.Cmd {
	return func() tea.Msg {
		return SearchDocumentationMsg{
			Term: strings.TrimSpace(term),
		}
	}
}

// explainFunction provides function explanation
func (cam *ContextActionManager) explainFunction(functionCall string) tea.Cmd {
	return func() tea.Msg {
		return ExplainFunctionMsg{
			Function: strings.TrimSpace(functionCall),
		}
	}
}

// formatJSON formats JSON content
func (cam *ContextActionManager) formatJSON(jsonContent string) tea.Cmd {
	return func() tea.Msg {
		// This would typically use a JSON formatting library
		// For now, just return a placeholder
		return FormatJSONMsg{
			Content: strings.TrimSpace(jsonContent),
		}
	}
}

// createFile creates a new file from path
func (cam *ContextActionManager) createFile(filePath string) tea.Cmd {
	return func() tea.Msg {
		cleanPath := strings.TrimSpace(filePath)

		// Check if file already exists
		if _, err := os.Stat(cleanPath); err == nil {
			return ContextActionResultMsg{
				Success: false,
				Message: fmt.Sprintf("File already exists: %s", cleanPath),
			}
		}

		// Create directory if it doesn't exist
		dir := filepath.Dir(cleanPath)
		if err := os.MkdirAll(dir, 0755); err != nil {
			return ContextActionResultMsg{
				Success: false,
				Message: fmt.Sprintf("Failed to create directory: %v", err),
			}
		}

		// Create empty file
		file, err := os.Create(cleanPath)
		if err != nil {
			return ContextActionResultMsg{
				Success: false,
				Message: fmt.Sprintf("Failed to create file: %v", err),
			}
		}
		defer file.Close()

		return ContextActionResultMsg{
			Success: true,
			Message: fmt.Sprintf("Created file: %s", cleanPath),
		}
	}
}

// copyToClipboard copies content to system clipboard
func (cam *ContextActionManager) copyToClipboard(content string) tea.Cmd {
	return func() tea.Msg {
		return cam.copyToClipboardSync(content)
	}
}

// copyToClipboardSync synchronously copies content to clipboard
func (cam *ContextActionManager) copyToClipboardSync(content string) ContextActionResultMsg {
	var cmd *exec.Cmd
	switch runtime.GOOS {
	case "darwin":
		cmd = exec.Command("pbcopy")
	case "linux":
		if _, err := exec.LookPath("xclip"); err == nil {
			cmd = exec.Command("xclip", "-selection", "clipboard")
		} else if _, err := exec.LookPath("xsel"); err == nil {
			cmd = exec.Command("xsel", "--clipboard", "--input")
		} else {
			return ContextActionResultMsg{
				Success: false,
				Message: "No clipboard utility found (install xclip or xsel)",
			}
		}
	case "windows":
		cmd = exec.Command("clip")
	default:
		return ContextActionResultMsg{
			Success: false,
			Message: "Unsupported operating system",
		}
	}

	cmd.Stdin = strings.NewReader(content)
	err := cmd.Run()
	if err != nil {
		return ContextActionResultMsg{
			Success: false,
			Message: fmt.Sprintf("Failed to copy to clipboard: %v", err),
		}
	}

	return ContextActionResultMsg{
		Success: true,
		Message: fmt.Sprintf("Copied %d characters to clipboard", len(content)),
	}
}

// GetStyles returns the context action styles
func (cam *ContextActionManager) GetStyles() ContextActionStyles {
	return cam.styles
}

// SetStyles sets the context action styles
func (cam *ContextActionManager) SetStyles(styles ContextActionStyles) {
	cam.styles = styles
}

// RenderContextMenu renders a context menu for the given actions
func (cam *ContextActionManager) RenderContextMenu(actions []ContextAction, selected int) string {
	if len(actions) == 0 {
		return ""
	}

	var content strings.Builder

	for i, action := range actions {
		var line strings.Builder

		// Icon
		line.WriteString(cam.styles.Icon.Render(action.Icon))
		line.WriteString(" ")

		// Description
		line.WriteString(action.Description)

		// Apply style
		if i == selected {
			content.WriteString(cam.styles.MenuSelect.Render(line.String()))
		} else {
			content.WriteString(cam.styles.MenuItem.Render(line.String()))
		}

		if i < len(actions)-1 {
			content.WriteString("\n")
		}
	}

	return cam.styles.Menu.Render(content.String())
}

// Message types for context actions
type (
	ContextActionResultMsg struct {
		Success bool
		Message string
	}
	ShowErrorDetailsMsg struct {
		Line    int
		Content string
		Details string
	}
	SearchDocumentationMsg struct {
		Term string
	}
	ExplainFunctionMsg struct {
		Function string
	}
	FormatJSONMsg struct {
		Content string
	}
)
