// Package components provides UI components for error display.
package components

import (
	"fmt"
	"strings"
	"time"

	"github.com/charmbracelet/lipgloss"
	"github.com/common-creation/coda/internal/ai"
	"github.com/common-creation/coda/internal/errors"
)

// ErrorDisplay provides user-friendly error display functionality.
type ErrorDisplay struct {
	handler      *errors.ErrorHandler
	currentError error
	showDetails  bool
	styles       ErrorStyles
}

// ErrorStyles defines the styling for error display components.
type ErrorStyles struct {
	ErrorBox     lipgloss.Style
	ErrorTitle   lipgloss.Style
	ErrorMessage lipgloss.Style
	ErrorDetail  lipgloss.Style
	ActionHint   lipgloss.Style
	Timestamp    lipgloss.Style
}

// NewErrorDisplay creates a new error display component.
func NewErrorDisplay(handler *errors.ErrorHandler) *ErrorDisplay {
	return &ErrorDisplay{
		handler:      handler,
		currentError: nil,
		showDetails:  false,
		styles:       DefaultErrorStyles(),
	}
}

// DefaultErrorStyles returns the default styling for error display.
func DefaultErrorStyles() ErrorStyles {
	return ErrorStyles{
		ErrorBox: lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(lipgloss.Color("9")).
			Padding(1, 2).
			Margin(1, 0).
			Background(lipgloss.Color("52")).
			Foreground(lipgloss.Color("15")),

		ErrorTitle: lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color("9")),

		ErrorMessage: lipgloss.NewStyle().
			Foreground(lipgloss.Color("15")).
			Margin(1, 0),

		ErrorDetail: lipgloss.NewStyle().
			Foreground(lipgloss.Color("8")).
			Italic(true).
			Margin(1, 0, 0, 2),

		ActionHint: lipgloss.NewStyle().
			Foreground(lipgloss.Color("10")).
			Bold(true).
			Margin(1, 0, 0, 0),

		Timestamp: lipgloss.NewStyle().
			Foreground(lipgloss.Color("8")).
			Align(lipgloss.Right),
	}
}

// SetError sets the current error to display.
func (e *ErrorDisplay) SetError(err error) {
	e.currentError = err
}

// ToggleDetails toggles the display of error details.
func (e *ErrorDisplay) ToggleDetails() {
	e.showDetails = !e.showDetails
}

// Render renders the error display.
func (e *ErrorDisplay) Render(width int) string {
	if e.currentError == nil {
		return ""
	}

	// Get user-friendly message
	userMessage := e.handler.UserMessage(e.currentError)

	// Build the error display
	var content strings.Builder

	// Title
	title := e.styles.ErrorTitle.Render("⚠ エラーが発生しました")
	content.WriteString(title + "\n\n")

	// User-friendly message
	message := e.styles.ErrorMessage.Render(userMessage)
	content.WriteString(message + "\n")

	// Action hints based on error type
	actionHint := e.getActionHint(e.currentError)
	if actionHint != "" {
		hint := e.styles.ActionHint.Render("💡 " + actionHint)
		content.WriteString("\n" + hint + "\n")
	}

	// Details (if requested)
	if e.showDetails {
		details := e.getErrorDetails(e.currentError)
		if details != "" {
			detail := e.styles.ErrorDetail.Render("詳細: " + details)
			content.WriteString("\n" + detail + "\n")
		}
	}

	// Instructions
	instructions := e.getInstructions()
	content.WriteString("\n" + instructions)

	// Timestamp
	timestamp := e.styles.Timestamp.Render(time.Now().Format("15:04:05"))
	content.WriteString("\n" + timestamp)

	// Wrap in error box
	errorBox := e.styles.ErrorBox.Width(width - 4).Render(content.String())

	return errorBox
}

// getActionHint returns appropriate action hints based on error type.
func (e *ErrorDisplay) getActionHint(err error) string {
	if err == nil {
		return ""
	}

	category := e.handler.ClassifyError(err)

	switch category {
	case errors.UserError:
		return "入力内容を確認してください"
	case errors.NetworkError:
		return "インターネット接続を確認してください"
	case errors.ConfigError:
		return "設定ファイルを確認するか、'config' コマンドを実行してください"
	case errors.SecurityError:
		return "システム管理者にお問い合わせください"
	case errors.AIServiceError:
		return "しばらく待ってから再試行してください"
	case errors.SystemError:
		return "問題が続く場合はサポートにお問い合わせください"
	default:
		return "詳細を確認するには 'd' キーを押してください"
	}
}

// getErrorDetails returns detailed error information.
func (e *ErrorDisplay) getErrorDetails(err error) string {
	if err == nil {
		return ""
	}

	// For AI errors, provide specific details
	if aiErr, ok := err.(*ai.Error); ok {
		details := fmt.Sprintf("種類: %s", aiErr.Type)
		if aiErr.RequestID != "" {
			details += fmt.Sprintf(", リクエストID: %s", aiErr.RequestID)
		}
		if aiErr.StatusCode != 0 {
			details += fmt.Sprintf(", ステータス: %d", aiErr.StatusCode)
		}
		return details
	}

	// For regular errors, just return the error message
	return err.Error()
}

