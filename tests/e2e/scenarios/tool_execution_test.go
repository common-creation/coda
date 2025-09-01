//go:build integration
// +build integration

package scenarios

import (
	"fmt"
	"testing"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/common-creation/coda/internal/ui"
	"github.com/common-creation/coda/tests/e2e/helpers"
)

func TestBasicToolExecution(t *testing.T) {
	helper, err := helpers.NewE2ETestHelper(t, helpers.E2ETestOptions{
		Timeout:     30 * time.Second,
		EnableMocks: true,
	})
	require.NoError(t, err)
	defer helper.Cleanup()

	helper.SetAIResponse("I'll help you read that file using the file reading tool.")
	require.NoError(t, helper.StartApp())

	// Create test file
	require.NoError(t, helper.CreateTestFile("test.txt", "Hello from test file!"))

	t.Run("Automatic tool execution", func(t *testing.T) {
		// Disable approval mode for automatic execution
		toolManager := helper.GetMockToolManager()
		toolManager.SetApprovalMode(false)

		helper.SendKeyMsg("i")
		helper.TypeText("Please read the test.txt file")
		helper.PressEnter()

		require.NoError(t, helper.WaitForResponse(10*time.Second))

		// Verify tool was executed
		calls := toolManager.GetCallHistory()
		found := false
		for _, call := range calls {
			if call.ToolName == "read_file" && call.Approved {
				found = true
				break
			}
		}
		assert.True(t, found, "Tool should have been executed automatically")

		helper.AssertLastMessageRole("assistant")
	})

	t.Run("Tool execution with result", func(t *testing.T) {
		// Set custom tool result
		toolManager := helper.GetMockToolManager()
		toolManager.SetToolResult("read_file", "Custom file content for testing")

		helper.SetAIResponse("I can see the file contains: Custom file content for testing")

		helper.SendKeyMsg("i")
		helper.TypeText("Read the test.txt file and tell me what it contains")
		helper.PressEnter()

		require.NoError(t, helper.WaitForResponse(10*time.Second))

		// Verify the custom result was used
		helper.AssertLastMessageContains("Custom file content")
	})
}

func TestToolApprovalSystem(t *testing.T) {
	helper, err := helpers.NewE2ETestHelper(t, helpers.E2ETestOptions{
		Timeout:     30 * time.Second,
		EnableMocks: true,
	})
	require.NoError(t, err)
	defer helper.Cleanup()

	require.NoError(t, helper.StartApp())

	t.Run("Tool approval UI appears", func(t *testing.T) {
		// Enable approval mode
		toolManager := helper.GetMockToolManager()
		toolManager.SetApprovalMode(true)

		helper.SetAIResponse("I need to use the file reading tool to help you with that.")

		helper.SendKeyMsg("i")
		helper.TypeText("Read the config.json file")
		helper.PressEnter()

		// Wait for approval UI to appear
		err := helper.WaitForCondition(func() bool {
			view := helper.GetModel().View()
			return toolContains(view, "Tool Approval") || toolContains(view, "approve") || toolContains(view, "y/n")
		}, 10*time.Second, "tool approval UI to appear")

		require.NoError(t, err, "Tool approval UI should appear")

		// Verify approval UI is shown
		helper.AssertOutput("approve")
	})

	t.Run("Approve tool execution", func(t *testing.T) {
		toolManager := helper.GetMockToolManager()
		toolManager.SetApprovalMode(true)
		toolManager.ClearHistory()

		helper.SetAIResponse("I'll read the file for you after approval.")

		helper.SendKeyMsg("i")
		helper.TypeText("Please read test.txt")
		helper.PressEnter()

		// Wait for approval prompt
		time.Sleep(2 * time.Second)

		// Approve the tool execution
		helper.SendKeyMsg("y")

		require.NoError(t, helper.WaitForResponse(10*time.Second))

		// Verify tool was approved and executed
		calls := toolManager.GetCallHistory()
		found := false
		for _, call := range calls {
			if call.ToolName == "read_file" && call.Approved {
				found = true
				break
			}
		}
		assert.True(t, found, "Tool should have been approved and executed")
	})

	t.Run("Deny tool execution", func(t *testing.T) {
		toolManager := helper.GetMockToolManager()
		toolManager.SetApprovalMode(true)
		toolManager.ClearHistory()

		helper.SetAIResponse("I understand you don't want me to read the file. Is there another way I can help?")

		helper.SendKeyMsg("i")
		helper.TypeText("Please read sensitive.txt")
		helper.PressEnter()

		// Wait for approval prompt
		time.Sleep(2 * time.Second)

		// Deny the tool execution
		helper.SendKeyMsg("n")

		require.NoError(t, helper.WaitForResponse(10*time.Second))

		// Verify tool was denied
		calls := toolManager.GetCallHistory()
		for _, call := range calls {
			if call.ToolName == "read_file" {
				assert.False(t, call.Approved, "Tool should have been denied")
			}
		}
	})
}

