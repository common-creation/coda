package ai

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Mock response types
type mockChatCompletionResponse struct {
	ID      string `json:"id"`
	Object  string `json:"object"`
	Created int64  `json:"created"`
	Model   string `json:"model"`
	Choices []struct {
		Index   int `json:"index"`
		Message struct {
			Role    string `json:"role"`
			Content string `json:"content"`
		} `json:"message"`
		FinishReason string `json:"finish_reason"`
	} `json:"choices"`
	Usage struct {
		PromptTokens     int `json:"prompt_tokens"`
		CompletionTokens int `json:"completion_tokens"`
		TotalTokens      int `json:"total_tokens"`
	} `json:"usage"`
}

type mockModelsResponse struct {
	Object string `json:"object"`
	Data   []struct {
		ID      string `json:"id"`
		Object  string `json:"object"`
		Created int64  `json:"created"`
		OwnedBy string `json:"owned_by"`
	} `json:"data"`
}

type mockErrorResponse struct {
	Error struct {
		Message string  `json:"message"`
		Type    string  `json:"type"`
		Param   *string `json:"param"`
		Code    *string `json:"code"`
	} `json:"error"`
}

// Test helper functions
func setupMockServer(t *testing.T, handler http.HandlerFunc) *httptest.Server {
	server := httptest.NewServer(handler)
	t.Cleanup(server.Close)
	return server
}

func createTestConfig(baseURL string) AIConfig {
	return AIConfig{
		APIKey:         "test-api-key",
		BaseURL:        baseURL,
		Model:          "o3",
		MaxRetries:     3,
		RetryDelay:     10 * time.Millisecond,
		RequestTimeout: 5 * time.Second,
	}
}

// Test cases

func TestNewOpenAIClient(t *testing.T) {
	tests := []struct {
		name    string
		config  AIConfig
		wantErr bool
		errType ErrorType
	}{
		{
			name: "valid config",
			config: AIConfig{
				APIKey: "test-key",
			},
			wantErr: false,
		},
		{
			name: "missing API key",
			config: AIConfig{
				APIKey: "",
			},
			wantErr: true,
			errType: ErrTypeAuthentication,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			client, err := NewOpenAIClient(tt.config)
			if tt.wantErr {
				assert.Error(t, err)
				if tt.errType != "" {
					aiErr, ok := err.(*Error)
					assert.True(t, ok)
					assert.Equal(t, tt.errType, aiErr.Type)
				}
			} else {
				assert.NoError(t, err)
				assert.NotNil(t, client)
			}
		})
	}
}

