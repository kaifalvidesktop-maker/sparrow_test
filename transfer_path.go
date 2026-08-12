package main

import "sync"

// transferPathRegistry maps a transfer's ID to the file path it was saved to
// (receive side) or read from (send side). This lets other parts of the app
// — the "Open Folder" button in history, the receive-window's "download
// complete" state — look up where a given transfer's file actually lives
// without threading the path through every function call.
type transferPathRegistry struct {
	mu    sync.RWMutex
	paths map[string]string
}

// globalTransferPaths is the process-wide registry. It is safe for
// concurrent use from webview bind callbacks, the TCP receiver's accept
// loop, and the sender's chunk goroutines all at once.
var globalTransferPaths = &transferPathRegistry{paths: make(map[string]string)}

// SetTransferPath records (or overwrites) the on-disk path for a transfer
// ID. Call this as soon as the path is known — e.g. right after a send
// request is accepted, or right after the receiver creates the destination
// file — so lookups never race with the transfer's own I/O.
func SetTransferPath(transferID, path string) {
	globalTransferPaths.mu.Lock()
	defer globalTransferPaths.mu.Unlock()
	globalTransferPaths.paths[transferID] = path
}

// GetTransferPath returns the path stored for transferID and whether one
// was found. Uses a read lock so concurrent lookups (e.g. the UI polling
// several transfers' rows at once) don't block each other.
func GetTransferPath(transferID string) (string, bool) {
	globalTransferPaths.mu.RLock()
	defer globalTransferPaths.mu.RUnlock()
	p, ok := globalTransferPaths.paths[transferID]
	return p, ok
}

// RemoveTransferPath deletes the stored path for a single transfer ID.
// Call this once a transfer is finalized (completed, cancelled, or failed)
// and its path has been copied into history, so the registry doesn't grow
// unbounded over a long-running session.
func RemoveTransferPath(transferID string) {
	globalTransferPaths.mu.Lock()
	defer globalTransferPaths.mu.Unlock()
	delete(globalTransferPaths.paths, transferID)
}

// HasTransferPath reports whether a path is currently stored for
// transferID, without needing to discard the "ok" bool from GetTransferPath
// at call sites that only care about presence (e.g. enabling/disabling an
// "Open Folder" button in the UI).
func HasTransferPath(transferID string) bool {
	globalTransferPaths.mu.RLock()
	defer globalTransferPaths.mu.RUnlock()
	_, ok := globalTransferPaths.paths[transferID]
	return ok
}

// ClearTransferPaths wipes every stored path. Intended for app shutdown or
// for a "clear history" action in Settings, where completed transfers'
// paths no longer need to be resolvable.
func ClearTransferPaths() {
	globalTransferPaths.mu.Lock()
	defer globalTransferPaths.mu.Unlock()
	globalTransferPaths.paths = make(map[string]string)
}