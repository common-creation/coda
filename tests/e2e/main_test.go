package e2e

import (
	"flag"
	"fmt"
	"os"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/common-creation/coda/tests/e2e/helpers"
	"github.com/common-creation/coda/tests/e2e/scenarios"
)

var (
	// Test configuration flags
	testTimeout = flag.Duration("test-timeout", 60*time.Second, "Default timeout for E2E tests")
	verbose     = flag.Bool("verbose", false, "Enable verbose test output")
	skipSlow    = flag.Bool("skip-slow", false, "Skip slow-running tests")
	testFilter  = flag.String("filter", "", "Run only tests matching this pattern")
)

// TestMain provides setup and teardown for the entire E2E test suite
func TestMain(m *testing.M) {
	flag.Parse()

	// Setup test environment
	if err := setupTestEnvironment(); err != nil {
		fmt.Fprintf(os.Stderr, "Failed to setup test environment: %v\n", err)
		os.Exit(1)
	}

	// Run tests
	code := m.Run()

	// Cleanup test environment
	if err := cleanupTestEnvironment(); err != nil {
		fmt.Fprintf(os.Stderr, "Failed to cleanup test environment: %v\n", err)
	}

	os.Exit(code)
}

// setupTestEnvironment initializes the test environment
func setupTestEnvironment() error {
	if *verbose {
		fmt.Println("Setting up E2E test environment...")
	}

	// Create temporary directories for tests
	if err := os.MkdirAll("tmp/e2e", 0755); err != nil {
		return fmt.Errorf("failed to create temp directory: %w", err)
	}

	// Set environment variables for testing
	os.Setenv("CODA_TEST_MODE", "true")
	os.Setenv("CODA_LOG_LEVEL", "debug")

	if *verbose {
		fmt.Println("E2E test environment setup complete")
	}

	return nil
}

// cleanupTestEnvironment cleans up after all tests
func cleanupTestEnvironment() error {
	if *verbose {
		fmt.Println("Cleaning up E2E test environment...")
	}

	// Remove temporary directories
	if err := os.RemoveAll("tmp/e2e"); err != nil {
		return fmt.Errorf("failed to remove temp directory: %w", err)
	}

	if *verbose {
		fmt.Println("E2E test environment cleanup complete")
	}

	return nil
}

// TestE2EApplicationStartup tests basic application startup and shutdown
func TestE2EApplicationStartup(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping E2E test in short mode")
	}

	helper, err := helpers.NewE2ETestHelper(t, helpers.E2ETestOptions{
		Timeout:     *testTimeout,
		EnableMocks: true,
	})
	require.NoError(t, err)
	defer helper.Cleanup()

	t.Run("Application starts successfully", func(t *testing.T) {
		require.NoError(t, helper.StartApp())

		// Verify app is running
		assert.True(t, helper.GetModel().GetCurrentMode() != 0, "App should be initialized")
	})

	t.Run("Application shuts down gracefully", func(t *testing.T) {
		require.NoError(t, helper.StartApp())
		require.NoError(t, helper.Shutdown())
	})
}

