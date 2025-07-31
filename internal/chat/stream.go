package chat

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"strings"
	"sync"
	"time"

	"github.com/common-creation/coda/internal/ai"
)

// StreamHandler manages streaming responses from the AI
type StreamHandler struct {
	output     io.Writer
	buffer     *bytes.Buffer
	onChunk    func(string)
	onComplete func(string)
	onError    func(error)
	onToolCall func(ai.ToolCall)
	mu         sync.Mutex

	// Performance optimization
	lastFlush   time.Time
	flushDelay  time.Duration
	partialJSON strings.Builder
}

// StreamHandlerOption is a function that configures a StreamHandler
type StreamHandlerOption func(*StreamHandler)

// WithOutput sets the output writer for the stream handler
func WithOutput(w io.Writer) StreamHandlerOption {
	return func(h *StreamHandler) {
		h.output = w
	}
}

// WithChunkHandler sets the chunk handler callback
func WithChunkHandler(fn func(string)) StreamHandlerOption {
	return func(h *StreamHandler) {
		h.onChunk = fn
	}
}

// WithCompleteHandler sets the completion handler callback
func WithCompleteHandler(fn func(string)) StreamHandlerOption {
	return func(h *StreamHandler) {
		h.onComplete = fn
	}
}

// WithErrorHandler sets the error handler callback
func WithErrorHandler(fn func(error)) StreamHandlerOption {
	return func(h *StreamHandler) {
		h.onError = fn
	}
}

// WithToolCallHandler sets the tool call handler callback
func WithToolCallHandler(fn func(ai.ToolCall)) StreamHandlerOption {
	return func(h *StreamHandler) {
		h.onToolCall = fn
	}
}

// NewStreamHandler creates a new stream handler with the given options
func NewStreamHandler(opts ...StreamHandlerOption) *StreamHandler {
	h := &StreamHandler{
		buffer:     &bytes.Buffer{},
		flushDelay: 50 * time.Millisecond, // Default flush delay
		lastFlush:  time.Now(),
	}

	// Apply options
	for _, opt := range opts {
		opt(h)
	}

	// Set default output if not provided
	if h.output == nil {
		h.output = io.Discard
	}

	return h
}

// ProcessStream processes a streaming response from the AI
func (h *StreamHandler) ProcessStream(ctx context.Context, stream ai.StreamReader) error {
	defer h.flush() // Ensure final flush

	var fullContent strings.Builder
	var toolCalls []ai.ToolCall
	var currentToolCall *ai.ToolCall

	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
			chunk, err := stream.Read()
			if err == io.EOF {
				// Complete processing
				if h.onComplete != nil {
					h.onComplete(fullContent.String())
				}
				return nil
			}
			if err != nil {
				if h.onError != nil {
					h.onError(err)
				}
				return fmt.Errorf("error reading stream: %w", err)
			}

			// Process chunk
			if err := h.processChunk(chunk, &fullContent, &toolCalls, &currentToolCall); err != nil {
				if h.onError != nil {
					h.onError(err)
				}
				return err
			}
		}
	}
}

// processChunk processes a single chunk from the stream
func (h *StreamHandler) processChunk(chunk *ai.StreamChunk, fullContent *strings.Builder, toolCalls *[]ai.ToolCall, currentToolCall **ai.ToolCall) error {
	h.mu.Lock()
	defer h.mu.Unlock()

	if chunk.Choices == nil || len(chunk.Choices) == 0 {
		return nil
	}

	delta := chunk.Choices[0].Delta

	// Handle content
	if delta.Content != "" {
		fullContent.WriteString(delta.Content)
		h.buffer.WriteString(delta.Content)

		if h.onChunk != nil {
			h.onChunk(delta.Content)
		}

		// Check if we should flush
		if time.Since(h.lastFlush) > h.flushDelay {
			h.flushLocked()
		}
	}

	// Handle tool calls
	if delta.ToolCalls != nil && len(delta.ToolCalls) > 0 {
		for _, tc := range delta.ToolCalls {
			// Check if this is a continuation of an existing tool call
			if tc.Index >= 0 && tc.Index < len(*toolCalls) {
				// Update existing tool call
				existingCall := &(*toolCalls)[tc.Index]
				if tc.Function.Name != "" {
					existingCall.Function.Name = tc.Function.Name
				}
				if tc.Function.Arguments != "" {
					existingCall.Function.Arguments += tc.Function.Arguments
				}
				*currentToolCall = existingCall
			} else {
				// New tool call
				*toolCalls = append(*toolCalls, tc)
				*currentToolCall = &(*toolCalls)[len(*toolCalls)-1]
			}

			// Try to detect complete tool calls
			if *currentToolCall != nil && h.isCompleteToolCall(*currentToolCall) {
				if h.onToolCall != nil {
					h.onToolCall(**currentToolCall)
				}
				*currentToolCall = nil
			}
		}
	}

	// Handle finish reason
	if chunk.Choices[0].FinishReason != nil && *chunk.Choices[0].FinishReason != "" {
		h.flushLocked()

		// Process any remaining tool calls
		for _, tc := range *toolCalls {
			if h.onToolCall != nil && h.isCompleteToolCall(&tc) {
				h.onToolCall(tc)
			}
		}
	}

	return nil
}

