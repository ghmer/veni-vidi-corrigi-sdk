package codec

import (
	"encoding/binary"
	"errors"
	"image"
	"testing"
)

func TestDetectFormat(t *testing.T) {
	tests := []struct {
		name    string
		data    []byte
		want    string
		wantErr error
	}{
		{name: "png", data: encodePNG(t, solid(2, 2, 5)), want: FormatPNG},
		{name: "jpeg", data: encodeJPEG(t, solid(2, 2, 5)), want: FormatJPEG},
		{name: "gif rejected", data: []byte("GIF89a\x00\x00"), wantErr: ErrUnsupportedFormat},
		{name: "garbage rejected", data: []byte("hello world"), wantErr: ErrUnsupportedFormat},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			format, err := DetectFormat(tt.data)
			if tt.wantErr != nil {
				if !errors.Is(err, tt.wantErr) {
					t.Fatalf("error = %v, want %v", err, tt.wantErr)
				}
				return
			}
			if err != nil {
				t.Fatalf("DetectFormat: %v", err)
			}
			if format != tt.want {
				t.Fatalf("format = %s, want %s", format, tt.want)
			}
		})
	}
}

func TestDecodeConfig(t *testing.T) {
	images := []struct {
		format string
		data   []byte
	}{
		{FormatPNG, encodePNG(t, solid(7, 5, 9))},
		{FormatJPEG, encodeJPEG(t, solid(7, 5, 9))},
	}
	for _, tt := range images {
		t.Run(tt.format, func(t *testing.T) {
			w, h, format, err := DecodeConfig(tt.data)
			if err != nil {
				t.Fatalf("DecodeConfig: %v", err)
			}
			if w != 7 || h != 5 || format != tt.format {
				t.Fatalf("got %dx%d %s, want 7x5 %s", w, h, format, tt.format)
			}
		})
	}
}

func TestDecodeConfigRejectsGarbage(t *testing.T) {
	if _, _, _, err := DecodeConfig([]byte{0x89, 0x50, 0x4E}); err == nil {
		t.Fatal("accepted truncated png bytes")
	}
}

// exifTiff builds a minimal little-endian TIFF with an orientation entry.
func exifTiff(orient uint16) []byte {
	b := []byte("II")
	b = binary.LittleEndian.AppendUint16(b, 42)
	b = binary.LittleEndian.AppendUint32(b, 8)
	b = binary.LittleEndian.AppendUint16(b, 1)
	b = binary.LittleEndian.AppendUint16(b, 0x0112)
	b = binary.LittleEndian.AppendUint16(b, 3)
	b = binary.LittleEndian.AppendUint32(b, 1)
	b = binary.LittleEndian.AppendUint16(b, orient)
	b = binary.LittleEndian.AppendUint16(b, 0)
	b = binary.LittleEndian.AppendUint32(b, 0)
	return b
}

// jpegWithOrientation wraps a JPEG payload plus an APP1 EXIF segment.
//
//nolint:gosec // test fixture: narrowing conversions on small controlled values
func jpegWithOrientation(t *testing.T, img image.Image, orient uint16) []byte {
	t.Helper()
	base := encodeJPEG(t, img)
	payload := append([]byte("Exif\x00\x00"), exifTiff(orient)...)
	out := append([]byte{0xFF, 0xD8}, 0xFF, 0xE1)
	out = binary.BigEndian.AppendUint16(out, uint16(len(payload)+2))
	out = append(out, payload...)
	out = append(out, base[2:]...) // strip the base SOI
	return out
}
