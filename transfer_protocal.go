package main

import (
	"fmt"
	"strconv"
	"strings"
)

// ============================================================
// transfer_protocol.go — raw TCP wire protocol on port 53318
// ============================================================

const (
	ChunkCount     = 8
	CopyBufferSize = 1 << 20 // 1 MiB
	MinChunkSize   = 256 * 1024
)

type MsgType int

const (
	MsgUnknown MsgType = iota
	MsgOffer
	MsgAccept
	MsgReject
	MsgChunk
)

type OfferMsg struct {
	TransferID string
	FileName   string
	TotalSize  int64
	ChunkCount int
	Encrypted  bool
	KeyHex     string
}

type ChunkMsg struct {
	TransferID string
	ChunkIndex int
	Start      int64
	End        int64
}

func EncodeOffer(o OfferMsg) string {
	enc := "0"
	key := "-"
	if o.Encrypted {
		enc = "1"
		key = o.KeyHex
	}
	return fmt.Sprintf("OFFER|%s|%s|%d|%d|%s|%s\n",
		o.TransferID, o.FileName, o.TotalSize, o.ChunkCount, enc, key)
}

func EncodeAccept() string { return "ACCEPT\n" }

func EncodeReject(reason string) string { return fmt.Sprintf("REJECT|%s\n", reason) }

func EncodeChunkHeader(c ChunkMsg) string {
	return fmt.Sprintf("CHUNK|%s|%d|%d|%d\n", c.TransferID, c.ChunkIndex, c.Start, c.End)
}

func ParseLine(line string) MsgType {
	switch {
	case strings.HasPrefix(line, "OFFER|"):
		return MsgOffer
	case strings.HasPrefix(line, "ACCEPT"):
		return MsgAccept
	case strings.HasPrefix(line, "REJECT|"):
		return MsgReject
	case strings.HasPrefix(line, "CHUNK|"):
		return MsgChunk
	default:
		return MsgUnknown
	}
}

func ParseOffer(line string) (OfferMsg, error) {
	parts := strings.SplitN(line, "|", 7)
	if len(parts) != 7 {
		return OfferMsg{}, fmt.Errorf("malformed OFFER line: %q", line)
	}
	totalSize, err := strconv.ParseInt(parts[3], 10, 64)
	if err != nil {
		return OfferMsg{}, fmt.Errorf("bad totalSize in OFFER: %w", err)
	}
	chunkCount, err := strconv.Atoi(parts[4])
	if err != nil {
		return OfferMsg{}, fmt.Errorf("bad chunkCount in OFFER: %w", err)
	}
	return OfferMsg{
		TransferID: parts[1], FileName: parts[2], TotalSize: totalSize,
		ChunkCount: chunkCount, Encrypted: parts[5] == "1", KeyHex: parts[6],
	}, nil
}

func ParseChunkHeader(line string) (ChunkMsg, error) {
	parts := strings.SplitN(line, "|", 5)
	if len(parts) != 5 {
		return ChunkMsg{}, fmt.Errorf("malformed CHUNK line: %q", line)
	}
	index, err := strconv.Atoi(parts[2])
	if err != nil {
		return ChunkMsg{}, fmt.Errorf("bad chunkIndex in CHUNK: %w", err)
	}
	start, err := strconv.ParseInt(parts[3], 10, 64)
	if err != nil {
		return ChunkMsg{}, fmt.Errorf("bad start in CHUNK: %w", err)
	}
	end, err := strconv.ParseInt(parts[4], 10, 64)
	if err != nil {
		return ChunkMsg{}, fmt.Errorf("bad end in CHUNK: %w", err)
	}
	return ChunkMsg{TransferID: parts[1], ChunkIndex: index, Start: start, End: end}, nil
}

func ParseReject(line string) string {
	parts := strings.SplitN(line, "|", 2)
	if len(parts) == 2 {
		return parts[1]
	}
	return "unknown"
}

func computeChunkBoundaries(totalSize int64, count int) []ChunkMsg {
	if count < 1 {
		count = 1
	}
	if totalSize < MinChunkSize {
		count = 1
	}
	base := totalSize / int64(count)
	chunks := make([]ChunkMsg, 0, count)
	var offset int64
	for i := 0; i < count; i++ {
		start := offset
		end := start + base
		if i == count-1 {
			end = totalSize
		}
		chunks = append(chunks, ChunkMsg{ChunkIndex: i, Start: start, End: end})
		offset = end
	}
	return chunks
}