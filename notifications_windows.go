package main

import (
	"strings"
	"sync"
	"syscall"
	"time"
	"unsafe"
)

// Tray structure references
type TrayIcon struct {
	hwnd syscall.Handle
}

var globalTray *TrayIcon

var shellNotifyIcon *syscall.Proc

type notifyIconDataW struct {
	cbSize           uint32
	hWnd             syscall.Handle
	uID              uint32
	uFlags           uint32
	uCallbackMessage uint32
	hIcon            syscall.Handle
	szTip            [128]uint16
	dwState          uint32
	dwStateMask      uint32
	szInfo           [256]uint16
	uTimeoutVersion  uint32
	szInfoTitle      [64]uint16
	dwInfoFlags      uint32
	guidItem         [16]byte
	hBalloonIcon     syscall.Handle
}

var seenNotifyIDs = struct {
	mu   sync.Mutex
	seen map[string]bool
}{seen: make(map[string]bool)}

func StartTransferNotifications() {
	go func() {
		for {
			time.Sleep(1 * time.Second)
			if globalTray == nil || globalTray.hwnd == 0 {
				continue
			}
			for _, snap := range globalTransferManager.List() {
				if snap.Status != StatusCompleted && snap.Status != StatusFailed {
					continue
				}
				seenNotifyIDs.mu.Lock()
				already := seenNotifyIDs.seen[snap.ID]
				if !already {
					seenNotifyIDs.seen[snap.ID] = true
				}
				seenNotifyIDs.mu.Unlock()
				if already {
					continue
				}

				title := "Sparrow"
				var body string
				if snap.Status == StatusCompleted {
					if snap.Direction == DirectionReceive {
						body = "Received " + snap.FileName + " from " + snap.PeerName
					} else {
						body = "Sent " + snap.FileName + " to " + snap.PeerName
					}
				} else {
					body = "Transfer failed: " + snap.FileName
				}
				showBalloonNotification(title, body)
			}
		}
	}()
}

func showBalloonNotification(title, body string) {
	if globalTray == nil || globalTray.hwnd == 0 || shellNotifyIcon == nil {
		return
	}
	var nid notifyIconDataW
	nid.cbSize = uint32(unsafe.Sizeof(nid))
	nid.hWnd = globalTray.hwnd
	nid.uID = 1
	nid.uFlags = 0x00000010 // NIF_INFO

	titleUTF16, _ := syscall.UTF16FromString(truncateForBuffer(title, 63))
	copy(nid.szInfoTitle[:], titleUTF16)

	bodyUTF16, _ := syscall.UTF16FromString(truncateForBuffer(body, 255))
	copy(nid.szInfo[:], bodyUTF16)

	shellNotifyIcon.Call(nimModifyConst, uintptr(unsafe.Pointer(&nid)))
}

const nimModifyConst = 0x00000001

func truncateForBuffer(s string, max int) string {
	if len(s) <= max {
		return s
	}
	return s[:max]
}

func hasSuffixFold(s, suffix string) bool {
	return strings.HasSuffix(strings.ToLower(s), strings.ToLower(suffix))
}