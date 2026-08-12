package main

import (
	"encoding/json"
	"os"
	"path/filepath"
	"sync"
)

// Lets the user assign a custom nickname to a discovered device (keyed by
// the device's stable ID), persisted independently of the main config.

type NicknameStore struct {
	mu    sync.Mutex
	names map[string]string
	path  string
}

var globalNicknames *NicknameStore

func InitNicknames() {
	dir := configDirPath()
	ns := &NicknameStore{
		names: make(map[string]string),
		path:  filepath.Join(dir, "nicknames.json"),
	}
	data, err := os.ReadFile(ns.path)
	if err == nil {
		_ = json.Unmarshal(data, &ns.names)
	}
	globalNicknames = ns
}

func (n *NicknameStore) Set(deviceID, nickname string) error {
	n.mu.Lock()
	if nickname == "" {
		delete(n.names, deviceID)
	} else {
		n.names[deviceID] = nickname
	}
	snapshot := make(map[string]string, len(n.names))
	for k, v := range n.names {
		snapshot[k] = v
	}
	n.mu.Unlock()

	data, err := json.MarshalIndent(snapshot, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(n.path, data, 0644)
}

func (n *NicknameStore) Get(deviceID string) string {
	n.mu.Lock()
	defer n.mu.Unlock()
	return n.names[deviceID]
}

func (n *NicknameStore) All() map[string]string {
	n.mu.Lock()
	defer n.mu.Unlock()
	out := make(map[string]string, len(n.names))
	for k, v := range n.names {
		out[k] = v
	}
	return out
}

// DisplayNameFor resolves a device's nickname if set, otherwise its raw
// broadcast name — the function other code should call for any user-facing
// device label.
func DisplayNameFor(dev DeviceInfo) string {
	if nick := globalNicknames.Get(dev.ID); nick != "" {
		return nick
	}
	return dev.Name
}