// isCompleteToolCall checks if a tool call has all required fields
func (h *StreamHandler) isCompleteToolCall(tc *ai.ToolCall) bool {
	if tc == nil || tc.Function.Name == "" {
		return false
	}

	// Check if arguments form valid JSON
	if tc.Function.Arguments != "" {
		var temp interface{}
		if err := json.Unmarshal([]byte(tc.Function.Arguments), &temp); err != nil {
			// Not complete JSON yet
			return false
		}
	}

	return true
}

// flush writes any buffered content to the output
func (h *StreamHandler) flush() {
	h.mu.Lock()
	defer h.mu.Unlock()
	h.flushLocked()
}

// flushLocked writes buffered content while holding the lock
func (h *StreamHandler) flushLocked() {
	if h.buffer.Len() > 0 {
		h.output.Write(h.buffer.Bytes())
		h.buffer.Reset()
		h.lastFlush = time.Now()
	}
}

// Reset resets the stream handler state
func (h *StreamHandler) Reset() {
	h.mu.Lock()
	defer h.mu.Unlock()

	h.buffer.Reset()
	h.partialJSON.Reset()
	h.lastFlush = time.Now()
}

// StreamProcessor provides a higher-level interface for stream processing
type StreamProcessor struct {
	handler     *StreamHandler
	formatters  []ContentFormatter
	rateLimiter *RateLimiter
}

// ContentFormatter defines an interface for formatting streamed content
type ContentFormatter interface {
	Format(content string) string
}

// MarkdownFormatter provides basic markdown formatting
type MarkdownFormatter struct{}

func (f *MarkdownFormatter) Format(content string) string {
	// Basic markdown processing
	// This is simplified - a full implementation would handle more cases
	return content
}

// RateLimiter provides rate limiting for stream output
type RateLimiter struct {
	ticker     *time.Ticker
	lastUpdate time.Time
	mu         sync.Mutex
}

// NewRateLimiter creates a new rate limiter
func NewRateLimiter(rate time.Duration) *RateLimiter {
	return &RateLimiter{
		ticker:     time.NewTicker(rate),
		lastUpdate: time.Now(),
	}
}

// Allow checks if an update is allowed based on the rate limit
func (r *RateLimiter) Allow() bool {
	r.mu.Lock()
	defer r.mu.Unlock()

	select {
	case <-r.ticker.C:
		r.lastUpdate = time.Now()
		return true
	default:
		return false
	}
}

// Stop stops the rate limiter
func (r *RateLimiter) Stop() {
	r.ticker.Stop()
}

// NewStreamProcessor creates a new stream processor
func NewStreamProcessor(handler *StreamHandler, formatters ...ContentFormatter) *StreamProcessor {
	return &StreamProcessor{
		handler:     handler,
		formatters:  formatters,
		rateLimiter: NewRateLimiter(30 * time.Millisecond),
	}
}

// Process processes a stream with formatting and rate limiting
func (p *StreamProcessor) Process(ctx context.Context, stream ai.StreamReader) error {
	defer p.rateLimiter.Stop()

	// Wrap the chunk handler to apply formatting
	originalHandler := p.handler.onChunk
	p.handler.onChunk = func(chunk string) {
		// Apply formatters
		formatted := chunk
		for _, formatter := range p.formatters {
			formatted = formatter.Format(formatted)
		}

		// Apply rate limiting if needed
		if p.rateLimiter.Allow() && originalHandler != nil {
			originalHandler(formatted)
		}
	}

	return p.handler.ProcessStream(ctx, stream)
}

// StreamMetrics tracks streaming performance metrics
type StreamMetrics struct {
	StartTime      time.Time
	EndTime        time.Time
	BytesReceived  int64
	ChunksReceived int64
	Errors         int64
	mu             sync.Mutex
}

// NewStreamMetrics creates a new metrics tracker
func NewStreamMetrics() *StreamMetrics {
	return &StreamMetrics{
		StartTime: time.Now(),
	}
}

// RecordChunk records a received chunk
func (m *StreamMetrics) RecordChunk(size int) {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.ChunksReceived++
	m.BytesReceived += int64(size)
}

// RecordError records an error
func (m *StreamMetrics) RecordError() {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.Errors++
}

// Complete marks the streaming as complete
func (m *StreamMetrics) Complete() {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.EndTime = time.Now()
}

// Duration returns the streaming duration
func (m *StreamMetrics) Duration() time.Duration {
	m.mu.Lock()
	defer m.mu.Unlock()

	if m.EndTime.IsZero() {
		return time.Since(m.StartTime)
	}
	return m.EndTime.Sub(m.StartTime)
}

// Throughput returns the bytes per second
func (m *StreamMetrics) Throughput() float64 {
	duration := m.Duration().Seconds()
	if duration == 0 {
		return 0
	}

	m.mu.Lock()
	defer m.mu.Unlock()

	return float64(m.BytesReceived) / duration
}
