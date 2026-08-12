package main

import (
	"os"
	"path/filepath"
)


func (h *HistoryManager) Clear() error {
	h.entries = []HistoryEntry{}
	return nil
}

func ResetSparrow() error {
	dir := configDirPath()

	files := []string{
		"config.json", "history.json", "nicknames.json",
		"favorites.json", "background.json", "bandwidth.json",
	}
	for _, f := range files {
		_ = os.Remove(filepath.Join(dir, f))
	}

	_ = globalConfig.Update(func(c *Config) {
		*c = defaultConfig()
	})
	_ = globalHistory.Clear()

	return nil
}