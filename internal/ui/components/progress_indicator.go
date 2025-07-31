package components

import (
	"fmt"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/progress"
	"github.com/charmbracelet/bubbles/spinner"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"github.com/common-creation/coda/internal/styles"
)

// ProgressType defines the type of progress indicator
type ProgressType int

const (
	ProgressSpinner ProgressType = iota
	ProgressBar
	ProgressTypeStep
	ProgressComposite
)

// ProgressStatus represents the current status
type ProgressStatus int

// tickMsg is used for time updates
type tickMsg time.Time

const (
	StatusRunning ProgressStatus = iota
	StatusCompleted
	StatusFailed
	StatusPaused
)

// ProgressIndicator displays progress information
type ProgressIndicator struct {
	// Core components
	spinner     spinner.Model
	progressBar progress.Model
	styles      styles.Styles

	// Configuration
	progressType    ProgressType
	status          ProgressStatus
	isIndeterminate bool

	// Progress data
	message    string
	percentage float64
	current    int
	total      int
	startTime  time.Time

	// Step progress
	steps       []ProgressStep
	currentStep int

	// Additional info
	errorCount   int
	warningCount int
	elapsedTime  time.Duration

	// Display properties
	width       int
	showDetails bool
	compact     bool
}

// ProgressStep represents a step in a multi-step process
type ProgressStep struct {
	Name        string
	Description string
	Status      ProgressStatus
	Duration    time.Duration
}

// ProgressUpdate represents a progress update message
type ProgressUpdate struct {
	Message    string
	Percentage float64
	Current    int
	Total      int
	Step       int
	Status     ProgressStatus
	Error      error
}

// NewProgressIndicator creates a new progress indicator
func NewProgressIndicator(progressType ProgressType, styles styles.Styles) *ProgressIndicator {
	// Initialize spinner
	s := spinner.New()
	s.Spinner = spinner.Dot
	s.Style = styles.StatusLoading

	// Initialize progress bar
	p := progress.New(progress.WithDefaultGradient())
	p.Width = 40

	return &ProgressIndicator{
		spinner:         s,
		progressBar:     p,
		styles:          styles,
		progressType:    progressType,
		status:          StatusRunning,
		isIndeterminate: true,
		message:         "Loading...",
		startTime:       time.Now(),
		width:           80,
		steps:           make([]ProgressStep, 0),
	}
}

// SetType sets the progress indicator type
func (pi *ProgressIndicator) SetType(progressType ProgressType) {
	pi.progressType = progressType
}

// SetMessage sets the current message
func (pi *ProgressIndicator) SetMessage(message string) {
	pi.message = message
	pi.elapsedTime = time.Since(pi.startTime)
}

// SetProgress sets the progress percentage (0.0 to 1.0)
func (pi *ProgressIndicator) SetProgress(percentage float64, current, total int) {
	pi.percentage = percentage
	pi.current = current
	pi.total = total
	pi.isIndeterminate = false
	pi.elapsedTime = time.Since(pi.startTime)
}

// SetStatus sets the current status
func (pi *ProgressIndicator) SetStatus(status ProgressStatus) {
	pi.status = status

	// Update spinner style based on status
	switch status {
	case StatusRunning:
		pi.spinner.Style = pi.styles.StatusLoading
	case StatusCompleted:
		pi.spinner.Style = pi.styles.StatusActive
	case StatusFailed:
		pi.spinner.Style = pi.styles.StatusError
	case StatusPaused:
		pi.spinner.Style = pi.styles.Muted
	}
}

// SetSteps sets up step-based progress
func (pi *ProgressIndicator) SetSteps(steps []string) {
	pi.steps = make([]ProgressStep, len(steps))
	for i, step := range steps {
		pi.steps[i] = ProgressStep{
			Name:   step,
			Status: StatusRunning,
		}
	}
	pi.currentStep = 0
}

