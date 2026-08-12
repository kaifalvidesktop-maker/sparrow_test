package main

import (
	"encoding/base64"
	"fmt"

	qrcode "github.com/skip2/go-qrcode"
)

// generateQRCodeBase64 builds a PNG QR code encoding the given text (the
// phone will scan this to open Sparrow's mobile web page) and returns it
// as a base64 string ready to drop into an <img src="data:image/png;base64,...">
func generateQRCodeBase64(content string, sizePx int) (string, error) {
	png, err := qrcode.Encode(content, qrcode.Medium, sizePx)
	if err != nil {
		return "", fmt.Errorf("failed to generate QR code: %w", err)
	}
	return base64.StdEncoding.EncodeToString(png), nil
}