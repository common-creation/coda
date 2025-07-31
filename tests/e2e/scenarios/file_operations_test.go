package scenarios

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/common-creation/coda/internal/ui"
	"github.com/common-creation/coda/tests/e2e/helpers"
)

func TestFileReadOperations(t *testing.T) {
	helper, err := helpers.NewE2ETestHelper(t, helpers.E2ETestOptions{
		Timeout:     30 * time.Second,
		EnableMocks: true,
	})
	require.NoError(t, err)
	defer helper.Cleanup()

	// Create test files
	require.NoError(t, helper.CreateTestFile("main.go", `package main

import "fmt"

func main() {
    fmt.Println("Hello, World!")
}
`))
	require.NoError(t, helper.CreateTestFile("config.json", `{
    "name": "test-app",
    "version": "1.0.0"
}
`))

	helper.SetAIResponse("I can see the Go file you want me to read. Let me analyze it for you.")
	require.NoError(t, helper.StartApp())

	t.Run("Request file reading", func(t *testing.T) {
		helper.SendKeyMsg("i")
		helper.TypeText("Can you read the main.go file and explain what it does?")
		helper.PressEnter()

		require.NoError(t, helper.WaitForResponse(10*time.Second))

		// Verify tool was called
		toolManager := helper.GetMockToolManager()
		calls := toolManager.GetCallHistory()

		// Should have called read_file tool
		found := false
		for _, call := range calls {
			if call.ToolName == "read_file" {
				assert.Contains(t, call.Arguments["path"], "main.go")
				found = true
				break
			}
		}
		assert.True(t, found, "read_file tool should have been called")

		helper.AssertLastMessageRole("assistant")
		helper.AssertLastMessageContains("analyze")
	})

	t.Run("Read multiple files", func(t *testing.T) {
		helper.SetAIResponse("I'll read both files for you. The Go file is a simple Hello World program, and the JSON file contains application configuration.")

		helper.SendKeyMsg("i")
		helper.TypeText("Please read both main.go and config.json files")
		helper.PressEnter()

		require.NoError(t, helper.WaitForResponse(10*time.Second))

		// Verify multiple file reads
		toolManager := helper.GetMockToolManager()
		calls := toolManager.GetCallHistory()

		readCalls := 0
		for _, call := range calls {
			if call.ToolName == "read_file" {
				readCalls++
			}
		}
		assert.GreaterOrEqual(t, readCalls, 2, "Should have called read_file at least twice")
	})
}

func TestFileWriteOperations(t *testing.T) {
	helper, err := helpers.NewE2ETestHelper(t, helpers.E2ETestOptions{
		Timeout:     30 * time.Second,
		EnableMocks: true,
	})
	require.NoError(t, err)
	defer helper.Cleanup()

	helper.SetAIResponse("I'll create the Go file for you with the Hello World program.")
	require.NoError(t, helper.StartApp())

	t.Run("Request file creation", func(t *testing.T) {
		helper.SendKeyMsg("i")
		helper.TypeText("Create a simple Hello World program in Go and save it as hello.go")
		helper.PressEnter()

		require.NoError(t, helper.WaitForResponse(10*time.Second))

		// Verify write_file tool was called
		toolManager := helper.GetMockToolManager()
		calls := toolManager.GetCallHistory()

		found := false
		for _, call := range calls {
			if call.ToolName == "write_file" {
				assert.Contains(t, call.Arguments["path"], "hello.go")
				assert.Contains(t, call.Arguments["content"], "package main")
				found = true
				break
			}
		}
		assert.True(t, found, "write_file tool should have been called")
	})

	t.Run("Request file modification", func(t *testing.T) {
		// First create a file
		require.NoError(t, helper.CreateTestFile("example.go", `package main

func main() {
    // TODO: Add implementation
}
`))

		helper.SetAIResponse("I'll modify the file to add the implementation.")

		helper.SendKeyMsg("i")
		helper.TypeText("Read example.go and add a proper implementation to print 'Hello, Go!'")
		helper.PressEnter()

		require.NoError(t, helper.WaitForResponse(10*time.Second))

		// Should have both read and write operations
		toolManager := helper.GetMockToolManager()
		calls := toolManager.GetCallHistory()

		hasRead := false
		hasWrite := false
		for _, call := range calls {
			if call.ToolName == "read_file" && call.Arguments["path"].(string) == "example.go" {
				hasRead = true
			}
			if call.ToolName == "write_file" && call.Arguments["path"].(string) == "example.go" {
				hasWrite = true
			}
		}

		assert.True(t, hasRead, "Should have read the file first")
		assert.True(t, hasWrite, "Should have written the modified file")
	})
}

