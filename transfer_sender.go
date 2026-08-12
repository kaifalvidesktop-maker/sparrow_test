package main

import (
	"bufio"
	"fmt"
	"io"
	"log"
	"net"
	"os"
	"path/filepath"
	"sync"
	"time"
)

// ============================================================
// transfer_sender.go — splits a file into 8 chunks, sends via
// 8 parallel goroutines/TCP connections[cite: 14]
// ============================================================

func SendFile(filePath, destIP string, destPort int, turbo bool) (*TransferState, error) {
	if IsSelfIP(destIP) {
		return nil, fmt.Errorf("cannot send to your own device")
	}

	info, err := os.Stat(filePath)
	if err != nil {
		return nil, fmt.Errorf("cannot read file: %w", err)
	}
	if info.IsDir() {
		return nil, fmt.Errorf("%s is a directory — use SendFolder instead", filePath)
	}

	totalSize := info.Size()
	fileName := filepath.Base(filePath)

	transferID, err := generateTransferID()
	if err != nil {
		return nil, fmt.Errorf("failed to generate transfer id: %w", err)
	}

	chunkCount := ChunkCount
	if totalSize < MinChunkSize {
		chunkCount = 1
	}

	var key []byte
	encrypted := !turbo
	if encrypted {
		key, err = generateAESKey()
		if err != nil {
			return nil, fmt.Errorf("failed to generate encryption key: %w", err)
		}
	}

	peerName := destIP
	if dev, ok := globalDeviceRegistry.findByIP(destIP); ok {
		peerName = dev.Name
	}

	ts := globalTransferManager.Register(transferID, fileName, totalSize, DirectionSend, peerName, destIP, encrypted)

	if err := performOfferHandshake(destIP, destPort, OfferMsg{
		TransferID: transferID, FileName: fileName, TotalSize: totalSize,
		ChunkCount: chunkCount, Encrypted: encrypted, KeyHex: boolKeyHex(encrypted, key),
	}); err != nil {
		globalTransferManager.SetStatus(transferID, StatusFailed, err.Error())
		return ts, err
	}

	globalTransferManager.SetStatus(transferID, StatusActive, "")

	go sendChunksParallel(filePath, destIP, destPort, transferID, chunkCount, totalSize, key, ts)

	return ts, nil
}

func boolKeyHex(encrypted bool, key []byte) string {
	if !encrypted {
		return "-"
	}
	return keyToHex(key)
}

func performOfferHandshake(destIP string, destPort int, offer OfferMsg) error {
	addr := fmt.Sprintf("%s:%d", destIP, destPort)
	conn, err := net.DialTimeout("tcp", addr, 8*time.Second)
	if err != nil {
		return fmt.Errorf("could not connect to %s: %w", addr, err)
	}
	defer conn.Close()

	_ = conn.SetDeadline(time.Now().Add(10 * time.Second))

	if _, err := conn.Write([]byte(EncodeOffer(offer))); err != nil {
		return fmt.Errorf("failed to send offer: %w", err)
	}

	reader := bufio.NewReader(conn)
	line, err := reader.ReadString('\n')
	if err != nil {
		return fmt.Errorf("no response from receiver: %w", err)
	}

	switch ParseLine(line) {
	case MsgAccept:
		return nil
	case MsgReject:
		return fmt.Errorf("receiver declined: %s", ParseReject(line))
	default:
		return fmt.Errorf("unexpected response from receiver")
	}
}

func sendChunksParallel(filePath, destIP string, destPort int, transferID string, chunkCount int, totalSize int64, key []byte, ts *TransferState) {
	chunks := computeChunkBoundaries(totalSize, chunkCount)

	var wg sync.WaitGroup
	errCh := make(chan error, len(chunks))

	for _, c := range chunks {
		c.TransferID = transferID
		wg.Add(1)
		go func(chunk ChunkMsg) {
			defer wg.Done()
			if err := sendOneChunk(filePath, destIP, destPort, chunk, key, ts); err != nil {
				errCh <- err
			}
		}(c)
	}

	wg.Wait()
	close(errCh)

	if firstErr, hasErr := <-errCh; hasErr {
		globalTransferManager.SetStatus(transferID, StatusFailed, firstErr.Error())
		log.Printf("[transfer] send %s failed: %v", transferID, firstErr)
		return
	}

	if ts.control.IsCancelled() {
		globalTransferManager.SetStatus(transferID, StatusCancelled, "")
		return
	}

	globalTransferManager.SetStatus(transferID, StatusCompleted, "")
	log.Printf("[transfer] send %s completed (%d bytes, %d chunks)", transferID, totalSize, len(chunks))
}

func sendOneChunk(filePath, destIP string, destPort int, chunk ChunkMsg, key []byte, ts *TransferState) error {
	f, err := os.Open(filePath)
	if err != nil {
		return fmt.Errorf("chunk %d: failed to open file: %w", chunk.ChunkIndex, err)
	}
	defer f.Close()

	if _, err := f.Seek(chunk.Start, os.SEEK_SET); err != nil {
		return fmt.Errorf("chunk %d: seek failed: %w", chunk.ChunkIndex, err)
	}

	addr := fmt.Sprintf("%s:%d", destIP, destPort)
	conn, err := net.DialTimeout("tcp", addr, 8*time.Second)
	if err != nil {
		return fmt.Errorf("chunk %d: connect failed: %w", chunk.ChunkIndex, err)
	}
	defer conn.Close()

	if _, err := conn.Write([]byte(EncodeChunkHeader(chunk))); err != nil {
		return fmt.Errorf("chunk %d: failed to send header: %w", chunk.ChunkIndex, err)
	}

	limitedReader := &io.LimitedReader{R: f, N: chunk.End - chunk.Start}

	onProgress := func(n int64) { ts.AddBytes(n) }

	var copyErr error
	if key != nil {
		_, copyErr = encryptedCopy(conn, limitedReader, key, ts.control, onProgress)
	} else {
		_, copyErr = plainCopy(conn, limitedReader, ts.control, onProgress)
	}

	if copyErr != nil {
		return fmt.Errorf("chunk %d: %w", chunk.ChunkIndex, copyErr)
	}
	return nil
}