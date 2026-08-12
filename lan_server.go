package main

import (
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"time"
)

// LANPort is the port phones connect to over Wi-Fi to reach Sparrow's
// mobile bridge web page. Kept distinct from the internal 127.0.0.1 UI
// server so the desktop webview and the LAN-facing server never collide.
const LANPort = 53317

// LANServer is the HTTP server bound to 0.0.0.0:LANPort — reachable from
// any device on the same Wi-Fi network. It serves the mobile web page and
// the upload/download/text-share API endpoints.
type LANServer struct {
	listener net.Listener
	server   *http.Server
	Port     int
}

// StartLANServer boots the LAN-facing HTTP server. Unlike StartUIServer
// (which binds only to 127.0.0.1), this one binds to 0.0.0.0 so phones on
// the same Wi-Fi can reach it using the PC's LAN IP address.
func StartLANServer() (*LANServer, error) {
	mux := http.NewServeMux()

	mux.HandleFunc("/", serveText(mobileHTML, "text/html; charset=utf-8"))
	mux.HandleFunc("/api/upload", handleUpload)
	mux.HandleFunc("/api/outbox", handleOutboxList)
	mux.HandleFunc("/api/download", handleDownload)
	mux.HandleFunc("/api/paste", handlePasteText)
	mux.HandleFunc("/api/whoami", handleWhoAmI)

	addr := fmt.Sprintf("0.0.0.0:%d", LANPort)
	listener, err := net.Listen("tcp", addr)
	if err != nil {
		return nil, fmt.Errorf("failed to bind LAN server on %s: %w", addr, err)
	}

	httpServer := &http.Server{Handler: mux}

	lanSrv := &LANServer{
		listener: listener,
		server:   httpServer,
		Port:     LANPort,
	}

	go func() {
		if err := httpServer.Serve(listener); err != nil && err != http.ErrServerClosed {
			log.Printf("[lan-server] stopped: %v", err)
		}
	}()

	return lanSrv, nil
}

// Shutdown gracefully stops the LAN-facing server on app exit.
func (s *LANServer) Shutdown() {
	if s.server != nil {
		_ = s.server.Close()
	}
}

// handleUpload receives a file POSTed from the phone's browser (via
// multipart/form-data) and saves it into the user's configured download
// folder. This is the "phone -> PC" direction.
func handleUpload(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Safety cap: refuse absurdly large uploads (2GB). Real chunked
	// transfer for huge files arrives with the full transfer engine.
	r.Body = http.MaxBytesReader(w, r.Body, 2<<30)

	if err := r.ParseMultipartForm(32 << 20); err != nil {
		http.Error(w, "failed to parse upload: "+err.Error(), http.StatusBadRequest)
		return
	}

	file, header, err := r.FormFile("file")
	if err != nil {
		http.Error(w, "no file field in request: "+err.Error(), http.StatusBadRequest)
		return
	}
	defer file.Close()

	destDir := state.downloadDir
	if destDir == "" {
		destDir = "."
	}
	_ = os.MkdirAll(destDir, 0755)

	safeName := sanitizeFileName(header.Filename)
	destPath := uniquePath(filepath.Join(destDir, safeName))

	out, err := os.Create(destPath)
	if err != nil {
		http.Error(w, "failed to save file: "+err.Error(), http.StatusInternalServerError)
		return
	}
	defer out.Close()

	written, err := io.Copy(out, file)
	if err != nil {
		http.Error(w, "failed while writing file: "+err.Error(), http.StatusInternalServerError)
		return
	}

	log.Printf("[lan-server] received %q (%d bytes) from %s", safeName, written, r.RemoteAddr)

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(map[string]interface{}{
		"ok":   true,
		"name": safeName,
		"size": written,
	})
}

// handleOutboxList returns the list of files currently available for the
// phone to download (the "PC -> phone" direction). Files are read from a
// dedicated SparrowOutbox folder — later, the desktop "Send" screen will
// place selected files here automatically before generating the QR link.
func handleOutboxList(w http.ResponseWriter, r *http.Request) {
	entries, err := os.ReadDir(outboxDir())
	files := []map[string]interface{}{}
	if err == nil {
		for _, e := range entries {
			if e.IsDir() {
				continue
			}
			info, infoErr := e.Info()
			size := int64(0)
			if infoErr == nil {
				size = info.Size()
			}
			files = append(files, map[string]interface{}{
				"name": e.Name(),
				"size": size,
			})
		}
	}
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(files)
}

// handleDownload streams a single outbox file to the phone's browser.
func handleDownload(w http.ResponseWriter, r *http.Request) {
	name := r.URL.Query().Get("name")
	if name == "" {
		http.Error(w, "missing name parameter", http.StatusBadRequest)
		return
	}
	safeName := sanitizeFileName(name)
	path := filepath.Join(outboxDir(), safeName)

	if _, err := os.Stat(path); err != nil {
		http.Error(w, "file not found", http.StatusNotFound)
		return
	}

	w.Header().Set("Content-Disposition", "attachment; filename=\""+safeName+"\"")
	http.ServeFile(w, r, path)
}

// pastedTexts holds text/link snippets shared from the phone, in RAM only.
var pastedTexts []string

// handlePasteText accepts a plain-text POST from the phone (text/link
// share) and stores it in memory, or returns the stored list on GET.
func handlePasteText(w http.ResponseWriter, r *http.Request) {
	if r.Method == http.MethodPost {
		body, _ := io.ReadAll(r.Body)
		text := string(body)
		pastedTexts = append(pastedTexts, text)
		log.Printf("[lan-server] received text (%d chars) from %s", len(text), r.RemoteAddr)
		w.WriteHeader(http.StatusOK)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(pastedTexts)
}

// handleWhoAmI lets the mobile page show which PC it connected to.
func handleWhoAmI(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(map[string]string{
		"deviceName": state.deviceName,
		"appVersion": state.appVersion,
	})
}

// outboxDir returns (and creates if needed) the folder phones can browse
// and download from.
func outboxDir() string {
	base := state.downloadDir
	if base == "" {
		base = "."
	}
	dir := filepath.Join(filepath.Dir(base), "SparrowOutbox")
	_ = os.MkdirAll(dir, 0755)
	return dir
}

// sanitizeFileName strips directory components so an uploaded file name
// can never be used to write outside the intended download folder.
func sanitizeFileName(name string) string {
	name = filepath.Base(name)
	if name == "" || name == "." || name == ".." {
		return "file_" + strconv.FormatInt(time.Now().UnixNano(), 10)
	}
	return name
}

// uniquePath appends " (1)", " (2)", etc. if a file with the same name
// already exists, so incoming uploads never silently overwrite anything.
func uniquePath(path string) string {
	if _, err := os.Stat(path); os.IsNotExist(err) {
		return path
	}
	ext := filepath.Ext(path)
	base := path[:len(path)-len(ext)]
	for i := 1; ; i++ {
		candidate := fmt.Sprintf("%s (%d)%s", base, i, ext)
		if _, err := os.Stat(candidate); os.IsNotExist(err) {
			return candidate
		}
	}
}