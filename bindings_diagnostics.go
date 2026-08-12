package main

import webview "github.com/webview/webview_go"

func bindDiagnosticsFunctions(w webview.WebView) {
	_ = w.Bind("goGetLaunchFile", func() string {
		return ConsumeLaunchFile()
	})

	_ = w.Bind("goToggleFavorite", func(deviceID string) bool {
		return globalFavorites.Toggle(deviceID)
	})
	_ = w.Bind("goListFavorites", func() []string {
		return globalFavorites.List()
	})

	_ = w.Bind("goPingDevice", func(ip string, port int) PingResult {
		return PingDevice(ip, port)
	})
	_ = w.Bind("goCheckLANHealth", func() LANHealth {
		return CheckLANHealth()
	})

	_ = w.Bind("goExportSettings", func(destPath string) error {
		return ExportSettings(destPath)
	})
	_ = w.Bind("goImportSettings", func(srcPath string) error {
		return ImportSettings(srcPath)
	})

	_ = w.Bind("goGetLogs", func() []LogEntry {
		return globalLogRing.List()
	})
}