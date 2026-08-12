package main

import (
	"encoding/json"
	"fmt"
	"net/http"
	"time"
)

// Simple version-check client: fetches a small JSON manifest from your
// own hosted endpoint (e.g. served from kaifalvi.pages.app) and compares
// against the running app version.

type UpdateManifest struct {
	LatestVersion string `json:"latestVersion"`
	DownloadURL   string `json:"downloadUrl"`
	ReleaseNotes  string `json:"releaseNotes"`
}

type UpdateCheckResult struct {
	UpdateAvailable bool   `json:"updateAvailable"`
	CurrentVersion  string `json:"currentVersion"`
	LatestVersion   string `json:"latestVersion,omitempty"`
	DownloadURL     string `json:"downloadUrl,omitempty"`
	ReleaseNotes    string `json:"releaseNotes,omitempty"`
	Error           string `json:"error,omitempty"`
}

// CheckForUpdate hits manifestURL (point this at a JSON file you host,
// e.g. https://kaifalvi.pages.app/sparrow/version.json) and reports
// whether a newer version is available than state.appVersion.
func CheckForUpdate(manifestURL string) UpdateCheckResult {
	client := http.Client{Timeout: 6 * time.Second}
	resp, err := client.Get(manifestURL)
	if err != nil {
		return UpdateCheckResult{CurrentVersion: state.appVersion, Error: err.Error()}
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		return UpdateCheckResult{
			CurrentVersion: state.appVersion,
			Error:          fmt.Sprintf("update server returned status %d", resp.StatusCode),
		}
	}

	var manifest UpdateManifest
	if err := json.NewDecoder(resp.Body).Decode(&manifest); err != nil {
		return UpdateCheckResult{CurrentVersion: state.appVersion, Error: "invalid manifest: " + err.Error()}
	}

	return UpdateCheckResult{
		UpdateAvailable: manifest.LatestVersion != "" && manifest.LatestVersion != state.appVersion,
		CurrentVersion:  state.appVersion,
		LatestVersion:   manifest.LatestVersion,
		DownloadURL:     manifest.DownloadURL,
		ReleaseNotes:    manifest.ReleaseNotes,
	}
}