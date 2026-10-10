// Package codec decodes the supported input formats (JPEG, PNG) and encodes
// results with a hard size limit backed by a deterministic PNG-to-JPEG
// fallback ladder. Decoding applies the EXIF orientation of phone photos
// exactly, so all downstream operations always see an upright image.
package codec

import (
	"bytes"
	"errors"
	"fmt"
	"image"
	"image/jpeg"
	"image/png"
	"net/http"
)

// Canonical format names used across the SDK.
const (
	FormatPNG  = "png"
	FormatJPEG = "jpeg"
)

// Errors surfaced by the codec. The facade re-exports them on the public
// surface via aliases.
var (
	// ErrUnsupportedFormat is returned for bytes that are neither clean
	// JPEG nor PNG.
	ErrUnsupportedFormat = errors.New("unsupported image format")
	// ErrSourceTooLarge is returned when an input exceeds the pixel budget.
	ErrSourceTooLarge = errors.New("source image too large")
	// ErrInvalidImage is returned when decoded dimensions are degenerate.
	ErrInvalidImage = errors.New("invalid image")
)

// DetectFormat sniffs the canonical format from the magic bytes, regardless
// of the file name.
func DetectFormat(data []byte) (string, error) {
	head := data
	if len(head) > 512 {
		head = head[:512]
	}
	switch contentType := http.DetectContentType(head); contentType {
	case "image/jpeg":
		return FormatJPEG, nil
	case "image/png":
		return FormatPNG, nil
	default:
		return "", fmt.Errorf("%w: detected content type %q (need jpeg or png)", ErrUnsupportedFormat, contentType)
	}
}

// DecodeConfig returns the pixel dimensions and the detected format without
// decoding pixel data.
func DecodeConfig(data []byte) (width, height int, format string, err error) {
	format, err = DetectFormat(data)
	if err != nil {
		return 0, 0, "", err
	}
	var cfg image.Config
	switch format {
	case FormatJPEG:
		cfg, err = jpeg.DecodeConfig(bytes.NewReader(data))
	case FormatPNG:
		cfg, err = png.DecodeConfig(bytes.NewReader(data))
	}
	if err != nil {
		return 0, 0, "", fmt.Errorf("%w: %w", ErrInvalidImage, err)
	}
	return cfg.Width, cfg.Height, format, nil
}