// SetCurrentStep sets the current step
func (pi *ProgressIndicator) SetCurrentStep(step int, message string) {
	if step >= 0 && step < len(pi.steps) {
		// Mark previous steps as completed
		for i := 0; i < step; i++ {
			if pi.steps[i].Status == StatusRunning {
				pi.steps[i].Status = StatusCompleted
			}
		}

		pi.currentStep = step
		pi.steps[step].Status = StatusRunning
		pi.steps[step].Description = message
		pi.message = message
	}
}

// CompleteStep marks a step as completed
func (pi *ProgressIndicator) CompleteStep(step int) {
	if step >= 0 && step < len(pi.steps) {
		pi.steps[step].Status = StatusCompleted
		pi.steps[step].Duration = time.Since(pi.startTime)
	}
}

// FailStep marks a step as failed
func (pi *ProgressIndicator) FailStep(step int, err error) {
	if step >= 0 && step < len(pi.steps) {
		pi.steps[step].Status = StatusFailed
		pi.steps[step].Description = err.Error()
		pi.errorCount++
	}
}

// AddError increments the error count
func (pi *ProgressIndicator) AddError() {
	pi.errorCount++
}

// AddWarning increments the warning count
func (pi *ProgressIndicator) AddWarning() {
	pi.warningCount++
}

// SetWidth sets the display width
func (pi *ProgressIndicator) SetWidth(width int) {
	pi.width = width
	pi.progressBar.Width = width - 20 // Leave space for percentage and details
	pi.compact = width < 60
}

// SetShowDetails controls detail visibility
func (pi *ProgressIndicator) SetShowDetails(show bool) {
	pi.showDetails = show
}

// Init implements tea.Model
func (pi *ProgressIndicator) Init() tea.Cmd {
	return tea.Batch(pi.spinner.Tick, pi.tickCmd())
}

// Update implements tea.Model
func (pi *ProgressIndicator) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmd tea.Cmd
	var cmds []tea.Cmd

	switch msg := msg.(type) {
	case ProgressUpdate:
		pi.handleProgressUpdate(msg)

	case tea.WindowSizeMsg:
		pi.SetWidth(msg.Width)

	case tickMsg:
		pi.elapsedTime = time.Since(pi.startTime)
		cmds = append(cmds, pi.tickCmd())
	}

	// Update spinner
	pi.spinner, cmd = pi.spinner.Update(msg)
	cmds = append(cmds, cmd)

	return pi, tea.Batch(cmds...)
}

// handleProgressUpdate handles progress update messages
func (pi *ProgressIndicator) handleProgressUpdate(update ProgressUpdate) {
	if update.Message != "" {
		pi.SetMessage(update.Message)
	}

	if update.Total > 0 {
		pi.SetProgress(update.Percentage, update.Current, update.Total)
	}

	if update.Step >= 0 {
		pi.SetCurrentStep(update.Step, update.Message)
	}

	if update.Status != StatusRunning {
		pi.SetStatus(update.Status)
	}

	if update.Error != nil {
		pi.AddError()
	}
}

// View implements tea.Model
func (pi *ProgressIndicator) View() string {
	switch pi.progressType {
	case ProgressSpinner:
		return pi.renderSpinner()
	case ProgressBar:
		return pi.renderProgressBar()
	case ProgressTypeStep:
		return pi.renderStepProgress()
	case ProgressComposite:
		return pi.renderComposite()
	default:
		return pi.renderSpinner()
	}
}

// renderSpinner renders a spinner indicator
func (pi *ProgressIndicator) renderSpinner() string {
	var content strings.Builder

	// Spinner and message
	spinnerText := pi.spinner.View() + " " + pi.message
	content.WriteString(spinnerText)

	// Add details if not compact
	if !pi.compact && pi.showDetails {
		details := pi.renderDetails()
		if details != "" {
			content.WriteString("\n" + details)
		}
	}

	return content.String()
}

