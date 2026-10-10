// Package exif extracts the JPEG EXIF orientation tag with a minimal,
// allocation-free parser. It reads no more than the markers and the first
// image file directory, so it never decodes pixel data.
package exif

import (
	"encoding/binary"
	"errors"
	"fmt"
)

// Orientation values as defined by the EXIF specification (tag 0x0112):
// 1 = normal, 2 = flip horizontal, 3 = rotate 180, 4 = flip vertical,
// 5 = transpose, 6 = rotate 90 CW, 7 = transverse, 8 = rotate 90 CCW.
const (
	normal      = 1
	flipH       = 2
	rotate180   = 3
	flipV       = 4
	transpose   = 5
	rotate90CW  = 6
	transverse  = 7
	rotate90CCW = 8
)

// Orientation returns the EXIF orientation (1..8) found in JPEG data.
// It returns 1 when the file carries no EXIF orientation and an error when
// the JPEG or EXIF structure itself is malformed. The caller decides whether
// parse failures are fatal; the SDK falls back to 1 and logs a warning.
func Orientation(data []byte) (int, error) {
	if len(data) < 4 || data[0] != 0xFF || data[1] != 0xD8 {
		return 0, errors.New("exif: not a JPEG (missing SOI marker)")
	}
	pos := 2
	for pos+4 <= len(data) {
		if data[pos] != 0xFF {
			return 0, fmt.Errorf("exif: bad marker byte 0x%02X at offset %d", data[pos], pos)
		}
		marker := data[pos+1]
		if marker == 0xD8 { // embedded thumbnail SOI: not our file's structure
			return 0, errors.New("exif: unexpected SOI inside segment scan")
		}
		if marker == 0xD9 { // EOI without any APP1
			return normal, nil
		}
		if marker == 0xDA { // start of scan: no more metadata segments
			return normal, nil
		}
		segLen := int(binary.BigEndian.Uint16(data[pos+2 : pos+4]))
		if segLen < 2 || pos+2+segLen > len(data) {
			return 0, fmt.Errorf("exif: bad segment length %d at offset %d", segLen, pos)
		}
		if marker == 0xE1 {
			payload := data[pos+4 : pos+2+segLen]
			if hasExifHeader(payload) {
				return parseTIFF(payload[6:])
			}
		}
		pos += 2 + segLen
	}
	return normal, nil
}

// hasExifHeader reports whether the APP1 payload starts with "Exif\0\0".
func hasExifHeader(payload []byte) bool {
	return len(payload) >= 6 && string(payload[:6]) == "Exif\x00\x00"
}

// parseTIFF reads the orientation tag (0x0112) from the first image file
// directory of an embedded TIFF structure.
func parseTIFF(tiff []byte) (int, error) {
	if len(tiff) < 8 {
		return 0, errors.New("exif: truncated TIFF header")
	}
	var order binary.ByteOrder = binary.LittleEndian
	switch string(tiff[:2]) {
	case "II":
	case "MM":
		order = binary.BigEndian
	default:
		return 0, fmt.Errorf("exif: bad TIFF byte order 0x%02X%02X", tiff[0], tiff[1])
	}
	if magic := order.Uint16(tiff[2:4]); magic != 42 {
		return 0, fmt.Errorf("exif: bad TIFF magic %d", magic)
	}
	ifd0 := int(order.Uint32(tiff[4:8]))
	if ifd0 < 8 || ifd0+2 > len(tiff) {
		return 0, fmt.Errorf("exif: bad IFD0 offset %d", ifd0)
	}
	count := int(order.Uint16(tiff[ifd0 : ifd0+2]))
	entryStart := ifd0 + 2
	if entryStart+12*count > len(tiff) {
		return 0, fmt.Errorf("exif: IFD overruns buffer (%d entries)", count)
	}
	for i := range count {
		entry := tiff[entryStart+12*i : entryStart+12*i+12]
		if tag := order.Uint16(entry[0:2]); tag == 0x0112 {
			return readOrientation(order, entry)
		}
	}
	return normal, nil
}

// readOrientation extracts the value of one orientation tag entry. The value
// is stored inline in the entry for short types with a single value, which
// covers all real-world orientation tags.
func readOrientation(order binary.ByteOrder, entry []byte) (int, error) {
	typ := order.Uint16(entry[2:4])
	count := order.Uint32(entry[4:8])
	if count == 0 {
		return 0, errors.New("exif: orientation tag with zero count")
	}
	if count != 1 {
		// With multiple values the value field holds an offset into the
		// TIFF structure instead of the value; misreading an offset whose
		// low bits fall into 1..8 would corrupt a page with a bogus
		// rotation.
		return 0, fmt.Errorf("exif: orientation tag with count %d (need 1)", count)
	}
	var raw uint64
	switch typ {
	case 3: // SHORT
		raw = uint64(order.Uint16(entry[8:10]))
	case 4: // LONG
		raw = uint64(order.Uint32(entry[8:12]))
	default:
		return 0, fmt.Errorf("exif: orientation tag has unexpected type %d", typ)
	}
	if raw < 1 || raw > 8 {
		// 9..12 exist in some draft specs but carry no defined transform.
		return 0, fmt.Errorf("exif: orientation value %d out of range 1..8", raw)
	}
	return int(raw), nil
}

// ApplyDescription maps an orientation value to a short description, used in
// Result notes and logs. Unknown values yield "invalid orientation".
func ApplyDescription(orientation int) string {
	switch orientation {
	case normal:
		return "normal"
	case flipH:
		return "flip horizontal"
	case rotate180:
		return "rotate 180"
	case flipV:
		return "flip vertical"
	case transpose:
		return "transpose"
	case rotate90CW:
		return "rotate 90 CW"
	case transverse:
		return "transverse"
	case rotate90CCW:
		return "rotate 90 CCW"
	default:
		return "invalid orientation"
	}
}
