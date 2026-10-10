package codec

import (
	"bytes"
	"errors"
	"fmt"
	"image"
	"image/jpeg"
	"image/png"
)

// ErrNilImage is returned when callers pass a nil image; the stdlib codecs
// panic on nil inputs, so the codec guards them explicitly.
var ErrNilImage = errors.New("nil image")

// ErrSizeLimitExceeded is returned when neither the PNG encoding nor the
// JPEG fallback ladder fits the size limit; SizeLimit is a hard cap, so
// oversized results must fail the operation instead of slipping through.
var ErrSizeLimitExceeded = errors.New("encoded size exceeds limit")

// defaultJPEGQuality is the JPEG fallback quality used when the caller
// passes an out-of-range quality (the facade validates first, so this is a
// defensive default).
const defaultJPEGQuality = 90

// jpegRetryQualities is the fixed ladder of lower qualities tried after the
// requested one. The ladder is static, so retries stay deterministic
// (same call, same result) while giving the size limit a real chance.
var jpegRetryQualities = []int{75, 50}

// Encoded bundles serialized result bytes with the chosen format.
type Encoded struct {
	Data   []byte
	Format string
	Notes  []string
}

// Encode serializes img and enforces the size budget as a hard cap: PNG is
// encoded first; when it exceeds sizeLimit, JPEG fallbacks (the requested
// quality, then the fixed lower-quality ladder) are tried and the first
// encoding that fits wins. When no encoding fits, ErrSizeLimitExceeded is
// returned. sizeLimit <= 0 disables the limit.
func Encode(img image.Image, sizeLimit int64, quality int) (*Encoded, error) {
	if img == nil {
		return nil, fmt.Errorf("%w", ErrNilImage)
	}
	quality = normalizeQuality(quality)
	pngData, err := EncodePNG(img)
	if err != nil {
		return nil, fmt.Errorf("encode png: %w", err)
	}
	if sizeLimit <= 0 || int64(len(pngData)) <= sizeLimit {
		return &Encoded{Data: pngData, Format: FormatPNG}, nil
	}
	qualities := append([]int{quality}, lowerQualities(quality)...)
	smallest := int64(len(pngData))
	for _, q := range qualities {
		jpegData, err := EncodeJPEG(img, q)
		if err != nil {
			return nil, fmt.Errorf("encode jpeg fallback: %w", err)
		}
		if int64(len(jpegData)) <= sizeLimit {
			return &Encoded{
				Data:   jpegData,
				Format: FormatJPEG,
				Notes:  []string{fmt.Sprintf("PNG size %d exceeds the limit of %d bytes: JPEG q%d fallback", len(pngData), sizeLimit, q)},
			}, nil
		}
		smallest = min(smallest, int64(len(jpegData)))
	}
	return nil, fmt.Errorf(
		"%w: smallest encoding is %d bytes, limit is %d bytes; raise SizeLimit or lower MaxSide",
		ErrSizeLimitExceeded, smallest, sizeLimit)
}

// lowerQualities lists the fixed fallback qualities below the requested
// one, so no quality is tried twice.
func lowerQualities(quality int) []int {
	out := make([]int, 0, len(jpegRetryQualities))
	for _, q := range jpegRetryQualities {
		if q < quality {
			out = append(out, q)
		}
	}
	return out
}

// normalizeQuality maps every out-of-range quality to the default.
func normalizeQuality(quality int) int {
	if quality < 1 || quality > 100 {
		return defaultJPEGQuality
	}
	return quality
}

// EncodePNG serializes an image as lossless PNG with default compression,
// which keeps text excerpts crisp.
func EncodePNG(img image.Image) ([]byte, error) {
	if img == nil {
		return nil, fmt.Errorf("%w", ErrNilImage)
	}
	var buf bytes.Buffer
	if err := png.Encode(&buf, img); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

// EncodeJPEG serializes an image as JPEG with the given quality.
func EncodeJPEG(img image.Image, quality int) ([]byte, error) {
	if img == nil {
		return nil, fmt.Errorf("%w", ErrNilImage)
	}
	var buf bytes.Buffer
	if err := jpeg.Encode(&buf, img, &jpeg.Options{Quality: quality}); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}
