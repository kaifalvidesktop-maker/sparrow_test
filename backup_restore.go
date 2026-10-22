package main

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"
)

// Full backup / restore: bookmarks, favorites, history, downloads, settings,
// shortcuts, autofill, custom theme, home animation and wallpaper in one JSON.
// Passwords are intentionally excluded (use the encrypted vault export).

type BackupSettings struct {
	Theme               string `json:"theme"`
	DefaultSearchEngine string `json:"defaultSearchEngine"`
	Homepage            string `json:"homepage"`
	JSEnabled           bool   `json:"jsEnabled"`
	ImagesEnabled       bool   `json:"imagesEnabled"`
	DownloadPath        string `json:"downloadPath"`
}

type BackupBundle struct {
	Version       string               `json:"version"`
	CreatedAt     string               `json:"createdAt"`
	Bookmarks     []ExportedBookmark   `json:"bookmarks"`
	Settings      map[string]any       `json:"settings"`
	SettingsData  *BackupSettings      `json:"settingsData,omitempty"`
	Shortcuts     []PinnedShortcut     `json:"shortcuts"`
	Favorites     []FavoriteEntry      `json:"favorites,omitempty"`
	History       []BackupHistoryEntry `json:"history,omitempty"`
	Downloads     []BackupDownload     `json:"downloads,omitempty"`
	Autofill      *AutofillData        `json:"autofill,omitempty"`
	CustomTheme   *CustomThemeSnapshot `json:"customTheme,omitempty"`
	HomeAnimation string               `json:"homeAnimation,omitempty"`
	HomeWallpaper string               `json:"homeWallpaper,omitempty"`
}

type BackupHistoryEntry struct {
	URL   string    `json:"url"`
	Title string    `json:"title"`
	Time  time.Time `json:"time"`
}

type BackupDownload struct {
	ID         int       `json:"id"`
	FileName   string    `json:"fileName"`
	URL        string    `json:"url"`
	Progress   int       `json:"progress"`
	Status     string    `json:"status"`
	StartedAt  time.Time `json:"startedAt"`
	FinishedAt time.Time `json:"finishedAt,omitempty"`
}

// ---- shared helpers (also used by persistence.go) ----

func exportBookmarks() []ExportedBookmark {
	all := bookmarkManager.GetAll()
	out := make([]ExportedBookmark, 0, len(all))
	for _, b := range all {
		out = append(out, ExportedBookmark{Title: b.Title, URL: b.URL, Folder: b.Folder, CreatedAt: b.CreatedAt})
	}
	return out
}

func importBookmarks(list []ExportedBookmark) int {
	n := 0
	for _, b := range list {
		if strings.TrimSpace(b.URL) == "" || bookmarkManager.IsBookmarked(b.URL) {
			continue
		}
		bm := bookmarkManager.Add(b.Title, b.URL, b.Folder)
		if bm != nil && !b.CreatedAt.IsZero() {
			bm.CreatedAt = b.CreatedAt
		}
		n++
	}
	return n
}

func exportDownloads(onlyFinished bool) []BackupDownload {
	items := downloadManager.GetAll()
	out := make([]BackupDownload, 0, len(items))
	for _, it := range items {
		if it == nil || (onlyFinished && it.FinishedAt.IsZero()) {
			continue
		}
		out = append(out, BackupDownload{
			ID: it.ID, FileName: it.FileName, URL: it.URL, Progress: it.Progress,
			Status: it.Status, StartedAt: it.StartedAt, FinishedAt: it.FinishedAt,
		})
	}
	return out
}

func importDownloads(list []BackupDownload) int {
	items := make([]*DownloadItem, 0, len(list))
	for _, d := range list {
		items = append(items, &DownloadItem{
			ID: d.ID, FileName: d.FileName, URL: d.URL, Progress: d.Progress,
			Status: d.Status, StartedAt: d.StartedAt, FinishedAt: d.FinishedAt,
		})
	}
	return downloadManager.Import(items)
}

func snapshotSettings() *BackupSettings {
	settings.mu.Lock()
	defer settings.mu.Unlock()
	return &BackupSettings{
		Theme: settings.Theme, DefaultSearchEngine: settings.DefaultSearchEngine,
		Homepage: settings.Homepage, JSEnabled: settings.JSEnabled,
		ImagesEnabled: settings.ImagesEnabled, DownloadPath: settings.DownloadPath,
	}
}

func validTheme(t string) bool { return t == "dark" || t == "light" }

func validEngine(e string) bool {
	switch e {
	case "google", "bing", "brave", "duckduckgo", "yahoo":
		return true
	}
	return false
}

// ---- backup ----

