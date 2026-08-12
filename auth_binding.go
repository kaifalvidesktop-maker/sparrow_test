package main

import webview "github.com/webview/webview_go"

func bindAuthFunctions(w webview.WebView) {
	_ = w.Bind("goVerifyPin", func(pin string) bool {
		return VerifySecret(pin, globalConfig.Get().PinHash)
	})

	_ = w.Bind("goVerifyAppPassword", func(pw string) bool {
		return VerifySecret(pw, globalConfig.Get().AppPasswordHash)
	})

	_ = w.Bind("goHasAppPassword", func() bool {
		return globalConfig.Get().AppPasswordHash != ""
	})

	_ = w.Bind("goHasPin", func() bool {
		return globalConfig.Get().PinHash != ""
	})
}