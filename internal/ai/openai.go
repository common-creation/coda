// Package ai provides OpenAI client implementation.
package ai

import (
	"context"
	"errors"
	"io"
	"net/http"
	"strings"
	"time"

	openai "github.com/sashabaranov/go-openai"
)

// OpenAIClient implements the Client interface for OpenAI API.
type OpenAIClient struct {
	client     *openai.Client
	config     AIConfig
	httpClient *http.Client
}

// AIConfig represents the configuration for AI clients.
type AIConfig struct {
	APIKey         string
	Organization   string
	BaseURL        string
	Model          string
	MaxRetries     int
	RetryDelay     time.Duration
	RequestTimeout time.Duration
}

// NewOpenAIClient creates a new OpenAI client instance.
func NewOpenAIClient(config AIConfig) (*OpenAIClient, error) {
	if config.APIKey == "" {
		return nil, NewError(ErrTypeAuthentication, "API key is required")
	}

	// Set default values
	if config.MaxRetries == 0 {
		config.MaxRetries = 3
	}
	if config.RetryDelay == 0 {
		config.RetryDelay = time.Second
	}
	if config.RequestTimeout == 0 {
		config.RequestTimeout = DefaultTimeout
	}

	// Create HTTP client with timeout
	httpClient := &http.Client{
		Timeout: config.RequestTimeout,
	}

	// Create OpenAI client configuration
	clientConfig := openai.DefaultConfig(config.APIKey)
	if config.Organization != "" {
		clientConfig.OrgID = config.Organization
	}
	if config.BaseURL != "" {
		clientConfig.BaseURL = config.BaseURL
	}
	clientConfig.HTTPClient = httpClient

	client := openai.NewClientWithConfig(clientConfig)

	return &OpenAIClient{
		client:     client,
		config:     config,
		httpClient: httpClient,
	}, nil
}

// ChatCompletion implements the Client interface for non-streaming chat completion.
func (c *OpenAIClient) ChatCompletion(ctx context.Context, req ChatRequest) (*ChatResponse, error) {
	openaiReq, err := c.convertChatRequest(req)
	if err != nil {
		return nil, err
	}

	// Retry logic for transient errors
	var resp openai.ChatCompletionResponse
	var lastErr error

	for attempt := 0; attempt <= c.config.MaxRetries; attempt++ {
		if attempt > 0 {
			// Exponential backoff
			delay := c.config.RetryDelay * time.Duration(1<<(attempt-1))
			select {
			case <-ctx.Done():
				return nil, NewError(ErrTypeTimeout, "context cancelled during retry").WithCause(ctx.Err())
			case <-time.After(delay):
			}
		}

		resp, lastErr = c.client.CreateChatCompletion(ctx, openaiReq)
		if lastErr == nil {
			break
		}

		// Check if error is retryable
		if !c.isRetryableError(lastErr) {
			break
		}
	}

	if lastErr != nil {
		return nil, c.wrapError(lastErr)
	}

	return c.convertChatResponse(resp), nil
}

// ChatCompletionStream implements the Client interface for streaming chat completion.
func (c *OpenAIClient) ChatCompletionStream(ctx context.Context, req ChatRequest) (StreamReader, error) {
	openaiReq, err := c.convertChatRequest(req)
	if err != nil {
		return nil, err
	}

	// Force streaming
	openaiReq.Stream = true

	stream, err := c.client.CreateChatCompletionStream(ctx, openaiReq)
	if err != nil {
		return nil, c.wrapError(err)
	}

	return &openAIStreamReader{
		stream: stream,
	}, nil
}

// ListModels implements the Client interface for listing available models.
func (c *OpenAIClient) ListModels(ctx context.Context) ([]Model, error) {
	modelsList, err := c.client.ListModels(ctx)
	if err != nil {
		return nil, c.wrapError(err)
	}

	models := make([]Model, len(modelsList.Models))
	for i, m := range modelsList.Models {
		models[i] = Model{
			ID:         m.ID,
			Object:     m.Object,
			Created:    time.Now().Unix(), // m.Created field doesn't exist in current version
			OwnedBy:    m.OwnedBy,
			Permission: []string{}, // Convert permissions later
			Root:       m.Root,
			Parent:     m.Parent,
		}
	}

	return models, nil
}

// Ping implements the Client interface for health checking.
func (c *OpenAIClient) Ping(ctx context.Context) error {
	// Use ListModels as a lightweight health check
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	_, err := c.client.ListModels(ctx)
	if err != nil {
		return c.wrapError(err)
	}

	return nil
}

