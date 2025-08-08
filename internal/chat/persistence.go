package chat

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"
)

// Persistence interface defines methods for session persistence
type Persistence interface {
	SaveSession(session *Session) error
	LoadSession(id string) (*Session, error)
	ListSessions() ([]string, error)
	DeleteSession(id string) error
}

// FilePersistence implements file-based session persistence
type FilePersistence struct {
	basePath     string
	mu           sync.RWMutex
	autoSave     bool
	saveInterval time.Duration
}

// NewFilePersistence creates a new file-based persistence manager
func NewFilePersistence(basePath string, autoSave bool, saveInterval time.Duration) (*FilePersistence, error) {
	// Create base directory if it doesn't exist
	if err := os.MkdirAll(basePath, 0755); err != nil {
		return nil, fmt.Errorf("failed to create persistence directory: %w", err)
	}

	// Create subdirectories
	dirs := []string{"sessions", "metadata", "backup", "temp"}
	for _, dir := range dirs {
		path := filepath.Join(basePath, dir)
		if err := os.MkdirAll(path, 0755); err != nil {
			return nil, fmt.Errorf("failed to create %s directory: %w", dir, err)
		}
	}

	return &FilePersistence{
		basePath:     basePath,
		autoSave:     autoSave,
		saveInterval: saveInterval,
	}, nil
}

// SaveSession saves a session to persistent storage
func (fp *FilePersistence) SaveSession(session *Session) error {
	fp.mu.Lock()
	defer fp.mu.Unlock()

	// Validate session
	if session == nil || session.ID == "" {
		return fmt.Errorf("invalid session")
	}

	// Save to temp file first (atomic write)
	tempPath := filepath.Join(fp.basePath, "temp", fmt.Sprintf("%s.tmp", session.ID))
	finalPath := filepath.Join(fp.basePath, "sessions", fmt.Sprintf("%s.json", session.ID))

	// Create temp file
	tempFile, err := os.Create(tempPath)
	if err != nil {
		return fmt.Errorf("failed to create temp file: %w", err)
	}
	defer os.Remove(tempPath) // Clean up temp file

	// Encode session data
	encoder := json.NewEncoder(tempFile)
	encoder.SetIndent("", "  ")

	if err := encoder.Encode(session); err != nil {
		tempFile.Close()
		return fmt.Errorf("failed to encode session: %w", err)
	}

	// Close temp file
	if err := tempFile.Close(); err != nil {
		return fmt.Errorf("failed to close temp file: %w", err)
	}

	// Calculate checksum
	checksum, err := fp.calculateChecksum(tempPath)
	if err != nil {
		return fmt.Errorf("failed to calculate checksum: %w", err)
	}

	// Save metadata
	metadata := SessionMetadata{
		ID:           session.ID,
		Checksum:     checksum,
		SavedAt:      time.Now(),
		Version:      "1.0",
		MessageCount: len(session.Messages),
		TokenCount:   session.TokenCount,
	}

	if err := fp.saveMetadata(session.ID, metadata); err != nil {
		return fmt.Errorf("failed to save metadata: %w", err)
	}

	// Create backup if file already exists
	if _, err := os.Stat(finalPath); err == nil {
		backupPath := filepath.Join(fp.basePath, "backup", fmt.Sprintf("%s_%d.json", session.ID, time.Now().Unix()))
		if err := fp.copyFile(finalPath, backupPath); err != nil {
			// Log error but don't fail the save
			fmt.Printf("Warning: failed to create backup: %v\n", err)
		}
	}

	// Atomic rename
	if err := os.Rename(tempPath, finalPath); err != nil {
		return fmt.Errorf("failed to save session: %w", err)
	}

	return nil
}

// LoadSession loads a session from persistent storage
func (fp *FilePersistence) LoadSession(id string) (*Session, error) {
	fp.mu.RLock()
	defer fp.mu.RUnlock()

	sessionPath := filepath.Join(fp.basePath, "sessions", fmt.Sprintf("%s.json", id))

	// Check if file exists
	if _, err := os.Stat(sessionPath); os.IsNotExist(err) {
		return nil, fmt.Errorf("session not found: %s", id)
	}

	// Load metadata for validation
	metadata, err := fp.loadMetadata(id)
	if err != nil {
		// Continue without metadata validation
		fmt.Printf("Warning: failed to load metadata: %v\n", err)
	}

	// Open session file
	file, err := os.Open(sessionPath)
	if err != nil {
		return nil, fmt.Errorf("failed to open session file: %w", err)
	}
	defer file.Close()

	// Decode session
	var session Session
	decoder := json.NewDecoder(file)

	if err := decoder.Decode(&session); err != nil {
		return nil, fmt.Errorf("failed to decode session: %w", err)
	}

	// Validate checksum if metadata exists
	if metadata != nil {
		checksum, err := fp.calculateChecksum(sessionPath)
		if err == nil && checksum != metadata.Checksum {
			// Try to recover from backup
			if recoveredSession, err := fp.recoverFromBackup(id); err == nil {
				return recoveredSession, nil
			}
			return nil, fmt.Errorf("session data corrupted (checksum mismatch)")
		}
	}

	return &session, nil
}