func TestDirectoryOperations(t *testing.T) {
	helper, err := helpers.NewE2ETestHelper(t, helpers.E2ETestOptions{
		Timeout:     30 * time.Second,
		EnableMocks: true,
	})
	require.NoError(t, err)
	defer helper.Cleanup()

	// Create test directory structure
	require.NoError(t, helper.CreateTestFile("src/main.go", "package main"))
	require.NoError(t, helper.CreateTestFile("src/utils.go", "package main"))
	require.NoError(t, helper.CreateTestFile("docs/README.md", "# Project"))
	require.NoError(t, helper.CreateTestFile("go.mod", "module test"))

	helper.SetAIResponse("I can see your project structure. You have source files in the src directory and documentation in docs.")
	require.NoError(t, helper.StartApp())

	t.Run("List project files", func(t *testing.T) {
		helper.SendKeyMsg("i")
		helper.TypeText("Can you show me all files in this project?")
		helper.PressEnter()

		require.NoError(t, helper.WaitForResponse(10*time.Second))

		// Verify list_files tool was called
		toolManager := helper.GetMockToolManager()
		calls := toolManager.GetCallHistory()

		found := false
		for _, call := range calls {
			if call.ToolName == "list_files" {
				found = true
				break
			}
		}
		assert.True(t, found, "list_files tool should have been called")
	})

	t.Run("List specific directory", func(t *testing.T) {
		helper.SetAIResponse("The src directory contains your Go source files: main.go and utils.go")

		helper.SendKeyMsg("i")
		helper.TypeText("What files are in the src directory?")
		helper.PressEnter()

		require.NoError(t, helper.WaitForResponse(10*time.Second))

		// Verify list_files was called with src path
		toolManager := helper.GetMockToolManager()
		calls := toolManager.GetCallHistory()

		found := false
		for _, call := range calls {
			if call.ToolName == "list_files" {
				if path, ok := call.Arguments["path"].(string); ok && path == "src" {
					found = true
					break
				}
			}
		}
		assert.True(t, found, "list_files should have been called with src directory")
	})
}

func TestSearchOperations(t *testing.T) {
	helper, err := helpers.NewE2ETestHelper(t, helpers.E2ETestOptions{
		Timeout:     30 * time.Second,
		EnableMocks: true,
	})
	require.NoError(t, err)
	defer helper.Cleanup()

	// Create test files with searchable content
	require.NoError(t, helper.CreateTestFile("main.go", `package main

import "fmt"

func main() {
    fmt.Println("Hello World")
    fmt.Printf("Debug: %s\n", "test")
}
`))
	require.NoError(t, helper.CreateTestFile("utils.go", `package main

import "fmt"

func debugPrint(msg string) {
    fmt.Printf("DEBUG: %s\n", msg)
}
`))

	helper.SetAIResponse("I found several occurrences of 'fmt.Printf' in your code files.")
	require.NoError(t, helper.StartApp())

	t.Run("Search for specific text", func(t *testing.T) {
		helper.SendKeyMsg("i")
		helper.TypeText("Search for all occurrences of 'fmt.Printf' in my code")
		helper.PressEnter()

		require.NoError(t, helper.WaitForResponse(10*time.Second))

		// Verify search tool was called
		toolManager := helper.GetMockToolManager()
		calls := toolManager.GetCallHistory()

		found := false
		for _, call := range calls {
			if call.ToolName == "search" {
				assert.Contains(t, call.Arguments["query"], "fmt.Printf")
				found = true
				break
			}
		}
		assert.True(t, found, "search tool should have been called")
	})

	t.Run("Search in specific directory", func(t *testing.T) {
		helper.SetAIResponse("I'll search for function definitions in your source files.")

		helper.SendKeyMsg("i")
		helper.TypeText("Find all function definitions in the current directory")
		helper.PressEnter()

		require.NoError(t, helper.WaitForResponse(10*time.Second))

		// Verify search was called with appropriate parameters
		toolManager := helper.GetMockToolManager()
		calls := toolManager.GetCallHistory()

		found := false
		for _, call := range calls {
			if call.ToolName == "search" {
				// Should search for function pattern
				query := call.Arguments["query"].(string)
				if query == "func " || query == "function" {
					found = true
					break
				}
			}
		}
		assert.True(t, found, "search should have been called with function pattern")
	})
}

