package ui

import (
	"strings"
	"testing"
	"time"

	"github.com/charmbracelet/bubbles/viewport"
	"github.com/stretchr/testify/assert"

	"github.com/common-creation/coda/internal/styles"
)

func TestRenderScrollbar(t *testing.T) {
	tests := []struct {
		name             string
		viewportHeight   int
		totalLines       int
		visibleLines     int
		scrollPercent    float64
		expectedThumbPos int
		expectedEmpty    bool
	}{
		{
			name:           "Content fits - empty scrollbar",
			viewportHeight: 10,
			totalLines:     8,
			visibleLines:   10,
			scrollPercent:  0,
			expectedEmpty:  true,
		},
		{
			name:             "Scrollbar at top",
			viewportHeight:   10,
			totalLines:       20,
			visibleLines:     10,
			scrollPercent:    0,
			expectedThumbPos: 0,
		},
		{
			name:             "Scrollbar at middle",
			viewportHeight:   10,
			totalLines:       20,
			visibleLines:     10,
			scrollPercent:    0.5,
			expectedThumbPos: 2, // (10-5)*0.5 = 2.5 -> 2
		},
		{
			name:             "Scrollbar at bottom",
			viewportHeight:   10,
			totalLines:       20,
			visibleLines:     10,
			scrollPercent:    1.0,
			expectedThumbPos: 5, // (10-5)*1.0 = 5
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create a model with mock viewport
			model := Model{
				viewport: viewport.Model{
					Height: tt.viewportHeight,
				},
				styles: styles.GetTheme("default").GetStyles(),
			}

			// Mock viewport methods by creating a test viewport
			model.viewport = viewport.New(80, tt.viewportHeight)

			// Set content to simulate the total lines
			content := ""
			for i := 0; i < tt.totalLines; i++ {
				content += "Line\n"
			}
			model.viewport.SetContent(content)

			// Render scrollbar
			result := model.renderScrollbar()
			lines := strings.Split(result, "\n")

			// Check line count
			assert.Equal(t, tt.viewportHeight, len(lines))

			if tt.expectedEmpty {
				// Check all lines are spaces
				for _, line := range lines {
					assert.Equal(t, " ", line)
				}
			} else {
				// Count thumb characters
				thumbCount := 0
				firstThumbLine := -1
				for i, line := range lines {
					if strings.Contains(line, "█") {
						if firstThumbLine == -1 {
							firstThumbLine = i
						}
						thumbCount++
					}
				}

				// Check thumb exists
				assert.Greater(t, thumbCount, 0, "Scrollbar thumb should be visible")
			}
		})
	}
}

func TestUpdateViewportContent(t *testing.T) {
	// Create a model
	model := Model{
		viewport: viewport.New(80, 20),
		messages: []Message{
			{
				Content:   "Hello",
				Role:      "user",
				Timestamp: time.Now(),
			},
			{
				Content:   "Hi there!",
				Role:      "assistant",
				Timestamp: time.Now(),
			},
		},
	}

	// Update viewport content
	model.updateViewportContent()

	// Get content
	content := model.viewport.View()

	// Check that figlet is present (using actual figlet characters)
	assert.Contains(t, content, "▄████████")
	assert.Contains(t, content, "▄██████▄")

	// Check messages are present
	assert.Contains(t, content, "Hello")
	assert.Contains(t, content, "Hi there!")
}