// ListSessions returns a list of all session IDs
func (fp *FilePersistence) ListSessions() ([]string, error) {
	fp.mu.RLock()
	defer fp.mu.RUnlock()

	sessionsPath := filepath.Join(fp.basePath, "sessions")

	entries, err := os.ReadDir(sessionsPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read sessions directory: %w", err)
	}

	var sessionIDs []string
	for _, entry := range entries {
		if !entry.IsDir() && filepath.Ext(entry.Name()) == ".json" {
			// Extract ID from filename
			id := entry.Name()[:len(entry.Name())-5] // Remove .json extension
			sessionIDs = append(sessionIDs, id)
		}
	}

	return sessionIDs, nil
}

// DeleteSession removes a session from persistent storage
func (fp *FilePersistence) DeleteSession(id string) error {
	fp.mu.Lock()
	defer fp.mu.Unlock()

	// Create backup before deletion
	sessionPath := filepath.Join(fp.basePath, "sessions", fmt.Sprintf("%s.json", id))
	if _, err := os.Stat(sessionPath); err == nil {
		backupPath := filepath.Join(fp.basePath, "backup", fmt.Sprintf("%s_deleted_%d.json", id, time.Now().Unix()))
		if err := fp.copyFile(sessionPath, backupPath); err != nil {
			// Log error but don't fail the deletion
			fmt.Printf("Warning: failed to create deletion backup: %v\n", err)
		}
	}

	// Delete session file
	if err := os.Remove(sessionPath); err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("failed to delete session: %w", err)
	}

	// Delete metadata
	metadataPath := filepath.Join(fp.basePath, "metadata", fmt.Sprintf("%s.json", id))
	if err := os.Remove(metadataPath); err != nil && !os.IsNotExist(err) {
		// Log error but don't fail
		fmt.Printf("Warning: failed to delete metadata: %v\n", err)
	}

	return nil
}

// SessionMetadata contains metadata about a saved session
type SessionMetadata struct {
	ID           string    `json:"id"`
	Checksum     string    `json:"checksum"`
	SavedAt      time.Time `json:"saved_at"`
	Version      string    `json:"version"`
	MessageCount int       `json:"message_count"`
	TokenCount   int       `json:"token_count"`
}

// saveMetadata saves session metadata
func (fp *FilePersistence) saveMetadata(id string, metadata SessionMetadata) error {
	metadataPath := filepath.Join(fp.basePath, "metadata", fmt.Sprintf("%s.json", id))

	data, err := json.MarshalIndent(metadata, "", "  ")
	if err != nil {
		return err
	}

	return os.WriteFile(metadataPath, data, 0644)
}

// loadMetadata loads session metadata
func (fp *FilePersistence) loadMetadata(id string) (*SessionMetadata, error) {
	metadataPath := filepath.Join(fp.basePath, "metadata", fmt.Sprintf("%s.json", id))

	data, err := os.ReadFile(metadataPath)
	if err != nil {
		return nil, err
	}

	var metadata SessionMetadata
	if err := json.Unmarshal(data, &metadata); err != nil {
		return nil, err
	}

	return &metadata, nil
}

// calculateChecksum calculates SHA256 checksum of a file
func (fp *FilePersistence) calculateChecksum(path string) (string, error) {
	file, err := os.Open(path)
	if err != nil {
		return "", err
	}
	defer file.Close()

	hash := sha256.New()
	if _, err := io.Copy(hash, file); err != nil {
		return "", err
	}

	return hex.EncodeToString(hash.Sum(nil)), nil
}

// copyFile copies a file from src to dst
func (fp *FilePersistence) copyFile(src, dst string) error {
	sourceFile, err := os.Open(src)
	if err != nil {
		return err
	}
	defer sourceFile.Close()

	destFile, err := os.Create(dst)
	if err != nil {
		return err
	}
	defer destFile.Close()

	_, err = io.Copy(destFile, sourceFile)
	return err
}

