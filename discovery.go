package main

import (
	"sync"
	"time"
)

// ============================================================
// discovery.go
// UDP-based LAN device discovery for Sparrow.
// ============================================================

const DiscoveryPort = 53317
const TransferPort = 53318
const discoveryBroadcastInterval = 3 * time.Second
const discoveryExpireAfter = 12 * time.Second

// DeviceInfo describes one device discovered on the LAN.
type DeviceInfo struct {
	ID         string    `json:"id"`
	Name       string    `json:"name"`
	IP         string    `json:"ip"`
	Port       int       `json:"port"`
	Avatar     string    `json:"avatar"`
	Version    string    `json:"version"`
	LastSeenMs int64     `json:"lastSeenMs"`
	lastSeen   time.Time
}

// DeviceRegistry is a thread-safe in-memory table of currently known LAN devices.
type DeviceRegistry struct {
	mu      sync.RWMutex
	devices map[string]*DeviceInfo
}

func NewDeviceRegistry() *DeviceRegistry {
	return &DeviceRegistry{devices: make(map[string]*DeviceInfo)}
}

func (r *DeviceRegistry) Upsert(info DeviceInfo) {
	r.mu.Lock()
	defer r.mu.Unlock()
	info.lastSeen = time.Now()
	info.LastSeenMs = info.lastSeen.UnixMilli()
	r.devices[info.ID] = &info
}

func (r *DeviceRegistry) Remove(id string) {
	r.mu.Lock()
	defer r.mu.Unlock()
	delete(r.devices, id)
}

func (r *DeviceRegistry) PruneExpired() {
	r.mu.Lock()
	defer r.mu.Unlock()
	cutoff := time.Now().Add(-discoveryExpireAfter)
	for id, dev := range r.devices {
		if dev.lastSeen.Before(cutoff) {
			delete(r.devices, id)
		}
	}
}

func (r *DeviceRegistry) List() []DeviceInfo {
	r.mu.RLock()
	defer r.mu.RUnlock()
	out := make([]DeviceInfo, 0, len(r.devices))
	for _, dev := range r.devices {
		out = append(out, *dev)
	}
	for i := 1; i < len(out); i++ {
		for j := i; j > 0 && out[j].Name < out[j-1].Name; j-- {
			out[j], out[j-1] = out[j-1], out[j]
		}
	}
	return out
}

func (r *DeviceRegistry) Get(id string) (DeviceInfo, bool) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	dev, ok := r.devices[id]
	if !ok {
		return DeviceInfo{}, false
	}
	return *dev, true
}

func (r *DeviceRegistry) Count() int {
	r.mu.RLock()
	defer r.mu.RUnlock()
	return len(r.devices)
}