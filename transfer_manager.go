package main

import (
	"context"
	"fmt"
	"sync"
	"sync/atomic"
	"time"
)

// ============================================================
// transfer_manager.go — pause/resume/cancel, progress, state
// ============================================================

type TransferDirection string

const (
	DirectionSend    TransferDirection = "send"
	DirectionReceive TransferDirection = "receive"
)

type TransferStatus string

const (
	StatusQueued    TransferStatus = "queued"
	StatusActive    TransferStatus = "active"
	StatusPaused    TransferStatus = "paused"
	StatusCompleted TransferStatus = "completed"
	StatusFailed    TransferStatus = "failed"
	StatusCancelled TransferStatus = "cancelled"
)

type TransferControl struct {
	ctx       context.Context
	cancelFn  context.CancelFunc
	paused    atomic.Bool
	pauseCond *sync.Cond
	pauseMu   sync.Mutex
}

func NewTransferControl() *TransferControl {
	ctx, cancel := context.WithCancel(context.Background())
	tc := &TransferControl{ctx: ctx, cancelFn: cancel}
	tc.pauseCond = sync.NewCond(&tc.pauseMu)
	return tc
}

func (tc *TransferControl) Cancel() {
	tc.cancelFn()
	tc.pauseMu.Lock()
	tc.pauseCond.Broadcast()
	tc.pauseMu.Unlock()
}

func (tc *TransferControl) Pause() { tc.paused.Store(true) }

func (tc *TransferControl) Resume() {
	tc.paused.Store(false)
	tc.pauseMu.Lock()
	tc.pauseCond.Broadcast()
	tc.pauseMu.Unlock()
}

func (tc *TransferControl) IsCancelled() bool {
	select {
	case <-tc.ctx.Done():
		return true
	default:
		return false
	}
}

func (tc *TransferControl) WaitIfPaused() {
	for tc.paused.Load() && !tc.IsCancelled() {
		tc.pauseMu.Lock()
		if tc.paused.Load() && !tc.IsCancelled() {
			tc.pauseCond.Wait()
		}
		tc.pauseMu.Unlock()
	}
}

type TransferState struct {
	ID          string
	Direction   TransferDirection
	FileName    string
	TotalSize   int64
	Status      TransferStatus
	PeerName    string
	PeerIP      string
	SpeedBps    float64
	Encrypted   bool
	StartedAt   int64
	UpdatedAt   int64
	ErrorReason string

	control            *TransferControl
	bytesDone          atomic.Int64
	lastSampleAt       time.Time
	lastSampleAt2Bytes int64
	mu                 sync.Mutex
}

// TransferSnapshot is a plain, copy-safe view for JSON/UI use.
type TransferSnapshot struct {
	ID          string            `json:"id"`
	Direction   TransferDirection `json:"direction"`
	FileName    string            `json:"fileName"`
	TotalSize   int64             `json:"totalSize"`
	BytesDone   int64             `json:"bytesDone"`
	Status      TransferStatus    `json:"status"`
	PeerName    string            `json:"peerName"`
	PeerIP      string            `json:"peerIp"`
	SpeedBps    float64           `json:"speedBps"`
	Encrypted   bool              `json:"encrypted"`
	StartedAt   int64             `json:"startedAtMs"`
	UpdatedAt   int64             `json:"updatedAtMs"`
	ErrorReason string            `json:"errorReason,omitempty"`
}

func (t *TransferState) Snapshot() TransferSnapshot {
	t.mu.Lock()
	defer t.mu.Unlock()
	return TransferSnapshot{
		ID: t.ID, Direction: t.Direction, FileName: t.FileName,
		TotalSize: t.TotalSize, BytesDone: t.bytesDone.Load(),
		Status: t.Status, PeerName: t.PeerName, PeerIP: t.PeerIP,
		SpeedBps: t.SpeedBps, Encrypted: t.Encrypted,
		StartedAt: t.StartedAt, UpdatedAt: t.UpdatedAt,
		ErrorReason: t.ErrorReason,
	}
}

