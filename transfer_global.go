package main

import (
	"crypto/rand"
	"encoding/hex"
)

var globalDeviceRegistry = NewDeviceRegistry()
var globalTransferManager = NewTransferManager()

func incomingManager() *TransferManager { return globalTransferManager }

func generateTransferID() (string, error) {
	b := make([]byte, 16)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return hex.EncodeToString(b), nil
}

func generateDeviceID() string {
	b := make([]byte, 8)
	if _, err := rand.Read(b); err != nil {
		return "sparrow-device"
	}
	return "sp-" + hex.EncodeToString(b)
}

func (r *DeviceRegistry) findByIP(ip string) (DeviceInfo, bool) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	for _, dev := range r.devices {
		if dev.IP == ip {
			return *dev, true
		}
	}
	return DeviceInfo{}, false
}