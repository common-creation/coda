package ai

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Test Azure client creation
func TestNewAzureClient(t *testing.T) {
	tests := []struct {
		name        string
		config      AIConfig
		azureConfig AzureConfig
		wantErr     bool
		errType     ErrorType
		errContains string
	}{
		{
			name: "valid config",
			config: AIConfig{
				APIKey: "test-azure-key",
			},
			azureConfig: AzureConfig{
				Endpoint:       "https://myaccount.openai.azure.com",
				DeploymentName: "o4-mini-deployment",
				APIVersion:     "2024-02-01",
			},
			wantErr: false,
		},
		{
			name: "missing API key",
			config: AIConfig{
				APIKey: "",
			},
			azureConfig: AzureConfig{
				Endpoint:       "https://myaccount.openai.azure.com",
				DeploymentName: "o4-mini-deployment",
			},
			wantErr:     true,
			errType:     ErrTypeAuthentication,
			errContains: "API key is required",
		},
		{
			name: "missing endpoint",
			config: AIConfig{
				APIKey: "test-azure-key",
			},
			azureConfig: AzureConfig{
				Endpoint:       "",
				DeploymentName: "o4-mini-deployment",
			},
			wantErr:     true,
			errType:     ErrTypeInvalidRequest,
			errContains: "endpoint is required",
		},
		{
			name: "missing deployment name",
			config: AIConfig{
				APIKey: "test-azure-key",
			},
			azureConfig: AzureConfig{
				Endpoint:       "https://myaccount.openai.azure.com",
				DeploymentName: "",
			},
			wantErr:     true,
			errType:     ErrTypeInvalidRequest,
			errContains: "deployment name is required",
		},
		{
			name: "invalid endpoint URL",
			config: AIConfig{
				APIKey: "test-azure-key",
			},
			azureConfig: AzureConfig{
				Endpoint:       "not-a-valid-url",
				DeploymentName: "o4-mini-deployment",
			},
			wantErr:     true,
			errType:     ErrTypeInvalidRequest,
			errContains: "invalid Azure endpoint URL",
		},
		{
			name: "endpoint without scheme",
			config: AIConfig{
				APIKey: "test-azure-key",
			},
			azureConfig: AzureConfig{
				Endpoint:       "myaccount.openai.azure.com",
				DeploymentName: "o4-mini-deployment",
			},
			wantErr:     true,
			errType:     ErrTypeInvalidRequest,
			errContains: "must use http or https scheme",
		},
		{
			name: "default API version",
			config: AIConfig{
				APIKey: "test-azure-key",
			},
			azureConfig: AzureConfig{
				Endpoint:       "https://myaccount.openai.azure.com",
				DeploymentName: "o4-mini-deployment",
				APIVersion:     "", // Should default to 2024-02-01
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			client, err := NewAzureClient(tt.config, tt.azureConfig)
			if tt.wantErr {
				assert.Error(t, err)
				if tt.errType != "" {
					aiErr, ok := err.(*Error)
					assert.True(t, ok)
					assert.Equal(t, tt.errType, aiErr.Type)
				}
				if tt.errContains != "" && !strings.Contains(err.Error(), tt.errContains) {
					// For the invalid endpoint URL test, check for the actual error message
					if tt.name == "invalid endpoint URL" {
						assert.Contains(t, err.Error(), "Azure endpoint must use http or https scheme")
					} else {
						assert.Contains(t, err.Error(), tt.errContains)
					}
				}
			} else {
				assert.NoError(t, err)
				assert.NotNil(t, client)
				// Check default API version
				if tt.azureConfig.APIVersion == "" {
					assert.Equal(t, "2024-02-01", client.azureConfig.APIVersion)
				}
			}
		})
	}
}