func (t *TransferState) Progress() float64 {
	if t.TotalSize <= 0 {
		return 0
	}
	done := t.bytesDone.Load()
	return (float64(done) / float64(t.TotalSize)) * 100
}

func (t *TransferState) AddBytes(n int64) {
	t.bytesDone.Add(n)
	t.mu.Lock()
	defer t.mu.Unlock()

	now := time.Now()
	if t.lastSampleAt.IsZero() {
		t.lastSampleAt = now
		t.lastSampleAt2Bytes = t.bytesDone.Load()
		return
	}
	elapsed := now.Sub(t.lastSampleAt).Seconds()
	if elapsed >= 0.25 {
		deltaBytes := t.bytesDone.Load() - t.lastSampleAt2Bytes
		t.SpeedBps = float64(deltaBytes) / elapsed
		t.lastSampleAt = now
		t.lastSampleAt2Bytes = t.bytesDone.Load()
	}
}

type TransferManager struct {
	mu        sync.RWMutex
	transfers map[string]*TransferState
	onChange  func(*TransferState)
}

func NewTransferManager() *TransferManager {
	return &TransferManager{transfers: make(map[string]*TransferState)}
}

func (m *TransferManager) Register(id, fileName string, totalSize int64, dir TransferDirection, peerName, peerIP string, encrypted bool) *TransferState {
	t := &TransferState{
		ID: id, Direction: dir, FileName: fileName, TotalSize: totalSize,
		Status: StatusQueued, PeerName: peerName, PeerIP: peerIP,
		Encrypted: encrypted, StartedAt: time.Now().UnixMilli(),
		control: NewTransferControl(),
	}
	m.mu.Lock()
	m.transfers[id] = t
	m.mu.Unlock()
	m.notify(t)
	return t
}

func (m *TransferManager) Get(id string) (*TransferState, bool) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	t, ok := m.transfers[id]
	return t, ok
}

func (m *TransferManager) SetStatus(id string, status TransferStatus, errReason string) {
	m.mu.RLock()
	t, ok := m.transfers[id]
	m.mu.RUnlock()
	if !ok {
		return
	}
	t.mu.Lock()
	t.Status = status
	t.ErrorReason = errReason
	t.UpdatedAt = time.Now().UnixMilli()
	t.mu.Unlock()
	m.notify(t)
}

func (m *TransferManager) Pause(id string) error {
	t, ok := m.Get(id)
	if !ok {
		return fmt.Errorf("transfer %s not found", id)
	}
	t.control.Pause()
	m.SetStatus(id, StatusPaused, "")
	return nil
}

func (m *TransferManager) Resume(id string) error {
	t, ok := m.Get(id)
	if !ok {
		return fmt.Errorf("transfer %s not found", id)
	}
	t.control.Resume()
	m.SetStatus(id, StatusActive, "")
	return nil
}

func (m *TransferManager) Cancel(id string) error {
	t, ok := m.Get(id)
	if !ok {
		return fmt.Errorf("transfer %s not found", id)
	}
	t.control.Cancel()
	m.SetStatus(id, StatusCancelled, "")
	return nil
}

func (m *TransferManager) List() []TransferSnapshot {
	m.mu.RLock()
	defer m.mu.RUnlock()
	out := make([]TransferSnapshot, 0, len(m.transfers))
	for _, t := range m.transfers {
		out = append(out, t.Snapshot())
	}
	for i := 1; i < len(out); i++ {
		for j := i; j > 0 && out[j].StartedAt > out[j-1].StartedAt; j-- {
			out[j], out[j-1] = out[j-1], out[j]
		}
	}
	return out
}

func (m *TransferManager) notify(t *TransferState) {
	if m.onChange == nil {
		return
	}
	m.onChange(t)
}