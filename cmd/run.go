/*
Copyright © 2025 CODA Project
*/
package cmd

import (
	"context"
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/spf13/cobra"
	"github.com/common-creation/coda/internal/chat"
)

var (
	inputFile  string
	outputFile string
	prompt     string
)

// runCmd represents the run command for non-interactive mode
var runCmd = &cobra.Command{
	Use:   "run [prompt]",
	Short: "Run CODA in non-interactive mode",
	Long: `Run CODA in non-interactive mode with a single prompt.

This command allows you to use CODA for one-off tasks without entering
the interactive chat interface. It's useful for scripting and automation.

Examples:
  # Simple prompt
  coda run "Explain this Python code: print('Hello')"
  
  # Read input from file
  coda run --input code.py "Explain this code"
  
  # Save output to file
  coda run --output result.txt "Generate a README template"
  
  # Pipe input
  cat code.js | coda run "Review this JavaScript code"`,
	Args: cobra.MaximumNArgs(1),
	RunE: runNonInteractive,
}

func init() {
	rootCmd.AddCommand(runCmd)

	// Command flags
	runCmd.Flags().StringVarP(&inputFile, "input", "i", "", "read input from file")
	runCmd.Flags().StringVarP(&outputFile, "output", "o", "", "save output to file")
	runCmd.Flags().StringVarP(&model, "model", "m", "", "AI model to use")
	runCmd.Flags().BoolVar(&noTools, "no-tools", false, "disable tool execution")
}

func runNonInteractive(cmd *cobra.Command, args []string) error {
	ctx := context.Background()

	// Get prompt from arguments or flags
	userPrompt := ""
	if len(args) > 0 {
		userPrompt = args[0]
	}

	// Read additional input if specified
	var additionalInput string
	if inputFile != "" {
		content, err := os.ReadFile(inputFile)
		if err != nil {
			return fmt.Errorf("failed to read input file: %w", err)
		}
		additionalInput = string(content)
	} else if !isTerminal(os.Stdin) {
		// Read from stdin if it's piped
		content, err := io.ReadAll(os.Stdin)
		if err != nil {
			return fmt.Errorf("failed to read from stdin: %w", err)
		}
		additionalInput = string(content)
	}

	// Construct final prompt
	finalPrompt := constructPrompt(userPrompt, additionalInput, inputFile)
	if finalPrompt == "" {
		return fmt.Errorf("no prompt provided")
	}

	// Setup output
	output := os.Stdout
	if outputFile != "" {
		file, err := os.Create(outputFile)
		if err != nil {
			return fmt.Errorf("failed to create output file: %w", err)
		}
		defer file.Close()
		output = file
	}

	// Setup chat handler
	handler, err := setupChatHandler(ctx)
	if err != nil {
		return fmt.Errorf("failed to setup chat handler: %w", err)
	}

	// Create a custom stream handler that writes to output
	streamHandler := &nonInteractiveStreamHandler{
		output: output,
	}

	// Process the prompt
	if err := processNonInteractivePrompt(ctx, handler, finalPrompt, streamHandler); err != nil {
		return fmt.Errorf("failed to process prompt: %w", err)
	}

	// Add newline at the end if writing to stdout
	if output == os.Stdout {
		fmt.Fprintln(output)
	}

	return nil
}

func constructPrompt(userPrompt, additionalInput, filename string) string {
	parts := []string{}

	if userPrompt != "" {
		parts = append(parts, userPrompt)
	}

	if additionalInput != "" {
		if filename != "" {
			parts = append(parts, fmt.Sprintf("\n\nFile: %s\n```\n%s\n```", filename, additionalInput))
		} else {
			parts = append(parts, fmt.Sprintf("\n\nInput:\n```\n%s\n```", additionalInput))
		}
	}

	return strings.Join(parts, "\n")
}

func processNonInteractivePrompt(ctx context.Context, handler *chat.ChatHandler, prompt string, streamHandler *nonInteractiveStreamHandler) error {
	// Override the default stream handler
	// This would need to be implemented in the handler
	// For now, we'll use the standard handler
	return handler.HandleMessage(ctx, prompt)
}

// nonInteractiveStreamHandler handles streaming output for non-interactive mode
type nonInteractiveStreamHandler struct {
	output io.Writer
}

func (h *nonInteractiveStreamHandler) Write(p []byte) (n int, err error) {
	return h.output.Write(p)
}

// isTerminal checks if the file descriptor is a terminal
func isTerminal(f *os.File) bool {
	// Simple check - in production, use golang.org/x/term
	stat, err := f.Stat()
	if err != nil {
		return true
	}
	return (stat.Mode() & os.ModeCharDevice) != 0
}
