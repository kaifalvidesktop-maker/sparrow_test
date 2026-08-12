package main

import "time"

// Lightweight in-memory session info (not persisted) for a future
// dashboard: uptime, live counts of active transfers by status.

var sparrowStartTime = time.Now()

type SessionStats struct {
	UptimeSeconds   int64 `json:"uptimeSeconds"`
	ActiveTransfers int   `json:"activeTransfers"`
	PausedTransfers int   `json:"pausedTransfers"`
	DevicesOnline   int   `json:"devicesOnline"`
}

func GetSessionStats() SessionStats {
	var active, paused int
	for _, t := range globalTransferManager.List() {
		switch t.Status {
		case StatusActive:
			active++
		case StatusPaused:
			paused++
		}
	}
	return SessionStats{
		UptimeSeconds:   int64(time.Since(sparrowStartTime).Seconds()),
		ActiveTransfers: active,
		PausedTransfers: paused,
		DevicesOnline:   globalDeviceRegistry.Count(),
	}
}