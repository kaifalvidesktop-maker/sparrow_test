package main

import (
	"encoding/base64"
	"encoding/json"
	"os"
	"path/filepath"
	"sync"
)

// Stores the user's imported custom background image as a base64 data
// URL, kept in its own small JSON file (separate from config.json since
// image data can be large and shouldn't bloat every settings read/write).

type BackgroundStore struct {
	mu     sync.Mutex
	dataURL string
	path   string
}

var globalBackground *BackgroundStore

func InitBackground() {
	dir := configDirPath()
	bs := &BackgroundStore{path: filepath.Join(dir, "background.json")}
	data, err := os.ReadFile(bs.path)
	if err == nil {
		var stored struct {
			DataURL string `json:"dataUrl"`
		}
		if json.Unmarshal(data, &stored) == nil {
			bs.dataURL = stored.DataURL
		}
	}
	globalBackground = bs
}

// SetBackgroundImage reads an image file from disk and stores it as a
// base64 data URL so the frontend can apply it directly via CSS
// background-image without a second file-serving round trip.
func (b *BackgroundStore) SetBackgroundImage(imagePath string) error {
	raw, err := os.ReadFile(imagePath)
	if err != nil {
		return err
	}

	mime := "image/png"
	ext := filepath.Ext(imagePath)
	switch ext {
	case ".jpg", ".jpeg":
		mime = "image/jpeg"
	case ".gif":
		mime = "image/gif"
	case ".webp":
		mime = "image/webp"
	}

	encoded := base64.StdEncoding.EncodeToString(raw)
	dataURL := "data:" + mime + ";base64," + encoded

	b.mu.Lock()
	b.dataURL = dataURL
	b.mu.Unlock()

	return b.save()
}

func (b *BackgroundStore) save() error {
	b.mu.Lock()
	payload := struct {
		DataURL string `json:"dataUrl"`
	}{DataURL: b.dataURL}
	b.mu.Unlock()

	data, err := json.Marshal(payload)
	if err != nil {
		return err
	}
	return os.WriteFile(b.path, data, 0644)
}

func (b *BackgroundStore) Get() string {
	b.mu.Lock()
	defer b.mu.Unlock()
	return b.dataURL
}

func (b *BackgroundStore) Clear() error {
	b.mu.Lock()
	b.dataURL = ""
	b.mu.Unlock()
	return b.save()
}