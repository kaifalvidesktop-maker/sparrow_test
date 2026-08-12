package main

import (
	"os"
	"sync"
)

var launchFile = struct {
	mu   sync.Mutex
	path string
}{}

// InitLaunchArgs checks if Sparrow was launched via the "Share with
// Sparrow" right-click menu (os.Args[1] = file/folder path) and stores it
// so the UI can preload it into the Home tab's selection on startup.
func InitLaunchArgs() {
	if len(os.Args) < 2 {
		return
	}
	path := os.Args[1]
	if _, err := os.Stat(path); err != nil {
		return
	}
	launchFile.mu.Lock()
	launchFile.path = path
	launchFile.mu.Unlock()
}

// ConsumeLaunchFile returns the launch-time file path once, then clears
// it so it is never re-applied on subsequent polls from the frontend.
func ConsumeLaunchFile() string {
	launchFile.mu.Lock()
	defer launchFile.mu.Unlock()
	p := launchFile.path
	launchFile.path = ""
	return p
}