package main

import (
	"sync"
	"time"
)

type LogEntry struct {
	Message   string `json:"message"`
	Timestamp int64  `json:"timestampMs"`
}

type LogRing struct {
	mu      sync.Mutex
	entries []LogEntry
	max     int
}

var globalLogRing = &LogRing{max: 300}

func (r *LogRing) Add(message string) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.entries = append(r.entries, LogEntry{Message: message, Timestamp: time.Now().UnixMilli()})
	if len(r.entries) > r.max {
		r.entries = r.entries[len(r.entries)-r.max:]
	}
}

func (r *LogRing) List() []LogEntry {
	r.mu.Lock()
	defer r.mu.Unlock()
	out := make([]LogEntry, len(r.entries))
	copy(out, r.entries)
	return out
}