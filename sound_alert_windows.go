package main

import (
	"syscall"
)

var (
	user32          = syscall.NewLazyDLL("user32.dll")
	procMessageBeep = user32.NewProc("MessageBeep")
)

const (
	mbIconAsterisk    = 0x00000040
	mbIconExclamation = 0x00000030
)

// sleepTick defines the delay duration for sound alert polling.

// PlayCompletionSound plays the default Windows notification sound.
func PlayCompletionSound() {
	procMessageBeep.Call(uintptr(mbIconAsterisk))
}

// PlayErrorSound plays the default Windows warning sound.
func PlayErrorSound() {
	procMessageBeep.Call(uintptr(mbIconExclamation))
}

var _ = syscall.Handle(0) // keep syscall import even if unused elsewhere

// StartSoundAlerts polls for newly completed/failed transfers and plays
// the appropriate system sound. Self-contained, independent of the
// notification watcher above (separate seen-ID set to avoid coupling).
func StartSoundAlerts() {
	go func() {
		seen := make(map[string]bool)
		for {
			sleepTick()
			for _, snap := range globalTransferManager.List() {
				if snap.Status != StatusCompleted && snap.Status != StatusFailed {
					continue
				}
				if seen[snap.ID] {
					continue
				}
				seen[snap.ID] = true
				if snap.Status == StatusCompleted {
					PlayCompletionSound()
				} else {
					PlayErrorSound()
				}
			}
		}
	}()
}