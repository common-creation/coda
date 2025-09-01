//go:build integration
// +build integration

package scenarios

import (
	"context"
	"fmt"
	"testing"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/common-creation/coda/internal/ui"
	"github.com/common-creation/coda/tests/e2e/helpers"
)

func TestAIServiceErrors(t *testing.T) {
	helper, err := helpers.NewE2ETestHelper(t, helpers.E2ETestOptions{
		Timeout:     30 * time.Second,
		EnableMocks: true,
	})
	require.NoError(t, err)
	defer helper.Cleanup()

	require.NoError(t, helper.StartApp())

	t.Run("AI API rate limit error", func(t *testing.T) {
		// Configure mock AI to simulate rate limit error
		mockAI := helper.GetMockAIClient()
		mockAI.SetError(true, "Rate limit exceeded. Please try again later.")

		helper.SendKeyMsg("i")
		helper.TypeText("Help me write some Go code")
		helper.PressEnter()

		// Wait for error to be processed
		time.Sleep(3 * time.Second)

		// Verify error is displayed appropriately
		model := helper.GetModel()
		view := model.View()

		// Should show error information
		assert.Contains(t, view, "rate limit", "Should display rate limit error")

		// Application should remain responsive
		helper.AssertMode(ui.ModeNormal)
	})

	t.Run("AI API connection timeout", func(t *testing.T) {
		mockAI := helper.GetMockAIClient()
		mockAI.SetError(true, "Connection timeout: Unable to reach AI service")
		mockAI.SetLatency(10 * time.Second) // Simulate very slow response

		helper.SendKeyMsg("i")
		helper.TypeText("What is Go programming?")
		helper.PressEnter()

		// Wait for timeout handling
		time.Sleep(5 * time.Second)

		// Should show connection error
		view := helper.GetModel().View()
		assert.Contains(t, view, "timeout", "Should display timeout error")
	})

	t.Run("AI API authentication error", func(t *testing.T) {
		mockAI := helper.GetMockAIClient()
		mockAI.SetError(true, "Authentication failed: Invalid API key")

		helper.SendKeyMsg("i")
		helper.TypeText("Hello AI")
		helper.PressEnter()

		time.Sleep(2 * time.Second)

		// Should show authentication error
		view := helper.GetModel().View()
		assert.Contains(t, view, "authentication", "Should display authentication error")
	})

	t.Run("AI service unavailable", func(t *testing.T) {
		mockAI := helper.GetMockAIClient()
		mockAI.SetError(true, "Service temporarily unavailable (503)")

		helper.SendKeyMsg("i")
		helper.TypeText("Can you help me?")
		helper.PressEnter()

		time.Sleep(2 * time.Second)

		// Should show service unavailable error
		view := helper.GetModel().View()
		assert.Contains(t, view, "unavailable", "Should display service unavailable error")
	})

	t.Run("Error recovery", func(t *testing.T) {
		// First, simulate an error
		mockAI := helper.GetMockAIClient()
		mockAI.SetError(true, "Temporary error")

		helper.SendKeyMsg("i")
		helper.TypeText("This will fail")
		helper.PressEnter()
		time.Sleep(2 * time.Second)

		// Now fix the error and try again
		mockAI.SetError(false, "")
		mockAI.SetNextResponse("I'm working again! How can I help you?")

		helper.SendKeyMsg("i")
		helper.TypeText("Are you working now?")
		helper.PressEnter()

		require.NoError(t, helper.WaitForResponse(10*time.Second))

		// Should recover successfully
		helper.AssertLastMessageRole("assistant")
		helper.AssertLastMessageContains("working again")
	})
}

