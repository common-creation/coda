package views

import (
	"fmt"
	"path/filepath"
	"strings"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/charmbracelet/log"

	"github.com/common-creation/coda/internal/styles"
)

// StatusView displays application status information
type StatusView struct {
	// Core components
	styles styles.Styles
	logger *log.Logger

	// Status information
	mode         string
	aiModel      string
	provider     string
	tokenCount   int
	totalTokens  int
	sessionID    string
	workingDir   string
	connected    bool
	lastActivity time.Time
	error        error

	// Display properties
	width       int
	height      int
	showDetails bool
	compact     bool

	// Progress and notifications
	loading      bool
	loadingText  string
	progress     float64
	notification string
	notifyExpiry time.Time

	// Performance metrics
	responseTime time.Duration
	uptime       time.Duration
	startTime    time.Time
}

// StatusInfo contains status information
type StatusInfo struct {
	Mode         string
	AIModel      string
	Provider     string
	TokenCount   int
	TotalTokens  int
	SessionID    string
	WorkingDir   string
	Connected    bool
	Loading      bool
	LoadingText  string
	Progress     float64
	Error        error
	ResponseTime time.Duration
}

// NewStatusView creates a new status view
func NewStatusView(width, height int, styles styles.Styles, logger *log.Logger) *StatusView {
	return &StatusView{
		styles:     styles,
		logger:     logger,
		width:      width,
		height:     height,
		mode:       "Chat",
		aiModel:    "Unknown",
		provider:   "Unknown",
		workingDir: getWorkingDir(),
		connected:  false,
		startTime:  time.Now(),
		compact:    width < 100, // Use compact mode for narrow screens
	}
}

// SetSize updates the status view dimensions
func (sv *StatusView) SetSize(width, height int) {
	sv.width = width
	sv.height = height
	sv.compact = width < 100
}

// UpdateStatus updates the status information
func (sv *StatusView) UpdateStatus(info StatusInfo) {
	sv.mode = info.Mode
	sv.aiModel = info.AIModel
	sv.provider = info.Provider
	sv.tokenCount = info.TokenCount
	sv.totalTokens = info.TotalTokens
	sv.sessionID = info.SessionID
	sv.connected = info.Connected
	sv.loading = info.Loading
	sv.loadingText = info.LoadingText
	sv.progress = info.Progress
	sv.error = info.Error
	sv.responseTime = info.ResponseTime
	sv.lastActivity = time.Now()

	if info.WorkingDir != "" {
		sv.workingDir = info.WorkingDir
	}

	sv.logger.Debug("Status updated",
		"mode", sv.mode,
		"model", sv.aiModel,
		"tokens", sv.tokenCount,
		"connected", sv.connected)
}

// SetMode sets the current mode
func (sv *StatusView) SetMode(mode string) {
	sv.mode = mode
}

// SetModel sets the AI model information
func (sv *StatusView) SetModel(model, provider string) {
	sv.aiModel = model
	sv.provider = provider
}

// SetTokenCount sets the token count
func (sv *StatusView) SetTokenCount(current, total int) {
	sv.tokenCount = current
	sv.totalTokens = total
}

// SetConnected sets the connection status
func (sv *StatusView) SetConnected(connected bool) {
	sv.connected = connected
}

// SetLoading sets the loading state
func (sv *StatusView) SetLoading(loading bool, text string) {
	sv.loading = loading
	sv.loadingText = text
}

// SetProgress sets the progress value (0.0 to 1.0)
func (sv *StatusView) SetProgress(progress float64) {
	sv.progress = progress
}

// SetError sets an error status
func (sv *StatusView) SetError(err error) {
	sv.error = err
	if err != nil {
		sv.ShowNotification(fmt.Sprintf("Error: %s", err.Error()), 5*time.Second)
	}
}

// ShowNotification shows a temporary notification
func (sv *StatusView) ShowNotification(message string, duration time.Duration) {
	sv.notification = message
	sv.notifyExpiry = time.Now().Add(duration)
}

// ToggleDetails toggles detailed status view
func (sv *StatusView) ToggleDetails() {
	sv.showDetails = !sv.showDetails
}

// Init implements tea.Model
func (sv *StatusView) Init() tea.Cmd {
	return tea.Tick(time.Second, func(t time.Time) tea.Msg {
		return tickMsg(t)
	})
}

