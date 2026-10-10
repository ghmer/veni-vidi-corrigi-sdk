package exif

import (
	"encoding/binary"
	"testing"
)

// tiffBytes builds a minimal TIFF structure with a single IFD0 entry for the
// orientation tag. Only meant for tests inside this package.
//
//nolint:gosec // test fixture: narrowing conversions on small controlled values
func tiffBytes(big bool, orientVal uint16, typ uint16) []byte {
	magic := []byte("II")
	if big {
		magic = []byte("MM")
	}
	u16 := func(v uint16) []byte {
		if big {
			return []byte{byte(v >> 8), byte(v)}
		}
		return []byte{byte(v), byte(v >> 8)}
	}
	u32 := func(v uint32) []byte {
		hi := u16(uint16(v >> 16))
		lo := u16(uint16(v))
		if big {
			return append(hi, lo...)
		}
		return append(lo, hi...)
	}
	b := make([]byte, 0, 26)
	b = append(b, magic...)
	b = append(b, u16(42)...)
	b = append(b, u32(8)...) // IFD0 follows the 8-byte header
	b = append(b, u16(1)...) // one entry
	b = append(b, u16(0x0112)...)
	b = append(b, u16(typ)...)
	b = append(b, u32(1)...)
	switch typ {
	case 3: // SHORT: value inline in the first two bytes of the value field
		b = append(b, u16(orientVal)...)
		b = append(b, u16(0)...)
	case 4: // LONG: value occupies the full four-byte value field
		b = append(b, u32(uint32(orientVal))...)
	}
	b = append(b, u32(0)...) // next IFD: none
	return b
}

// jpegBytes wraps payloads into a minimal JPEG segment chain.
//
//nolint:gosec // test fixture: narrowing conversions on small controlled values
func jpegBytes(app1Payload []byte) []byte {
	b := []byte{0xFF, 0xD8}
	if app1Payload != nil {
		b = append(b, 0xFF, 0xE1)
		b = binary.BigEndian.AppendUint16(b, uint16(2+len(app1Payload)))
		b = append(b, app1Payload...)
	}
	b = append(b, 0xFF, 0xFE, 0x00, 0x02) // empty COM segment
	b = append(b, 0xFF, 0xDA, 0x00, 0x02) // SOS ends the metadata scan
	return b
}

func exifPayload(tiff []byte) []byte {
	return append([]byte("Exif\x00\x00"), tiff...)
}

func TestOrientationLittleEndian(t *testing.T) {
	got, err := Orientation(jpegBytes(exifPayload(tiffBytes(false, 6, 3))))
	if err != nil {
		t.Fatalf("Orientation: %v", err)
	}
	if got != 6 {
		t.Fatalf("orientation = %d, want 6", got)
	}
}

func TestOrientationBigEndian(t *testing.T) {
	got, err := Orientation(jpegBytes(exifPayload(tiffBytes(true, 8, 3))))
	if err != nil {
		t.Fatalf("Orientation: %v", err)
	}
	if got != 8 {
		t.Fatalf("orientation = %d, want 8", got)
	}
}

func TestOrientationLongType(t *testing.T) {
	got, err := Orientation(jpegBytes(exifPayload(tiffBytes(false, 3, 4))))
	if err != nil {
		t.Fatalf("Orientation: %v", err)
	}
	if got != 3 {
		t.Fatalf("orientation = %d, want 3", got)
	}
}

func TestOrientationWithoutExif(t *testing.T) {
	for _, data := range [][]byte{
		jpegBytes(nil),
		jpegBytes([]byte("JFIF-not-exif-app1")),
	} {
		got, err := Orientation(data)
		if err != nil {
			t.Fatalf("Orientation: %v", err)
		}
		if got != 1 {
			t.Fatalf("orientation = %d, want 1", got)
		}
	}
}

func TestOrientationErrors(t *testing.T) {
	tests := []struct {
		name string
		data []byte
	}{
		{"not jpg", []byte("GIF89a")},
		{"too short", []byte{0xFF, 0xD8}},
		{"exif without tiff magic", jpegBytes(exifPayload([]byte("XX\x2a\x00\x08\x00")))},
		{"value out of range", jpegBytes(exifPayload(tiffBytes(false, 9, 3)))},
		{"bad ifd offset", func() []byte {
			raw := tiffBytes(false, 6, 3)
			raw[4] = 0xFF // corrupt IFD0 offset
			return jpegBytes(exifPayload(raw))
		}()},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if _, err := Orientation(tt.data); err == nil {
				t.Fatalf("expected error for %s", tt.name)
			}
		})
	}
}

func TestApplyDescriptionCoversAll(t *testing.T) {
	for o := 1; o <= 8; o++ {
		if ApplyDescription(o) == "invalid orientation" {
			t.Fatalf("description missing for orientation %d", o)
		}
	}
	if ApplyDescription(0) != "invalid orientation" {
		t.Fatalf("description for invalid value wrong")
	}
}