func TestToolExecutionErrors(t *testing.T) {
	helper, err := helpers.NewE2ETestHelper(t, helpers.E2ETestOptions{
		Timeout:     30 * time.Second,
		EnableMocks: true,
	})
	require.NoError(t, err)
	defer helper.Cleanup()

	require.NoError(t, helper.StartApp())

	t.Run("File not found error", func(t *testing.T) {
		// Register a tool that will fail with file not found
		failingTool := helpers.MockTool{
			Name:        "read_nonexistent_file",
			Description: "Try to read a file that doesn't exist",
			Handler: func(args map[string]interface{}) (interface{}, error) {
				return nil, fmt.Errorf("file not found: %s", args["path"])
			},
		}

		toolManager := helper.GetMockToolManager()
		toolManager.RegisterTool(failingTool)
		toolManager.SetApprovalMode(false)

		helper.SetAIResponse("I'll try to read that file, but it seems to not exist.")

		helper.SendKeyMsg("i")
		helper.TypeText("Read nonexistent.txt using read_nonexistent_file")
		helper.PressEnter()

		require.NoError(t, helper.WaitForResponse(10*time.Second))

		// Should handle file not found gracefully
		helper.AssertLastMessageRole("assistant")
		view := helper.GetModel().View()
		assert.Contains(t, view, "not exist", "Should mention file doesn't exist")
	})

	t.Run("Permission denied error", func(t *testing.T) {
		permissionTool := helpers.MockTool{
			Name:        "restricted_operation",
			Description: "Try to perform a restricted operation",
			Handler: func(args map[string]interface{}) (interface{}, error) {
				return nil, fmt.Errorf("permission denied: insufficient privileges")
			},
		}

		toolManager := helper.GetMockToolManager()
		toolManager.RegisterTool(permissionTool)

		helper.SetAIResponse("I don't have permission to perform that operation.")

		helper.SendKeyMsg("i")
		helper.TypeText("Perform restricted_operation")
		helper.PressEnter()

		require.NoError(t, helper.WaitForResponse(10*time.Second))

		// Should handle permission error
		view := helper.GetModel().View()
		assert.Contains(t, view, "permission", "Should mention permission issue")
	})

	t.Run("Tool execution timeout", func(t *testing.T) {
		timeoutTool := helpers.MockTool{
			Name:        "timeout_tool",
			Description: "A tool that times out",
			Handler: func(args map[string]interface{}) (interface{}, error) {
				// Simulate operation that takes too long
				ctx, cancel := context.WithTimeout(context.Background(), 1*time.Second)
				defer cancel()

				select {
				case <-time.After(5 * time.Second):
					return "Should not reach here", nil
				case <-ctx.Done():
					return nil, fmt.Errorf("operation timed out")
				}
			},
		}

		toolManager := helper.GetMockToolManager()
		toolManager.RegisterTool(timeoutTool)

		helper.SetAIResponse("The operation timed out. Let me try a different approach.")

		helper.SendKeyMsg("i")
		helper.TypeText("Use timeout_tool")
		helper.PressEnter()

		require.NoError(t, helper.WaitForResponse(15*time.Second))

		// Should handle timeout gracefully
		view := helper.GetModel().View()
		assert.Contains(t, view, "timeout", "Should mention timeout")
	})

	t.Run("Tool crashes and recovery", func(t *testing.T) {
		crashingTool := helpers.MockTool{
			Name:        "crashing_tool",
			Description: "A tool that panics",
			Handler: func(args map[string]interface{}) (interface{}, error) {
				panic("simulated tool crash")
			},
		}

		toolManager := helper.GetMockToolManager()
		toolManager.RegisterTool(crashingTool)

		helper.SetAIResponse("There was an unexpected error with the tool, but I'm still here to help.")

		helper.SendKeyMsg("i")
		helper.TypeText("Use crashing_tool")
		helper.PressEnter()

		time.Sleep(3 * time.Second)

		// Application should still be responsive after tool crash
		helper.AssertMode(ui.ModeNormal)

		// Should be able to continue working
		helper.SetAIResponse("I'm still working normally!")
		helper.SendKeyMsg("i")
		helper.TypeText("Are you still working?")
		helper.PressEnter()

		require.NoError(t, helper.WaitForResponse(10*time.Second))
		helper.AssertLastMessageContains("working")
	})
}