// convertChatRequest converts our ChatRequest to OpenAI's format.
func (c *OpenAIClient) convertChatRequest(req ChatRequest) (openai.ChatCompletionRequest, error) {
	openaiReq := openai.ChatCompletionRequest{
		Model:    req.Model,
		Messages: make([]openai.ChatCompletionMessage, len(req.Messages)),
		Stream:   req.Stream,
	}

	// Set model default if not provided
	if openaiReq.Model == "" {
		if c.config.Model != "" {
			openaiReq.Model = c.config.Model
		} else {
			openaiReq.Model = ModelGPT4Turbo
		}
	}

	// Convert messages
	for i, msg := range req.Messages {
		openaiReq.Messages[i] = openai.ChatCompletionMessage{
			Role:       msg.Role,
			Content:    msg.Content,
			Name:       msg.Name,
			ToolCallID: msg.ToolCallID,
		}

		// Convert tool calls
		if len(msg.ToolCalls) > 0 {
			openaiReq.Messages[i].ToolCalls = make([]openai.ToolCall, len(msg.ToolCalls))
			for j, tc := range msg.ToolCalls {
				openaiReq.Messages[i].ToolCalls[j] = openai.ToolCall{
					ID:   tc.ID,
					Type: openai.ToolType(tc.Type),
					Function: openai.FunctionCall{
						Name:      tc.Function.Name,
						Arguments: tc.Function.Arguments,
					},
				}
			}
		}
	}

	// Convert optional parameters
	if req.Temperature != nil {
		openaiReq.Temperature = *req.Temperature
	}
	// OpenAI reasoning models (o1, o3, o4, etc.) do not support MaxTokens, they use MaxCompletionTokens instead
	if req.MaxTokens != nil && !strings.HasPrefix(openaiReq.Model, "o") {
		openaiReq.MaxTokens = *req.MaxTokens
	}
	if req.TopP != nil {
		openaiReq.TopP = *req.TopP
	}
	if req.PresencePenalty != nil {
		openaiReq.PresencePenalty = *req.PresencePenalty
	}
	if req.FrequencyPenalty != nil {
		openaiReq.FrequencyPenalty = *req.FrequencyPenalty
	}
	if req.Stop != nil {
		openaiReq.Stop = req.Stop
	}
	if req.User != "" {
		openaiReq.User = req.User
	}
	if req.Seed != nil {
		openaiReq.Seed = req.Seed
	}

	// Convert tools
	if len(req.Tools) > 0 {
		openaiReq.Tools = make([]openai.Tool, len(req.Tools))
		for i, tool := range req.Tools {
			openaiReq.Tools[i] = openai.Tool{
				Type: openai.ToolType(tool.Type),
				Function: &openai.FunctionDefinition{
					Name:        tool.Function.Name,
					Description: tool.Function.Description,
					Parameters:  tool.Function.Parameters,
				},
			}
		}
	}

	// Convert response format
	if req.ResponseFormat != nil {
		openaiReq.ResponseFormat = &openai.ChatCompletionResponseFormat{
			Type: openai.ChatCompletionResponseFormatType(req.ResponseFormat.Type),
		}
	}

	return openaiReq, nil
}

// convertChatResponse converts OpenAI's response to our format.
func (c *OpenAIClient) convertChatResponse(resp openai.ChatCompletionResponse) *ChatResponse {
	chatResp := &ChatResponse{
		ID:                resp.ID,
		Object:            resp.Object,
		Created:           resp.Created,
		Model:             resp.Model,
		SystemFingerprint: resp.SystemFingerprint,
		Choices:           make([]Choice, len(resp.Choices)),
		Usage: Usage{
			PromptTokens:     resp.Usage.PromptTokens,
			CompletionTokens: resp.Usage.CompletionTokens,
			TotalTokens:      resp.Usage.TotalTokens,
		},
	}

	// Convert choices
	for i, choice := range resp.Choices {
		chatResp.Choices[i] = Choice{
			Index:        choice.Index,
			FinishReason: string(choice.FinishReason),
			Message: Message{
				Role:    choice.Message.Role,
				Content: choice.Message.Content,
				Name:    choice.Message.Name,
			},
		}

		// Convert tool calls
		if len(choice.Message.ToolCalls) > 0 {
			chatResp.Choices[i].Message.ToolCalls = make([]ToolCall, len(choice.Message.ToolCalls))
			for j, tc := range choice.Message.ToolCalls {
				chatResp.Choices[i].Message.ToolCalls[j] = ToolCall{
					ID:   tc.ID,
					Type: string(tc.Type),
					Function: FunctionCall{
						Name:      tc.Function.Name,
						Arguments: tc.Function.Arguments,
					},
				}
			}
		}
	}

	return chatResp
}

