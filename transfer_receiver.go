package main

import (
	"bufio"
	"fmt"
	"log"
	"net"
	"os"
	"path/filepath"
	"sync"
	"sync/atomic"
)

type pendingTransfer struct {
	offer       OfferMsg
	file        *os.File
	finalPath   string
	key         []byte
	state       *TransferState
	chunksDone  atomic.Int32
	totalChunks int
	peerIP      string
}

type receiverRegistry struct {
	mu      sync.Mutex
	pending map[string]*pendingTransfer
}

var incomingTransfers = &receiverRegistry{pending: make(map[string]*pendingTransfer)}

type TransferReceiver struct {
	listener   net.Listener
	manager    *TransferManager
	stopCh     chan struct{}
	requirePin func() bool
	checkPin   func(pin string) bool
}

func NewTransferReceiver(manager *TransferManager) *TransferReceiver {
	return &TransferReceiver{manager: manager, stopCh: make(chan struct{})}
}

func (r *TransferReceiver) Start() error {
	addr := fmt.Sprintf("0.0.0.0:%d", TransferPort)
	ln, err := net.Listen("tcp", addr)
	if err != nil {
		return fmt.Errorf("failed to bind transfer server on %s: %w", addr, err)
	}
	r.listener = ln
	go r.acceptLoop()
	log.Println("[transfer] receiver listening on TCP", TransferPort)
	return nil
}

func (r *TransferReceiver) Stop() {
	close(r.stopCh)
	if r.listener != nil {
		_ = r.listener.Close()
	}
}

func (r *TransferReceiver) acceptLoop() {
	for {
		conn, err := r.listener.Accept()
		if err != nil {
			select {
			case <-r.stopCh:
				return
			default:
				log.Println("[transfer] accept error:", err)
				continue
			}
		}
		go r.handleConn(conn)
	}
}

func (r *TransferReceiver) handleConn(conn net.Conn) {
	reader := bufio.NewReader(conn)
	line, err := reader.ReadString('\n')
	if err != nil {
		_ = conn.Close()
		return
	}
	switch ParseLine(line) {
	case MsgOffer:
		r.handleOffer(conn, line)
	case MsgChunk:
		r.handleChunk(conn, reader, line)
	default:
		log.Println("[transfer] unrecognized message, closing connection")
		_ = conn.Close()
	}
}

func (r *TransferReceiver) handleOffer(conn net.Conn, line string) {
	offer, err := ParseOffer(line)
	if err != nil {
		_, _ = conn.Write([]byte(EncodeReject("bad_offer")))
		return
	}

	if r.requirePin != nil && r.requirePin() {
		_, _ = conn.Write([]byte(EncodeReject("pin_required")))
		return
	}

	destDir := state.downloadDir
	if destDir == "" {
		destDir = "."
	}
	_ = os.MkdirAll(destDir, 0755)

	safeName := sanitizeFileName(offer.FileName)
	finalPath := uniquePath(filepath.Join(destDir, safeName))

	f, err := os.Create(finalPath)
	if err != nil {
		_, _ = conn.Write([]byte(EncodeReject("cannot_create_file")))
		return
	}
	if err := f.Truncate(offer.TotalSize); err != nil {
		_ = f.Close()
		_, _ = conn.Write([]byte(EncodeReject("cannot_allocate_space")))
		return
	}

	var key []byte
	if offer.Encrypted {
		key, err = keyFromHex(offer.KeyHex)
		if err != nil {
			_ = f.Close()
			_, _ = conn.Write([]byte(EncodeReject("bad_key")))
			return
		}
	}

	peerIP, _, _ := net.SplitHostPort(conn.RemoteAddr().String())
	peerName := peerIP
	if dev, ok := globalDeviceRegistry.findByIP(peerIP); ok {
		peerName = dev.Name
	}

	ts := incomingManager().Register(offer.TransferID, offer.FileName, offer.TotalSize, DirectionReceive, peerName, peerIP, offer.Encrypted)
	incomingManager().SetStatus(offer.TransferID, StatusActive, "")

	pt := &pendingTransfer{
		offer: offer, file: f, finalPath: finalPath, key: key,
		state: ts, totalChunks: offer.ChunkCount, peerIP: peerIP,
	}

	incomingTransfers.mu.Lock()
	incomingTransfers.pending[offer.TransferID] = pt
	incomingTransfers.mu.Unlock()

	SetTransferPath(offer.TransferID, finalPath)

	_, _ = conn.Write([]byte(EncodeAccept()))
	log.Printf("[transfer] accepted incoming %q (%d bytes, %d chunks) from %s",
		offer.FileName, offer.TotalSize, offer.ChunkCount, peerIP)
}

func (r *TransferReceiver) handleChunk(conn net.Conn, reader *bufio.Reader, line string) {
	defer conn.Close()

	chunk, err := ParseChunkHeader(line)
	if err != nil {
		log.Println("[transfer] bad chunk header:", err)
		return
	}

	incomingTransfers.mu.Lock()
	pt, ok := incomingTransfers.pending[chunk.TransferID]
	incomingTransfers.mu.Unlock()
	if !ok {
		log.Println("[transfer] chunk for unknown transfer:", chunk.TransferID)
		return
	}

	writer := &fileOffsetWriter{file: pt.file, offset: chunk.Start}
	onProgress := func(n int64) { pt.state.AddBytes(n) }

	var copyErr error
	if pt.key != nil {
		_, copyErr = decryptedCopy(writer, reader, pt.key, pt.state.control, onProgress)
	} else {
		_, copyErr = plainCopy(writer, reader, pt.state.control, onProgress)
	}

	if copyErr != nil {
		log.Printf("[transfer] chunk %d of %s failed: %v", chunk.ChunkIndex, chunk.TransferID, copyErr)
		incomingManager().SetStatus(chunk.TransferID, StatusFailed, copyErr.Error())
		return
	}

	done := pt.chunksDone.Add(1)
	if int(done) >= pt.totalChunks {
		r.finalizeIncoming(pt)
	}
}

func (r *TransferReceiver) finalizeIncoming(pt *pendingTransfer) {
	_ = pt.file.Close()
	incomingTransfers.mu.Lock()
	delete(incomingTransfers.pending, pt.offer.TransferID)
	incomingTransfers.mu.Unlock()
	incomingManager().SetStatus(pt.offer.TransferID, StatusCompleted, "")
	log.Printf("[transfer] completed incoming %q -> %s", pt.offer.FileName, pt.finalPath)
}

type fileOffsetWriter struct {
	file   *os.File
	offset int64
}

func (w *fileOffsetWriter) Write(p []byte) (int, error) {
	n, err := w.file.WriteAt(p, w.offset)
	w.offset += int64(n)
	return n, err
}