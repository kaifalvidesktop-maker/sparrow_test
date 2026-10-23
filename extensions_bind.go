package main

import webview "github.com/webview/webview_go"

// registerExtensionFetchBinding exposes the restricted network bridge used by
// sparrowExt.fetch. The request runs in a goroutine so the UI never blocks,
// and the answer is sent back into the page with Eval.
func registerExtensionFetchBinding(w webview.WebView) {
	w.Bind("extFetchStart", func(id, rawURL, reqID string) {
		go func() {
			result := extFetchJSON(id, rawURL)
			w.Dispatch(func() {
				w.Eval("window.__sparrowExtDone&&window.__sparrowExtDone(" + jsString(reqID) + "," + jsString(result) + ");")
			})
		}()
	})
}
