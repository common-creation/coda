package ui

import (
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"

	"github.com/common-creation/coda/internal/styles"
)

func TestRenderChatWithExplicitNewlines(t *testing.T) {
	// Create a model with messages containing explicit newlines
	model := Model{
		width:  60,
		height: 24,
		ready:  true,
		styles: styles.GetTheme("default").GetStyles(),
		messages: []Message{
			{
				ID:        "1",
				Content:   "This is a message\nwith multiple lines\nexplicitly separated.",
				Role:      "user",
				Timestamp: time.Now(),
			},
			{
				ID:        "2",
				Content:   "これは日本語のメッセージです。\n複数行に分かれています。\n明示的な改行があります。",
				Role:      "assistant",
				Timestamp: time.Now(),
			},
		},
	}

	// Render the chat
	output := model.renderChat()

	// Verify that the output contains multiple lines
	lines := strings.Split(output, "\n")

	// Check that we have multiple lines due to explicit newlines
	assert.Greater(t, len(lines), 6, "Messages with newlines should produce multiple lines")

	// Check that subsequent lines are indented
	hasIndentedLine := false
	for _, line := range lines {
		if strings.HasPrefix(line, strings.Repeat(" ", 10)) { // Approximate indent
			hasIndentedLine = true
			break
		}
	}
	assert.True(t, hasIndentedLine, "Lines after newlines should be indented")
}

func TestRenderChatPrefixWidth(t *testing.T) {
	// Test that multi-byte characters in role names are handled correctly
	model := Model{
		width:  80,
		height: 24,
		ready:  true,
		styles: styles.GetTheme("default").GetStyles(),
		messages: []Message{
			{
				ID:        "1",
				Content:   "Test message",
				Role:      "ユーザー", // Japanese role name
				Timestamp: time.Now(),
			},
		},
	}

	// Render the chat
	output := model.renderChat()

	// Should render without panic
	assert.NotEmpty(t, output)
	assert.Contains(t, output, "ユーザー")
}

func TestRenderChatEmptyMessages(t *testing.T) {
	model := Model{
		width:    80,
		height:   24,
		ready:    true,
		styles:   styles.GetTheme("default").GetStyles(),
		messages: []Message{},
	}

	// Should render welcome message
	output := model.renderChat()
	assert.NotContains(t, output, "[")
	assert.NotContains(t, output, "]")
}

func TestRenderChatWithStyles(t *testing.T) {
	// Test that styles are applied correctly
	theme := styles.GetTheme("default")
	model := Model{
		width:  80,
		height: 24,
		ready:  true,
		styles: theme.GetStyles(),
		messages: []Message{
			{
				ID:        "1",
				Content:   "User message",
				Role:      "user",
				Timestamp: time.Now(),
			},
			{
				ID:        "2",
				Content:   "Assistant message",
				Role:      "assistant",
				Timestamp: time.Now(),
			},
		},
	}

	output := model.renderChat()

	// Check that output is not empty
	assert.NotEmpty(t, output)

	// Verify both messages are rendered
	assert.Contains(t, output, "User message")
	assert.Contains(t, output, "Assistant message")
}
