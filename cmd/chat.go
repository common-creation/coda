/*
Copyright © 2025 CODA Project
*/
package cmd

import (
	"bufio"
	"context"
	"fmt"
	"io"
	"os"
	"os/signal"
	"path/filepath"
	"strings"
	"syscall"
	"time"

	"github.com/spf13/cobra"

	"github.com/common-creation/coda/internal/ai"
	"github.com/common-creation/coda/internal/chat"
	"github.com/common-creation/coda/internal/config"
	"github.com/common-creation/coda/internal/security"
	"github.com/common-creation/coda/internal/tools"
	"github.com/common-creation/coda/internal/ui"
)

// These variables are shared between root.go and chat.go
// to support both "coda chat" and direct "coda" invocation
var (
	model           string
	noStream        bool
	sessionID       string
	continueSession bool
	noTools         bool
	autoApprove     bool
	useTUI          bool
	initialMessage  string  // Initial message to send when starting chat
)

// chatCmd represents the chat command
var chatCmd = &cobra.Command{
	Use:   "chat",
	Short: "Start an interactive chat session",
	Long: `Start an interactive chat session with the AI assistant.

The chat command provides an interactive interface for conversing with AI models.
You can ask questions, request code analysis, and perform various development tasks
through natural language interaction.

Examples:
  coda chat                    # Start a new chat session
  coda chat --continue         # Continue the last session
  coda chat --model gpt-4      # Use a specific model
  coda chat --no-tools         # Disable tool execution`,
	RunE: runChat,
}

func init() {
	rootCmd.AddCommand(chatCmd)

	// Command flags
	chatCmd.Flags().StringVar(&model, "model", "", "AI model to use (overrides config)")
	chatCmd.Flags().BoolVar(&noStream, "no-stream", false, "disable streaming responses")
	chatCmd.Flags().StringVar(&sessionID, "session", "", "specify session ID to load")
	chatCmd.Flags().BoolVar(&continueSession, "continue", false, "continue last session")
	chatCmd.Flags().BoolVar(&noTools, "no-tools", false, "disable tool execution")
	chatCmd.Flags().BoolVar(&autoApprove, "auto-approve", false, "auto-approve all tool executions (use with caution)")
	chatCmd.Flags().BoolVar(&useTUI, "tui", true, "use interactive TUI interface (default: true)")
	chatCmd.Flags().BoolVar(&useTUI, "ui", true, "use interactive TUI interface (alias for --tui)")
	chatCmd.Flags().Bool("no-tui", false, "disable TUI interface")
}

func runChat(cmd *cobra.Command, args []string) error {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Setup signal handling
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)
	go func() {
		<-sigChan
		ShowInfo("\nReceived interrupt signal. Exiting...")
		cancel()
	}()

	// If args are provided, use them as the initial message
	if len(args) > 0 {
		initialMessage = strings.Join(args, " ")
	}

	// Setup chat components
	handler, err := setupChatHandler(ctx)
	if err != nil {
		return fmt.Errorf("failed to setup chat handler: %w", err)
	}

	// Check if --no-tui flag was set
	noTUI, _ := cmd.Flags().GetBool("no-tui")
	
	// Use TUI mode by default or if explicitly enabled
	if useTUI && !noTUI {
		return runTUIChat(ctx, handler)
	}

	// Run traditional CLI mode
	return runCLIChat(ctx, handler)
}

func runTUIChat(ctx context.Context, handler *chat.ChatHandler) error {
	// Create tool manager (same as in setupChatHandler)
	cfg := GetConfig()
	validator := security.NewDefaultValidator(".")
	logger := &simpleLogger{}
	wrappedValidator := &securityValidatorWrapper{validator: validator}
	toolManager := tools.NewManager(wrappedValidator, logger)
	
	// Register tools unless disabled
	if !noTools {
		toolManager.Register(tools.NewReadFileTool(wrappedValidator))
		toolManager.Register(tools.NewWriteFileTool(wrappedValidator))
		toolManager.Register(tools.NewEditFileTool(wrappedValidator))
		toolManager.Register(tools.NewListFilesTool(wrappedValidator))
		toolManager.Register(tools.NewSearchFilesTool(wrappedValidator))
	}
	
	// Create and run the Bubbletea UI app
	app, err := ui.NewApp(ui.AppOptions{
		Config:         cfg,
		ChatHandler:    handler,
		ToolManager:    toolManager,
		Logger:         nil, // Will use default logger
		InitialMessage: initialMessage,
	})
	if err != nil {
		return fmt.Errorf("failed to create app: %w", err)
	}
	
	return app.Run()
}

