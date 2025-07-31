// Package ai provides Azure OpenAI client implementation.
package ai

import (
	"context"
	"errors"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"

	openai "github.com/sashabaranov/go-openai"
)

// AzureClient implements the Client interface for Azure OpenAI Service.
type AzureClient struct {
	client         *openai.Client
	config         AIConfig
	azureConfig    AzureConfig
	deploymentName string
	httpClient     *http.Client
}

// AzureConfig represents Azure-specific configuration.
type AzureConfig struct {
	Endpoint       string
	DeploymentName string
	APIVersion     string
}

// NewAzureClient creates a new Azure OpenAI client instance.
func NewAzureClient(config AIConfig, azureConfig AzureConfig) (*AzureClient, error) {
	// Validate configuration
	if config.APIKey == "" {
		return nil, NewError(ErrTypeAuthentication, "Azure API key is required")
	}

	if azureConfig.Endpoint == "" {
		return nil, NewError(ErrTypeInvalidRequest, "Azure endpoint is required")
	}

	if azureConfig.DeploymentName == "" {
		return nil, NewError(ErrTypeInvalidRequest, "Azure deployment name is required")
	}

	// Validate endpoint URL
	endpointURL, err := url.Parse(azureConfig.Endpoint)
	if err != nil {
		return nil, NewError(ErrTypeInvalidRequest, "invalid Azure endpoint URL").WithCause(err)
	}

	// Ensure endpoint has proper scheme
	if endpointURL.Scheme != "https" && endpointURL.Scheme != "http" {
		return nil, NewError(ErrTypeInvalidRequest, "Azure endpoint must use http or https scheme")
	}

	// Set default values
	if azureConfig.APIVersion == "" {
		azureConfig.APIVersion = "2024-02-01"
	}

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

	// Create Azure OpenAI client configuration
	azureConfig.Endpoint = strings.TrimRight(azureConfig.Endpoint, "/")
	clientConfig := openai.DefaultAzureConfig(config.APIKey, azureConfig.Endpoint)
	clientConfig.APIVersion = azureConfig.APIVersion
	clientConfig.HTTPClient = httpClient

	// Azure uses api-key header instead of Authorization Bearer
	clientConfig.APIType = openai.APITypeAzure

	client := openai.NewClientWithConfig(clientConfig)

	return &AzureClient{
		client:         client,
		config:         config,
		azureConfig:    azureConfig,
		deploymentName: azureConfig.DeploymentName,
		httpClient:     httpClient,
	}, nil
}