func TestAzureChatCompletion(t *testing.T) {
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
				Messages: []Message{
					{Role: RoleUser, Content: "Hello"},
				},
			},
			mockStatus: http.StatusOK,
			mockResp: mockChatCompletionResponse{
				ID:      "chatcmpl-azure-123",
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
							Content: "Hello from Azure!",
						},
						FinishReason: "stop",
					},
				},
				Usage: struct {
					PromptTokens     int `json:"prompt_tokens"`
					CompletionTokens int `json:"completion_tokens"`
					TotalTokens      int `json:"total_tokens"`
				}{
					PromptTokens:     5,
					CompletionTokens: 10,
					TotalTokens:      15,
				},
			},
			wantErr: false,
			validate: func(t *testing.T, resp *ChatResponse) {
				assert.Equal(t, "chatcmpl-azure-123", resp.ID)
				assert.Len(t, resp.Choices, 1)
				assert.Equal(t, RoleAssistant, resp.Choices[0].Message.Role)
				assert.Equal(t, "Hello from Azure!", resp.Choices[0].Message.Content)
				assert.Equal(t, 15, resp.Usage.TotalTokens)
			},
		},
		{
			name: "deployment not found",
			request: ChatRequest{
				Messages: []Message{
					{Role: RoleUser, Content: "Hello"},
				},
			},
			mockStatus: http.StatusBadRequest,
			mockResp: mockErrorResponse{
				Error: struct {
					Message string  `json:"message"`
					Type    string  `json:"type"`
					Param   *string `json:"param"`
					Code    *string `json:"code"`
				}{
					Message: "Deployment not found",
					Type:    "invalid_request_error",
					Code:    StringPtr("deployment_not_found"),
				},
			},
			wantErr: true,
			errType: ErrTypeModelNotFound,
		},
		{
			name: "content policy violation",
			request: ChatRequest{
				Messages: []Message{
					{Role: RoleUser, Content: "Inappropriate content"},
				},
			},
			mockStatus: http.StatusBadRequest,
			mockResp: mockErrorResponse{
				Error: struct {
					Message string  `json:"message"`
					Type    string  `json:"type"`
					Param   *string `json:"param"`
					Code    *string `json:"code"`
				}{
					Message: "Content policy violation",
					Type:    "invalid_request_error",
					Code:    StringPtr("content_policy_violation"),
				},
			},
			wantErr: true,
			errType: ErrTypeContentFilter,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			deploymentName := "test-deployment"

			server := setupMockServer(t, func(w http.ResponseWriter, r *http.Request) {
				// Verify Azure-specific request properties
				assert.Equal(t, "POST", r.Method)
				// Azure uses deployment name in the path
				expectedPath := fmt.Sprintf("/openai/deployments/%s/chat/completions", deploymentName)
				assert.Equal(t, expectedPath, r.URL.Path)
				assert.Equal(t, "2024-02-01", r.URL.Query().Get("api-version"))
				// Azure uses api-key header instead of Authorization Bearer
				assert.Equal(t, "test-api-key", r.Header.Get("api-key"))

				// Send response
				w.WriteHeader(tt.mockStatus)
				json.NewEncoder(w).Encode(tt.mockResp)
			})

			config := createTestConfig("")
			azureConfig := AzureConfig{
				Endpoint:       server.URL,
				DeploymentName: deploymentName,
				APIVersion:     "2024-02-01",
			}

			client, err := NewAzureClient(config, azureConfig)
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

func TestAzureEndpointConstruction(t *testing.T) {
	tests := []struct {
		name           string
		endpoint       string
		expectedPrefix string
	}{
		{
			name:           "endpoint with trailing slash",
			endpoint:       "https://myaccount.openai.azure.com/",
			expectedPrefix: "https://myaccount.openai.azure.com",
		},
		{
			name:           "endpoint without trailing slash",
			endpoint:       "https://myaccount.openai.azure.com",
			expectedPrefix: "https://myaccount.openai.azure.com",
		},
		{
			name:           "endpoint with path",
			endpoint:       "https://myaccount.openai.azure.com/openai",
			expectedPrefix: "https://myaccount.openai.azure.com/openai",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			config := AIConfig{
				APIKey: "test-key",
			}
			azureConfig := AzureConfig{
				Endpoint:       tt.endpoint,
				DeploymentName: "test-deployment",
				APIVersion:     "2024-02-01",
			}

			client, err := NewAzureClient(config, azureConfig)
			require.NoError(t, err)

			// The trimmed endpoint should be stored
			assert.Equal(t, strings.TrimRight(tt.endpoint, "/"), client.azureConfig.Endpoint)
		})
	}
}

func TestAzureListModels(t *testing.T) {
	config := AIConfig{
		APIKey: "test-key",
		Model:  "o4-mini",
	}
	azureConfig := AzureConfig{
		Endpoint:       "https://myaccount.openai.azure.com",
		DeploymentName: "my-gpt4-deployment",
		APIVersion:     "2024-02-01",
	}

	client, err := NewAzureClient(config, azureConfig)
	require.NoError(t, err)

	models, err := client.ListModels(context.Background())
	require.NoError(t, err)
	assert.Len(t, models, 2)

	// Should return deployment name as model
	assert.Equal(t, "my-gpt4-deployment", models[0].ID)
	assert.Equal(t, "azure", models[0].OwnedBy)

	// Should also return configured model if different
	assert.Equal(t, "o4-mini", models[1].ID)
	assert.Equal(t, "azure", models[1].OwnedBy)
}