func runCLIChat(ctx context.Context, handler *chat.ChatHandler) error {
	// Show welcome message
	showWelcomeMessage()

	// Process initial message if provided
	if initialMessage != "" {
		ShowInfo("> %s", initialMessage)
		if err := handler.HandleMessage(ctx, initialMessage); err != nil {
			ShowError("Failed to process initial message: %v", err)
		}
	}

	// Main chat loop
	reader := bufio.NewReader(os.Stdin)

	for {
		// Check context cancellation
		select {
		case <-ctx.Done():
			return nil
		default:
		}

		// Show prompt
		fmt.Print("\n> ")

		// Read input
		input, err := readInput(reader)
		if err != nil {
			if err == io.EOF {
				ShowInfo("\nGoodbye!")
				return nil
			}
			ShowError("Failed to read input: %v", err)
			continue
		}

		// Check for exit commands
		if shouldExit(input) {
			ShowInfo("Goodbye!")
			return nil
		}

		// Handle empty input
		if strings.TrimSpace(input) == "" {
			continue
		}

		// Process message
		if err := handler.HandleMessage(ctx, input); err != nil {
			ShowError("Failed to process message: %v", err)
			continue
		}
	}
}

func setupChatHandler(ctx context.Context) (*chat.ChatHandler, error) {
	cfg := GetConfig()

	// Override model if specified
	if model != "" {
		cfg.AI.Model = model
	}

	// Create AI client
	aiClient, err := createAIClient(cfg)
	if err != nil {
		return nil, fmt.Errorf("failed to create AI client: %w", err)
	}

	// Create tool manager
	toolManager, err := createToolManager(cfg)
	if err != nil {
		return nil, fmt.Errorf("failed to create tool manager: %w", err)
	}

	// Create session manager
	// Use default values for now as SessionConfig doesn't have MaxAge and MaxTokens
	sessionManager := chat.NewSessionManager(30*24*60*60, 1000000) // 30 days, 1M tokens

	// Handle session continuation
	if continueSession || sessionID != "" {
		if err := loadPreviousSession(sessionManager, sessionID); err != nil {
			ShowWarning("Failed to load previous session: %v", err)
		}
	}

	// Create history manager
	historyPath := filepath.Join(getDataDir(), "history")
	history, err := chat.NewHistory(historyPath)
	if err != nil {
		ShowWarning("Failed to initialize history: %v", err)
		// Continue without history
		history = nil
	}

	// Create chat handler
	handler := chat.NewChatHandler(aiClient, toolManager, sessionManager, cfg, history)

	// Create and set prompt builder
	promptBuilder := chat.NewPromptBuilder(cfg.AI.MaxTokens, nil)

	// Add tool prompts
	for _, tool := range toolManager.GetAll() {
		promptBuilder.AddToolPrompt(tool.Name(), tool.Description())
	}

	// Apply workspace config if available
	workspaceLoader := chat.NewWorkspaceLoader()
	if workspaceConfig, err := workspaceLoader.LoadWorkspaceConfig("."); err == nil && workspaceConfig != nil {
		chat.ApplyWorkspaceConfig(workspaceConfig, promptBuilder)
	}

	// Build and set system prompt
	systemPrompt, err := promptBuilder.Build()
	if err != nil {
		ShowWarning("Failed to build system prompt: %v", err)
		// Use default prompt
	} else {
		handler.SetSystemPrompt(systemPrompt)
	}

	return handler, nil
}

func createAIClient(cfg *config.Config) (ai.Client, error) {
	// Check if API key is available
	if cfg.AI.APIKey == "" {
		ShowError("No API key configured!")
		ShowError("")
		ShowError("Please set up your API key in one of the following ways:")
		ShowError("1. Create config.yaml in the current directory with:")
		ShowError("   ai:")
		ShowError("     provider: openai")
		ShowError("     api_key: sk-xxxxxxxxxxxxx")
		ShowError("")
		ShowError("2. Set environment variable:")
		ShowError("   export OPENAI_API_KEY=sk-xxxxxxxxxxxxx")
		ShowError("")
		ShowError("3. Create ~/.config/coda/config.yaml")
		ShowError("")
		ShowError("See config.example.yaml for a complete configuration example.")
		return nil, fmt.Errorf("API key not configured")
	}

	// Use the standard AI client factory
	return ai.NewClient(cfg.AI)
}

func createToolManager(cfg *config.Config) (*tools.Manager, error) {
	// Create security validator
	validator := security.NewDefaultValidator(".")

	// Create logger (placeholder)
	logger := &simpleLogger{}

	// Wrap validator to match tools.SecurityValidator interface
	wrappedValidator := &securityValidatorWrapper{validator: validator}

	// Create tool manager
	manager := tools.NewManager(wrappedValidator, logger)

	// Register tools unless disabled
	if !noTools {
		// Register file tools
		manager.Register(tools.NewReadFileTool(wrappedValidator))
		manager.Register(tools.NewWriteFileTool(wrappedValidator))
		manager.Register(tools.NewEditFileTool(wrappedValidator))
		manager.Register(tools.NewListFilesTool(wrappedValidator))
		manager.Register(tools.NewSearchFilesTool(wrappedValidator))
	}

	return manager, nil
}

