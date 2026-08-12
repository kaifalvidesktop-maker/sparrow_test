package main

import (
	"encoding/json"
	"os"
	"path/filepath"
	"sync"
)

type FavoritesStore struct {
	mu   sync.Mutex
	ids  map[string]bool
	path string
}

var globalFavorites *FavoritesStore

func InitFavorites() {
	dir := configDirPath()
	fs := &FavoritesStore{
		ids:  make(map[string]bool),
		path: filepath.Join(dir, "favorites.json"),
	}
	data, err := os.ReadFile(fs.path)
	if err == nil {
		var list []string
		if json.Unmarshal(data, &list) == nil {
			for _, id := range list {
				fs.ids[id] = true
			}
		}
	}
	globalFavorites = fs
}

func (f *FavoritesStore) save() {
	f.mu.Lock()
	list := make([]string, 0, len(f.ids))
	for id := range f.ids {
		list = append(list, id)
	}
	f.mu.Unlock()

	data, err := json.MarshalIndent(list, "", "  ")
	if err != nil {
		return
	}
	_ = os.WriteFile(f.path, data, 0644)
}

func (f *FavoritesStore) Toggle(deviceID string) bool {
	f.mu.Lock()
	f.ids[deviceID] = !f.ids[deviceID]
	isFav := f.ids[deviceID]
	if !isFav {
		delete(f.ids, deviceID)
	}
	f.mu.Unlock()
	f.save()
	return isFav
}

func (f *FavoritesStore) IsFavorite(deviceID string) bool {
	f.mu.Lock()
	defer f.mu.Unlock()
	return f.ids[deviceID]
}

func (f *FavoritesStore) List() []string {
	f.mu.Lock()
	defer f.mu.Unlock()
	out := make([]string, 0, len(f.ids))
	for id := range f.ids {
		out = append(out, id)
	}
	return out
}