// TestE2EFullWorkflow tests a complete user workflow
func TestE2EFullWorkflow(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping E2E test in short mode")
	}

	if *skipSlow {
		t.Skip("Skipping slow E2E test")
	}

	helper, err := helpers.NewE2ETestHelper(t, helpers.E2ETestOptions{
		Timeout:     2 * *testTimeout, // Double timeout for complex workflow
		EnableMocks: true,
	})
	require.NoError(t, err)
	defer helper.Cleanup()

	// Create test project
	require.NoError(t, helper.CreateTestFile("main.go", `package main

import "fmt"

func main() {
	fmt.Println("Hello, World!")
}
`))

	require.NoError(t, helper.StartApp())

	scenario := helpers.UserScenario{
		Name:        "Complete development workflow",
		Description: "User performs a complete coding session with CODA",
		Steps: []helpers.TestStep{
			{
				Description: "Greet the AI",
				Action:      "switch_mode",
				Input:       "insert",
			},
			{
				Description: "Send greeting",
				Action:      "type",
				Input:       "Hello! I need help with my Go project.",
			},
			{
				Description: "Send greeting message",
				Action:      "enter",
			},
			{
				Description: "Wait for greeting response",
				Action:      "wait_for_response",
			},
			{
				Description: "Request file analysis",
				Action:      "switch_mode",
				Input:       "insert",
			},
			{
				Description: "Ask for code review",
				Action:      "type",
				Input:       "Please read and review my main.go file.",
			},
			{
				Description: "Send review request",
				Action:      "enter",
			},
			{
				Description: "Wait for code review",
				Action:      "wait_for_response",
			},
			{
				Description: "Request improvements",
				Action:      "switch_mode",
				Input:       "insert",
			},
			{
				Description: "Ask for improvements",
				Action:      "type",
				Input:       "Can you suggest some improvements to make this more robust?",
			},
			{
				Description: "Send improvement request",
				Action:      "enter",
			},
			{
				Description: "Wait for improvement suggestions",
				Action:      "wait_for_response",
			},
		},
	}

	// Set up AI responses
	responses := []string{
		"Hello! I'd be happy to help you with your Go project. What would you like to work on?",
		"I'll read your main.go file and provide a code review.",
		"Here are some improvements I suggest: Add error handling, use proper logging, and consider adding tests.",
	}

	for i, response := range responses {
		if i == 0 {
			helper.SetAIResponse(response)
		} else {
			// For subsequent responses, we'd need a more sophisticated mock
			// that can handle sequential responses
			helper.SetAIResponse(response)
		}
	}

	// Execute the complete workflow
	err = helper.SimulateUserInteraction(scenario)
	require.NoError(t, err)

	// Verify the workflow completed successfully
	messages := helper.GetModel().GetMessages()
	assert.GreaterOrEqual(t, len(messages), 6, "Should have completed full conversation")

	// Verify tool usage
	toolManager := helper.GetMockToolManager()
	calls := toolManager.GetCallHistory()
	assert.Greater(t, len(calls), 0, "Should have used tools during the workflow")
}

// TestE2EErrorRecovery tests error recovery scenarios
func TestE2EErrorRecovery(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping E2E test in short mode")
	}

	helper, err := helpers.NewE2ETestHelper(t, helpers.E2ETestOptions{
		Timeout:     *testTimeout,
		EnableMocks: true,
	})
	require.NoError(t, err)
	defer helper.Cleanup()

	require.NoError(t, helper.StartApp())

	// Test AI service error recovery
	t.Run("AI service recovery", func(t *testing.T) {
		mockAI := helper.GetMockAIClient()

		// Simulate service failure
		mockAI.SetError(true, "Service temporarily unavailable")

		helper.SendMessage(helper.GetModel())
		helper.TypeText("This should fail")
		helper.PressEnter()

		time.Sleep(2 * time.Second)

		// Recover service
		mockAI.SetError(false, "")
		mockAI.SetNextResponse("I'm back online and ready to help!")

		helper.SendMessage(helper.GetModel())
		helper.TypeText("Are you working now?")
		helper.PressEnter()

		require.NoError(t, helper.WaitForResponse(10*time.Second))
		helper.AssertLastMessageContains("back online")
	})
}