// renderProgressBar renders a progress bar
func (pi *ProgressIndicator) renderProgressBar() string {
	var content strings.Builder

	// Progress bar
	if pi.isIndeterminate {
		// Show spinner for indeterminate progress
		content.WriteString(pi.spinner.View() + " " + pi.message)
	} else {
		// Show progress bar
		bar := pi.progressBar.ViewAs(pi.percentage)
		percentage := fmt.Sprintf("%.1f%%", pi.percentage*100)

		if pi.total > 0 {
			countInfo := fmt.Sprintf("(%d/%d)", pi.current, pi.total)
			content.WriteString(bar + " " + percentage + " " + countInfo)
		} else {
			content.WriteString(bar + " " + percentage)
		}

		if pi.message != "" {
			content.WriteString("\n" + pi.message)
		}
	}

	// Add details
	if pi.showDetails {
		details := pi.renderDetails()
		if details != "" {
			content.WriteString("\n" + details)
		}
	}

	return content.String()
}

// renderStepProgress renders step-based progress
func (pi *ProgressIndicator) renderStepProgress() string {
	var content strings.Builder

	// Current step header
	if pi.currentStep < len(pi.steps) {
		step := pi.steps[pi.currentStep]
		stepHeader := fmt.Sprintf("Step %d/%d: %s",
			pi.currentStep+1, len(pi.steps), step.Name)
		content.WriteString(pi.styles.Bold.Render(stepHeader))

		if step.Description != "" {
			content.WriteString("\n" + step.Description)
		}
		content.WriteString("\n")
	}

	// Step list
	if !pi.compact {
		for i, step := range pi.steps {
			var icon string
			var style lipgloss.Style

			switch step.Status {
			case StatusCompleted:
				icon = "✓"
				style = pi.styles.StatusActive
			case StatusRunning:
				icon = pi.spinner.View()
				style = pi.styles.StatusLoading
			case StatusFailed:
				icon = "✗"
				style = pi.styles.StatusError
			default:
				icon = "○"
				style = pi.styles.Muted
			}

			stepLine := fmt.Sprintf("%s %s", icon, step.Name)
			if i == pi.currentStep && step.Description != "" {
				stepLine += " - " + step.Description
			}

			content.WriteString(style.Render(stepLine) + "\n")
		}
	}

	// Overall progress
	if len(pi.steps) > 0 {
		completed := 0
		for _, step := range pi.steps {
			if step.Status == StatusCompleted {
				completed++
			}
		}

		progress := float64(completed) / float64(len(pi.steps))
		bar := pi.progressBar.ViewAs(progress)
		percentage := fmt.Sprintf("%.0f%%", progress*100)

		content.WriteString(bar + " " + percentage)
	}

	return content.String()
}

// renderComposite renders a composite indicator
func (pi *ProgressIndicator) renderComposite() string {
	var content strings.Builder

	// Main message with spinner/status
	switch pi.status {
	case StatusRunning:
		content.WriteString(pi.spinner.View() + " " + pi.message)
	case StatusCompleted:
		content.WriteString(pi.styles.StatusActive.Render("✓") + " " + pi.message)
	case StatusFailed:
		content.WriteString(pi.styles.StatusError.Render("✗") + " " + pi.message)
	case StatusPaused:
		content.WriteString(pi.styles.Muted.Render("⏸") + " " + pi.message)
	}

	// Progress bar if determinate
	if !pi.isIndeterminate && pi.status == StatusRunning {
		content.WriteString("\n")
		bar := pi.progressBar.ViewAs(pi.percentage)
		percentage := fmt.Sprintf("%.1f%%", pi.percentage*100)
		content.WriteString(bar + " " + percentage)

		if pi.total > 0 {
			countInfo := fmt.Sprintf(" (%d/%d)", pi.current, pi.total)
			content.WriteString(countInfo)
		}
	}

	// Details
	if pi.showDetails {
		details := pi.renderDetails()
		if details != "" {
			content.WriteString("\n" + details)
		}
	}

	return content.String()
}

