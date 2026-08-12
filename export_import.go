package main

import (
	"encoding/json"
	"fmt"
	"os"
)

// HistoryEntry defines the structure for transfer history items.

// HistoryManager handles listing and storing history items.
type HistoryManager struct {
	entries []HistoryEntry
}

var globalHistory = &HistoryManager{}

func (h *HistoryManager) List() []HistoryEntry {
	if h.entries == nil {
		return []HistoryEntry{}
	}
	return h.entries
}

type ExportBundle struct {
	Config  Config         `json:"config"`
	History []HistoryEntry `json:"history"`
}

func ExportSettings(destPath string) error {
	bundle := ExportBundle{
		Config:  globalConfig.Get(),
		History: globalHistory.List(),
	}
	data, err := json.MarshalIndent(bundle, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to encode settings: %w", err)
	}
	return os.WriteFile(destPath, data, 0644)
}

func ImportSettings(srcPath string) error {
	data, err := os.ReadFile(srcPath)
	if err != nil {
		return fmt.Errorf("failed to read import file: %w", err)
	}
	var bundle ExportBundle
	if err := json.Unmarshal(data, &bundle); err != nil {
		return fmt.Errorf("invalid settings file: %w", err)
	}
	return globalConfig.Update(func(c *Config) {
		*c = bundle.Config
	})
}