// TestE2EPerformance tests performance-related scenarios
func TestE2EPerformance(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping E2E test in short mode")
	}

	if *skipSlow {
		t.Skip("Skipping slow performance test")
	}

	helper, err := helpers.NewE2ETestHelper(t, helpers.E2ETestOptions{
		Timeout:     3 * *testTimeout, // Extended timeout for performance tests
		EnableMocks: true,
	})
	require.NoError(t, err)
	defer helper.Cleanup()

	require.NoError(t, helper.StartApp())

	t.Run("Large conversation handling", func(t *testing.T) {
		helper.SetAIResponse("Message processed.")

		// Send many messages to test performance
		for i := 0; i < 50; i++ {
			helper.SendMessage(helper.GetModel())
			helper.TypeText(fmt.Sprintf("Performance test message %d", i))
			helper.PressEnter()

			if i%10 == 0 {
				time.Sleep(100 * time.Millisecond) // Brief pause every 10 messages
			}
		}

		// Verify all messages were handled
		require.NoError(t, helper.WaitForResponse(30*time.Second))
		messages := helper.GetModel().GetMessages()
		assert.GreaterOrEqual(t, len(messages), 50, "Should handle large number of messages")
	})

	t.Run("Rapid input handling", func(t *testing.T) {
		helper.SetAIResponse("Rapid input handled successfully.")

		// Test rapid typing
		text := "This is a test of rapid input handling with lots of text to simulate fast typing"
		for _, char := range text {
			helper.SendKeyMsg(string(char))
			time.Sleep(1 * time.Millisecond) // Very fast typing
		}

		helper.PressEnter()
		require.NoError(t, helper.WaitForResponse(10*time.Second))

		// Verify rapid input was handled correctly
		messages := helper.GetModel().GetMessages()
		if len(messages) > 0 {
			lastUserMessage := messages[len(messages)-2] // Second to last should be user message
			assert.Contains(t, lastUserMessage.Content, "rapid input", "Should contain the typed text")
		}
	})
}

// benchmarkE2EOperations runs benchmark tests for E2E operations
func BenchmarkE2EOperations(b *testing.B) {
	helper, err := helpers.NewE2ETestHelper(&testing.T{}, helpers.E2ETestOptions{
		Timeout:     30 * time.Second,
		EnableMocks: true,
	})
	if err != nil {
		b.Fatalf("Failed to create test helper: %v", err)
	}
	defer helper.Cleanup()

	if err := helper.StartApp(); err != nil {
		b.Fatalf("Failed to start app: %v", err)
	}

	helper.SetAIResponse("Benchmark response")

	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		helper.SendMessage(helper.GetModel())
		helper.TypeText(fmt.Sprintf("Benchmark message %d", i))
		helper.PressEnter()

		// Don't wait for response in benchmark to measure throughput
	}
}

// Example test demonstrating best practices
func ExampleE2ETest() {
	// This example shows how to write a comprehensive E2E test

	// Create test helper with appropriate timeout and mocks
	helper, err := helpers.NewE2ETestHelper(&testing.T{}, helpers.E2ETestOptions{
		Timeout:     30 * time.Second,
		EnableMocks: true,
	})
	if err != nil {
		fmt.Printf("Failed to create helper: %v\n", err)
		return
	}
	defer helper.Cleanup()

	// Set up test data
	helper.CreateTestFile("example.go", "package main\n\nfunc main() {\n\tprintln(\"Hello!\")\n}")

	// Configure expected behavior
	helper.SetAIResponse("I can help you improve this Go code.")

	// Start the application
	if err := helper.StartApp(); err != nil {
		fmt.Printf("Failed to start app: %v\n", err)
		return
	}

	// Simulate user interaction
	scenario := helpers.UserScenario{
		Name: "Code improvement request",
		Steps: []helpers.TestStep{
			{Action: "switch_mode", Input: "insert"},
			{Action: "type", Input: "Please review example.go"},
			{Action: "enter"},
			{Action: "wait_for_response"},
		},
	}

	if err := helper.SimulateUserInteraction(scenario); err != nil {
		fmt.Printf("Scenario failed: %v\n", err)
		return
	}

	// Verify results
	messages := helper.GetModel().GetMessages()
	if len(messages) >= 2 {
		fmt.Println("Test completed successfully")
	}

	// Output: Test completed successfully
}