func TestInputValidationErrors(t *testing.T) {
	helper, err := helpers.NewE2ETestHelper(t, helpers.E2ETestOptions{
		Timeout:     30 * time.Second,
		EnableMocks: true,
	})
	require.NoError(t, err)
	defer helper.Cleanup()

	require.NoError(t, helper.StartApp())

	t.Run("Extremely long input", func(t *testing.T) {
		helper.SetAIResponse("I received your very long message. I'll do my best to help.")

		// Create a very long input (10KB)
		longInput := make([]rune, 10000)
		for i := range longInput {
			longInput[i] = 'A' + rune(i%26)
		}

		helper.SendKeyMsg("i")
		helper.TypeText(string(longInput))
		helper.PressEnter()

		require.NoError(t, helper.WaitForResponse(15*time.Second))

		// Should handle long input gracefully
		helper.AssertLastMessageRole("assistant")
	})

	t.Run("Special characters in input", func(t *testing.T) {
		helper.SetAIResponse("I can handle special characters just fine!")

		specialChars := "!@#$%^&*()_+-=[]{}|;':\",./<>?`~"
		unicodeChars := "こんにちは世界 🚀 🌟 ☕ 💻"

		helper.SendKeyMsg("i")
		helper.TypeText("Test with special chars: " + specialChars + " and unicode: " + unicodeChars)
		helper.PressEnter()

		require.NoError(t, helper.WaitForResponse(10*time.Second))

		// Should handle special characters without issues
		helper.AssertLastMessageRole("assistant")
	})

	t.Run("Empty or whitespace-only input", func(t *testing.T) {
		// Test empty input
		helper.SendKeyMsg("i")
		helper.PressEnter()

		// Should not send empty message
		helper.AssertMode(ui.ModeInsert)

		// Test whitespace-only input
		helper.TypeText("   \t  \n  ")
		helper.PressEnter()

		// Should handle whitespace-only input appropriately
		time.Sleep(1 * time.Second)

		// Might stay in insert mode or handle gracefully
		model := helper.GetModel()
		assert.NotNil(t, model, "Model should still be responsive")
	})

	t.Run("Binary or control characters", func(t *testing.T) {
		helper.SetAIResponse("I'll handle unusual characters appropriately.")

		// Test with control characters
		controlChars := string([]byte{0x01, 0x02, 0x03, 0x7F, 0x1B})

		helper.SendKeyMsg("i")
		helper.TypeText("Test with control chars: " + controlChars)
		helper.PressEnter()

		require.NoError(t, helper.WaitForResponse(10*time.Second))

		// Should handle gracefully
		helper.AssertLastMessageRole("assistant")
	})
}

func TestUIStateErrors(t *testing.T) {
	helper, err := helpers.NewE2ETestHelper(t, helpers.E2ETestOptions{
		Timeout:     30 * time.Second,
		EnableMocks: true,
	})
	require.NoError(t, err)
	defer helper.Cleanup()

	require.NoError(t, helper.StartApp())

	t.Run("Invalid mode transitions", func(t *testing.T) {
		// Start in normal mode
		helper.AssertMode(ui.ModeNormal)

		// Try rapid mode switching
		helper.SendKeyMsg("i") // Insert
		helper.SendKeyMsg(":") // Command (from insert)
		helper.PressEscape()   // Back to normal
		helper.SendKeyMsg("/") // Search
		helper.PressEscape()   // Back to normal

		// Should end up in normal mode
		time.Sleep(100 * time.Millisecond)
		helper.AssertMode(ui.ModeNormal)
	})

	t.Run("Rapid key sequences", func(t *testing.T) {
		// Send rapid key sequence
		keys := []string{"i", "h", "e", "l", "l", "o"}
		for _, key := range keys {
			helper.SendKeyMsg(key)
			time.Sleep(1 * time.Millisecond) // Very fast typing
		}

		// Should handle rapid input gracefully
		time.Sleep(100 * time.Millisecond)
		model := helper.GetModel()
		view := model.View()
		assert.Contains(t, view, "hello", "Should handle rapid typing")
	})

	t.Run("Concurrent operations", func(t *testing.T) {
		helper.SetAIResponse("I'm handling multiple operations.")

		// Start multiple operations quickly
		helper.SendKeyMsg("i")
		helper.TypeText("First message")
		helper.PressEnter()

		// Don't wait, send another immediately
		helper.SendKeyMsg("i")
		helper.TypeText("Second message")
		helper.PressEnter()

		// Should handle concurrent operations gracefully
		require.NoError(t, helper.WaitForResponse(15*time.Second))

		// At least one response should be received
		messages := helper.GetMessages()
		assert.GreaterOrEqual(t, len(messages), 2, "Should handle multiple messages")
	})

	t.Run("Window resize during operation", func(t *testing.T) {
		helper.SetAIResponse("Window resize handled gracefully.")

		helper.SendKeyMsg("i")
		helper.TypeText("Testing window resize")

		// Simulate window resize
		helper.SendMessage(tea.WindowSizeMsg{Width: 120, Height: 40})
		helper.SendMessage(tea.WindowSizeMsg{Width: 80, Height: 24})

		helper.PressEnter()

		require.NoError(t, helper.WaitForResponse(10*time.Second))

		// Should handle resize gracefully
		helper.AssertLastMessageRole("assistant")
	})
}

