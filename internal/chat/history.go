package chat

import (
	"compress/gzip"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"sync"
	"time"
)

// History manages chat session history
type History struct {
	Sessions    []SessionSummary `json:"sessions"`
	TotalChats  int              `json:"total_chats"`
	StoragePath string           `json:"storage_path"`
	mu          sync.RWMutex
}

// SessionSummary contains summary information about a session
type SessionSummary struct {
	ID        string    `json:"id"`
	Title     string    `json:"title"`
	StartTime time.Time `json:"start_time"`
	EndTime   time.Time `json:"end_time"`
	Messages  int       `json:"messages"`
	Tags      []string  `json:"tags"`
}

// NewHistory creates a new history manager
func NewHistory(storagePath string) (*History, error) {
	// Create storage directory if it doesn't exist
	if err := os.MkdirAll(storagePath, 0755); err != nil {
		return nil, fmt.Errorf("failed to create storage directory: %w", err)
	}

	h := &History{
		Sessions:    make([]SessionSummary, 0),
		StoragePath: storagePath,
	}

	// Load existing history index
	if err := h.loadIndex(); err != nil {
		// If index doesn't exist, that's okay
		if !os.IsNotExist(err) {
			return nil, fmt.Errorf("failed to load history index: %w", err)
		}
	}

	return h, nil
}

// Save saves a session to history
func (h *History) Save(session *Session) error {
	h.mu.Lock()
	defer h.mu.Unlock()

	// Generate title if not set
	title := h.generateTitle(session)

	// Create session summary
	summary := SessionSummary{
		ID:        session.ID,
		Title:     title,
		StartTime: session.StartedAt,
		EndTime:   time.Now(),
		Messages:  len(session.Messages),
		Tags:      h.extractTags(session),
	}

	// Save session data
	sessionPath := filepath.Join(h.StoragePath, fmt.Sprintf("%s.json.gz", session.ID))
	if err := h.saveSessionData(sessionPath, session); err != nil {
		return fmt.Errorf("failed to save session data: %w", err)
	}

	// Update index
	h.Sessions = append(h.Sessions, summary)
	h.TotalChats++

	// Save updated index
	if err := h.saveIndex(); err != nil {
		return fmt.Errorf("failed to save history index: %w", err)
	}

	return nil
}

// Load loads a session from history
func (h *History) Load(id string) (*Session, error) {
	h.mu.RLock()
	defer h.mu.RUnlock()

	// Check if session exists in index
	found := false
	for _, s := range h.Sessions {
		if s.ID == id {
			found = true
			break
		}
	}

	if !found {
		return nil, fmt.Errorf("session not found in history: %s", id)
	}

	// Load session data
	sessionPath := filepath.Join(h.StoragePath, fmt.Sprintf("%s.json.gz", id))
	session, err := h.loadSessionData(sessionPath)
	if err != nil {
		return nil, fmt.Errorf("failed to load session data: %w", err)
	}

	return session, nil
}

// Search searches for sessions matching the query
func (h *History) Search(query string) ([]SessionSummary, error) {
	h.mu.RLock()
	defer h.mu.RUnlock()

	// Compile regex if query looks like a regex
	var regex *regexp.Regexp
	if strings.HasPrefix(query, "/") && strings.HasSuffix(query, "/") {
		pattern := query[1 : len(query)-1]
		var err error
		regex, err = regexp.Compile(pattern)
		if err != nil {
			return nil, fmt.Errorf("invalid regex pattern: %w", err)
		}
	}

	results := make([]SessionSummary, 0)

	for _, summary := range h.Sessions {
		// Check title
		if regex != nil {
			if regex.MatchString(summary.Title) {
				results = append(results, summary)
				continue
			}
		} else {
			if strings.Contains(strings.ToLower(summary.Title), strings.ToLower(query)) {
				results = append(results, summary)
				continue
			}
		}

		// Check tags
		for _, tag := range summary.Tags {
			if regex != nil {
				if regex.MatchString(tag) {
					results = append(results, summary)
					break
				}
			} else {
				if strings.Contains(strings.ToLower(tag), strings.ToLower(query)) {
					results = append(results, summary)
					break
				}
			}
		}
	}

	return results, nil
}

// SearchByDateRange searches for sessions within a date range
func (h *History) SearchByDateRange(start, end time.Time) []SessionSummary {
	h.mu.RLock()
	defer h.mu.RUnlock()

	results := make([]SessionSummary, 0)

	for _, summary := range h.Sessions {
		if summary.StartTime.After(start) && summary.StartTime.Before(end) {
			results = append(results, summary)
		}
	}

	// Sort by start time
	sort.Slice(results, func(i, j int) bool {
		return results[i].StartTime.Before(results[j].StartTime)
	})

	return results
}

// SearchByTags searches for sessions with specific tags
func (h *History) SearchByTags(tags []string) []SessionSummary {
	h.mu.RLock()
	defer h.mu.RUnlock()

	results := make([]SessionSummary, 0)

	for _, summary := range h.Sessions {
		// Check if session has all requested tags
		hasAllTags := true
		for _, searchTag := range tags {
			found := false
			for _, sessionTag := range summary.Tags {
				if sessionTag == searchTag {
					found = true
					break
				}
			}
			if !found {
				hasAllTags = false
				break
			}
		}

		if hasAllTags {
			results = append(results, summary)
		}
	}

	return results
}

