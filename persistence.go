package main

import (
	"encoding/json"
	"log"
	"os"
	"path/filepath"
	"sync"
	"time"
)

// Persistence layer: state.json under the user's config dir
// (%AppData%\Sparrow\state.json on Windows). Autosaved every 30s and on shutdown.
// History stays RAM-only by design; bookmarks and finished downloads are persisted
// so a restored backup survives restarts.

type persistedState struct {
	SavedAt time.Time `json:"savedAt"`

	Theme               string `json:"theme"`
	DefaultSearchEngine string `json:"defaultSearchEngine"`
	Homepage            string `json:"homepage"`
	JSEnabled           bool   `json:"jsEnabled"`
	ImagesEnabled       bool   `json:"imagesEnabled"`
	DownloadPath        string `json:"downloadPath"`

	CustomTheme   *CustomThemeSnapshot `json:"customTheme,omitempty"`
	HomeAnimation string               `json:"homeAnimation,omitempty"`
	HomeWallpaper string               `json:"homeWallpaper,omitempty"`

	Autofill AutofillData `json:"autofill"`

	Favorites []FavoriteEntry    `json:"favorites"`
	Bookmarks []ExportedBookmark `json:"bookmarks,omitempty"`
	Downloads []BackupDownload   `json:"downloads,omitempty"`
	SavedLinks []PinnedShortcut  `json:"savedLinks"`

	Passwords    []StoredPassword `json:"passwords"`
	PasswordSalt []byte           `json:"passwordSalt,omitempty"`

	AppLock AppLockSnapshot `json:"appLock,omitempty"`
}

var (
	autosaveStopChan chan struct{}
	autosaveOnce     sync.Once
)

func stateFilePath() (string, error) {
	dir, err := os.UserConfigDir()
	if err != nil {
		return "", err
	}
	appDir := filepath.Join(dir, "Sparrow")
	if err := os.MkdirAll(appDir, 0o755); err != nil {
		return "", err
	}
	return filepath.Join(appDir, "state.json"), nil
}

func buildPersistedState() *persistedState {
	settings.mu.Lock()
	s := persistedState{
		SavedAt:             time.Now(),
		Theme:               settings.Theme,
		DefaultSearchEngine: settings.DefaultSearchEngine,
		Homepage:            settings.Homepage,
		JSEnabled:           settings.JSEnabled,
		ImagesEnabled:       settings.ImagesEnabled,
		DownloadPath:        settings.DownloadPath,
	}
	settings.mu.Unlock()

	ct := customTheme.Snapshot()
	s.CustomTheme = &ct
	s.HomeAnimation = homeAnimationManager.Get()
	s.HomeWallpaper = wallpaperManager.GetSelectedID()
	s.Autofill = getAutofillProfile()
	s.Favorites = favoriteManager.GetAll()
	s.Bookmarks = exportBookmarks()
	s.Downloads = exportDownloads(true)
	s.SavedLinks = shortcutManagerNewTab.GetShortcuts()
	s.Passwords = passwordVault.Snapshot()
	s.PasswordSalt = passwordVault.SaltSnapshot()
	s.AppLock = appLock.Snapshot()
	return &s
}

func applyPersistedState(s *persistedState) {
	if s == nil {
		return
	}
	settings.mu.Lock()
	if s.Theme != "" {
		settings.Theme = s.Theme
	}
	if s.DefaultSearchEngine != "" {
		settings.DefaultSearchEngine = s.DefaultSearchEngine
	}
	if s.Homepage != "" {
		settings.Homepage = s.Homepage
	}
	settings.JSEnabled = s.JSEnabled
	settings.ImagesEnabled = s.ImagesEnabled
	if s.DownloadPath != "" {
		settings.DownloadPath = s.DownloadPath
	}
	settings.mu.Unlock()

	if s.CustomTheme != nil {
		customTheme.Restore(*s.CustomTheme)
	}
	homeAnimationManager.Restore(s.HomeAnimation)
	wallpaperManager.Restore(s.HomeWallpaper)
	saveAutofillProfile(s.Autofill)
	favoriteManager.ReplaceAll(s.Favorites)
	importBookmarks(s.Bookmarks)
	importDownloads(s.Downloads)
	shortcutManagerNewTab.ReplaceAll(s.SavedLinks)
	passwordVault.Restore(s.Passwords)
	passwordVault.SetSalt(s.PasswordSalt)
	appLock.Restore(s.AppLock)
}

func loadPersistedState() *persistedState {
	path, err := stateFilePath()
	if err != nil {
		log.Println("[persistence] could not resolve config dir:", err)
		return nil
	}
	data, err := os.ReadFile(path)
	if err != nil {
		if !os.IsNotExist(err) {
			log.Println("[persistence] read failed:", err)
		}
		return nil
	}
	var s persistedState
	if err := json.Unmarshal(data, &s); err != nil {
		log.Println("[persistence] corrupt state.json, ignoring:", err)
		return nil
	}
	return &s
}

// Atomic write: temp file then rename.
func savePersistedState() error {
	path, err := stateFilePath()
	if err != nil {
		return err
	}
	data, err := json.MarshalIndent(buildPersistedState(), "", " ")
	if err != nil {
		return err
	}
	tmp := path + ".tmp"
	if err := os.WriteFile(tmp, data, 0o644); err != nil {
		return err
	}
	return os.Rename(tmp, path)
}

func startAutosave(interval time.Duration, stop <-chan struct{}) {
	ticker := time.NewTicker(interval)
	go func() {
		defer ticker.Stop()
		for {
			select {
			case <-ticker.C:
				if err := savePersistedState(); err != nil {
					log.Println("[persistence] autosave failed:", err)
				}
			case <-stop:
				return
			}
		}
	}()
}

func InitPersistence() {
	if state := loadPersistedState(); state != nil {
		applyPersistedState(state)
		log.Println("[persistence] persisted state loaded successfully")
	}
	autosaveOnce.Do(func() {
		autosaveStopChan = make(chan struct{})
		startAutosave(30*time.Second, autosaveStopChan)
		log.Println("[persistence] 30s autosave timer started")
	})
}

func StopPersistence() {
	if autosaveStopChan != nil {
		select {
		case <-autosaveStopChan:
		default:
			close(autosaveStopChan)
		}
	}
	if err := savePersistedState(); err != nil {
		log.Println("[persistence] final shutdown save failed:", err)
	} else {
		log.Println("[persistence] state saved successfully on shutdown")
	}
}

// --- tiny local helpers (kept: may be used elsewhere) ---

func persistIntToStr(n int) string {
	if n == 0 {
		return "0"
	}
	neg := n < 0
	if neg {
		n = -n
	}
	var buf [20]byte
	i := len(buf)
	for n > 0 {
		i--
		buf[i] = byte('0' + n%10)
		n /= 10
	}
	if neg {
		i--
		buf[i] = '-'
	}
	return string(buf[i:])
}

func persistStrToInt(s string) (int, bool) {
	if s == "" {
		return 0, false
	}
	neg := false
	i := 0
	if s[0] == '-' {
		neg = true
		i = 1
	}
	if i == len(s) {
		return 0, false
	}
	n := 0
	for ; i < len(s); i++ {
		c := s[i]
		if c < '0' || c > '9' {
			return 0, false
		}
		n = n*10 + int(c-'0')
	}
	if neg {
		n = -n
	}
	return n, true
}
