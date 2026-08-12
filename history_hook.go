package main

import (
	"encoding/json"
	"os"
	"path/filepath"
	"sync"
	"time"
)

// HistoryEntry represents a single transfer or message history item.
type HistoryEntry struct {
	ID        string    `json:"id"`
	Type      string    `json:"type"` // "send" or "receive"
	Name      string    `json:"name"`
	Size      int64     `json:"size"`
	Path      string    `json:"path"`
	Timestamp time.Time `json:"timestamp"`
	IsText    bool      `json:"is_text"`
	TextContent string  `json:"text_content,omitempty"`
}

// HistoryStore manages saving and loading transfer histories.
type HistoryStore struct {
	mu      sync.Mutex
	entries []HistoryEntry
	filePath string
}

const folderTransferSuffix = ".zip"

// NewHistoryStore initializes and loads history from disk.
func NewHistoryStore() *HistoryStore {
	dir := configDirPath()
	_ = os.MkdirAll(dir, 0755)
	fp := filepath.Join(dir, "history.json")

	store := &HistoryStore{
		filePath: fp,
		entries:  []HistoryEntry{},
	}
	store.load()
	return store
}

func (s *HistoryStore) load() {
	s.mu.Lock()
	defer s.mu.Unlock()

	data, err := os.ReadFile(s.filePath)
	if err != nil {
		return
	}
	_ = json.Unmarshal(data, &s.entries)
}

func (s *HistoryStore) SaveEntry(entry HistoryEntry) {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.entries = append([]HistoryEntry{entry}, s.entries...)
	
	// Keep last 100 entries max
	if len(s.entries) > 100 {
		s.entries = s.entries[:100]
	}

	data, err := json.MarshalIndent(s.entries, "", "  ")
	if err == nil {
		_ = os.WriteFile(s.filePath, data, 0644)
	}
}

// Helper utility functions referenced in hooks
func isTextTransfer(dataType string) bool {
	return dataType == "text"
}

func handleReceivedText(content string) {
	// Logic to handle incoming text payloads
}

func extractFolderZip(zipPath string, destDir string) error {
	// Logic to extract received folder zips
	return nil
}