func loadPreviousSession(sessionManager *chat.SessionManager, specificID string) error {
	// Get project-specific session path
	sessionPath, err := chat.GetProjectSessionPath()
	if err != nil {
		return fmt.Errorf("failed to get session path: %w", err)
	}

	// Create persistence manager
	persistence, err := chat.NewFilePersistence(sessionPath, true, 5*time.Minute)
	if err != nil {
		return fmt.Errorf("failed to create persistence: %w", err)
	}

	// If specific ID provided, load it
	if specificID != "" {
		session, err := persistence.LoadSession(specificID)
		if err != nil {
			return fmt.Errorf("failed to load session %s: %w", specificID, err)
		}
		// TODO: Add session to sessionManager
		ShowInfo("Loaded session: %s", session.ID)
		return nil
	}

	// Otherwise, list available sessions
	sessions, err := persistence.ListSessions()
	if err != nil {
		return fmt.Errorf("failed to list sessions: %w", err)
	}

	if len(sessions) == 0 {
		ShowInfo("No previous sessions found for this project")
		_, err := sessionManager.CreateSession()
		return err
	}

	// For now, load the most recent session
	// TODO: Implement TUI session selector
	if len(sessions) > 0 {
		session, err := persistence.LoadSession(sessions[0])
		if err != nil {
			return fmt.Errorf("failed to load session: %w", err)
		}
		ShowInfo("Loaded most recent session: %s", session.ID)
	}

	return nil
}

func showWelcomeMessage() {
	ShowInfo(`
Welcome to CODA - Your AI Coding Assistant

Type your questions or requests below. Special commands:
  /help    - Show help information
  /clear   - Clear the current session
  /save    - Save the current session
  /exit    - Exit the chat

Press Ctrl+C to exit at any time.
`)
}

func readInput(reader *bufio.Reader) (string, error) {
	// Read single line for now
	// TODO: Implement multi-line input support
	line, err := reader.ReadString('\n')
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(line), nil
}

func shouldExit(input string) bool {
	input = strings.ToLower(strings.TrimSpace(input))
	return input == "exit" || input == "quit" || input == "/exit" || input == "/quit"
}

func getDataDir() string {
	// Get data directory
	home, err := os.UserHomeDir()
	if err != nil {
		return ".coda"
	}
	return filepath.Join(home, ".coda")
}

// simpleLogger is a placeholder logger implementation
type simpleLogger struct{}

func (l *simpleLogger) Info(msg string, keysAndValues ...interface{}) {
	if IsDebug() {
		fmt.Printf("[INFO] %s", msg)
		for i := 0; i < len(keysAndValues); i += 2 {
			if i+1 < len(keysAndValues) {
				fmt.Printf(" %v=%v", keysAndValues[i], keysAndValues[i+1])
			}
		}
		fmt.Println()
	}
}

func (l *simpleLogger) Error(msg string, keysAndValues ...interface{}) {
	fmt.Fprintf(os.Stderr, "[ERROR] %s", msg)
	for i := 0; i < len(keysAndValues); i += 2 {
		if i+1 < len(keysAndValues) {
			fmt.Fprintf(os.Stderr, " %v=%v", keysAndValues[i], keysAndValues[i+1])
		}
	}
	fmt.Fprintln(os.Stderr)
}

func (l *simpleLogger) Debug(msg string, keysAndValues ...interface{}) {
	if IsDebug() {
		fmt.Printf("[DEBUG] %s", msg)
		for i := 0; i < len(keysAndValues); i += 2 {
			if i+1 < len(keysAndValues) {
				fmt.Printf(" %v=%v", keysAndValues[i], keysAndValues[i+1])
			}
		}
		fmt.Println()
	}
}

func (l *simpleLogger) Warn(msg string, keysAndValues ...interface{}) {
	fmt.Printf("[WARN] %s", msg)
	for i := 0; i < len(keysAndValues); i += 2 {
		if i+1 < len(keysAndValues) {
			fmt.Printf(" %v=%v", keysAndValues[i], keysAndValues[i+1])
		}
	}
	fmt.Println()
}

// securityValidatorWrapper wraps security.DefaultValidator to implement tools.SecurityValidator
type securityValidatorWrapper struct {
	validator *security.DefaultValidator
}

func (w *securityValidatorWrapper) ValidatePath(path string) error {
	return w.validator.ValidatePath(path)
}

func (w *securityValidatorWrapper) ValidateOperation(op tools.Operation, path string) error {
	// Convert tools.Operation to security.Operation
	var secOp security.Operation
	switch op {
	case tools.OpRead:
		secOp = security.OpRead
	case tools.OpWrite:
		secOp = security.OpWrite
	case tools.OpDelete:
		secOp = security.OpDelete
	case tools.OpExecute:
		secOp = security.OpExecute
	case tools.OpList:
		secOp = security.OpList
	default:
		return fmt.Errorf("unknown operation: %s", op)
	}
	return w.validator.ValidateOperation(secOp, path)
}

func (w *securityValidatorWrapper) IsAllowedExtension(path string) bool {
	return w.validator.IsAllowedExtension(path)
}

func (w *securityValidatorWrapper) CheckContent(content []byte) error {
	return w.validator.CheckContent(content)
}