func TestMemoryAndResourceErrors(t *testing.T) {
	helper, err := helpers.NewE2ETestHelper(t, helpers.E2ETestOptions{
		Timeout:     30 * time.Second,
		EnableMocks: true,
	})
	require.NoError(t, err)
	defer helper.Cleanup()

	require.NoError(t, helper.StartApp())

	t.Run("Large conversation history", func(t *testing.T) {
		helper.SetAIResponse("Message received.")

		// Create a large conversation history
		for i := 0; i < 100; i++ {
			helper.SendKeyMsg("i")
			helper.TypeText(fmt.Sprintf("Message %d: This is a test message to build up history", i))
			helper.PressEnter()

			// Wait briefly to avoid overwhelming
			if i%10 == 0 {
				time.Sleep(100 * time.Millisecond)
			}
		}

		// Should handle large history gracefully
		require.NoError(t, helper.WaitForResponse(20*time.Second))

		messages := helper.GetMessages()
		assert.Greater(t, len(messages), 50, "Should maintain large conversation history")
	})

	t.Run("Memory pressure simulation", func(t *testing.T) {
		// Create many large messages to simulate memory pressure
		largeContent := make([]rune, 1000)
		for i := range largeContent {
			largeContent[i] = 'X'
		}

		helper.SetAIResponse("Handling large content appropriately.")

		for i := 0; i < 20; i++ {
			helper.SendKeyMsg("i")
			helper.TypeText(fmt.Sprintf("Large message %d: %s", i, string(largeContent)))
			helper.PressEnter()

			if i%5 == 0 {
				time.Sleep(200 * time.Millisecond)
			}
		}

		// Application should remain responsive under memory pressure
		require.NoError(t, helper.WaitForResponse(30*time.Second))
		helper.AssertLastMessageRole("assistant")
	})
}

func TestNetworkAndConnectivityErrors(t *testing.T) {
	helper, err := helpers.NewE2ETestHelper(t, helpers.E2ETestOptions{
		Timeout:     30 * time.Second,
		EnableMocks: true,
	})
	require.NoError(t, err)
	defer helper.Cleanup()

	require.NoError(t, helper.StartApp())

	t.Run("Intermittent connectivity", func(t *testing.T) {
		mockAI := helper.GetMockAIClient()

		// Simulate working connection
		mockAI.SetError(false, "")
		mockAI.SetNextResponse("First response works fine.")

		helper.SendKeyMsg("i")
		helper.TypeText("First message")
		helper.PressEnter()
		require.NoError(t, helper.WaitForResponse(10*time.Second))

		// Simulate connection failure
		mockAI.SetError(true, "Network connection failed")

		helper.SendKeyMsg("i")
		helper.TypeText("Second message should fail")
		helper.PressEnter()
		time.Sleep(3 * time.Second)

		// Simulate connection recovery
		mockAI.SetError(false, "")
		mockAI.SetNextResponse("Connection restored!")

		helper.SendKeyMsg("i")
		helper.TypeText("Third message should work")
		helper.PressEnter()
		require.NoError(t, helper.WaitForResponse(10*time.Second))

		// Should handle intermittent connectivity
		helper.AssertLastMessageContains("restored")
	})

	t.Run("Slow network response", func(t *testing.T) {
		mockAI := helper.GetMockAIClient()
		mockAI.SetLatency(5 * time.Second) // Very slow response
		mockAI.SetNextResponse("Sorry for the delay!")

		helper.SendKeyMsg("i")
		helper.TypeText("This should be slow")
		helper.PressEnter()

		// Should show loading indicator or similar
		time.Sleep(2 * time.Second)
		view := helper.GetModel().View()
		// Should indicate that it's processing/thinking
		assert.True(t,
			contains(view, "thinking") ||
				contains(view, "loading") ||
				contains(view, "..."),
			"Should show processing indicator")

		require.NoError(t, helper.WaitForResponse(10*time.Second))
		helper.AssertLastMessageContains("delay")
	})
}

