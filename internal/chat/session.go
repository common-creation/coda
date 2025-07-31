package chat

import (
	"fmt"
	"sync"
	"time"

	"github.com/common-creation/coda/internal/ai"
	"github.com/google/uuid"
)

// Session represents a chat session
type Session struct {
	ID         string                 `json:"id"`
	StartedAt  time.Time              `json:"started_at"`
	LastActive time.Time              `json:"last_active"`
	Messages   []ai.Message           `json:"messages"`
	Context    map[string]interface{} `json:"context"`
	MaxTokens  int                    `json:"max_tokens"`
	TokenCount int                    `json:"token_count"`
}

// SessionManager manages chat sessions
type SessionManager struct {
	sessions       map[string]*Session
	currentSession string
	mu             sync.RWMutex
	maxAge         time.Duration
	maxTokens      int
	tokenizer      TokenCounter
}

// TokenCounter interface for counting tokens in messages
type TokenCounter interface {
	CountTokens(text string) int
}

// SimpleTokenCounter provides a simple token counting implementation
type SimpleTokenCounter struct{}

// CountTokens estimates token count (roughly 4 chars per token)
func (s *SimpleTokenCounter) CountTokens(text string) int {
	// Simple estimation: ~4 characters per token on average
	return len(text) / 4
}

// NewSessionManager creates a new session manager
func NewSessionManager(maxAge time.Duration, maxTokens int) *SessionManager {
	sm := &SessionManager{
		sessions:  make(map[string]*Session),
		maxAge:    maxAge,
		maxTokens: maxTokens,
		tokenizer: &SimpleTokenCounter{},
	}

	// Start cleanup goroutine
	go sm.cleanupRoutine()

	return sm
}

// SetTokenCounter sets a custom token counter
func (sm *SessionManager) SetTokenCounter(counter TokenCounter) {
	sm.mu.Lock()
	defer sm.mu.Unlock()
	sm.tokenizer = counter
}

// NewSession creates a new chat session
func (sm *SessionManager) NewSession() *Session {
	sm.mu.Lock()
	defer sm.mu.Unlock()

	session := &Session{
		ID:         uuid.New().String(),
		StartedAt:  time.Now(),
		LastActive: time.Now(),
		Messages:   make([]ai.Message, 0),
		Context:    make(map[string]interface{}),
		MaxTokens:  sm.maxTokens,
		TokenCount: 0,
	}

	sm.sessions[session.ID] = session
	return session
}

// GetSession retrieves a session by ID
func (sm *SessionManager) GetSession(id string) (*Session, error) {
	sm.mu.RLock()
	defer sm.mu.RUnlock()

	session, exists := sm.sessions[id]
	if !exists {
		return nil, fmt.Errorf("session not found: %s", id)
	}

	// Update last active time
	session.LastActive = time.Now()

	return session, nil
}

// UpdateSession adds a message to the session
func (sm *SessionManager) UpdateSession(id string, msg ai.Message) error {
	sm.mu.Lock()
	defer sm.mu.Unlock()

	session, exists := sm.sessions[id]
	if !exists {
		return fmt.Errorf("session not found: %s", id)
	}

	// Count tokens in the new message
	msgTokens := sm.tokenizer.CountTokens(msg.Content)

	// Add message
	session.Messages = append(session.Messages, msg)
	session.TokenCount += msgTokens
	session.LastActive = time.Now()

	// Trim messages if token limit exceeded
	if session.TokenCount > session.MaxTokens {
		sm.trimMessages(session)
	}

	return nil
}

// trimMessages removes old messages to stay within token limit
func (sm *SessionManager) trimMessages(session *Session) {
	// Keep system message if it exists
	startIdx := 0
	if len(session.Messages) > 0 && session.Messages[0].Role == "system" {
		startIdx = 1
	}

	// Remove messages from the beginning (after system message)
	for session.TokenCount > session.MaxTokens && len(session.Messages) > startIdx+1 {
		removedMsg := session.Messages[startIdx]
		removedTokens := sm.tokenizer.CountTokens(removedMsg.Content)

		// Remove the message
		session.Messages = append(session.Messages[:startIdx], session.Messages[startIdx+1:]...)
		session.TokenCount -= removedTokens
	}
}

// SetContext sets a context value for the session
func (sm *SessionManager) SetContext(id string, key string, value interface{}) error {
	sm.mu.Lock()
	defer sm.mu.Unlock()

	session, exists := sm.sessions[id]
	if !exists {
		return fmt.Errorf("session not found: %s", id)
	}

	session.Context[key] = value
	session.LastActive = time.Now()

	return nil
}