// ChatCompletion implements the Client interface for non-streaming chat completion.
func (c *AzureClient) ChatCompletion(ctx context.Context, req ChatRequest) (*ChatResponse, error) {
	azureReq, err := c.convertChatRequest(req)
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

		resp, lastErr = c.client.CreateChatCompletion(ctx, azureReq)
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
func (c *AzureClient) ChatCompletionStream(ctx context.Context, req ChatRequest) (StreamReader, error) {
	azureReq, err := c.convertChatRequest(req)
	if err != nil {
		return nil, err
	}

	// Force streaming
	azureReq.Stream = true

	stream, err := c.client.CreateChatCompletionStream(ctx, azureReq)
	if err != nil {
		return nil, c.wrapError(err)
	}

	return &azureStreamReader{
		stream: stream,
	}, nil
}

// ListModels implements the Client interface for listing available models.
// Note: Azure OpenAI doesn't support listing models like OpenAI does.
// This returns the configured deployment as a model.
func (c *AzureClient) ListModels(ctx context.Context) ([]Model, error) {
	// Azure doesn't have a list models endpoint
	// Return the current deployment as a model
	models := []Model{
		{
			ID:      c.deploymentName,
			Object:  "model",
			Created: time.Now().Unix(),
			OwnedBy: "azure",
		},
	}

	// If a specific model is configured, add it to the list
	if c.config.Model != "" && c.config.Model != c.deploymentName {
		models = append(models, Model{
			ID:      c.config.Model,
			Object:  "model",
			Created: time.Now().Unix(),
			OwnedBy: "azure",
		})
	}

	return models, nil
}

// Ping implements the Client interface for health checking.
func (c *AzureClient) Ping(ctx context.Context) error {
	// Use a minimal chat completion request as health check
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	req := ChatRequest{
		Model: c.deploymentName,
		Messages: []Message{
			{Role: RoleSystem, Content: "ping"},
		},
		MaxTokens: IntPtr(1),
	}

	_, err := c.ChatCompletion(ctx, req)
	if err != nil {
		// If it's an authentication error, the service is reachable
		if IsAuthenticationError(err) {
			return nil
		}
		return err
	}

	return nil
}

// convertChatRequest converts our ChatRequest to Azure OpenAI's format.
func (c *AzureClient) convertChatRequest(req ChatRequest) (openai.ChatCompletionRequest, error) {
	azureReq := openai.ChatCompletionRequest{
		Model:    c.deploymentName, // Azure uses deployment name as model
		Messages: make([]openai.ChatCompletionMessage, len(req.Messages)),
		Stream:   req.Stream,
	}

	// Convert messages
	for i, msg := range req.Messages {
		azureReq.Messages[i] = openai.ChatCompletionMessage{
			Role:       msg.Role,
			Content:    msg.Content,
			Name:       msg.Name,
			ToolCallID: msg.ToolCallID,
		}

		// Convert tool calls
		if len(msg.ToolCalls) > 0 {
			azureReq.Messages[i].ToolCalls = make([]openai.ToolCall, len(msg.ToolCalls))
			for j, tc := range msg.ToolCalls {
				azureReq.Messages[i].ToolCalls[j] = openai.ToolCall{
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
		azureReq.Temperature = *req.Temperature
	} else if c.config.Model != "" {
		// Use configured temperature if not specified
		azureReq.Temperature = DefaultTemperature
	}

	// OpenAI reasoning models (o1, o3, o4, etc.) do not support MaxTokens, they use MaxCompletionTokens instead
	if req.MaxTokens != nil && !strings.HasPrefix(azureReq.Model, "o") {
		azureReq.MaxTokens = *req.MaxTokens
	}
	if req.TopP != nil {
		azureReq.TopP = *req.TopP
	}
	if req.PresencePenalty != nil {
		azureReq.PresencePenalty = *req.PresencePenalty
	}
	if req.FrequencyPenalty != nil {
		azureReq.FrequencyPenalty = *req.FrequencyPenalty
	}
	if req.Stop != nil {
		azureReq.Stop = req.Stop
	}
	if req.User != "" {
		azureReq.User = req.User
	}
	if req.Seed != nil {
		azureReq.Seed = req.Seed
	}

	// Convert tools
	if len(req.Tools) > 0 {
		azureReq.Tools = make([]openai.Tool, len(req.Tools))
		for i, tool := range req.Tools {
			azureReq.Tools[i] = openai.Tool{
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
		azureReq.ResponseFormat = &openai.ChatCompletionResponseFormat{
			Type: openai.ChatCompletionResponseFormatType(req.ResponseFormat.Type),
		}
	}

	return azureReq, nil
}

// convertChatResponse converts Azure OpenAI's response to our format.
func (c *AzureClient) convertChatResponse(resp openai.ChatCompletionResponse) *ChatResponse {
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
func (c *AzureClient) isRetryableError(err error) bool {
	if err == nil {
		return false
	}

	// Check for specific Azure error types
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

// wrapError converts Azure errors to our error types.
func (c *AzureClient) wrapError(err error) error {
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

// getErrorType maps Azure error to our error types.
func (c *AzureClient) getErrorType(apiErr *openai.APIError) ErrorType {
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
				if strings.Contains(codeStr, "deployment_not_found") || strings.Contains(codeStr, "model_not_found") {
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

// azureStreamReader implements StreamReader for Azure streaming responses.
type azureStreamReader struct {
	stream *openai.ChatCompletionStream
}

// Read reads the next chunk from the stream.
func (r *azureStreamReader) Read() (*StreamChunk, error) {
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
func (r *azureStreamReader) Close() error {
	if r.stream != nil {
		r.stream.Close()
	}
	return nil
}