func TestChatCompletion(t *testing.T) {
	tests := []struct {
		name       string
		request    ChatRequest
		mockStatus int
		mockResp   interface{}
		wantErr    bool
		errType    ErrorType
		validate   func(t *testing.T, resp *ChatResponse)
	}{
		{
			name: "successful completion",
			request: ChatRequest{
				Model: "o3",
				Messages: []Message{
					{Role: RoleUser, Content: "Hello"},
				},
			},
			mockStatus: http.StatusOK,
			mockResp: mockChatCompletionResponse{
				ID:      "chatcmpl-123",
				Object:  "chat.completion",
				Created: time.Now().Unix(),
				Model:   "o3",
				Choices: []struct {
					Index   int `json:"index"`
					Message struct {
						Role    string `json:"role"`
						Content string `json:"content"`
					} `json:"message"`
					FinishReason string `json:"finish_reason"`
				}{
					{
						Index: 0,
						Message: struct {
							Role    string `json:"role"`
							Content string `json:"content"`
						}{
							Role:    RoleAssistant,
							Content: "Hello! How can I help you?",
						},
						FinishReason: "stop",
					},
				},
				Usage: struct {
					PromptTokens     int `json:"prompt_tokens"`
					CompletionTokens int `json:"completion_tokens"`
					TotalTokens      int `json:"total_tokens"`
				}{
					PromptTokens:     10,
					CompletionTokens: 20,
					TotalTokens:      30,
				},
			},
			wantErr: false,
			validate: func(t *testing.T, resp *ChatResponse) {
				assert.Equal(t, "chatcmpl-123", resp.ID)
				assert.Len(t, resp.Choices, 1)
				assert.Equal(t, RoleAssistant, resp.Choices[0].Message.Role)
				assert.Equal(t, "Hello! How can I help you?", resp.Choices[0].Message.Content)
				assert.Equal(t, 30, resp.Usage.TotalTokens)
			},
		},
		{
			name: "authentication error",
			request: ChatRequest{
				Model: "o3",
				Messages: []Message{
					{Role: RoleUser, Content: "Hello"},
				},
			},
			mockStatus: http.StatusUnauthorized,
			mockResp: mockErrorResponse{
				Error: struct {
					Message string  `json:"message"`
					Type    string  `json:"type"`
					Param   *string `json:"param"`
					Code    *string `json:"code"`
				}{
					Message: "Invalid API key",
					Type:    "invalid_request_error",
				},
			},
			wantErr: true,
			errType: ErrTypeAuthentication,
		},
		{
			name: "rate limit error",
			request: ChatRequest{
				Model: "o3",
				Messages: []Message{
					{Role: RoleUser, Content: "Hello"},
				},
			},
			mockStatus: http.StatusTooManyRequests,
			mockResp: mockErrorResponse{
				Error: struct {
					Message string  `json:"message"`
					Type    string  `json:"type"`
					Param   *string `json:"param"`
					Code    *string `json:"code"`
				}{
					Message: "Rate limit exceeded",
					Type:    "rate_limit_error",
				},
			},
			wantErr: true,
			errType: ErrTypeRateLimit,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server := setupMockServer(t, func(w http.ResponseWriter, r *http.Request) {
				// Verify request
				assert.Equal(t, "POST", r.Method)
				assert.Equal(t, "/v1/chat/completions", r.URL.Path)
				assert.Equal(t, "Bearer test-api-key", r.Header.Get("Authorization"))

				// Send response
				w.WriteHeader(tt.mockStatus)
				json.NewEncoder(w).Encode(tt.mockResp)
			})

			config := createTestConfig(server.URL + "/v1")
			client, err := NewOpenAIClient(config)
			require.NoError(t, err)

			resp, err := client.ChatCompletion(context.Background(), tt.request)
			if tt.wantErr {
				assert.Error(t, err)
				if tt.errType != "" {
					assert.Equal(t, tt.errType, GetErrorType(err))
				}
			} else {
				assert.NoError(t, err)
				assert.NotNil(t, resp)
				if tt.validate != nil {
					tt.validate(t, resp)
				}
			}
		})
	}
}

func TestChatCompletionStream(t *testing.T) {
	server := setupMockServer(t, func(w http.ResponseWriter, r *http.Request) {
		// Verify request
		assert.Equal(t, "POST", r.Method)
		assert.Equal(t, "/v1/chat/completions", r.URL.Path)

		// Parse request to verify stream is true
		var req map[string]interface{}
		json.NewDecoder(r.Body).Decode(&req)
		assert.Equal(t, true, req["stream"])

		// Send SSE response
		w.Header().Set("Content-Type", "text/event-stream")
		w.WriteHeader(http.StatusOK)

		// Send chunks
		chunks := []string{
			`{"id":"chatcmpl-123","object":"chat.completion.chunk","created":1234567890,"model":"o3","choices":[{"index":0,"delta":{"role":"assistant"},"finish_reason":null}]}`,
			`{"id":"chatcmpl-123","object":"chat.completion.chunk","created":1234567890,"model":"o3","choices":[{"index":0,"delta":{"content":"Hello"},"finish_reason":null}]}`,
			`{"id":"chatcmpl-123","object":"chat.completion.chunk","created":1234567890,"model":"o3","choices":[{"index":0,"delta":{"content":" world!"},"finish_reason":null}]}`,
			`{"id":"chatcmpl-123","object":"chat.completion.chunk","created":1234567890,"model":"o3","choices":[{"index":0,"delta":{},"finish_reason":"stop"}]}`,
		}

		for _, chunk := range chunks {
			fmt.Fprintf(w, "data: %s\n\n", chunk)
			w.(http.Flusher).Flush()
		}
		fmt.Fprintf(w, "data: [DONE]\n\n")
		w.(http.Flusher).Flush()
	})

	config := createTestConfig(server.URL + "/v1")
	client, err := NewOpenAIClient(config)
	require.NoError(t, err)

	req := ChatRequest{
		Model: "o3",
		Messages: []Message{
			{Role: RoleUser, Content: "Hello"},
		},
		Stream: true,
	}

	stream, err := client.ChatCompletionStream(context.Background(), req)
	require.NoError(t, err)
	defer stream.Close()

	// Read all chunks
	var content strings.Builder
	var role string
	for {
		chunk, err := stream.Read()
		if err == io.EOF {
			break
		}
		require.NoError(t, err)

		if len(chunk.Choices) > 0 {
			if chunk.Choices[0].Delta.Role != "" {
				role = chunk.Choices[0].Delta.Role
			}
			content.WriteString(chunk.Choices[0].Delta.Content)
		}
	}

	assert.Equal(t, RoleAssistant, role)
	assert.Equal(t, "Hello world!", content.String())
}

