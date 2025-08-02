package ui

import (
	"fmt"
	"strings"

	"github.com/tiktoken-go/tokenizer"
	
	"github.com/common-creation/coda/internal/ai"
)

// EstimateTokens estimates the number of tokens for a prompt with messages
func EstimateTokens(messages []ai.Message, model string) (int, error) {
	// Get the appropriate encoding for the model
	encoding, err := getEncodingForModel(model)
	if err != nil {
		return 0, fmt.Errorf("failed to get encoding: %w", err)
	}

	totalTokens := 0
	
	// Count tokens for each message
	for _, msg := range messages {
		// Add tokens for message structure (role, content markers)
		// OpenAI models typically use ~4 tokens per message for structure
		totalTokens += 4
		
		// Count role tokens
		roleTokens, _, err := encoding.Encode(msg.Role)
		if err != nil {
			return 0, fmt.Errorf("failed to encode role: %w", err)
		}
		totalTokens += len(roleTokens)
		
		// Count content tokens
		if msg.Content != "" {
			contentTokens, _, err := encoding.Encode(msg.Content)
			if err != nil {
				return 0, fmt.Errorf("failed to encode content: %w", err)
			}
			totalTokens += len(contentTokens)
		}
		
		// Count tool calls if any
		if len(msg.ToolCalls) > 0 {
			// Estimate ~50 tokens per tool call (rough estimate)
			totalTokens += len(msg.ToolCalls) * 50
		}
	}
	
	// Add tokens for reply primer (~3 tokens)
	totalTokens += 3
	
	return totalTokens, nil
}

// EstimateUserMessageTokens estimates tokens for just the user message
func EstimateUserMessageTokens(message string, model string) (int, error) {
	if message == "" {
		return 0, fmt.Errorf("empty message")
	}
	
	encoding, err := getEncodingForModel(model)
	if err != nil {
		return 0, fmt.Errorf("failed to get encoding for model %s: %w", model, err)
	}
	
	tokens, _, err := encoding.Encode(message)
	if err != nil {
		return 0, fmt.Errorf("failed to encode message of length %d: %w", len(message), err)
	}
	
	// Add message structure overhead
	tokenCount := len(tokens) + 4
	return tokenCount, nil
}

// getEncodingForModel returns the appropriate tokenizer encoding for a model
func getEncodingForModel(model string) (tokenizer.Codec, error) {
	// Default to cl100k_base for GPT-4 and GPT-3.5-turbo models
	// This covers most modern OpenAI models
	encodingName := tokenizer.Cl100kBase
	
	// For o-series models (o1, o3, etc), use O200k_base
	if strings.HasPrefix(model, "o") {
		encodingName = tokenizer.O200kBase
	} else if strings.HasPrefix(model, "gpt-3") && !strings.Contains(model, "turbo") {
		encodingName = tokenizer.P50kBase
	} else if strings.HasPrefix(model, "text-davinci") {
		encodingName = tokenizer.P50kBase
	} else if strings.HasPrefix(model, "code-") {
		encodingName = tokenizer.P50kBase
	}
	
	codec, err := tokenizer.Get(encodingName)
	if err != nil {
		return nil, fmt.Errorf("failed to get tokenizer encoding %s for model %s: %w", encodingName, model, err)
	}
	return codec, nil
}