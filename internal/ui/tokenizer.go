package ui

import (
	"github.com/common-creation/coda/internal/ai"
	"github.com/common-creation/coda/internal/tokenizer"
)

// EstimateTokens estimates the number of tokens for a prompt with messages
// This is a wrapper for backward compatibility
func EstimateTokens(messages []ai.Message, model string) (int, error) {
	return tokenizer.EstimateTokens(messages, model)
}

// EstimateUserMessageTokens estimates tokens for just the user message
// This is a wrapper for backward compatibility
func EstimateUserMessageTokens(message string, model string) (int, error) {
	return tokenizer.EstimateUserMessageTokens(message, model)
}