// renderDetails renders additional progress details
func (pi *ProgressIndicator) renderDetails() string {
	var details []string

	// Time information
	if pi.elapsedTime > 0 {
		elapsed := pi.elapsedTime.Truncate(time.Second)
		details = append(details, fmt.Sprintf("Elapsed: %v", elapsed))

		// Estimate remaining time for determinate progress
		if !pi.isIndeterminate && pi.percentage > 0 && pi.percentage < 1 {
			remaining := time.Duration(float64(pi.elapsedTime) * (1 - pi.percentage) / pi.percentage)
			remaining = remaining.Truncate(time.Second)
			details = append(details, fmt.Sprintf("Est. remaining: %v", remaining))
		}
	}

	// Error/warning counts
	if pi.errorCount > 0 {
		details = append(details,
			pi.styles.StatusError.Render(fmt.Sprintf("Errors: %d", pi.errorCount)))
	}

	if pi.warningCount > 0 {
		details = append(details,
			pi.styles.StatusLoading.Render(fmt.Sprintf("Warnings: %d", pi.warningCount)))
	}

	if len(details) == 0 {
		return ""
	}

	return pi.styles.Muted.Render(strings.Join(details, " | "))
}

// tickCmd creates a tick command for time updates
func (pi *ProgressIndicator) tickCmd() tea.Cmd {
	return tea.Tick(time.Second, func(t time.Time) tea.Msg {
		return tickMsg(t)
	})
}

// Helper functions

// CreateProgressUpdate creates a progress update message
func CreateProgressUpdate(message string, percentage float64, current, total int) ProgressUpdate {
	return ProgressUpdate{
		Message:    message,
		Percentage: percentage,
		Current:    current,
		Total:      total,
		Status:     StatusRunning,
	}
}

// CreateStepUpdate creates a step progress update
func CreateStepUpdate(step int, message string) ProgressUpdate {
	return ProgressUpdate{
		Step:    step,
		Message: message,
		Status:  StatusRunning,
	}
}

// CreateCompletionUpdate creates a completion update
func CreateCompletionUpdate(message string) ProgressUpdate {
	return ProgressUpdate{
		Message: message,
		Status:  StatusCompleted,
	}
}

// CreateErrorUpdate creates an error update
func CreateErrorUpdate(message string, err error) ProgressUpdate {
	return ProgressUpdate{
		Message: message,
		Status:  StatusFailed,
		Error:   err,
	}
}

// IsCompleted returns whether the progress is completed
func (pi *ProgressIndicator) IsCompleted() bool {
	return pi.status == StatusCompleted
}

// IsFailed returns whether the progress has failed
func (pi *ProgressIndicator) IsFailed() bool {
	return pi.status == StatusFailed
}

// GetElapsedTime returns the elapsed time
func (pi *ProgressIndicator) GetElapsedTime() time.Duration {
	return pi.elapsedTime
}

// GetErrorCount returns the error count
func (pi *ProgressIndicator) GetErrorCount() int {
	return pi.errorCount
}

// GetWarningCount returns the warning count
func (pi *ProgressIndicator) GetWarningCount() int {
	return pi.warningCount
}

// Reset resets the progress indicator
func (pi *ProgressIndicator) Reset() {
	pi.percentage = 0
	pi.current = 0
	pi.total = 0
	pi.message = "Loading..."
	pi.status = StatusRunning
	pi.isIndeterminate = true
	pi.errorCount = 0
	pi.warningCount = 0
	pi.startTime = time.Now()
	pi.elapsedTime = 0
	pi.currentStep = 0

	// Reset steps
	for i := range pi.steps {
		pi.steps[i].Status = StatusRunning
		pi.steps[i].Description = ""
		pi.steps[i].Duration = 0
	}
}