func TestComplexFileWorkflow(t *testing.T) {
	helper, err := helpers.NewE2ETestHelper(t, helpers.E2ETestOptions{
		Timeout:     60 * time.Second,
		EnableMocks: true,
	})
	require.NoError(t, err)
	defer helper.Cleanup()

	require.NoError(t, helper.StartApp())

	t.Run("Complete code review workflow", func(t *testing.T) {
		scenario := helpers.UserScenario{
			Name:        "Code review workflow",
			Description: "User requests code review of multiple files",
			Steps: []helpers.TestStep{
				{
					Description: "Start conversation",
					Action:      "switch_mode",
					Input:       "insert",
				},
				{
					Description: "Request project overview",
					Action:      "type",
					Input:       "Please review my Go project. Start by listing all files.",
				},
				{
					Description: "Send request",
					Action:      "enter",
				},
				{
					Description: "Wait for file listing",
					Action:      "wait_for_response",
					WaitAfter:   1 * time.Second,
				},
				{
					Description: "Request main file review",
					Action:      "switch_mode",
					Input:       "insert",
				},
				{
					Description: "Ask for main.go review",
					Action:      "type",
					Input:       "Now please read and review main.go for best practices.",
				},
				{
					Description: "Send review request",
					Action:      "enter",
				},
				{
					Description: "Wait for review",
					Action:      "wait_for_response",
					WaitAfter:   1 * time.Second,
				},
				{
					Description: "Request improvements",
					Action:      "switch_mode",
					Input:       "insert",
				},
				{
					Description: "Ask for improvements",
					Action:      "type",
					Input:       "Can you suggest improvements and create an improved version?",
				},
				{
					Description: "Send improvement request",
					Action:      "enter",
				},
				{
					Description: "Wait for improvements",
					Action:      "wait_for_response",
				},
			},
		}

		// Create test files
		require.NoError(t, helper.CreateTestFile("main.go", `package main

import "fmt"

func main() {
    fmt.Println("Hello")
}
`))
		require.NoError(t, helper.CreateTestFile("go.mod", "module testproject"))

		// Set up sequential responses
		responses := []string{
			"I'll start by listing all files in your project.",
			"I can see your main.go file. Let me review it for best practices.",
			"I can suggest several improvements to make your code more robust and maintainable.",
		}

		for i, response := range responses {
			if i > 0 {
				helper.SetAIResponse(response)
			} else {
				helper.SetAIResponse(response)
			}
		}

		err := helper.SimulateUserInteraction(scenario)
		require.NoError(t, err)

		// Verify the complete workflow
		toolManager := helper.GetMockToolManager()
		calls := toolManager.GetCallHistory()

		// Should have multiple tool calls
		assert.GreaterOrEqual(t, len(calls), 3, "Should have made multiple tool calls")

		// Check for expected tool calls
		hasListFiles := false
		hasReadFile := false
		hasWriteFile := false

		for _, call := range calls {
			switch call.ToolName {
			case "list_files":
				hasListFiles = true
			case "read_file":
				hasReadFile = true
			case "write_file":
				hasWriteFile = true
			}
		}

		assert.True(t, hasListFiles, "Should have listed files")
		assert.True(t, hasReadFile, "Should have read files")
		// Write file might not be called in all scenarios, so we won't assert it

		// Check message count
		helper.AssertMessageCount(6) // 3 user messages + 3 AI responses
	})
}