// recoverFromBackup attempts to recover a session from backup
func (fp *FilePersistence) recoverFromBackup(id string) (*Session, error) {
	backupDir := filepath.Join(fp.basePath, "backup")

	entries, err := os.ReadDir(backupDir)
	if err != nil {
		return nil, fmt.Errorf("failed to read backup directory: %w", err)
	}

	// Find the most recent backup for this session
	var latestBackup string
	var latestTime int64

	for _, entry := range entries {
		if !entry.IsDir() && strings.HasPrefix(entry.Name(), id+"_") {
			info, err := entry.Info()
			if err != nil {
				continue
			}

			if info.ModTime().Unix() > latestTime {
				latestTime = info.ModTime().Unix()
				latestBackup = entry.Name()
			}
		}
	}

	if latestBackup == "" {
		return nil, fmt.Errorf("no backup found for session: %s", id)
	}

	// Load from backup
	backupPath := filepath.Join(backupDir, latestBackup)
	file, err := os.Open(backupPath)
	if err != nil {
		return nil, fmt.Errorf("failed to open backup: %w", err)
	}
	defer file.Close()

	var session Session
	decoder := json.NewDecoder(file)

	if err := decoder.Decode(&session); err != nil {
		return nil, fmt.Errorf("failed to decode backup: %w", err)
	}

	fmt.Printf("Warning: recovered session %s from backup\n", id)
	return &session, nil
}

// ValidateIntegrity checks the integrity of all saved sessions
func (fp *FilePersistence) ValidateIntegrity() ([]string, error) {
	fp.mu.RLock()
	defer fp.mu.RUnlock()

	var corruptedSessions []string

	sessions, err := fp.ListSessions()
	if err != nil {
		return nil, err
	}

	for _, id := range sessions {
		metadata, err := fp.loadMetadata(id)
		if err != nil {
			// Missing metadata is not critical
			continue
		}

		sessionPath := filepath.Join(fp.basePath, "sessions", fmt.Sprintf("%s.json", id))
		checksum, err := fp.calculateChecksum(sessionPath)
		if err != nil {
			corruptedSessions = append(corruptedSessions, id)
			continue
		}

		if checksum != metadata.Checksum {
			corruptedSessions = append(corruptedSessions, id)
		}
	}

	return corruptedSessions, nil
}

// CleanupBackups removes old backup files
func (fp *FilePersistence) CleanupBackups(maxAge time.Duration) error {
	fp.mu.Lock()
	defer fp.mu.Unlock()

	backupDir := filepath.Join(fp.basePath, "backup")

	entries, err := os.ReadDir(backupDir)
	if err != nil {
		return fmt.Errorf("failed to read backup directory: %w", err)
	}

	cutoff := time.Now().Add(-maxAge)

	for _, entry := range entries {
		if !entry.IsDir() {
			info, err := entry.Info()
			if err != nil {
				continue
			}

			if info.ModTime().Before(cutoff) {
				path := filepath.Join(backupDir, entry.Name())
				if err := os.Remove(path); err != nil {
					// Log error but continue
					fmt.Printf("Warning: failed to remove old backup %s: %v\n", entry.Name(), err)
				}
			}
		}
	}

	return nil
}

// GetProjectSessionPath returns the session storage path for the current project
func GetProjectSessionPath() (string, error) {
	// Get current working directory
	cwd, err := os.Getwd()
	if err != nil {
		return "", fmt.Errorf("failed to get current directory: %w", err)
	}

	// Create a hash of the directory path for unique project identification
	hash := sha256.Sum256([]byte(cwd))
	projectHash := hex.EncodeToString(hash[:])[:16] // Use first 16 chars

	// Get home directory
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("failed to get home directory: %w", err)
	}

	// Create session path: ~/.coda/sessions/{project-hash}/
	sessionPath := filepath.Join(homeDir, ".coda", "sessions", projectHash)

	// Create project info file to track which directory this hash represents
	infoPath := filepath.Join(homeDir, ".coda", "sessions", projectHash, ".project-info")
	if err := os.MkdirAll(filepath.Dir(infoPath), 0755); err != nil {
		return "", fmt.Errorf("failed to create session directory: %w", err)
	}

	// Write project info
	info := map[string]string{
		"path":    cwd,
		"name":    filepath.Base(cwd),
		"created": time.Now().Format(time.RFC3339),
	}

	data, err := json.MarshalIndent(info, "", "  ")
	if err == nil {
		_ = os.WriteFile(infoPath, data, 0644)
	}

	return sessionPath, nil
}
