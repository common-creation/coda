package ui

import (
	"testing"
	"time"

	"github.com/charmbracelet/bubbles/viewport"
	"github.com/stretchr/testify/assert"

	"github.com/common-creation/coda/internal/styles"
)

func TestRenderChatWithStyles(t *testing.T) {
	// Test that styles are applied correctly
	theme := styles.GetTheme("default")
	model := Model{
		viewport: viewport.New(80, 20),
		width:    80,
		height:   24,
		ready:    true,
		styles:   theme.GetStyles(),
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

	// Update viewport content to include messages
	model.updateViewportContent()

	output := model.renderChat()

	// Check that output is not empty
	assert.NotEmpty(t, output)

	// Verify both messages are rendered
	assert.Contains(t, output, "User message")
	assert.Contains(t, output, "Assistant message")
}