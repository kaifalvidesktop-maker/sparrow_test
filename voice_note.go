package main

import (
	"encoding/base64"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"
)

const voiceTransferSuffix = ".sparrowvoice"

type VoiceMessage struct {
	ID        string `json:"id"`
	PeerName  string `json:"peerName"`
	Outgoing  bool   `json:"outgoing"`
	Base64    string `json:"base64"`
	Timestamp int64  `json:"timestampMs"`
}

type VoiceInbox struct {
	mu       sync.Mutex
	messages []VoiceMessage
}

var globalVoiceInbox = &VoiceInbox{}

func (v *VoiceInbox) Add(msg VoiceMessage) {
	v.mu.Lock()
	defer v.mu.Unlock()
	msg.Timestamp = time.Now().UnixMilli()
	v.messages = append(v.messages, msg)
	const max = 100
	if len(v.messages) > max {
		v.messages = v.messages[len(v.messages)-max:]
	}
}

func (v *VoiceInbox) List() []VoiceMessage {
	v.mu.Lock()
	defer v.mu.Unlock()
	out := make([]VoiceMessage, len(v.messages))
	copy(out, v.messages)
	return out
}

func SendVoiceNote(base64Audio, destIP string, destPort int, turbo bool) (*TransferState, error) {
	raw, err := base64.StdEncoding.DecodeString(base64Audio)
	if err != nil {
		return nil, fmt.Errorf("invalid audio data: %w", err)
	}
	if len(raw) == 0 {
		return nil, fmt.Errorf("empty voice note")
	}

	tmpName := fmt.Sprintf("voice-%d%s", time.Now().UnixNano(), voiceTransferSuffix)
	tmpPath := filepath.Join(os.TempDir(), tmpName)
	if err := os.WriteFile(tmpPath, raw, 0644); err != nil {
		return nil, fmt.Errorf("failed to prepare voice note: %w", err)
	}

	ts, err := SendFile(tmpPath, destIP, destPort, turbo)
	if err != nil {
		_ = os.Remove(tmpPath)
		return nil, err
	}

	peerName := destIP
	if dev, ok := globalDeviceRegistry.findByIP(destIP); ok {
		peerName = dev.Name
	}
	globalVoiceInbox.Add(VoiceMessage{ID: ts.ID, PeerName: peerName, Outgoing: true, Base64: base64Audio})

	go cleanupTempZipWhenDone(ts.ID, tmpPath)
	go pollVoiceCompletion(ts.ID, tmpPath)

	return ts, nil
}

func pollVoiceCompletion(transferID, path string) {
	for {
		time.Sleep(500 * time.Millisecond)
		snap, ok := globalTransferManager.Get(transferID)
		if !ok {
			break
		}
		if snap.Status == StatusCompleted || snap.Status == StatusFailed || snap.Status == StatusCancelled {
			break
		}
	}
}

func cleanupTempZipWhenDone(transferID, path string) {
	for {
		time.Sleep(1 * time.Second)
		snap, ok := globalTransferManager.Get(transferID)
		if !ok || snap.Status == StatusCompleted || snap.Status == StatusFailed || snap.Status == StatusCancelled {
			_ = os.Remove(path)
			break
		}
	}
}

var voiceSeenIDs = struct {
	mu   sync.Mutex
	seen map[string]bool
}{seen: make(map[string]bool)}

func StartVoiceNoteWatcher() {
	go func() {
		for {
			time.Sleep(1 * time.Second)
			for _, snap := range globalTransferManager.List() {
				if snap.Status != StatusCompleted || snap.Direction != DirectionReceive {
					continue
				}
				voiceSeenIDs.mu.Lock()
				already := voiceSeenIDs.seen[snap.ID]
				if !already {
					voiceSeenIDs.seen[snap.ID] = true
				}
				voiceSeenIDs.mu.Unlock()
				if already {
					continue
				}

				path, ok := GetTransferPath(snap.ID)
				if !ok || !strings.HasSuffix(path, voiceTransferSuffix) {
					continue
				}

				data, err := os.ReadFile(path)
				_ = os.Remove(path)
				if err != nil {
					continue
				}
				globalVoiceInbox.Add(VoiceMessage{
					ID: snap.ID, PeerName: snap.PeerName, Outgoing: false,
					Base64: base64.StdEncoding.EncodeToString(data),
				})
			}
		}
	}()
}