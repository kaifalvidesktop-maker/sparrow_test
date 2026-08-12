package main

import (
	"encoding/json"
	"os"
	"path/filepath"
	"sync"
)

// All-time cumulative bytes sent/received, persisted to disk. Updated by
// a self-contained polling watcher (same seen-ID pattern used elsewhere)
// so it never needs to hook into the existing transfer callback chain.

type BandwidthStats struct {
	TotalBytesSent     int64 `json:"totalBytesSent"`
	TotalBytesReceived int64 `json:"totalBytesReceived"`
	TotalFilesSent     int64 `json:"totalFilesSent"`
	TotalFilesReceived int64 `json:"totalFilesReceived"`
}

type BandwidthStore struct {
	mu   sync.Mutex
	data BandwidthStats
	path string
}

var globalBandwidth *BandwidthStore

func InitBandwidthStats() {
	dir := configDirPath()
	bs := &BandwidthStore{path: filepath.Join(dir, "bandwidth.json")}
	data, err := os.ReadFile(bs.path)
	if err == nil {
		_ = json.Unmarshal(data, &bs.data)
	}
	globalBandwidth = bs
}

func (b *BandwidthStore) save() {
	data, err := json.MarshalIndent(b.data, "", "  ")
	if err != nil {
		return
	}
	_ = os.WriteFile(b.path, data, 0644)
}

func (b *BandwidthStore) recordSend(bytes int64) {
	b.mu.Lock()
	b.data.TotalBytesSent += bytes
	b.data.TotalFilesSent++
	b.mu.Unlock()
	b.save()
}

func (b *BandwidthStore) recordReceive(bytes int64) {
	b.mu.Lock()
	b.data.TotalBytesReceived += bytes
	b.data.TotalFilesReceived++
	b.mu.Unlock()
	b.save()
}

func (b *BandwidthStore) Get() BandwidthStats {
	b.mu.Lock()
	defer b.mu.Unlock()
	return b.data
}

// StartBandwidthTracking watches for newly completed transfers and
// accumulates their size into the persisted totals.
func StartBandwidthTracking() {
	go func() {
		seen := make(map[string]bool)
		for {
			sleepTick()
			for _, snap := range globalTransferManager.List() {
				if snap.Status != StatusCompleted {
					continue
				}
				if seen[snap.ID] {
					continue
				}
				seen[snap.ID] = true
				if snap.Direction == DirectionSend {
					globalBandwidth.recordSend(snap.TotalSize)
				} else {
					globalBandwidth.recordReceive(snap.TotalSize)
				}
			}
		}
	}()
}