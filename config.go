package main

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"os"
	"path/filepath"
	"sync"
)

type Config struct {
	DeviceName         string `json:"deviceName"`
	ThemeMode          string `json:"themeMode"`
	ColorTheme         string `json:"colorTheme"`
	DownloadDir        string `json:"downloadDir"`
	TurboMode          bool   `json:"turboMode"`
	RequirePinReceive  bool   `json:"requirePinReceive"`
	PinHash            string `json:"pinHash"`
	AppPasswordHash    string `json:"appPasswordHash"`
	MinimizeToTray     bool   `json:"minimizeToTray"`
	AutostartEnabled   bool   `json:"autostartEnabled"`
	ShareTargetEnabled bool   `json:"shareTargetEnabled"`
	QuickSave          bool   `json:"quickSave"`
	Animations         bool   `json:"animations"`
	DeviceAvatar       string `json:"deviceAvatar"`
}

type ConfigStore struct {
	mu   sync.Mutex
	data Config
	path string
}

var globalConfig *ConfigStore

func defaultConfig() Config {
	return Config{
		DeviceName:         defaultDeviceName(),
		ThemeMode:          "dark",
		ColorTheme:         "violet-default",
		DownloadDir:        defaultDownloadDirPath(),
		TurboMode:          true,
		RequirePinReceive:  false,
		MinimizeToTray:     false,
		AutostartEnabled:   false,
		ShareTargetEnabled: false,
		QuickSave:          false,
		Animations:         true,
	}
}

func configDirPath() string {
    dir, _ := os.UserConfigDir()
    return filepath.Join(dir, "sparrow")
}

func InitConfig() *ConfigStore {
	dir := configDirPath()
	cs := &ConfigStore{path: filepath.Join(dir, "config.json")}

	data, err := os.ReadFile(cs.path)
	if err != nil {
		cs.data = defaultConfig()
		_ = cs.save()
		globalConfig = cs
		return cs
	}

	var cfg Config
	if err := json.Unmarshal(data, &cfg); err != nil {
		cs.data = defaultConfig()
		_ = cs.save()
		globalConfig = cs
		return cs
	}

	cs.data = cfg
	globalConfig = cs
	return cs
}

func (c *ConfigStore) save() error {
	data, err := json.MarshalIndent(c.data, "", "  ")
	if err != nil {
		return err
	}
	tmp := c.path + ".tmp"
	if err := os.WriteFile(tmp, data, 0644); err != nil {
		return err
	}
	return os.Rename(tmp, c.path)
}

func (c *ConfigStore) Get() Config {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.data
}

func (c *ConfigStore) Update(mutate func(*Config)) error {
	c.mu.Lock()
	mutate(&c.data)
	c.mu.Unlock()
	return c.save()
}

func HashSecret(raw string) string {
	if raw == "" {
		return ""
	}
	sum := sha256.Sum256([]byte(raw))
	return hex.EncodeToString(sum[:])
}

func VerifySecret(raw, storedHash string) bool {
	if storedHash == "" {
		return true
	}
	return HashSecret(raw) == storedHash
}

func defaultDownloadDirPath() string {
	home, err := os.UserHomeDir()
	if err != nil || home == "" {
		return "."
	}
	dir := filepath.Join(home, "Downloads", "Sparrow")
	_ = os.MkdirAll(dir, 0755)
	return dir
}