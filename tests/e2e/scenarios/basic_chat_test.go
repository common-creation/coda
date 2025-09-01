//go:build integration
// +build integration

package scenarios

import (
	"testing"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/common-creation/coda/internal/ui"
	"github.com/common-creation/coda/tests/e2e/helpers"
)

func TestBasicChatFlow(t *testing.T) {
	helper, err := helpers.NewE2ETestHelper(t, helpers.E2ETestOptions{
		Timeout:     30 * time.Second,
		EnableMocks: true,
	})
	require.NoError(t, err)
	defer helper.Cleanup()

	// Set up expected AI response
	helper.SetAIResponse("Hello! I'm Claude, your coding assistant. How can I help you today?")

	// Start the application
	require.NoError(t, helper.StartApp())

	// Test basic conversation flow
	t.Run("User sends message and receives response", func(t *testing.T) {
		// Switch to insert mode
		helper.SendKeyMsg("i")
		helper.AssertMode(ui.ModeInsert)

		// Type a message
		helper.TypeText("Hello, can you help me with Go programming?")

		// Send the message
		helper.PressEnter()

		// Wait for response
		require.NoError(t, helper.WaitForResponse(10*time.Second))

		// Verify the conversation
		helper.AssertMessageCount(2) // User message + AI response
		helper.AssertLastMessageRole("assistant")
		helper.AssertLastMessageContains("Claude")

		// Should be back in normal mode after sending
		helper.AssertMode(ui.ModeNormal)
	})
}

func TestMultipleMessages(t *testing.T) {
	helper, err := helpers.NewE2ETestHelper(t, helpers.E2ETestOptions{
		Timeout:     30 * time.Second,
		EnableMocks: true,
	})
	require.NoError(t, err)
	defer helper.Cleanup()

	// Set up multiple AI responses
	helper.SetAIResponse("I can help you with Go! What specific topic would you like to learn about?")

	require.NoError(t, helper.StartApp())

	// Send first message
	helper.SendKeyMsg("i")
	helper.TypeText("I want to learn Go")
	helper.PressEnter()
	require.NoError(t, helper.WaitForResponse(5*time.Second))

	// Set next response
	helper.SetAIResponse("Goroutines are concurrent functions in Go. Here's a simple example...")

	// Send second message
	helper.SendKeyMsg("i")
	helper.TypeText("Tell me about goroutines")
	helper.PressEnter()
	require.NoError(t, helper.WaitForResponse(5*time.Second))

	// Verify conversation history
	helper.AssertMessageCount(4) // 2 user messages + 2 AI responses
	messages := helper.GetMessages()

	assert.Equal(t, "user", messages[0].Role)
	assert.Contains(t, messages[0].Content, "learn Go")

	assert.Equal(t, "assistant", messages[1].Role)
	assert.Contains(t, messages[1].Content, "Go!")

	assert.Equal(t, "user", messages[2].Role)
	assert.Contains(t, messages[2].Content, "goroutines")

	assert.Equal(t, "assistant", messages[3].Role)
	assert.Contains(t, messages[3].Content, "Goroutines")
}

func TestChatModes(t *testing.T) {
	helper, err := helpers.NewE2ETestHelper(t, helpers.E2ETestOptions{
		Timeout:     30 * time.Second,
		EnableMocks: true,
	})
	require.NoError(t, err)
	defer helper.Cleanup()

	require.NoError(t, helper.StartApp())

	t.Run("Normal mode navigation", func(t *testing.T) {
		// Should start in normal mode
		helper.AssertMode(ui.ModeNormal)

		// Test mode switching
		helper.SendKeyMsg("i")
		helper.AssertMode(ui.ModeInsert)

		helper.PressEscape()
		helper.AssertMode(ui.ModeNormal)
	})

	t.Run("Insert mode text input", func(t *testing.T) {
		helper.SendKeyMsg("i")
		helper.AssertMode(ui.ModeInsert)

		// Type some text
		helper.TypeText("Hello world")

		// Text should be visible in input
		helper.AssertOutput("Hello world")

		// Clear and try backspace
		for i := 0; i < 11; i++ {
			helper.PressKey(tea.KeyBackspace)
		}

		helper.TypeText("New text")
		helper.AssertOutput("New text")
	})

	t.Run("Command mode", func(t *testing.T) {
		helper.PressEscape() // Ensure we're in normal mode
		helper.SendKeyMsg(":")
		helper.AssertMode(ui.ModeCommand)

		// Command buffer should show ":"
		helper.AssertOutput(":")

		// Type a command
		helper.TypeText("help")
		helper.AssertOutput(":help")

		// Execute command
		helper.PressEnter()
		helper.AssertMode(ui.ModeNormal)
	})

	t.Run("Search mode", func(t *testing.T) {
		helper.PressEscape() // Ensure we're in normal mode
		helper.SendKeyMsg("/")
		helper.AssertMode(ui.ModeSearch)

		// Search buffer should show "/"
		helper.AssertOutput("/")

		// Type search query
		helper.TypeText("test")
		helper.AssertOutput("/test")

		// Execute search
		helper.PressEnter()
		helper.AssertMode(ui.ModeNormal)
	})
}

