package codec

import (
	"bytes"
	"fmt"
	"image"
	"image/jpeg"
	"image/png"

	"github.com/ghmer/veni-vidi-corrigi-sdk/internal/exif"
	"github.com/ghmer/veni-vidi-corrigi-sdk/internal/geometry"
)

// Decoded bundles the pixel data of one source file with its metadata.
type Decoded struct {
	Img    image.Image
	Format string
	// Orientation is the EXIF orientation applied for JPEG sources:
	// 1..8 valid, 0 means the EXIF data existed but was unreadable,
	// 1 is also reported when no transformation is needed.
	Orientation int
}

// Decode decodes pixel data and applies the EXIF orientation of JPEG
// sources with exact pixel permutations. maxPixels is the upper bound for
// width*height, protecting against accidental photo-stream inputs.
func Decode(data []byte, maxPixels int64) (*Decoded, error) {
	format, err := DetectFormat(data)
	if err != nil {
		return nil, err
	}
	width, height, _, err := DecodeConfig(data)
	if err != nil {
		return nil, err
	}
	if int64(width)*int64(height) > maxPixels {
		return nil, fmt.Errorf("%w: %dx%d = %d pixels exceeds the budget of %d pixels; crop first",
			ErrSourceTooLarge, width, height, int64(width)*int64(height), maxPixels)
	}

	var img image.Image
	orientation := 1
	switch format {
	case FormatJPEG:
		img, err = jpeg.Decode(bytes.NewReader(data))
		if err != nil {
			return nil, fmt.Errorf("%w: %w", ErrInvalidImage, err)
		}
		orientation = jpegOrientation(data)
		img = applyOrientation(img, orientation)
	case FormatPNG:
		img, err = png.Decode(bytes.NewReader(data))
		if err != nil {
			return nil, fmt.Errorf("%w: %w", ErrInvalidImage, err)
		}
	default:
		return nil, ErrUnsupportedFormat
	}
	return &Decoded{Img: img, Format: format, Orientation: orientation}, nil
}

// OrientationOf reports the raw EXIF orientation of the data without
// decoding pixels: 1..8 valid, 0 unreadable, 1 for anything non-JPEG.
func OrientationOf(data []byte) int {
	format, err := DetectFormat(data)
	if err != nil || format != FormatJPEG {
		return 1
	}
	return jpegOrientation(data)
}

// jpegOrientation returns 1..8 for a readable EXIF orientation and 0 when
// the EXIF block exists but is malformed.
func jpegOrientation(data []byte) int {
	orientation, err := exif.Orientation(data)
	if err != nil {
		return 0
	}
	return orientation
}

// applyOrientation maps an EXIF orientation value to its exact pixel
// permutation; values 1, 0 (unreadable) and anything else stay untouched.
func applyOrientation(img image.Image, orientation int) image.Image {
	switch orientation {
	case 2:
		return geometry.FlipH(img)
	case 3:
		return geometry.Rotate180(img)
	case 4:
		return geometry.FlipV(img)
	case 5:
		return geometry.Transpose(img)
	case 6:
		return geometry.Rotate90(img)
	case 7:
		return geometry.Transverse(img)
	case 8:
		return geometry.Rotate270(img)
	default:
		return img
	}
}

// DescribeOrientation renders an orientation value for Result notes.
func DescribeOrientation(orientation int) string {
	switch orientation {
	case 0:
		return "EXIF orientation unreadable, treated as normal"
	case 1:
		return ""
	default:
		return fmt.Sprintf("EXIF orientation %d applied (%s)", orientation, exif.ApplyDescription(orientation))
	}
}