// Update implements tea.Model
func (sv *StatusView) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "ctrl+s":
			sv.ToggleDetails()
		}

	case tea.WindowSizeMsg:
		sv.SetSize(msg.Width, msg.Height)

	case tickMsg:
		sv.uptime = time.Since(sv.startTime)
		// Clear expired notifications
		if !sv.notifyExpiry.IsZero() && time.Now().After(sv.notifyExpiry) {
			sv.notification = ""
			sv.notifyExpiry = time.Time{}
		}
		return sv, tea.Tick(time.Second, func(t time.Time) tea.Msg {
			return tickMsg(t)
		})
	}

	return sv, nil
}

// View implements tea.Model
func (sv *StatusView) View() string {
	if sv.showDetails {
		return sv.renderDetailedView()
	}

	if sv.compact {
		return sv.renderCompactView()
	}

	return sv.renderStandardView()
}

// renderStandardView renders the standard status bar
func (sv *StatusView) renderStandardView() string {
	var segments []string

	// Mode indicator
	modeStyle := sv.styles.StatusActive
	if sv.error != nil {
		modeStyle = sv.styles.StatusError
	} else if sv.loading {
		modeStyle = sv.styles.StatusLoading
	}
	segments = append(segments, modeStyle.Render(sv.mode))

	// AI Model
	if sv.aiModel != "Unknown" {
		modelText := sv.aiModel
		if sv.provider != "Unknown" && sv.provider != "" {
			modelText = fmt.Sprintf("%s (%s)", sv.aiModel, sv.provider)
		}
		segments = append(segments, sv.styles.StatusBar.Render(modelText))
	}

	// Token count
	if sv.tokenCount > 0 {
		tokenText := fmt.Sprintf("%s tokens", formatNumber(sv.tokenCount))
		if sv.totalTokens > 0 {
			tokenText = fmt.Sprintf("%s/%s tokens",
				formatNumber(sv.tokenCount),
				formatNumber(sv.totalTokens))
		}
		segments = append(segments, sv.styles.StatusBar.Render(tokenText))
	}

	// Working directory
	if sv.workingDir != "" {
		workDir := filepath.Base(sv.workingDir)
		if len(workDir) > 20 {
			workDir = "..." + workDir[len(workDir)-17:]
		}
		segments = append(segments, sv.styles.StatusBar.Render(workDir))
	}

	// Connection status
	connIndicator := "●"
	connStyle := sv.styles.StatusError
	if sv.connected {
		connStyle = sv.styles.StatusActive
	}

	// Loading indicator
	if sv.loading {
		spinner := []string{"⠋", "⠙", "⠹", "⠸", "⠼", "⠴", "⠦", "⠧", "⠇", "⠏"}
		frame := int(time.Now().UnixNano()/100000000) % len(spinner)
		connIndicator = spinner[frame]
		connStyle = sv.styles.StatusLoading
	}

	segments = append(segments, connStyle.Render(connIndicator))

	// Join segments
	content := strings.Join(segments, " │ ")

	// Add notification if present
	if sv.notification != "" {
		notifyStyle := sv.styles.StatusError
		if !strings.Contains(strings.ToLower(sv.notification), "error") {
			notifyStyle = sv.styles.StatusActive
		}
		content = notifyStyle.Render(sv.notification) + " │ " + content
	}

	// Create bordered status bar
	statusBar := sv.styles.StatusBar.
		Width(sv.width-2).
		Padding(0, 1).
		BorderStyle(lipgloss.NormalBorder()).
		BorderTop(true).
		BorderForeground(sv.styles.Colors.Border).
		Render(content)

	return statusBar
}

// renderCompactView renders a compact status bar for narrow screens
func (sv *StatusView) renderCompactView() string {
	var segments []string

	// Mode with status
	modeStyle := sv.styles.StatusActive
	if sv.error != nil {
		modeStyle = sv.styles.StatusError
	} else if sv.loading {
		modeStyle = sv.styles.StatusLoading
	}

	modeText := sv.mode
	if sv.loading {
		spinner := []string{"⠋", "⠙", "⠹", "⠸", "⠼", "⠴", "⠦", "⠧", "⠇", "⠏"}
		frame := int(time.Now().UnixNano()/100000000) % len(spinner)
		modeText = spinner[frame] + " " + sv.mode
	}
	segments = append(segments, modeStyle.Render(modeText))

	// Model (abbreviated)
	if sv.aiModel != "Unknown" {
		model := sv.aiModel
		if len(model) > 10 {
			model = model[:7] + "..."
		}
		segments = append(segments, sv.styles.StatusBar.Render(model))
	}

	// Tokens (abbreviated)
	if sv.tokenCount > 0 {
		segments = append(segments, sv.styles.StatusBar.Render(fmt.Sprintf("%dk", sv.tokenCount/1000)))
	}

	// Connection
	connIndicator := "●"
	connStyle := sv.styles.StatusError
	if sv.connected {
		connStyle = sv.styles.StatusActive
	}
	segments = append(segments, connStyle.Render(connIndicator))

	content := strings.Join(segments, "│")

	return sv.styles.StatusBar.
		Width(sv.width).
		Padding(0, 1).
		Render(content)
}