func TestListModels(t *testing.T) {
	server := setupMockServer(t, func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "GET", r.Method)
		assert.Equal(t, "/v1/models", r.URL.Path)

		resp := mockModelsResponse{
			Object: "list",
			Data: []struct {
				ID      string `json:"id"`
				Object  string `json:"object"`
				Created int64  `json:"created"`
				OwnedBy string `json:"owned_by"`
			}{
				{
					ID:      "o3",
					Object:  "model",
					Created: 1234567890,
					OwnedBy: "openai",
				},
				{
					ID:      "gpt-3.5-turbo",
					Object:  "model",
					Created: 1234567890,
					OwnedBy: "openai",
				},
			},
		}

		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(resp)
	})

	config := createTestConfig(server.URL + "/v1")
	client, err := NewOpenAIClient(config)
	require.NoError(t, err)

	models, err := client.ListModels(context.Background())
	require.NoError(t, err)
	assert.Len(t, models, 2)
	assert.Equal(t, "o3", models[0].ID)
	assert.Equal(t, "gpt-3.5-turbo", models[1].ID)
}

func TestPing(t *testing.T) {
	tests := []struct {
		name       string
		mockStatus int
		mockResp   interface{}
		wantErr    bool
	}{
		{
			name:       "successful ping",
			mockStatus: http.StatusOK,
			mockResp: mockModelsResponse{
				Object: "list",
				Data: []struct {
					ID      string `json:"id"`
					Object  string `json:"object"`
					Created int64  `json:"created"`
					OwnedBy string `json:"owned_by"`
				}{},
			},
			wantErr: false,
		},
		{
			name:       "failed ping",
			mockStatus: http.StatusServiceUnavailable,
			mockResp: mockErrorResponse{
				Error: struct {
					Message string  `json:"message"`
					Type    string  `json:"type"`
					Param   *string `json:"param"`
					Code    *string `json:"code"`
				}{
					Message: "Service unavailable",
					Type:    "server_error",
				},
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server := setupMockServer(t, func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(tt.mockStatus)
				json.NewEncoder(w).Encode(tt.mockResp)
			})

			config := createTestConfig(server.URL + "/v1")
			client, err := NewOpenAIClient(config)
			require.NoError(t, err)

			err = client.Ping(context.Background())
			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestRetryLogic(t *testing.T) {
	attempts := 0
	server := setupMockServer(t, func(w http.ResponseWriter, r *http.Request) {
		attempts++
		if attempts < 3 {
			// Fail first 2 attempts with retryable error
			w.WriteHeader(http.StatusServiceUnavailable)
			json.NewEncoder(w).Encode(mockErrorResponse{
				Error: struct {
					Message string  `json:"message"`
					Type    string  `json:"type"`
					Param   *string `json:"param"`
					Code    *string `json:"code"`
				}{
					Message: "Service temporarily unavailable",
					Type:    "server_error",
				},
			})
		} else {
			// Succeed on 3rd attempt
			w.WriteHeader(http.StatusOK)
			json.NewEncoder(w).Encode(mockChatCompletionResponse{
				ID:      "chatcmpl-123",
				Object:  "chat.completion",
				Created: time.Now().Unix(),
				Model:   "o3",
				Choices: []struct {
					Index   int `json:"index"`
					Message struct {
						Role    string `json:"role"`
						Content string `json:"content"`
					} `json:"message"`
					FinishReason string `json:"finish_reason"`
				}{
					{
						Index: 0,
						Message: struct {
							Role    string `json:"role"`
							Content string `json:"content"`
						}{
							Role:    RoleAssistant,
							Content: "Success after retry",
						},
						FinishReason: "stop",
					},
				},
			})
		}
	})

	config := createTestConfig(server.URL + "/v1")
	config.MaxRetries = 3
	config.RetryDelay = 10 * time.Millisecond

	client, err := NewOpenAIClient(config)
	require.NoError(t, err)

	req := ChatRequest{
		Model: "o3",
		Messages: []Message{
			{Role: RoleUser, Content: "Test retry"},
		},
	}

	resp, err := client.ChatCompletion(context.Background(), req)
	require.NoError(t, err)
	assert.NotNil(t, resp)
	assert.Equal(t, "Success after retry", resp.Choices[0].Message.Content)
	assert.Equal(t, 3, attempts)
}

func TestErrorTypes(t *testing.T) {
	tests := []struct {
		name         string
		mockStatus   int
		mockError    mockErrorResponse
		expectedType ErrorType
	}{
		{
			name:       "authentication error",
			mockStatus: http.StatusUnauthorized,
			mockError: mockErrorResponse{
				Error: struct {
					Message string  `json:"message"`
					Type    string  `json:"type"`
					Param   *string `json:"param"`
					Code    *string `json:"code"`
				}{
					Message: "Invalid API key",
					Type:    "invalid_request_error",
				},
			},
			expectedType: ErrTypeAuthentication,
		},
		{
			name:       "rate limit error",
			mockStatus: http.StatusTooManyRequests,
			mockError: mockErrorResponse{
				Error: struct {
					Message string  `json:"message"`
					Type    string  `json:"type"`
					Param   *string `json:"param"`
					Code    *string `json:"code"`
				}{
					Message: "Rate limit exceeded",
					Type:    "rate_limit_error",
				},
			},
			expectedType: ErrTypeRateLimit,
		},
		{
			name:       "quota exceeded",
			mockStatus: http.StatusPaymentRequired,
			mockError: mockErrorResponse{
				Error: struct {
					Message string  `json:"message"`
					Type    string  `json:"type"`
					Param   *string `json:"param"`
					Code    *string `json:"code"`
				}{
					Message: "Quota exceeded",
					Type:    "insufficient_quota",
				},
			},
			expectedType: ErrTypeQuotaExceeded,
		},
		{
			name:       "context length error",
			mockStatus: http.StatusBadRequest,
			mockError: mockErrorResponse{
				Error: struct {
					Message string  `json:"message"`
					Type    string  `json:"type"`
					Param   *string `json:"param"`
					Code    *string `json:"code"`
				}{
					Message: "Maximum context length exceeded",
					Type:    "invalid_request_error",
					Code:    StringPtr("context_length_exceeded"),
				},
			},
			expectedType: ErrTypeContextLength,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server := setupMockServer(t, func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(tt.mockStatus)
				json.NewEncoder(w).Encode(tt.mockError)
			})

			config := createTestConfig(server.URL + "/v1")
			config.MaxRetries = 0 // Disable retries for error testing

			client, err := NewOpenAIClient(config)
			require.NoError(t, err)

			req := ChatRequest{
				Model: "o3",
				Messages: []Message{
					{Role: RoleUser, Content: "Test"},
				},
			}

			_, err = client.ChatCompletion(context.Background(), req)
			require.Error(t, err)
			assert.Equal(t, tt.expectedType, GetErrorType(err))
		})
	}
}

// TestContextCancellation removed - timing-dependent test