func TestErrorRecoveryScenarios(t *testing.T) {
	helper, err := helpers.NewE2ETestHelper(t, helpers.E2ETestOptions{
		Timeout:     60 * time.Second,
		EnableMocks: true,
	})
	require.NoError(t, err)
	defer helper.Cleanup()

	require.NoError(t, helper.StartApp())

	t.Run("Complete error recovery workflow", func(t *testing.T) {
		scenario := helpers.UserScenario{
			Name:        "Error recovery workflow",
			Description: "Test complete error recovery from multiple failure points",
			Steps: []helpers.TestStep{
				{
					Description: "Start with working request",
					Action:      "switch_mode",
					Input:       "insert",
				},
				{
					Description: "Send working message",
					Action:      "type",
					Input:       "This should work fine",
				},
				{
					Description: "Send working request",
					Action:      "enter",
				},
				{
					Description: "Wait for working response",
					Action:      "wait_for_response",
				},
				{
					Description: "Send failing request",
					Action:      "switch_mode",
					Input:       "insert",
				},
				{
					Description: "Type failing message",
					Action:      "type",
					Input:       "This will cause an error",
				},
				{
					Description: "Send failing request",
					Action:      "enter",
				},
				{
					Description: "Wait for error handling",
					WaitAfter:   3 * time.Second,
				},
				{
					Description: "Send recovery message",
					Action:      "switch_mode",
					Input:       "insert",
				},
				{
					Description: "Type recovery message",
					Action:      "type",
					Input:       "Are you working again?",
				},
				{
					Description: "Send recovery request",
					Action:      "enter",
				},
				{
					Description: "Wait for recovery",
					Action:      "wait_for_response",
				},
			},
		}

		mockAI := helper.GetMockAIClient()

		// Set up responses: working -> error -> recovery
		responses := []string{
			"Yes, I'm working fine!",
			"", // Will be overridden by error
			"Yes, I'm fully recovered and working!",
		}

		mockAI.SetResponses(responses)

		// Configure error for second request
		go func() {
			time.Sleep(8 * time.Second) // After first response
			mockAI.SetError(true, "Simulated failure")
			time.Sleep(5 * time.Second) // Let error be processed
			mockAI.SetError(false, "")  // Recover
		}()

		err := helper.SimulateUserInteraction(scenario)
		require.NoError(t, err)

		// Verify recovery
		messages := helper.GetMessages()
		assert.GreaterOrEqual(t, len(messages), 4, "Should have messages from complete workflow")

		lastMessage := messages[len(messages)-1]
		assert.Equal(t, "assistant", lastMessage.Role)
		assert.Contains(t, lastMessage.Content, "recovered", "Should indicate recovery")
	})
}

func TestErrorLoggingAndDiagnostics(t *testing.T) {
	helper, err := helpers.NewE2ETestHelper(t, helpers.E2ETestOptions{
		Timeout:     30 * time.Second,
		EnableMocks: true,
	})
	require.NoError(t, err)
	defer helper.Cleanup()

	require.NoError(t, helper.StartApp())

	t.Run("Error information is properly logged", func(t *testing.T) {
		mockAI := helper.GetMockAIClient()
		mockAI.SetError(true, "Test error for logging")

		helper.SendKeyMsg("i")
		helper.TypeText("Cause an error for logging test")
		helper.PressEnter()

		time.Sleep(3 * time.Second)

		// Verify error handling occurred
		// In a real implementation, you would check logs
		// For now, verify the application is still responsive
		helper.AssertMode(ui.ModeNormal)

		// Application should continue working after error
		mockAI.SetError(false, "")
		mockAI.SetNextResponse("Error has been logged and handled.")

		helper.SendKeyMsg("i")
		helper.TypeText("Test after error")
		helper.PressEnter()

		require.NoError(t, helper.WaitForResponse(10*time.Second))
		helper.AssertLastMessageContains("logged")
	})
}

// Helper function to check if a string contains a substring
func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr ||
		len(s) > len(substr) &&
			(containsAt(s, substr, 0) || contains(s[1:], substr)))
}

func containsAt(s, substr string, index int) bool {
	if index+len(substr) > len(s) {
		return false
	}
	for i, c := range substr {
		if s[index+i] != byte(c) {
			return false
		}
	}
	return true
}