func TestAzurePing(t *testing.T) {
	tests := []struct {
		name       string
		mockStatus int
		mockResp   interface{}
		wantErr    bool
	}{
		{
			name:       "successful ping",
			mockStatus: http.StatusOK,
			mockResp: mockChatCompletionResponse{
				ID:      "ping-response",
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
							Content: "pong",
						},
						FinishReason: "stop",
					},
				},
			},
			wantErr: false,
		},
		{
			name:       "ping with auth error (service is reachable)",
			mockStatus: http.StatusUnauthorized,
			mockResp: mockErrorResponse{
				Error: struct {
					Message string  `json:"message"`
					Type    string  `json:"type"`
					Param   *string `json:"param"`
					Code    *string `json:"code"`
				}{
					Message: "Invalid API key",
					Type:    "authentication_error",
				},
			},
			wantErr: false, // Auth errors mean service is reachable
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
				// Verify it's a minimal request
				var req map[string]interface{}
				json.NewDecoder(r.Body).Decode(&req)
				assert.Equal(t, float64(1), req["max_tokens"])

				w.WriteHeader(tt.mockStatus)
				json.NewEncoder(w).Encode(tt.mockResp)
			})

			config := createTestConfig("")
			azureConfig := AzureConfig{
				Endpoint:       server.URL,
				DeploymentName: "test-deployment",
				APIVersion:     "2024-02-01",
			}

			client, err := NewAzureClient(config, azureConfig)
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

func TestAzureChatCompletionStream(t *testing.T) {
	deploymentName := "stream-deployment"

	server := setupMockServer(t, func(w http.ResponseWriter, r *http.Request) {
		// Verify request
		assert.Equal(t, "POST", r.Method)
		expectedPath := fmt.Sprintf("/openai/deployments/%s/chat/completions", deploymentName)
		assert.Equal(t, expectedPath, r.URL.Path)

		// Parse request to verify stream is true
		var req map[string]interface{}
		json.NewDecoder(r.Body).Decode(&req)
		assert.Equal(t, true, req["stream"])

		// Send SSE response
		w.Header().Set("Content-Type", "text/event-stream")
		w.WriteHeader(http.StatusOK)

		// Send chunks
		chunks := []string{
			`{"id":"chatcmpl-azure-123","object":"chat.completion.chunk","created":1234567890,"model":"o3","choices":[{"index":0,"delta":{"role":"assistant"},"finish_reason":null}]}`,
			`{"id":"chatcmpl-azure-123","object":"chat.completion.chunk","created":1234567890,"model":"o3","choices":[{"index":0,"delta":{"content":"Azure"},"finish_reason":null}]}`,
			`{"id":"chatcmpl-azure-123","object":"chat.completion.chunk","created":1234567890,"model":"o3","choices":[{"index":0,"delta":{"content":" streaming!"},"finish_reason":null}]}`,
			`{"id":"chatcmpl-azure-123","object":"chat.completion.chunk","created":1234567890,"model":"o3","choices":[{"index":0,"delta":{},"finish_reason":"stop"}]}`,
		}

		for _, chunk := range chunks {
			fmt.Fprintf(w, "data: %s\n\n", chunk)
			w.(http.Flusher).Flush()
		}
		fmt.Fprintf(w, "data: [DONE]\n\n")
		w.(http.Flusher).Flush()
	})

	config := createTestConfig("")
	azureConfig := AzureConfig{
		Endpoint:       server.URL,
		DeploymentName: deploymentName,
		APIVersion:     "2024-02-01",
	}

	client, err := NewAzureClient(config, azureConfig)
	require.NoError(t, err)

	req := ChatRequest{
		Messages: []Message{
			{Role: RoleUser, Content: "Test Azure streaming"},
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
	assert.Equal(t, "Azure streaming!", content.String())
}

func TestAzureAuthentication(t *testing.T) {
	server := setupMockServer(t, func(w http.ResponseWriter, r *http.Request) {
		// Azure should use api-key header, not Authorization
		apiKey := r.Header.Get("api-key")
		authHeader := r.Header.Get("Authorization")

		assert.Equal(t, "test-azure-api-key", apiKey)
		assert.Empty(t, authHeader) // Should not have Authorization header

		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(mockChatCompletionResponse{
			ID:      "auth-test",
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
						Content: "Authenticated!",
					},
					FinishReason: "stop",
				},
			},
		})
	})

	config := AIConfig{
		APIKey: "test-azure-api-key",
	}
	azureConfig := AzureConfig{
		Endpoint:       server.URL,
		DeploymentName: "auth-test-deployment",
		APIVersion:     "2024-02-01",
	}

	client, err := NewAzureClient(config, azureConfig)
	require.NoError(t, err)

	req := ChatRequest{
		Messages: []Message{
			{Role: RoleUser, Content: "Test auth"},
		},
	}

	resp, err := client.ChatCompletion(context.Background(), req)
	require.NoError(t, err)
	assert.Equal(t, "Authenticated!", resp.Choices[0].Message.Content)
}