func BuildBackup() BackupBundle {
	hist := history.GetAll()
	exportedHistory := make([]BackupHistoryEntry, 0, len(hist))
	for _, e := range hist {
		exportedHistory = append(exportedHistory, BackupHistoryEntry{URL: e.URL, Title: e.Title, Time: e.Time})
	}
	ct := customTheme.Snapshot()
	af := getAutofillProfile()
	return BackupBundle{
		Version:       sparrowVersion,
		CreatedAt:     time.Now().Format(time.RFC3339),
		Bookmarks:     exportBookmarks(),
		Settings:      settings.GetAll(),
		SettingsData:  snapshotSettings(),
		Shortcuts:     shortcutManagerNewTab.GetShortcuts(),
		Favorites:     favoriteManager.GetAll(),
		History:       exportedHistory,
		Downloads:     exportDownloads(false),
		Autofill:      &af,
		CustomTheme:   &ct,
		HomeAnimation: homeAnimationManager.Get(),
		HomeWallpaper: wallpaperManager.GetSelectedID(),
	}
}

func ExportBackupJSON() string {
	data, err := json.MarshalIndent(BuildBackup(), "", " ")
	if err != nil {
		return "{}"
	}
	return string(data)
}

func SaveBackupToFile() (string, error) {
	dir, err := userDownloadsDirectory()
	if err != nil {
		return "", err
	}
	if err := os.MkdirAll(dir, 0755); err != nil {
		return "", err
	}
	full := filepath.Join(dir, "sparrow-backup_"+time.Now().Format("2006-01-02_15-04-05")+".json")
	if err := os.WriteFile(full, []byte(ExportBackupJSON()), 0644); err != nil {
		return "", err
	}
	return full, nil
}

func userDownloadsDirectory() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	if strings.TrimSpace(home) == "" {
		return "", fmt.Errorf("could not determine the current user's home directory")
	}
	return filepath.Join(home, "Downloads"), nil
}

// ---- restore ----

func RestoreBackupJSON(jsonStr string) (int, error) {
	// tolerate a UTF-8 BOM
	jsonStr = strings.TrimPrefix(jsonStr, "\ufeff")
	var b BackupBundle
	if err := json.Unmarshal([]byte(jsonStr), &b); err != nil {
		return 0, err
	}
	if b.Version == "" && b.Settings == nil && b.SettingsData == nil && b.Bookmarks == nil &&
		b.Shortcuts == nil && b.Favorites == nil && b.History == nil &&
		b.Downloads == nil && b.Autofill == nil && b.CustomTheme == nil {
		return 0, fmt.Errorf("file does not contain a Sparrow backup")
	}

	count := importBookmarks(b.Bookmarks)

	// Settings: new typed block first, legacy map as fallback for old backups.
	settings.mu.Lock()
	if sd := b.SettingsData; sd != nil {
		if validTheme(sd.Theme) {
			settings.Theme = sd.Theme
		}
		if validEngine(sd.DefaultSearchEngine) {
			settings.DefaultSearchEngine = sd.DefaultSearchEngine
		}
		if hp := strings.TrimSpace(sd.Homepage); hp != "" {
			settings.Homepage = hp
		}
		settings.JSEnabled = sd.JSEnabled
		settings.ImagesEnabled = sd.ImagesEnabled
		if sd.DownloadPath != "" {
			settings.DownloadPath = sd.DownloadPath
		}
	} else if b.Settings != nil {
		if v, ok := b.Settings["theme"].(string); ok && validTheme(v) {
			settings.Theme = v
		}
		if v, ok := b.Settings["searchEngine"].(string); ok && validEngine(v) {
			settings.DefaultSearchEngine = v
		}
		if v, ok := b.Settings["homepage"].(string); ok && strings.TrimSpace(v) != "" {
			settings.Homepage = strings.TrimSpace(v)
		}
		if v, ok := b.Settings["jsEnabled"].(bool); ok {
			settings.JSEnabled = v
		}
		if v, ok := b.Settings["imagesEnabled"].(bool); ok {
			settings.ImagesEnabled = v
		}
		if v, ok := b.Settings["downloadPath"].(string); ok && v != "" {
			settings.DownloadPath = v
		}
	}
	settings.mu.Unlock()

	for _, s := range b.Shortcuts {
		if _, err := shortcutManagerNewTab.AddLink(s.Title, s.URL); err == nil {
			count++
		}
	}
	for _, f := range b.Favorites {
		if f.URL != "" && favoriteManager.Add(f.Title, f.URL) != nil {
			count++
		}
	}

	hist := make([]HistoryEntry, 0, len(b.History))
	for _, e := range b.History {
		hist = append(hist, HistoryEntry{URL: e.URL, Title: e.Title, Time: e.Time})
	}
	count += history.Import(hist)
	count += importDownloads(b.Downloads)

	if b.Autofill != nil {
		saveAutofillProfile(*b.Autofill)
	}
	if b.CustomTheme != nil {
		customTheme.Restore(*b.CustomTheme)
	}
	if b.HomeAnimation != "" {
		homeAnimationManager.Restore(b.HomeAnimation)
	}
	if b.HomeWallpaper != "" {
		wallpaperManager.Restore(b.HomeWallpaper)
	}

	if err := savePersistedState(); err != nil {
		return count, err
	}
	return count, nil
}
