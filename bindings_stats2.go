package main

import webview "github.com/webview/webview_go"

func bindStatsFunctions(w webview.WebView) {
	_ = w.Bind("goSetNickname", func(deviceID, nickname string) error {
		return globalNicknames.Set(deviceID, nickname)
	})
	_ = w.Bind("goGetNicknames", func() map[string]string {
		return globalNicknames.All()
	})

	_ = w.Bind("goGetBandwidthStats", func() BandwidthStats {
		return globalBandwidth.Get()
	})

	_ = w.Bind("goCheckForUpdate", func(manifestURL string) UpdateCheckResult {
		return CheckForUpdate(manifestURL)
	})

	_ = w.Bind("goGetSessionStats", func() SessionStats {
		return GetSessionStats()
	})
}