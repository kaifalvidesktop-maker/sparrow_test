package main

import webview "github.com/webview/webview_go"

func bindFinalFunctions(w webview.WebView) {
	_ = w.Bind("goGetWindowsThemePreference", func() string {
		return GetWindowsThemePreference()
	})

	_ = w.Bind("goSetBackgroundImage", func(imagePath string) error {
		return globalBackground.SetBackgroundImage(imagePath)
	})
	_ = w.Bind("goGetBackgroundImage", func() string {
		return globalBackground.Get()
	})
	_ = w.Bind("goClearBackgroundImage", func() error {
		return globalBackground.Clear()
	})
	_ = w.Bind("goPickBackgroundImage", func() (string, error) {
		paths, err := pickFilesNative()
		if err != nil || len(paths) == 0 {
			return "", err
		}
		if setErr := globalBackground.SetBackgroundImage(paths[0]); setErr != nil {
			return "", setErr
		}
		return globalBackground.Get(), nil
	})

	_ = w.Bind("goResetSparrow", func() error {
		return ResetSparrow()
	})
}