func TestMultipleToolExecution(t *testing.T) {
	helper, err := helpers.NewE2ETestHelper(t, helpers.E2ETestOptions{
		Timeout:     45 * time.Second,
		EnableMocks: true,
	})
	require.NoError(t, err)
	defer helper.Cleanup()

	// Create test files
	require.NoError(t, helper.CreateTestFile("file1.txt", "Content of file 1"))
	require.NoError(t, helper.CreateTestFile("file2.txt", "Content of file 2"))
	require.NoError(t, helper.CreateTestFile("output.txt", ""))

	require.NoError(t, helper.StartApp())

	t.Run("Sequential tool execution", func(t *testing.T) {
		toolManager := helper.GetMockToolManager()
		toolManager.SetApprovalMode(false) // Auto-approve for this test

		helper.SetAIResponse("I'll read both files and then combine their content into the output file.")

		helper.SendKeyMsg("i")
		helper.TypeText("Read file1.txt and file2.txt, then combine their content and write to output.txt")
		helper.PressEnter()

		require.NoError(t, helper.WaitForResponse(15*time.Second))

		// Verify multiple tools were executed
		calls := toolManager.GetCallHistory()

		readCalls := 0
		writeCalls := 0
		for _, call := range calls {
			switch call.ToolName {
			case "read_file":
				readCalls++
			case "write_file":
				writeCalls++
			}
		}

		assert.GreaterOrEqual(t, readCalls, 2, "Should have read at least 2 files")
		assert.GreaterOrEqual(t, writeCalls, 1, "Should have written at least 1 file")
	})

	t.Run("Complex workflow with approvals", func(t *testing.T) {
		toolManager := helper.GetMockToolManager()
		toolManager.SetApprovalMode(true)
		toolManager.ClearHistory()

		helper.SetAIResponse("I'll need to use several tools to analyze your project structure.")

		helper.SendKeyMsg("i")
		helper.TypeText("Analyze my project: list all files, read the main files, and create a summary")
		helper.PressEnter()

		// This would require multiple approvals in a real scenario
		// For testing, we'll simulate approving multiple tools
		for i := 0; i < 3; i++ {
			// Wait for approval prompt
			time.Sleep(1 * time.Second)

			// Check if approval is needed
			view := helper.GetModel().View()
			if toolContains(view, "approve") || toolContains(view, "y/n") {
				helper.SendKeyMsg("y")
			}
		}

		require.NoError(t, helper.WaitForResponse(20*time.Second))

		// Verify multiple tools were executed
		calls := toolManager.GetCallHistory()
		assert.GreaterOrEqual(t, len(calls), 2, "Should have executed multiple tools")
	})
}

func TestToolErrorHandling(t *testing.T) {
	helper, err := helpers.NewE2ETestHelper(t, helpers.E2ETestOptions{
		Timeout:     30 * time.Second,
		EnableMocks: true,
	})
	require.NoError(t, err)
	defer helper.Cleanup()

	require.NoError(t, helper.StartApp())

	t.Run("Tool execution failure", func(t *testing.T) {
		// Configure tool to fail
		toolManager := helper.GetMockToolManager()
		toolManager.SetToolResult("read_file", nil)

		// Register a failing tool
		failingTool := helpers.MockTool{
			Name:        "failing_tool",
			Description: "A tool that always fails",
			Handler: func(args map[string]interface{}) (interface{}, error) {
				return nil, fmt.Errorf("simulated tool failure")
			},
		}
		toolManager.RegisterTool(failingTool)

		helper.SetAIResponse("I encountered an error while trying to use the tool. Let me try a different approach.")

		helper.SendKeyMsg("i")
		helper.TypeText("Use the failing_tool to do something")
		helper.PressEnter()

		require.NoError(t, helper.WaitForResponse(10*time.Second))

		// Verify error was handled gracefully
		helper.AssertLastMessageRole("assistant")
		helper.AssertLastMessageContains("error")
	})

	t.Run("Non-existent tool handling", func(t *testing.T) {
		helper.SetAIResponse("I don't have access to that tool. Let me suggest alternative approaches.")

		helper.SendKeyMsg("i")
		helper.TypeText("Use the non_existent_tool to help me")
		helper.PressEnter()

		require.NoError(t, helper.WaitForResponse(10*time.Second))

		// Should handle gracefully
		helper.AssertLastMessageRole("assistant")
	})

	t.Run("Tool timeout handling", func(t *testing.T) {
		// Register a slow tool
		slowTool := helpers.MockTool{
			Name:        "slow_tool",
			Description: "A tool that takes a long time",
			Handler: func(args map[string]interface{}) (interface{}, error) {
				time.Sleep(5 * time.Second) // Simulate slow operation
				return "Finally done", nil
			},
		}

		toolManager := helper.GetMockToolManager()
		toolManager.RegisterTool(slowTool)

		helper.SetAIResponse("This tool is taking longer than expected, but I'll wait for it to complete.")

		helper.SendKeyMsg("i")
		helper.TypeText("Use the slow_tool for this task")
		helper.PressEnter()

		// Should eventually complete
		require.NoError(t, helper.WaitForResponse(15*time.Second))
		helper.AssertLastMessageRole("assistant")
	})
}