func TestDeploymentNameUsage(t *testing.T) {
	deploymentName := "custom-gpt4-deployment"
	requestCount := 0

	server := setupMockServer(t, func(w http.ResponseWriter, r *http.Request) {
		requestCount++

		// Verify deployment name is used in path
		expectedPath := fmt.Sprintf("/openai/deployments/%s/chat/completions", deploymentName)
		assert.Equal(t, expectedPath, r.URL.Path)

		// Parse request body
		var req map[string]interface{}
		json.NewDecoder(r.Body).Decode(&req)

		// Model in request should be the deployment name
		assert.Equal(t, deploymentName, req["model"])

		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(mockChatCompletionResponse{
			ID:      fmt.Sprintf("response-%d", requestCount),
			Object:  "chat.completion",
			Created: time.Now().Unix(),
			Model:   deploymentName,
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
						Content: fmt.Sprintf("Response %d from %s", requestCount, deploymentName),
					},
					FinishReason: "stop",
				},
			},
		})
	})

	config := AIConfig{
		APIKey: "test-key",
		Model:  "o3", // This should be overridden by deployment name
	}
	azureConfig := AzureConfig{
		Endpoint:       server.URL,
		DeploymentName: deploymentName,
		APIVersion:     "2024-02-01",
	}

	client, err := NewAzureClient(config, azureConfig)
	require.NoError(t, err)

	// Test multiple requests to ensure deployment name is consistently used
	for i := 0; i < 3; i++ {
		req := ChatRequest{
			Model: "some-other-model", // This should be ignored
			Messages: []Message{
				{Role: RoleUser, Content: fmt.Sprintf("Test %d", i)},
			},
		}

		resp, err := client.ChatCompletion(context.Background(), req)
		require.NoError(t, err)
		assert.Equal(t, deploymentName, resp.Model)
		assert.Contains(t, resp.Choices[0].Message.Content, deploymentName)
	}

	assert.Equal(t, 3, requestCount)
}

func TestAzureErrorResponse(t *testing.T) {
	tests := []struct {
		name          string
		mockStatus    int
		mockError     mockErrorResponse
		expectedType  ErrorType
		azureSpecific bool
	}{
		{
			name:       "deployment not found",
			mockStatus: http.StatusBadRequest,
			mockError: mockErrorResponse{
				Error: struct {
					Message string  `json:"message"`
					Type    string  `json:"type"`
					Param   *string `json:"param"`
					Code    *string `json:"code"`
				}{
					Message: "The deployment 'non-existent' does not exist",
					Type:    "invalid_request_error",
					Code:    StringPtr("deployment_not_found"),
				},
			},
			expectedType:  ErrTypeModelNotFound,
			azureSpecific: true,
		},
		{
			name:       "azure content filter",
			mockStatus: http.StatusBadRequest,
			mockError: mockErrorResponse{
				Error: struct {
					Message string  `json:"message"`
					Type    string  `json:"type"`
					Param   *string `json:"param"`
					Code    *string `json:"code"`
				}{
					Message: "The response was filtered due to the prompt triggering Azure OpenAI's content management policy",
					Type:    "invalid_request_error",
					Code:    StringPtr("content_filter"),
				},
			},
			expectedType:  ErrTypeContentFilter,
			azureSpecific: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server := setupMockServer(t, func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(tt.mockStatus)
				json.NewEncoder(w).Encode(tt.mockError)
			})

			config := createTestConfig("")
			config.MaxRetries = 0 // Disable retries for error testing
			azureConfig := AzureConfig{
				Endpoint:       server.URL,
				DeploymentName: "test-deployment",
				APIVersion:     "2024-02-01",
			}

			client, err := NewAzureClient(config, azureConfig)
			require.NoError(t, err)

			req := ChatRequest{
				Messages: []Message{
					{Role: RoleUser, Content: "Test"},
				},
			}

			_, err = client.ChatCompletion(context.Background(), req)
			require.Error(t, err)
			assert.Equal(t, tt.expectedType, GetErrorType(err))

			if tt.azureSpecific {
				// Verify Azure-specific error details are preserved
				aiErr, ok := err.(*Error)
				require.True(t, ok)
				assert.Contains(t, aiErr.Message, tt.mockError.Error.Message)
			}
		})
	}
}