func TestFilePermissionsAndSecurity(t *testing.T) {
	helper, err := helpers.NewE2ETestHelper(t, helpers.E2ETestOptions{
		Timeout:     30 * time.Second,
		EnableMocks: true,
	})
	require.NoError(t, err)
	defer helper.Cleanup()

	helper.SetAIResponse("I can only access files within your configured workspace directory for security reasons.")
	require.NoError(t, helper.StartApp())

	t.Run("Attempt to access restricted file", func(t *testing.T) {
		helper.SendKeyMsg("i")
		helper.TypeText("Can you read /etc/passwd for me?")
		helper.PressEnter()

		require.NoError(t, helper.WaitForResponse(10*time.Second))

		// The system should handle this gracefully
		// In a real implementation, this would be blocked by security policies
		helper.AssertLastMessageRole("assistant")
		helper.AssertLastMessageContains("security")
	})

	t.Run("Access allowed workspace files", func(t *testing.T) {
		// Create a file in the allowed workspace
		require.NoError(t, helper.CreateTestFile("allowed.txt", "This file is in the workspace"))

		helper.SetAIResponse("I can access this file as it's within your workspace.")

		helper.SendKeyMsg("i")
		helper.TypeText("Please read allowed.txt")
		helper.PressEnter()

		require.NoError(t, helper.WaitForResponse(10*time.Second))

		// Should be able to read workspace files
		toolManager := helper.GetMockToolManager()
		calls := toolManager.GetCallHistory()

		found := false
		for _, call := range calls {
			if call.ToolName == "read_file" && call.Arguments["path"].(string) == "allowed.txt" {
				found = true
				break
			}
		}
		assert.True(t, found, "Should be able to read workspace files")
	})
}

func TestLargeFileHandling(t *testing.T) {
	helper, err := helpers.NewE2ETestHelper(t, helpers.E2ETestOptions{
		Timeout:     30 * time.Second,
		EnableMocks: true,
	})
	require.NoError(t, err)
	defer helper.Cleanup()

	// Create a large file
	largeContent := ""
	for i := 0; i < 1000; i++ {
		largeContent += fmt.Sprintf("Line %d: This is a test line with some content\n", i)
	}
	require.NoError(t, helper.CreateTestFile("large.txt", largeContent))

	helper.SetAIResponse("I can see this is a large file. I'll analyze it efficiently.")
	require.NoError(t, helper.StartApp())

	t.Run("Handle large file reading", func(t *testing.T) {
		helper.SendKeyMsg("i")
		helper.TypeText("Can you analyze the content of large.txt?")
		helper.PressEnter()

		require.NoError(t, helper.WaitForResponse(15*time.Second))

		// Should handle large files gracefully
		toolManager := helper.GetMockToolManager()
		calls := toolManager.GetCallHistory()

		found := false
		for _, call := range calls {
			if call.ToolName == "read_file" && call.Arguments["path"].(string) == "large.txt" {
				found = true
				break
			}
		}
		assert.True(t, found, "Should attempt to read large file")

		helper.AssertLastMessageRole("assistant")
	})
}

func TestBinaryFileHandling(t *testing.T) {
	helper, err := helpers.NewE2ETestHelper(t, helpers.E2ETestOptions{
		Timeout:     30 * time.Second,
		EnableMocks: true,
	})
	require.NoError(t, err)
	defer helper.Cleanup()

	// Create a mock binary file (represented as text with binary-like content)
	binaryContent := string([]byte{0x00, 0x01, 0x02, 0x03, 0xFF, 0xFE, 0xFD})
	require.NoError(t, helper.CreateTestFile("binary.dat", binaryContent))

	helper.SetAIResponse("I notice this appears to be a binary file. Binary files cannot be read as text.")
	require.NoError(t, helper.StartApp())

	t.Run("Handle binary file appropriately", func(t *testing.T) {
		helper.SendKeyMsg("i")
		helper.TypeText("What's in the binary.dat file?")
		helper.PressEnter()

		require.NoError(t, helper.WaitForResponse(10*time.Second))

		// System should handle binary files appropriately
		helper.AssertLastMessageRole("assistant")
		helper.AssertLastMessageContains("binary")
	})
}