func TestToolApprovalUI(t *testing.T) {
	helper, err := helpers.NewE2ETestHelper(t, helpers.E2ETestOptions{
		Timeout:     30 * time.Second,
		EnableMocks: true,
	})
	require.NoError(t, err)
	defer helper.Cleanup()

	require.NoError(t, helper.StartApp())

	t.Run("Approval UI displays tool information", func(t *testing.T) {
		toolManager := helper.GetMockToolManager()
		toolManager.SetApprovalMode(true)

		helper.SetAIResponse("I need to read a file to help you.")

		helper.SendKeyMsg("i")
		helper.TypeText("Read important.txt")
		helper.PressEnter()

		// Wait for approval UI
		err := helper.WaitForCondition(func() bool {
			view := helper.GetModel().View()
			return toolContains(view, "read_file") && toolContains(view, "important.txt")
		}, 10*time.Second, "approval UI with tool details")

		require.NoError(t, err)

		// Verify tool details are shown
		helper.AssertOutput("read_file")
		helper.AssertOutput("important.txt")
	})

	t.Run("Multiple tool approvals", func(t *testing.T) {
		toolManager := helper.GetMockToolManager()
		toolManager.SetApprovalMode(true)
		toolManager.ClearHistory()

		helper.SetAIResponse("I'll need to list files and then read several of them.")

		helper.SendKeyMsg("i")
		helper.TypeText("Show me all files and then read the important ones")
		helper.PressEnter()

		// Approve first tool (list_files)
		time.Sleep(1 * time.Second)
		helper.SendKeyMsg("y")

		// Approve second tool (read_file)
		time.Sleep(1 * time.Second)
		helper.SendKeyMsg("y")

		require.NoError(t, helper.WaitForResponse(15*time.Second))

		// Verify both tools were approved
		calls := toolManager.GetCallHistory()
		approvedCalls := 0
		for _, call := range calls {
			if call.Approved {
				approvedCalls++
			}
		}
		assert.GreaterOrEqual(t, approvedCalls, 2, "Multiple tools should have been approved")
	})

	t.Run("Approval with keyboard shortcuts", func(t *testing.T) {
		toolManager := helper.GetMockToolManager()
		toolManager.SetApprovalMode(true)
		toolManager.ClearHistory()

		helper.SetAIResponse("I'll write a file after approval.")

		helper.SendKeyMsg("i")
		helper.TypeText("Create a new file called test_output.txt")
		helper.PressEnter()

		// Wait for approval UI
		time.Sleep(2 * time.Second)

		// Test different approval methods
		// First try 'Y' (uppercase)
		helper.SendKeyMsg("Y")

		require.NoError(t, helper.WaitForResponse(10*time.Second))

		calls := toolManager.GetCallHistory()
		assert.True(t, len(calls) > 0 && calls[0].Approved, "Tool should have been approved with uppercase Y")
	})
}

