// Package ai provides a unified interface for interacting with various AI providers.
// It abstracts the differences between OpenAI, Azure OpenAI, and other providers,
// offering a consistent API for chat completions, streaming, and model management.
package ai

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/common-creation/coda/internal/config"
)

// Client defines the unified interface for AI providers.
// All AI provider implementations must satisfy this interface.
type Client interface {
	// ChatCompletion sends a chat completion request and returns the full response.
	// This method blocks until the entire response is received.
	//
	// Example:
	//   resp, err := client.ChatCompletion(ctx, ChatRequest{
	//       Model: "o3",
	//       Messages: []Message{
	//           {Role: "user", Content: "Hello, AI!"},
	//       },
	//   })
	ChatCompletion(ctx context.Context, req ChatRequest) (*ChatResponse, error)

	// ChatCompletionStream sends a chat completion request and returns a stream reader.
	// This allows for real-time processing of the AI response as it's generated.
	//
	// Example:
	//   stream, err := client.ChatCompletionStream(ctx, req)
	//   if err != nil {
	//       return err
	//   }
	//   defer stream.Close()
	//
	//   for {
	//       chunk, err := stream.Read()
	//       if err == io.EOF {
	//           break
	//       }
	//       // Process chunk
	//   }
	ChatCompletionStream(ctx context.Context, req ChatRequest) (StreamReader, error)

	// ListModels retrieves the list of available models from the AI provider.
	// This can be used to validate model availability or present options to users.
	ListModels(ctx context.Context) ([]Model, error)

	// Ping checks if the AI service is accessible and responding.
	// It returns an error if the service is unavailable or authentication fails.
	Ping(ctx context.Context) error
}

// Note: Type definitions for ChatRequest, Message, ChatResponse, Choice, Usage,
// Model, Tool, FunctionTool, ToolCall, FunctionCall, ResponseFormat,
// StreamChunk, StreamChoice, and StreamDelta are in types.go

// ClientOptions contains options for creating a new AI client.
type ClientOptions struct {
	// Timeout for individual requests. If not set, defaults to 30 seconds.
	Timeout time.Duration

	// RetryPolicy defines how to handle retries for failed requests.
	RetryPolicy *RetryPolicy

	// HTTPClient allows using a custom HTTP client.
	// If not set, a default client will be used.
	HTTPClient interface{}
}

// RetryPolicy defines retry behavior for failed requests.
type RetryPolicy struct {
	// MaxRetries is the maximum number of retry attempts.
	MaxRetries int

	// InitialDelay is the initial delay between retry attempts.
	InitialDelay time.Duration

	// MaxDelay is the maximum delay between retry attempts.
	MaxDelay time.Duration

	// Multiplier is the factor by which the delay increases after each attempt.
	Multiplier float64

	// RetryableStatusCodes defines which HTTP status codes should trigger a retry.
	RetryableStatusCodes []int
}

// DefaultRetryPolicy returns a sensible default retry policy.
func DefaultRetryPolicy() *RetryPolicy {
	return &RetryPolicy{
		MaxRetries:           3,
		InitialDelay:         1 * time.Second,
		MaxDelay:             30 * time.Second,
		Multiplier:           2.0,
		RetryableStatusCodes: []int{408, 429, 500, 502, 503, 504},
	}
}

// NewClient creates a new AI client based on the provided configuration.
// It automatically selects the appropriate implementation based on the provider type.
//
// Example:
//
//	cfg := config.AIConfig{
//	    Provider: "openai",
//	    APIKey:   "sk-...",
//	}
//	client, err := ai.NewClient(cfg)
//	if err != nil {
//	    return err
//	}
func NewClient(cfg config.AIConfig, opts ...ClientOptions) (Client, error) {
	// Validate configuration
	if cfg.Provider == "" {
		return nil, errors.New("ai provider not specified")
	}

	if cfg.APIKey == "" {
		return nil, errors.New("api key not provided")
	}

	// Merge options
	var options ClientOptions
	if len(opts) > 0 {
		options = opts[0]
	}

	// Set defaults
	if options.Timeout == 0 {
		options.Timeout = 30 * time.Second
	}

	if options.RetryPolicy == nil {
		options.RetryPolicy = DefaultRetryPolicy()
	}

	// Convert config.AIConfig to AIConfig
	aiConfig := AIConfig{
		APIKey:         cfg.APIKey,
		Organization:   cfg.OpenAI.Organization,
		BaseURL:        cfg.OpenAI.BaseURL,
		Model:          cfg.Model,
		MaxRetries:     options.RetryPolicy.MaxRetries,
		RetryDelay:     options.RetryPolicy.InitialDelay,
		RequestTimeout: options.Timeout,
	}

	// Create client based on provider
	switch cfg.Provider {
	case "openai":
		return NewOpenAIClient(aiConfig)
	case "azure":
		azureConfig := AzureConfig{
			Endpoint:       cfg.Azure.Endpoint,
			DeploymentName: cfg.Azure.DeploymentName,
			APIVersion:     cfg.Azure.APIVersion,
		}
		return NewAzureClient(aiConfig, azureConfig)
	default:
		return nil, fmt.Errorf("unsupported ai provider: %s", cfg.Provider)
	}
}

// WithTimeout returns a context with the specified timeout.
// This is a convenience function for setting request timeouts.
//
// Example:
//
//	ctx, cancel := ai.WithTimeout(context.Background(), 10*time.Second)
//	defer cancel()
//	resp, err := client.ChatCompletion(ctx, req)
func WithTimeout(ctx context.Context, timeout time.Duration) (context.Context, context.CancelFunc) {
	return context.WithTimeout(ctx, timeout)
}

// WithCancel returns a context that can be cancelled.
// This is useful for implementing user-initiated cancellations.
//
// Example:
//
//	ctx, cancel := ai.WithCancel(context.Background())
//	defer cancel()
//
//	// In another goroutine or on user input:
//	cancel() // This will cancel the ongoing request
func WithCancel(ctx context.Context) (context.Context, context.CancelFunc) {
	return context.WithCancel(ctx)
}