// renderDetailedView renders a detailed multi-line status view
func (sv *StatusView) renderDetailedView() string {
	var lines []string

	// Header
	header := sv.styles.Bold.Render("CODA Status")
	lines = append(lines, header)
	lines = append(lines, strings.Repeat("─", lipgloss.Width(header)))

	// Basic info
	lines = append(lines, fmt.Sprintf("Mode: %s", sv.mode))
	lines = append(lines, fmt.Sprintf("AI Model: %s (%s)", sv.aiModel, sv.provider))
	lines = append(lines, fmt.Sprintf("Connected: %v", sv.connected))

	// Token usage
	if sv.tokenCount > 0 {
		lines = append(lines, fmt.Sprintf("Tokens: %s", formatNumber(sv.tokenCount)))
		if sv.totalTokens > 0 {
			percentage := float64(sv.tokenCount) / float64(sv.totalTokens) * 100
			lines = append(lines, fmt.Sprintf("Usage: %.1f%%", percentage))
		}
	}

	// Performance
	if sv.responseTime > 0 {
		lines = append(lines, fmt.Sprintf("Response Time: %v", sv.responseTime))
	}
	lines = append(lines, fmt.Sprintf("Uptime: %v", sv.uptime.Truncate(time.Second)))

	// Session info
	if sv.sessionID != "" {
		lines = append(lines, fmt.Sprintf("Session: %s", sv.sessionID[:8]+"..."))
	}
	lines = append(lines, fmt.Sprintf("Working Dir: %s", sv.workingDir))

	// Error info
	if sv.error != nil {
		lines = append(lines, "")
		lines = append(lines, sv.styles.ErrorMessage.Render("Error: "+sv.error.Error()))
	}

	// Loading info
	if sv.loading {
		lines = append(lines, "")
		loadingText := "Loading..."
		if sv.loadingText != "" {
			loadingText = sv.loadingText
		}
		lines = append(lines, sv.styles.StatusLoading.Render(loadingText))

		if sv.progress > 0 {
			progressBar := sv.renderProgressBar(sv.progress)
			lines = append(lines, progressBar)
		}
	}

	content := strings.Join(lines, "\n")

	return sv.styles.Container.
		Width(sv.width - 2).
		Padding(1).
		BorderStyle(lipgloss.RoundedBorder()).
		BorderForeground(sv.styles.Colors.Border).
		Render(content)
}

// renderProgressBar renders a progress bar
func (sv *StatusView) renderProgressBar(progress float64) string {
	width := sv.width - 10 // Account for borders and padding
	if width < 10 {
		width = 10
	}

	filled := int(progress * float64(width))
	if filled > width {
		filled = width
	}

	bar := strings.Repeat("█", filled) + strings.Repeat("░", width-filled)
	percentage := fmt.Sprintf("%.1f%%", progress*100)

	return sv.styles.Progress.Render(bar) + " " + sv.styles.StatusBar.Render(percentage)
}

// Helper functions

// tickMsg represents a timer tick
type tickMsg time.Time

// getWorkingDir gets the current working directory
func getWorkingDir() string {
	if wd, err := filepath.Abs("."); err == nil {
		return wd
	}
	return "Unknown"
}

// formatNumber formats a number with thousand separators
func formatNumber(n int) string {
	if n < 1000 {
		return fmt.Sprintf("%d", n)
	}
	if n < 1000000 {
		return fmt.Sprintf("%.1fK", float64(n)/1000)
	}
	return fmt.Sprintf("%.1fM", float64(n)/1000000)
}

// GetUptime returns the current uptime
func (sv *StatusView) GetUptime() time.Duration {
	return sv.uptime
}

// GetTokenCount returns the current token count
func (sv *StatusView) GetTokenCount() int {
	return sv.tokenCount
}

// IsConnected returns the connection status
func (sv *StatusView) IsConnected() bool {
	return sv.connected
}

// IsLoading returns the loading status
func (sv *StatusView) IsLoading() bool {
	return sv.loading
}
