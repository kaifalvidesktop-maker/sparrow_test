package main

import (
	"fmt"
	"log"
	"net"
	"net/http"
)

// UIServer wraps the local-only HTTP server that hosts the Go-string-based
// UI (shell + CSS + JS + tab partials). Nothing here is ever reachable from
// the LAN — it is bound to 127.0.0.1 purely to give webview's browser
// engine a real HTTP origin to load CSS/JS/fetch() from, since raw
// SetHtml() cannot resolve relative asset links or AJAX partial loads.
type UIServer struct {
	listener net.Listener
	server   *http.Server
	Port     int
}

// StartUIServer boots the local server on a random free 127.0.0.1 port and
// registers every route: the shell page, style.css, app.js, and one route
// per tab partial (home, devices, history, chat, settings, about).
func StartUIServer() (*UIServer, error) {
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		return nil, fmt.Errorf("failed to bind local UI server: %w", err)
	}

	mux := http.NewServeMux()

	// ---- Shell + static assets ----
	mux.HandleFunc("/", serveText(shellHTML, "text/html; charset=utf-8"))
	mux.HandleFunc("/style.css", serveText(appCSS, "text/css; charset=utf-8"))
	mux.HandleFunc("/app.js", serveText(appJS, "application/javascript; charset=utf-8"))

	// ---- Tab partials (fetched via AJAX from app.js, injected into #content-mount) ----
	mux.HandleFunc("/partial/home", serveText(homePartialHTML, "text/html; charset=utf-8"))
	mux.HandleFunc("/partial/devices", serveText(devicesPartialHTML, "text/html; charset=utf-8"))
	mux.HandleFunc("/partial/history", serveText(historyPartialHTML, "text/html; charset=utf-8"))
	mux.HandleFunc("/partial/chat", serveText(chatPartialHTML, "text/html; charset=utf-8"))
	mux.HandleFunc("/partial/settings", serveText(settingsPartialHTML, "text/html; charset=utf-8"))
	mux.HandleFunc("/partial/about", serveText(aboutPartialHTML, "text/html; charset=utf-8"))

	httpServer := &http.Server{Handler: mux}

	uiSrv := &UIServer{
		listener: listener,
		server:   httpServer,
		Port:     listener.Addr().(*net.TCPAddr).Port,
	}

	go func() {
		if err := httpServer.Serve(listener); err != nil && err != http.ErrServerClosed {
			log.Printf("[ui-server] stopped: %v", err)
		}
	}()

	return uiSrv, nil
}

// serveText returns an http.HandlerFunc that writes a fixed Go string
// constant as the response body with the given content type. This is how
// every UI piece (HTML/CSS/JS) — defined as const strings across the
// ui_*.go files — gets delivered to the webview browser engine.
func serveText(body string, contentType string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", contentType)
		w.Header().Set("Cache-Control", "no-store")
		_, _ = w.Write([]byte(body))
	}
}

// URL returns the local address the webview window should navigate to.
func (s *UIServer) URL() string {
	return fmt.Sprintf("http://127.0.0.1:%d/", s.Port)
}

// Shutdown gracefully stops the local UI server. Called on app exit.
func (s *UIServer) Shutdown() {
	if s.server != nil {
		_ = s.server.Close()
	}
}

// getLocalIP returns this machine's LAN IPv4 address (used later for LAN
// discovery and shown to the user in the Devices/Settings screens).
func getLocalIP() string {
	addrs, err := net.InterfaceAddrs()
	if err != nil {
		return "127.0.0.1"
	}
	for _, addr := range addrs {
		if ipNet, ok := addr.(*net.IPNet); ok && !ipNet.IP.IsLoopback() {
			if ipv4 := ipNet.IP.To4(); ipv4 != nil {
				return ipv4.String()
			}
		}
	}
	return "127.0.0.1"
}