func TestToolExecutionScenarios(t *testing.T) {
	helper, err := helpers.NewE2ETestHelper(t, helpers.E2ETestOptions{
		Timeout:     60 * time.Second,
		EnableMocks: true,
	})
	require.NoError(t, err)
	defer helper.Cleanup()

	require.NoError(t, helper.StartApp())

	t.Run("Complete development workflow", func(t *testing.T) {
		scenario := helpers.UserScenario{
			Name:        "Development workflow with tools",
			Description: "User requests code creation and modification with tool approvals",
			Steps: []helpers.TestStep{
				{
					Description: "Request project setup",
					Action:      "switch_mode",
					Input:       "insert",
				},
				{
					Description: "Ask for project creation",
					Action:      "type",
					Input:       "Create a new Go project with main.go and go.mod files",
				},
				{
					Description: "Send request",
					Action:      "enter",
				},
				{
					Description: "Wait for tool approval",
					WaitAfter:   2 * time.Second,
				},
				{
					Description: "Approve first tool",
					Action:      "key",
					Input:       "y",
				},
				{
					Description: "Wait for second approval",
					WaitAfter:   1 * time.Second,
				},
				{
					Description: "Approve second tool",
					Action:      "key",
					Input:       "y",
				},
				{
					Description: "Wait for completion",
					Action:      "wait_for_response",
				},
				{
					Description: "Request code review",
					Action:      "switch_mode",
					Input:       "insert",
				},
				{
					Description: "Ask for review",
					Action:      "type",
					Input:       "Please review the created files and suggest improvements",
				},
				{
					Description: "Send review request",
					Action:      "enter",
				},
				{
					Description: "Wait for review tools",
					WaitAfter:   2 * time.Second,
				},
				{
					Description: "Approve read operation",
					Action:      "key",
					Input:       "y",
				},
				{
					Description: "Wait for review completion",
					Action:      "wait_for_response",
				},
			},
		}

		// Enable approval mode
		toolManager := helper.GetMockToolManager()
		toolManager.SetApprovalMode(true)

		// Set up responses
		helper.SetAIResponse("I'll create the Go project files for you.")

		err := helper.SimulateUserInteraction(scenario)
		require.NoError(t, err)

		// Verify tools were executed with approvals
		calls := toolManager.GetCallHistory()
		assert.GreaterOrEqual(t, len(calls), 2, "Should have executed multiple tools")

		// Verify approvals
		for _, call := range calls {
			assert.True(t, call.Approved, "All tools should have been approved")
		}
	})

	t.Run("Mixed approval scenario", func(t *testing.T) {
		toolManager := helper.GetMockToolManager()
		toolManager.SetApprovalMode(true)
		toolManager.ClearHistory()

		helper.SetAIResponse("I'll help you with file operations, but I'll need your approval first.")

		helper.SendKeyMsg("i")
		helper.TypeText("Read file1.txt, but don't modify file2.txt, and create file3.txt")
		helper.PressEnter()

		// Simulate mixed approvals: approve read, deny modify, approve create
		approvals := []string{"y", "n", "y"}
		for _, approval := range approvals {
			time.Sleep(1 * time.Second)
			view := helper.GetModel().View()
			if toolContains(view, "approve") || toolContains(view, "y/n") {
				helper.SendKeyMsg(approval)
			}
		}

		require.NoError(t, helper.WaitForResponse(20*time.Second))

		// Verify mixed approval results
		calls := toolManager.GetCallHistory()
		approvedCount := 0
		deniedCount := 0

		for _, call := range calls {
			if call.Approved {
				approvedCount++
			} else {
				deniedCount++
			}
		}

		assert.Greater(t, approvedCount, 0, "Some tools should have been approved")
		// Note: Denied count might be 0 if the AI doesn't attempt denied operations
	})
}

func TestToolSecurityFeatures(t *testing.T) {
	helper, err := helpers.NewE2ETestHelper(t, helpers.E2ETestOptions{
		Timeout:     30 * time.Second,
		EnableMocks: true,
	})
	require.NoError(t, err)
	defer helper.Cleanup()

	require.NoError(t, helper.StartApp())

	t.Run("Dangerous operation approval", func(t *testing.T) {
		// Register a potentially dangerous tool
		dangerousTool := helpers.MockTool{
			Name:        "delete_file",
			Description: "Delete a file (dangerous operation)",
			Handler: func(args map[string]interface{}) (interface{}, error) {
				return "File deleted successfully", nil
			},
		}

		toolManager := helper.GetMockToolManager()
		toolManager.RegisterTool(dangerousTool)
		toolManager.SetApprovalMode(true)

		helper.SetAIResponse("I can delete the file, but this is a destructive operation that requires your approval.")

		helper.SendKeyMsg("i")
		helper.TypeText("Delete the old_file.txt")
		helper.PressEnter()

		// Wait for approval with emphasis on danger
		time.Sleep(2 * time.Second)

		// Verify dangerous operation is highlighted
		helper.AssertOutput("delete")

		// Approve the dangerous operation
		helper.SendKeyMsg("y")

		require.NoError(t, helper.WaitForResponse(10*time.Second))

		// Verify tool was executed
		calls := toolManager.GetCallHistory()
		found := false
		for _, call := range calls {
			if call.ToolName == "delete_file" && call.Approved {
				found = true
				break
			}
		}
		assert.True(t, found, "Dangerous tool should have been executed after approval")
	})
}

// Helper function to check if a string contains a substring (case-insensitive)
func toolContains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr ||
		len(s) > len(substr) &&
			(toolContainsAt(s, substr, 0) || toolContains(s[1:], substr)))
}

func toolContainsAt(s, substr string, index int) bool {
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