// Delete removes a session from history
func (h *History) Delete(id string) error {
	h.mu.Lock()
	defer h.mu.Unlock()

	// Find and remove from index
	found := false
	newSessions := make([]SessionSummary, 0, len(h.Sessions)-1)

	for _, s := range h.Sessions {
		if s.ID == id {
			found = true
			h.TotalChats--
		} else {
			newSessions = append(newSessions, s)
		}
	}

	if !found {
		return fmt.Errorf("session not found in history: %s", id)
	}

	h.Sessions = newSessions

	// Delete session file
	sessionPath := filepath.Join(h.StoragePath, fmt.Sprintf("%s.json.gz", id))
	if err := os.Remove(sessionPath); err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("failed to delete session file: %w", err)
	}

	// Save updated index
	if err := h.saveIndex(); err != nil {
		return fmt.Errorf("failed to save history index: %w", err)
	}

	return nil
}

// Export exports history in the specified format
func (h *History) Export(format string) ([]byte, error) {
	h.mu.RLock()
	defer h.mu.RUnlock()

	switch format {
	case "json":
		return json.MarshalIndent(h, "", "  ")
	case "csv":
		return h.exportCSV()
	default:
		return nil, fmt.Errorf("unsupported export format: %s", format)
	}
}

// GetRecent returns the most recent sessions
func (h *History) GetRecent(limit int) []SessionSummary {
	h.mu.RLock()
	defer h.mu.RUnlock()

	// Sort by start time (newest first)
	sorted := make([]SessionSummary, len(h.Sessions))
	copy(sorted, h.Sessions)

	sort.Slice(sorted, func(i, j int) bool {
		return sorted[i].StartTime.After(sorted[j].StartTime)
	})

	// Return limited results
	if limit > 0 && limit < len(sorted) {
		return sorted[:limit]
	}

	return sorted
}

// generateTitle generates a title for the session
func (h *History) generateTitle(session *Session) string {
	if len(session.Messages) == 0 {
		return "Empty Session"
	}

	// Find first user message
	for _, msg := range session.Messages {
		if msg.Role == "user" {
			// Truncate to reasonable length
			title := msg.Content
			if len(title) > 100 {
				title = title[:97] + "..."
			}
			// Remove newlines
			title = strings.ReplaceAll(title, "\n", " ")
			return title
		}
	}

	return fmt.Sprintf("Session %s", session.ID[:8])
}

// extractTags extracts tags from the session
func (h *History) extractTags(session *Session) []string {
	tags := make([]string, 0)

	// Add context-based tags
	if ctx, exists := session.Context["tags"]; exists {
		if contextTags, ok := ctx.([]string); ok {
			tags = append(tags, contextTags...)
		}
	}

	// Add auto-detected tags based on content
	// This is a simple implementation - could be enhanced with NLP
	for _, msg := range session.Messages {
		content := strings.ToLower(msg.Content)

		// Programming language tags
		if strings.Contains(content, "python") {
			tags = appendUnique(tags, "python")
		}
		if strings.Contains(content, "javascript") || strings.Contains(content, "js") {
			tags = appendUnique(tags, "javascript")
		}
		if strings.Contains(content, "golang") || strings.Contains(content, "go ") {
			tags = appendUnique(tags, "golang")
		}

		// Topic tags
		if strings.Contains(content, "debug") {
			tags = appendUnique(tags, "debugging")
		}
		if strings.Contains(content, "error") || strings.Contains(content, "fix") {
			tags = appendUnique(tags, "troubleshooting")
		}
		if strings.Contains(content, "test") {
			tags = appendUnique(tags, "testing")
		}
	}

	return tags
}

// appendUnique appends a string to slice if it doesn't exist
func appendUnique(slice []string, str string) []string {
	for _, s := range slice {
		if s == str {
			return slice
		}
	}
	return append(slice, str)
}

// saveSessionData saves session data to compressed JSON file
func (h *History) saveSessionData(path string, session *Session) error {
	file, err := os.Create(path)
	if err != nil {
		return err
	}
	defer file.Close()

	gzWriter := gzip.NewWriter(file)
	defer gzWriter.Close()

	encoder := json.NewEncoder(gzWriter)
	encoder.SetIndent("", "  ")

	return encoder.Encode(session)
}

// loadSessionData loads session data from compressed JSON file
func (h *History) loadSessionData(path string) (*Session, error) {
	file, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	gzReader, err := gzip.NewReader(file)
	if err != nil {
		return nil, err
	}
	defer gzReader.Close()

	var session Session
	decoder := json.NewDecoder(gzReader)

	if err := decoder.Decode(&session); err != nil {
		return nil, err
	}

	return &session, nil
}

// saveIndex saves the history index
func (h *History) saveIndex() error {
	indexPath := filepath.Join(h.StoragePath, "index.json")

	data, err := json.MarshalIndent(h, "", "  ")
	if err != nil {
		return err
	}

	return os.WriteFile(indexPath, data, 0644)
}

// loadIndex loads the history index
func (h *History) loadIndex() error {
	indexPath := filepath.Join(h.StoragePath, "index.json")

	data, err := os.ReadFile(indexPath)
	if err != nil {
		return err
	}

	return json.Unmarshal(data, h)
}

// exportCSV exports history to CSV format
func (h *History) exportCSV() ([]byte, error) {
	var builder strings.Builder

	// Header
	builder.WriteString("ID,Title,Start Time,End Time,Messages,Tags\n")

	// Data
	for _, s := range h.Sessions {
		builder.WriteString(fmt.Sprintf("%s,\"%s\",%s,%s,%d,\"%s\"\n",
			s.ID,
			strings.ReplaceAll(s.Title, "\"", "\"\""),
			s.StartTime.Format(time.RFC3339),
			s.EndTime.Format(time.RFC3339),
			s.Messages,
			strings.Join(s.Tags, ";"),
		))
	}

	return []byte(builder.String()), nil
}