func TestKeyboardShortcuts(t *testing.T) {
	helper, err := helpers.NewE2ETestHelper(t, helpers.E2ETestOptions{
		Timeout:     30 * time.Second,
		EnableMocks: true,
	})
	require.NoError(t, err)
	defer helper.Cleanup()

	helper.SetAIResponse("This is a test message")
	require.NoError(t, helper.StartApp())

	// Add some messages first
	helper.SendKeyMsg("i")
	helper.TypeText("Test message 1")
	helper.PressEnter()
	require.NoError(t, helper.WaitForResponse(5*time.Second))

	helper.SendKeyMsg("i")
	helper.TypeText("Test message 2")
	helper.PressEnter()
	require.NoError(t, helper.WaitForResponse(5*time.Second))

	helper.AssertMessageCount(4) // 2 user + 2 assistant

	t.Run("Clear history shortcut", func(t *testing.T) {
		// Clear history using Ctrl+L
		helper.PressKey(tea.KeyCtrlL)

		// Should have no messages
		helper.AssertMessageCount(0)
	})

	t.Run("Help toggle", func(t *testing.T) {
		// Show help
		helper.SendKeyMsg("?")
		helper.AssertOutput("Help")

		// Hide help
		helper.SendKeyMsg("?")
		// Help should be hidden (check by looking for chat view content)
		helper.AssertViewType(ui.ViewChat)
	})

	t.Run("New chat shortcut", func(t *testing.T) {
		// Add a message first
		helper.SendKeyMsg("i")
		helper.TypeText("Test")
		helper.PressEnter()
		require.NoError(t, helper.WaitForResponse(5*time.Second))

		// Start new chat
		helper.PressKey(tea.KeyCtrlN)
		helper.AssertMessageCount(0)
	})
}

func TestErrorHandling(t *testing.T) {
	helper, err := helpers.NewE2ETestHelper(t, helpers.E2ETestOptions{
		Timeout:     30 * time.Second,
		EnableMocks: true,
	})
	require.NoError(t, err)
	defer helper.Cleanup()

	require.NoError(t, helper.StartApp())

	t.Run("AI error response", func(t *testing.T) {
		// Configure mock to return an error
		helper.SetAIResponse("") // Empty response will trigger error handling

		// Set up error simulation
		mockAI := helper.GetMockAIClient()
		mockAI.SetError(true, "API rate limit exceeded")

		helper.SendKeyMsg("i")
		helper.TypeText("This should cause an error")
		helper.PressEnter()

		// Wait and check for error handling
		time.Sleep(2 * time.Second)

		// Error should be displayed
		helper.AssertOutput("error")
	})

	t.Run("Empty message handling", func(t *testing.T) {
		// Try to send empty message
		helper.SendKeyMsg("i")
		helper.PressEnter()

		// Nothing should happen, should stay in insert mode
		helper.AssertMode(ui.ModeInsert)
		helper.AssertMessageCount(0)
	})

	t.Run("Long message handling", func(t *testing.T) {
		helper.SetAIResponse("I can handle long messages too!")

		// Send a very long message
		longMessage := make([]rune, 1000)
		for i := range longMessage {
			longMessage[i] = 'A'
		}

		helper.SendKeyMsg("i")
		helper.TypeText(string(longMessage))
		helper.PressEnter()

		require.NoError(t, helper.WaitForResponse(10*time.Second))
		helper.AssertMessageCount(2)
	})
}

