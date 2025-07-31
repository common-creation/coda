package ai

import (
	"context"
	"fmt"
	"io"
	"time"
)

// DummyClient is a temporary AI client for testing TUI
type DummyClient struct {
	model string
}

// NewDummyClient creates a dummy AI client
func NewDummyClient(model string) *DummyClient {
	if model == "" {
		model = "dummy-model"
	}
	return &DummyClient{model: model}
}

// ChatCompletion implements Client interface
func (d *DummyClient) ChatCompletion(ctx context.Context, req ChatRequest) (*ChatResponse, error) {
	return &ChatResponse{
		ID:      "dummy-" + fmt.Sprint(time.Now().Unix()),
		Object:  "chat.completion",
		Created: time.Now().Unix(),
		Model:   d.model,
		Choices: []Choice{
			{
				Index: 0,
				Message: Message{
					Role:    "assistant",
					Content: "This is a dummy response. The AI client is not properly configured yet. To use real AI, please configure OpenAI or Azure OpenAI in the config file.",
				},
				FinishReason: "stop",
			},
		},
		Usage: Usage{
			PromptTokens:     10,
			CompletionTokens: 20,
			TotalTokens:      30,
		},
	}, nil
}

// dummyStreamReader implements StreamReader interface
type dummyStreamReader struct {
	chunks   []string
	index    int
	model    string
	finished bool
}

func (r *dummyStreamReader) Read() (*StreamChunk, error) {
	if r.finished {
		return nil, io.EOF
	}
	
	if r.index >= len(r.chunks) {
		r.finished = true
		finishReason := "stop"
		return &StreamChunk{
			ID:      "dummy-stream",
			Object:  "chat.completion.chunk",
			Created: time.Now().Unix(),
			Model:   r.model,
			Choices: []StreamChoice{
				{
					Index:        0,
					Delta:        StreamDelta{},
					FinishReason: &finishReason,
				},
			},
		}, nil
	}
	
	chunk := &StreamChunk{
		ID:      "dummy-stream",
		Object:  "chat.completion.chunk",
		Created: time.Now().Unix(),
		Model:   r.model,
		Choices: []StreamChoice{
			{
				Index: 0,
				Delta: StreamDelta{
					Content: r.chunks[r.index],
				},
			},
		},
	}
	
	r.index++
	return chunk, nil
}

func (r *dummyStreamReader) Close() error {
	return nil
}

// ChatCompletionStream implements Client interface
func (d *DummyClient) ChatCompletionStream(ctx context.Context, req ChatRequest) (StreamReader, error) {
	message := "This is a dummy streaming response. The AI client is not properly configured yet. Please configure OpenAI or Azure OpenAI."
	
	// Split message into words for streaming
	chunks := []string{}
	words := []string{}
	current := ""
	for _, ch := range message {
		if ch == ' ' {
			if current != "" {
				words = append(words, current+" ")
				current = ""
			}
		} else {
			current += string(ch)
		}
	}
	if current != "" {
		words = append(words, current)
	}
	
	// Convert words to chunks
	for _, word := range words {
		chunks = append(chunks, word)
	}
	
	return &dummyStreamReader{
		chunks: chunks,
		model:  d.model,
	}, nil
}

// ListModels implements Client interface
func (d *DummyClient) ListModels(ctx context.Context) ([]Model, error) {
	return []Model{
		{
			ID:         d.model,
			Object:     "model",
			Created:    time.Now().Unix(),
			OwnedBy:    "dummy",
			Permission: []string{"read"},
		},
	}, nil
}

// Ping implements Client interface
func (d *DummyClient) Ping(ctx context.Context) error {
	return nil
}