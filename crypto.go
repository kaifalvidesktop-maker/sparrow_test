package main

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"encoding/binary"
	"encoding/hex"
	"fmt"
	"io"
)

// ============================================================
// crypto.go — AES-256-GCM framed stream encryption (Turbo Mode OFF)
// ============================================================

const (
	aesKeySize   = 32
	gcmNonceSize = 12
)

func generateAESKey() ([]byte, error) {
	key := make([]byte, aesKeySize)
	if _, err := rand.Read(key); err != nil {
		return nil, fmt.Errorf("failed to generate AES key: %w", err)
	}
	return key, nil
}

func keyToHex(key []byte) string           { return hex.EncodeToString(key) }
func keyFromHex(s string) ([]byte, error) { return hex.DecodeString(s) }

func newGCM(key []byte) (cipher.AEAD, error) {
	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, fmt.Errorf("failed to create AES cipher: %w", err)
	}
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, fmt.Errorf("failed to create GCM mode: %w", err)
	}
	return gcm, nil
}

func encryptedCopy(dst io.Writer, src io.Reader, key []byte, control *TransferControl, onProgress func(n int64)) (int64, error) {
	gcm, err := newGCM(key)
	if err != nil {
		return 0, err
	}
	buf := make([]byte, CopyBufferSize)
	var total int64

	for {
		if control != nil {
			control.WaitIfPaused()
			if control.IsCancelled() {
				return total, fmt.Errorf("transfer cancelled")
			}
		}

		n, readErr := src.Read(buf)
		if n > 0 {
			nonce := make([]byte, gcmNonceSize)
			if _, rErr := rand.Read(nonce); rErr != nil {
				return total, fmt.Errorf("failed to generate nonce: %w", rErr)
			}
			ciphertext := gcm.Seal(nil, nonce, buf[:n], nil)

			frame := make([]byte, 4+gcmNonceSize+len(ciphertext))
			binary.BigEndian.PutUint32(frame[0:4], uint32(gcmNonceSize+len(ciphertext)))
			copy(frame[4:4+gcmNonceSize], nonce)
			copy(frame[4+gcmNonceSize:], ciphertext)

			if _, wErr := dst.Write(frame); wErr != nil {
				return total, fmt.Errorf("failed writing encrypted frame: %w", wErr)
			}
			total += int64(n)
			if onProgress != nil {
				onProgress(int64(n))
			}
		}
		if readErr == io.EOF {
			break
		}
		if readErr != nil {
			return total, fmt.Errorf("read error during encrypted copy: %w", readErr)
		}
	}
	return total, nil
}

func decryptedCopy(dst io.Writer, src io.Reader, key []byte, control *TransferControl, onProgress func(n int64)) (int64, error) {
	gcm, err := newGCM(key)
	if err != nil {
		return 0, err
	}
	lenBuf := make([]byte, 4)
	var total int64

	for {
		if control != nil {
			control.WaitIfPaused()
			if control.IsCancelled() {
				return total, fmt.Errorf("transfer cancelled")
			}
		}

		_, err := io.ReadFull(src, lenBuf)
		if err == io.EOF {
			break
		}
		if err != nil {
			return total, fmt.Errorf("failed reading frame length: %w", err)
		}

		frameLen := binary.BigEndian.Uint32(lenBuf)
		if frameLen < gcmNonceSize {
			return total, fmt.Errorf("corrupt frame: length %d too small", frameLen)
		}

		frame := make([]byte, frameLen)
		if _, err := io.ReadFull(src, frame); err != nil {
			return total, fmt.Errorf("failed reading frame body: %w", err)
		}

		nonce := frame[:gcmNonceSize]
		ciphertext := frame[gcmNonceSize:]

		plaintext, err := gcm.Open(nil, nonce, ciphertext, nil)
		if err != nil {
			return total, fmt.Errorf("decryption failed (corrupt data or wrong key): %w", err)
		}

		if _, err := dst.Write(plaintext); err != nil {
			return total, fmt.Errorf("failed writing decrypted plaintext: %w", err)
		}
		total += int64(len(plaintext))
		if onProgress != nil {
			onProgress(int64(len(plaintext)))
		}
	}
	return total, nil
}

func plainCopy(dst io.Writer, src io.Reader, control *TransferControl, onProgress func(n int64)) (int64, error) {
	buf := make([]byte, CopyBufferSize)
	var total int64
	for {
		if control != nil {
			control.WaitIfPaused()
			if control.IsCancelled() {
				return total, fmt.Errorf("transfer cancelled")
			}
		}

		n, readErr := src.Read(buf)
		if n > 0 {
			if _, wErr := dst.Write(buf[:n]); wErr != nil {
				return total, fmt.Errorf("write error during plain copy: %w", wErr)
			}
			total += int64(n)
			if onProgress != nil {
				onProgress(int64(n))
			}
		}
		if readErr == io.EOF {
			break
		}
		if readErr != nil {
			return total, fmt.Errorf("read error during plain copy: %w", readErr)
		}
	}
	return total, nil
}