// getInstructions returns user instructions for handling the error.
func (e *ErrorDisplay) getInstructions() string {
	instructions := []string{
		"Enter: エラーを閉じる",
		"r: 再試行",
	}

	if !e.showDetails {
		instructions = append(instructions, "d: 詳細表示")
	} else {
		instructions = append(instructions, "d: 詳細を隠す")
	}

	instructions = append(instructions, "q: 終了")

	return strings.Join(instructions, " | ")
}

// ClassifyError is a helper method to classify errors (exposed for UI use).
func (e *ErrorDisplay) ClassifyError(err error) errors.ErrorCategory {
	if e.handler != nil {
		// This would need to be exposed in the handler
		// For now, use a simple classification
		return e.classifyErrorSimple(err)
	}
	return errors.SystemError
}

// classifyErrorSimple provides a simple error classification for UI purposes.
func (e *ErrorDisplay) classifyErrorSimple(err error) errors.ErrorCategory {
	if err == nil {
		return errors.SystemError
	}

	errMsg := strings.ToLower(err.Error())

	switch {
	case strings.Contains(errMsg, "permission denied") ||
		strings.Contains(errMsg, "unauthorized"):
		return errors.SecurityError

	case strings.Contains(errMsg, "network") ||
		strings.Contains(errMsg, "connection") ||
		strings.Contains(errMsg, "timeout"):
		return errors.NetworkError

	case strings.Contains(errMsg, "config") ||
		strings.Contains(errMsg, "invalid key"):
		return errors.ConfigError

	case strings.Contains(errMsg, "file not found") ||
		strings.Contains(errMsg, "invalid argument"):
		return errors.UserError

	case strings.Contains(errMsg, "api") ||
		strings.Contains(errMsg, "rate limit") ||
		strings.Contains(errMsg, "quota"):
		return errors.AIServiceError

	default:
		return errors.SystemError
	}
}

// ErrorBanner provides a simple banner-style error display.
type ErrorBanner struct {
	styles BannerStyles
}

// BannerStyles defines styling for error banners.
type BannerStyles struct {
	Banner  lipgloss.Style
	Message lipgloss.Style
	Icon    lipgloss.Style
}

// NewErrorBanner creates a new error banner component.
func NewErrorBanner() *ErrorBanner {
	return &ErrorBanner{
		styles: DefaultBannerStyles(),
	}
}

// DefaultBannerStyles returns default banner styles.
func DefaultBannerStyles() BannerStyles {
	return BannerStyles{
		Banner: lipgloss.NewStyle().
			Background(lipgloss.Color("1")).
			Foreground(lipgloss.Color("15")).
			Padding(0, 1).
			Margin(0),

		Message: lipgloss.NewStyle().
			Foreground(lipgloss.Color("15")),

		Icon: lipgloss.NewStyle().
			Foreground(lipgloss.Color("9")).
			Bold(true),
	}
}

// Render renders a simple error banner.
func (b *ErrorBanner) Render(message string, width int) string {
	if message == "" {
		return ""
	}

	icon := b.styles.Icon.Render("⚠")
	msg := b.styles.Message.Render(message)

	content := fmt.Sprintf("%s %s", icon, msg)
	banner := b.styles.Banner.Width(width).Render(content)

	return banner
}

// ToastNotification provides toast-style error notifications.
type ToastNotification struct {
	message   string
	timestamp time.Time
	duration  time.Duration
	styles    ToastStyles
}

// ToastStyles defines styling for toast notifications.
type ToastStyles struct {
	Toast   lipgloss.Style
	Message lipgloss.Style
}

// NewToastNotification creates a new toast notification.
func NewToastNotification(message string, duration time.Duration) *ToastNotification {
	return &ToastNotification{
		message:   message,
		timestamp: time.Now(),
		duration:  duration,
		styles:    DefaultToastStyles(),
	}
}

// DefaultToastStyles returns default toast styles.
func DefaultToastStyles() ToastStyles {
	return ToastStyles{
		Toast: lipgloss.NewStyle().
			Background(lipgloss.Color("52")).
			Foreground(lipgloss.Color("15")).
			Border(lipgloss.RoundedBorder()).
			BorderForeground(lipgloss.Color("9")).
			Padding(0, 2).
			Margin(1),

		Message: lipgloss.NewStyle().
			Foreground(lipgloss.Color("15")),
	}
}

// IsExpired returns whether the toast has expired.
func (t *ToastNotification) IsExpired() bool {
	return time.Since(t.timestamp) > t.duration
}

// Render renders the toast notification.
func (t *ToastNotification) Render() string {
	if t.IsExpired() {
		return ""
	}

	message := t.styles.Message.Render(t.message)
	toast := t.styles.Toast.Render(message)

	return toast
}

// GetRemainingTime returns the remaining display time.
func (t *ToastNotification) GetRemainingTime() time.Duration {
	elapsed := time.Since(t.timestamp)
	if elapsed >= t.duration {
		return 0
	}
	return t.duration - elapsed
}