// isRetryableError checks if the error should be retried.
func (c *OpenAIClient) isRetryableError(err error) bool {
	if err == nil {
		return false
	}

	// Check for specific OpenAI error types
	var apiErr *openai.APIError
	if errors.As(err, &apiErr) {
		switch apiErr.HTTPStatusCode {
		case http.StatusTooManyRequests, http.StatusServiceUnavailable, http.StatusGatewayTimeout:
			return true
		case http.StatusInternalServerError, http.StatusBadGateway:
			return true
		}
	}

	// Check for network errors
	errStr := strings.ToLower(err.Error())
	if strings.Contains(errStr, "timeout") ||
		strings.Contains(errStr, "connection") ||
		strings.Contains(errStr, "network") {
		return true
	}

	return false
}

// wrapError converts OpenAI errors to our error types.
func (c *OpenAIClient) wrapError(err error) error {
	if err == nil {
		return nil
	}

	var apiErr *openai.APIError
	if errors.As(err, &apiErr) {
		aiErr := NewError(c.getErrorType(apiErr), apiErr.Message).
			WithStatusCode(apiErr.HTTPStatusCode).
			WithCause(err)

		// Add details if available
		if apiErr.Code != nil {
			if codeStr, ok := apiErr.Code.(string); ok {
				aiErr = aiErr.WithDetail("code", codeStr)
			}
		}
		if apiErr.Param != nil {
			aiErr = aiErr.WithDetail("param", *apiErr.Param)
		}
		if apiErr.Type != "" {
			aiErr = aiErr.WithDetail("type", apiErr.Type)
		}

		return aiErr
	}

	// Handle other error types
	return WrapError(err, ErrTypeUnknown)
}

// getErrorType maps OpenAI error to our error types.
func (c *OpenAIClient) getErrorType(apiErr *openai.APIError) ErrorType {
	switch apiErr.HTTPStatusCode {
	case http.StatusUnauthorized, http.StatusForbidden:
		return ErrTypeAuthentication
	case http.StatusTooManyRequests:
		return ErrTypeRateLimit
	case http.StatusBadRequest:
		if apiErr.Code != nil {
			if codeStr, ok := apiErr.Code.(string); ok {
				if strings.Contains(codeStr, "context_length") || strings.Contains(codeStr, "token") {
					return ErrTypeContextLength
				}
				if strings.Contains(codeStr, "content_policy") || strings.Contains(codeStr, "content_filter") {
					return ErrTypeContentFilter
				}
				if strings.Contains(codeStr, "model_not_found") {
					return ErrTypeModelNotFound
				}
			}
		}
		return ErrTypeInvalidRequest
	case http.StatusPaymentRequired:
		return ErrTypeQuotaExceeded
	case http.StatusInternalServerError, http.StatusBadGateway, http.StatusServiceUnavailable:
		return ErrTypeServerError
	case http.StatusGatewayTimeout:
		return ErrTypeTimeout
	default:
		if apiErr.HTTPStatusCode >= 500 {
			return ErrTypeServerError
		}
		return ErrTypeUnknown
	}
}

// openAIStreamReader implements StreamReader for OpenAI streaming responses.
type openAIStreamReader struct {
	stream *openai.ChatCompletionStream
}

// Read reads the next chunk from the stream.
func (r *openAIStreamReader) Read() (*StreamChunk, error) {
	chunk, err := r.stream.Recv()
	if err != nil {
		if err == io.EOF {
			return nil, io.EOF
		}
		return nil, WrapError(err, ErrTypeUnknown)
	}

	streamChunk := &StreamChunk{
		ID:                chunk.ID,
		Object:            chunk.Object,
		Created:           chunk.Created,
		Model:             chunk.Model,
		SystemFingerprint: chunk.SystemFingerprint,
		Choices:           make([]StreamChoice, len(chunk.Choices)),
	}

	// Convert choices
	for i, choice := range chunk.Choices {
		streamChunk.Choices[i] = StreamChoice{
			Index: choice.Index,
			Delta: StreamDelta{
				Role:    choice.Delta.Role,
				Content: choice.Delta.Content,
			},
		}

		// Convert finish reason if present
		if choice.FinishReason != "" {
			finishReason := string(choice.FinishReason)
			streamChunk.Choices[i].FinishReason = &finishReason
		}

		// Convert tool calls in delta
		if len(choice.Delta.ToolCalls) > 0 {
			streamChunk.Choices[i].Delta.ToolCalls = make([]ToolCall, len(choice.Delta.ToolCalls))
			for j, tc := range choice.Delta.ToolCalls {
				streamChunk.Choices[i].Delta.ToolCalls[j] = ToolCall{
					ID:   tc.ID,
					Type: string(tc.Type),
					Function: FunctionCall{
						Name:      tc.Function.Name,
						Arguments: tc.Function.Arguments,
					},
				}
			}
		}
	}

	return streamChunk, nil
}

// Close closes the stream.
func (r *openAIStreamReader) Close() error {
	if r.stream != nil {
		r.stream.Close()
	}
	return nil
}