func TestUserScenarios(t *testing.T) {
	helper, err := helpers.NewE2ETestHelper(t, helpers.E2ETestOptions{
		Timeout:     30 * time.Second,
		EnableMocks: true,
	})
	require.NoError(t, err)
	defer helper.Cleanup()

	require.NoError(t, helper.StartApp())

	t.Run("Complete coding conversation", func(t *testing.T) {
		scenario := helpers.UserScenario{
			Name:        "Coding conversation",
			Description: "User asks for help with Go code and receives assistance",
			Steps: []helpers.TestStep{
				{
					Description: "Switch to insert mode",
					Action:      "switch_mode",
					Input:       "insert",
				},
				{
					Description: "Ask about Go functions",
					Action:      "type",
					Input:       "How do I write a function in Go?",
				},
				{
					Description: "Send message",
					Action:      "enter",
				},
				{
					Description: "Wait for AI response",
					Action:      "wait_for_response",
				},
				{
					Description: "Verify response received",
					Action:      "assert_message_count",
					Expected:    "2",
				},
				{
					Description: "Ask follow-up question",
					Action:      "switch_mode",
					Input:       "insert",
				},
				{
					Description: "Type follow-up",
					Action:      "type",
					Input:       "Can you show me an example?",
				},
				{
					Description: "Send follow-up",
					Action:      "enter",
				},
				{
					Description: "Wait for follow-up response",
					Action:      "wait_for_response",
				},
				{
					Description: "Verify conversation length",
					Action:      "assert_message_count",
					Expected:    "4",
				},
			},
		}

		// Set up responses
		helper.SetAIResponse("In Go, you write functions using the 'func' keyword...")
		err := helper.SimulateUserInteraction(scenario)
		require.NoError(t, err)

		// Additional verification
		messages := helper.GetMessages()
		assert.Len(t, messages, 4)
		assert.Contains(t, messages[1].Content, "func")
	})

	t.Run("Mode switching scenario", func(t *testing.T) {
		scenario := helpers.UserScenario{
			Name:        "Mode switching",
			Description: "User switches between different modes",
			Steps: []helpers.TestStep{
				{
					Description: "Start in normal mode",
					Action:      "assert_mode",
					Expected:    "normal",
				},
				{
					Description: "Switch to command mode",
					Action:      "switch_mode",
					Input:       "command",
				},
				{
					Description: "Type help command",
					Action:      "type",
					Input:       "help",
				},
				{
					Description: "Execute command",
					Action:      "enter",
				},
				{
					Description: "Should be back in normal mode",
					Action:      "assert_mode",
					Expected:    "normal",
				},
				{
					Description: "Switch to search mode",
					Action:      "switch_mode",
					Input:       "search",
				},
				{
					Description: "Type search query",
					Action:      "type",
					Input:       "test",
				},
				{
					Description: "Execute search",
					Action:      "enter",
				},
			},
		}

		err := helper.SimulateUserInteraction(scenario)
		require.NoError(t, err)
	})
}

func TestSessionPersistence(t *testing.T) {
	helper, err := helpers.NewE2ETestHelper(t, helpers.E2ETestOptions{
		Timeout:     30 * time.Second,
		EnableMocks: true,
	})
	require.NoError(t, err)
	defer helper.Cleanup()

	helper.SetAIResponse("Session test response")
	require.NoError(t, helper.StartApp())

	// Add some messages
	helper.SendKeyMsg("i")
	helper.TypeText("Test message for session")
	helper.PressEnter()
	require.NoError(t, helper.WaitForResponse(5*time.Second))

	// Verify messages exist
	helper.AssertMessageCount(2)

	// Test session context (messages should be preserved within the session)
	helper.SendKeyMsg("i")
	helper.TypeText("Follow-up message")
	helper.PressEnter()
	require.NoError(t, helper.WaitForResponse(5*time.Second))

	helper.AssertMessageCount(4)

	// Verify message order and content
	messages := helper.GetMessages()
	assert.Equal(t, "user", messages[0].Role)
	assert.Equal(t, "assistant", messages[1].Role)
	assert.Equal(t, "user", messages[2].Role)
	assert.Equal(t, "assistant", messages[3].Role)
}
