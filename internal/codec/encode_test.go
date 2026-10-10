package codec

import (
	"bytes"
	"errors"
	"image"
	"image/png"
	"math/rand"
	"strings"
	"testing"
)

// errContains verifies that err exists and mentions text.
func errContains(t *testing.T, err error, text string) {
	t.Helper()
	if err == nil {
		t.Fatalf("expected error containing %q", text)
	}
	if !strings.Contains(err.Error(), text) {
		t.Fatalf("error %v does not contain %q", err, text)
	}
}

func TestEncodeChoosesPngBelowLimit(t *testing.T) {
	enc, err := Encode(solid(4, 4, 200), 0, 90)
	if err != nil {
		t.Fatalf("Encode: %v", err)
	}
	if enc.Format != FormatPNG || len(enc.Notes) != 0 {
		t.Fatalf("format = %s notes = %v, want png/no notes", enc.Format, enc.Notes)
	}
}

func TestEncodeFallsBackToJpegOverLimit(t *testing.T) {
	// Compressor-independent setup: the limit is derived from the actual
	// byte sizes, so the test does not depend on encoder versions.
	src := noisyGray()
	pngData, err := EncodePNG(src)
	if err != nil {
		t.Fatalf("EncodePNG: %v", err)
	}
	jpegData, err := EncodeJPEG(src, 90)
	if err != nil {
		t.Fatalf("EncodeJPEG: %v", err)
	}
	if len(jpegData) >= len(pngData) {
		t.Fatalf("fixture must compress better as JPEG: png %d vs jpeg %d", len(pngData), len(jpegData))
	}
	limit := int64(len(pngData)) - 1

	enc, err := Encode(src, limit, 90)
	if err != nil {
		t.Fatalf("Encode: %v", err)
	}
	if enc.Format != FormatJPEG {
		t.Fatalf("format = %s, want jpeg fallback", enc.Format)
	}
	if len(enc.Notes) != 1 || !strings.Contains(enc.Notes[0], "JPEG") {
		t.Fatalf("notes = %v, want a single JPEG fallback note", enc.Notes)
	}
	if int64(len(enc.Data)) > limit {
		t.Fatalf("fallback output %d exceeds the limit %d", len(enc.Data), limit)
	}
}

// noisyGray returns a 64x64 fine-grained deterministic noise image: PNG
// compresses it poorly while JPEG stays much smaller, which makes the
// fallback test encoder-version independent.
//
//nolint:gosec // test fixture: narrowing conversion on controlled random values
func noisyGray() *image.NRGBA {
	rnd := rand.New(rand.NewSource(1))
	img := image.NewNRGBA(image.Rect(0, 0, 64, 64))
	for y := range 64 {
		for x := range 64 {
			v := uint8(rnd.Intn(256))
			i := (y*64 + x) * 4
			img.Pix[i], img.Pix[i+1], img.Pix[i+2], img.Pix[i+3] = v, v, v, 255
		}
	}
	return img
}

func TestEncodeRejectsWhenNothingFits(t *testing.T) {
	// A flat 4x4 PNG (~70 bytes) and its JPEG are both far larger than the
	// 10-byte limit: SizeLimit is a hard cap, so the operation must fail
	// instead of returning an oversized result silently.
	_, err := Encode(solid(4, 4, 128), 10, 90)
	if !errors.Is(err, ErrSizeLimitExceeded) {
		t.Fatalf("error = %v, want ErrSizeLimitExceeded", err)
	}
	errContains(t, err, "smallest encoding")
}

func TestEncodeJPEGQualityLadder(t *testing.T) {
	// When the requested JPEG quality does not fit, the fixed lower-quality
	// ladder must be tried before failing, and the result must respect the
	// limit.
	src := noisyGray()
	q90, err := EncodeJPEG(src, 90)
	if err != nil {
		t.Fatalf("EncodeJPEG: %v", err)
	}
	q50, err := EncodeJPEG(src, 50)
	if err != nil {
		t.Fatalf("EncodeJPEG: %v", err)
	}
	if len(q50) >= len(q90) {
		t.Fatalf("fixture must compress smaller at quality 50: q50 %d vs q90 %d", len(q50), len(q90))
	}
	limit := int64(len(q90)) - 1 // q90 does not fit

	enc, err := Encode(src, limit, 90)
	if err != nil {
		t.Fatalf("Encode: %v", err)
	}
	if enc.Format != FormatJPEG {
		t.Fatalf("format = %s, want jpeg ladder fallback", enc.Format)
	}
	if int64(len(enc.Data)) > limit {
		t.Fatalf("ladder output %d exceeds the limit %d", len(enc.Data), limit)
	}
	if len(enc.Notes) != 1 || !strings.Contains(enc.Notes[0], "fallback") {
		t.Fatalf("notes = %v, want a fallback note", enc.Notes)
	}
}

func TestEncodeNormalizesQuality(t *testing.T) {
	src := verticalStripes(64, 64)
	withZero, err := Encode(src, 10<<20, 0)
	if err != nil {
		t.Fatalf("Encode: %v", err)
	}
	if withZero.Format != FormatPNG {
		t.Fatalf("limitless png must stay png, got %s", withZero.Format)
	}
}

func TestEncodePNGRoundTrip(t *testing.T) {
	src := verticalStripes(8, 8)
	data, err := EncodePNG(src)
	if err != nil {
		t.Fatalf("EncodePNG: %v", err)
	}
	dec, err := png.Decode(bytes.NewReader(data))
	if err != nil {
		t.Fatalf("roundtrip decode: %v", err)
	}
	if dec.Bounds().Dx() != 8 || dec.Bounds().Dy() != 8 {
		t.Fatalf("dims = %v", dec.Bounds())
	}
}

func TestEncodeRejectsNil(t *testing.T) {
	if _, err := EncodeJPEG(nil, 90); !errors.Is(err, ErrNilImage) {
		t.Fatalf("EncodeJPEG error = %v, want ErrNilImage", err)
	}
	if _, err := EncodePNG(nil); !errors.Is(err, ErrNilImage) {
		t.Fatalf("EncodePNG error = %v, want ErrNilImage", err)
	}
	if _, err := Encode(nil, 0, 90); !errors.Is(err, ErrNilImage) {
		t.Fatalf("Encode error = %v, want ErrNilImage", err)
	}
}