// GetContext retrieves a context value from the session
func (sm *SessionManager) GetContext(id string, key string) (interface{}, error) {
	sm.mu.RLock()
	defer sm.mu.RUnlock()

	session, exists := sm.sessions[id]
	if !exists {
		return nil, fmt.Errorf("session not found: %s", id)
	}

	value, exists := session.Context[key]
	if !exists {
		return nil, fmt.Errorf("context key not found: %s", key)
	}

	return value, nil
}

// GetMessages retrieves all messages from a session
func (sm *SessionManager) GetMessages(id string) ([]ai.Message, error) {
	sm.mu.RLock()
	defer sm.mu.RUnlock()

	session, exists := sm.sessions[id]
	if !exists {
		return nil, fmt.Errorf("session not found: %s", id)
	}

	// Return a copy to prevent external modification
	messages := make([]ai.Message, len(session.Messages))
	copy(messages, session.Messages)

	return messages, nil
}

// CleanupSessions removes expired sessions
func (sm *SessionManager) CleanupSessions() {
	sm.mu.Lock()
	defer sm.mu.Unlock()

	now := time.Now()
	for id, session := range sm.sessions {
		if now.Sub(session.LastActive) > sm.maxAge {
			delete(sm.sessions, id)
		}
	}
}

// cleanupRoutine runs periodic cleanup
func (sm *SessionManager) cleanupRoutine() {
	ticker := time.NewTicker(5 * time.Minute)
	defer ticker.Stop()

	for range ticker.C {
		sm.CleanupSessions()
	}
}

// ListSessions returns all active session IDs
func (sm *SessionManager) ListSessions() []string {
	sm.mu.RLock()
	defer sm.mu.RUnlock()

	ids := make([]string, 0, len(sm.sessions))
	for id := range sm.sessions {
		ids = append(ids, id)
	}

	return ids
}

// DeleteSession removes a specific session
func (sm *SessionManager) DeleteSession(id string) error {
	sm.mu.Lock()
	defer sm.mu.Unlock()

	if _, exists := sm.sessions[id]; !exists {
		return fmt.Errorf("session not found: %s", id)
	}

	delete(sm.sessions, id)
	return nil
}

// GetSessionInfo returns session metadata without messages
func (sm *SessionManager) GetSessionInfo(id string) (map[string]interface{}, error) {
	sm.mu.RLock()
	defer sm.mu.RUnlock()

	session, exists := sm.sessions[id]
	if !exists {
		return nil, fmt.Errorf("session not found: %s", id)
	}

	info := map[string]interface{}{
		"id":            session.ID,
		"started_at":    session.StartedAt,
		"last_active":   session.LastActive,
		"message_count": len(session.Messages),
		"token_count":   session.TokenCount,
		"max_tokens":    session.MaxTokens,
	}

	return info, nil
}

// ClearMessages removes all messages from a session except system message
func (sm *SessionManager) ClearMessages(id string) error {
	sm.mu.Lock()
	defer sm.mu.Unlock()

	session, exists := sm.sessions[id]
	if !exists {
		return fmt.Errorf("session not found: %s", id)
	}

	// Keep system message if it exists
	if len(session.Messages) > 0 && session.Messages[0].Role == "system" {
		systemMsg := session.Messages[0]
		session.Messages = []ai.Message{systemMsg}
		session.TokenCount = sm.tokenizer.CountTokens(systemMsg.Content)
	} else {
		session.Messages = []ai.Message{}
		session.TokenCount = 0
	}

	session.LastActive = time.Now()

	return nil
}

// GetCurrent returns the current active session
func (sm *SessionManager) GetCurrent() *Session {
	sm.mu.RLock()
	defer sm.mu.RUnlock()
	
	if sm.currentSession == "" {
		return nil
	}
	
	session, exists := sm.sessions[sm.currentSession]
	if !exists {
		return nil
	}
	
	return session
}

// CreateSession creates a new session and sets it as current
func (sm *SessionManager) CreateSession() (string, error) {
	session := sm.NewSession()
	
	sm.mu.Lock()
	sm.currentSession = session.ID
	sm.mu.Unlock()
	
	return session.ID, nil
}

// SetCurrent sets the current session by ID
func (sm *SessionManager) SetCurrent(id string) error {
	sm.mu.Lock()
	defer sm.mu.Unlock()
	
	if _, exists := sm.sessions[id]; !exists {
		return fmt.Errorf("session not found: %s", id)
	}
	
	sm.currentSession = id
	return nil
}

// AddMessage adds a message to a session
func (sm *SessionManager) AddMessage(sessionID string, msg ai.Message) error {
	return sm.UpdateSession(sessionID, msg)
}
