package ui

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"path/filepath"
	"runtime"
	"syscall"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/log"

	"github.com/common-creation/coda/internal/chat"
	"github.com/common-creation/coda/internal/config"
	"github.com/common-creation/coda/internal/tools"
)

// App represents the main TUI application
type App struct {
	program     *tea.Program
	model       Model
	config      *config.Config
	chatHandler *chat.ChatHandler
	toolManager *tools.Manager
	logger      *log.Logger
	ctx         context.Context
	cancel      context.CancelFunc
}

// AppOptions contains options for creating a new App
type AppOptions struct {
	Config      *config.Config
	ChatHandler *chat.ChatHandler
	ToolManager *tools.Manager
	Logger      *log.Logger
}

// NewApp creates a new TUI application instance
func NewApp(opts AppOptions) (*App, error) {
	if opts.Config == nil {
		return nil, fmt.Errorf("config is required")
	}
	if opts.ChatHandler == nil {
		return nil, fmt.Errorf("chat handler is required")
	}
	if opts.ToolManager == nil {
		return nil, fmt.Errorf("tool manager is required")
	}
	if opts.Logger == nil {
		opts.Logger = log.New(os.Stderr)
	}

	ctx, cancel := context.WithCancel(context.Background())

	// Create the model with dependencies
	model := NewModel(ModelOptions{
		Config:      opts.Config,
		ChatHandler: opts.ChatHandler,
		ToolManager: opts.ToolManager,
		Logger:      opts.Logger,
		Context:     ctx,
	})

	// Configure program options
	var programOpts []tea.ProgramOption

	// Enable mouse support if configured
	if opts.Config.UI.EnableMouse {
		programOpts = append(programOpts)
	}

	program := tea.NewProgram(model, programOpts...)

	app := &App{
		program:     program,
		model:       model,
		config:      opts.Config,
		chatHandler: opts.ChatHandler,
		toolManager: opts.ToolManager,
		logger:      opts.Logger,
		ctx:         ctx,
		cancel:      cancel,
	}

	// Setup panic recovery
	app.setupPanicRecovery()

	return app, nil
}

// Run starts the application and handles the main event loop
func (a *App) Run() error {
	a.logger.Info("Starting CODA TUI application")

	// Setup signal handlers
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	// Start the application in a goroutine
	errChan := make(chan error, 1)
	go func() {
		defer func() {
			if r := recover(); r != nil {
				a.handlePanic(r)
				errChan <- fmt.Errorf("application panicked: %v", r)
			}
		}()

		if _, err := a.program.Run(); err != nil {
			errChan <- fmt.Errorf("failed to run program: %w", err)
		} else {
			errChan <- nil
		}
	}()

	// Wait for completion or signal
	select {
	case sig := <-sigChan:
		a.logger.Info("Received signal", "signal", sig)
		return a.shutdown()
	case err := <-errChan:
		return err
	}
}

// Shutdown performs graceful shutdown of the application
func (a *App) Shutdown() error {
	return a.shutdown()
}

// shutdown performs the actual shutdown process
func (a *App) shutdown() error {
	a.logger.Info("Shutting down application")

	// Cancel the context
	a.cancel()

	// Give the program time to cleanup
	done := make(chan struct{})
	go func() {
		a.program.Quit()
		close(done)
	}()

	// Wait for cleanup with timeout
	timeout := time.Second * 2
	select {
	case <-done:
		a.logger.Info("Application shutdown complete")
		return nil
	case <-time.After(timeout):
		a.logger.Warn("Shutdown timeout, forcing exit")
		return fmt.Errorf("shutdown timeout after %v", timeout)
	}
}

// setupPanicRecovery sets up global panic recovery
func (a *App) setupPanicRecovery() {
	// This will catch panics in the main goroutine
	defer func() {
		if r := recover(); r != nil {
			a.handlePanic(r)
		}
	}()
}

// handlePanic handles application panics gracefully
func (a *App) handlePanic(r interface{}) {
	a.logger.Error("Application panic occurred", "panic", r)

	// Save current state if possible
	if err := a.saveState(); err != nil {
		a.logger.Error("Failed to save state during panic", "error", err)
	}

	// Generate crash report
	if err := a.generateCrashReport(r); err != nil {
		a.logger.Error("Failed to generate crash report", "error", err)
	}

	// Try to restore terminal
	if a.program != nil {
		a.program.Quit()
	}
}

// saveState saves the current application state
func (a *App) saveState() error {
	// Save chat history
	if a.chatHandler != nil {
		// This would need to be implemented in the chat handler
		// return a.chatHandler.SaveState()
	}

	// Save UI state
	if err := a.model.SaveState(); err != nil {
		return fmt.Errorf("failed to save model state: %w", err)
	}

	return nil
}

// generateCrashReport generates a crash report for debugging
func (a *App) generateCrashReport(panicInfo interface{}) error {
	crashDir := filepath.Join(os.TempDir(), "coda-crashes")
	if err := os.MkdirAll(crashDir, 0755); err != nil {
		return fmt.Errorf("failed to create crash directory: %w", err)
	}

	timestamp := time.Now().Format("2006-01-02_15-04-05")
	filename := filepath.Join(crashDir, fmt.Sprintf("crash_%s.log", timestamp))

	file, err := os.Create(filename)
	if err != nil {
		return fmt.Errorf("failed to create crash report file: %w", err)
	}
	defer file.Close()

	// Write crash information
	fmt.Fprintf(file, "CODA Crash Report\n")
	fmt.Fprintf(file, "================\n\n")
	fmt.Fprintf(file, "Timestamp: %s\n", time.Now().Format(time.RFC3339))
	fmt.Fprintf(file, "OS: %s\n", runtime.GOOS)
	fmt.Fprintf(file, "Arch: %s\n", runtime.GOARCH)
	fmt.Fprintf(file, "Go Version: %s\n", runtime.Version())
	fmt.Fprintf(file, "Panic: %v\n\n", panicInfo)

	// Write stack trace
	buf := make([]byte, 1024*1024) // 1MB buffer
	n := runtime.Stack(buf, true)
	fmt.Fprintf(file, "Stack Trace:\n%s\n", buf[:n])

	a.logger.Info("Crash report generated", "file", filename)
	return nil
}

// SendMessage sends a message through the application
func (a *App) SendMessage(msg tea.Msg) {
	if a.program != nil {
		a.program.Send(msg)
	}
}

// GetModel returns the current model (for testing purposes)
func (a *App) GetModel() Model {
	return a.model
}

// IsRunning returns whether the application is currently running
func (a *App) IsRunning() bool {
	select {
	case <-a.ctx.Done():
		return false
	default:
		return true
	}
}
