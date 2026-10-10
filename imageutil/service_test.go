package imageutil

import (
	"bytes"
	"context"
	"encoding/base64"
	"errors"
	"image"
	"math/rand"
	"strings"
	"testing"
)

func TestNewValidatesSettings(t *testing.T) {
	tests := []struct {
		name   string
		opt    Option
		errMsg string
	}{
		{"maxSide below", WithMaxSide(100), "maxSide"},
		{"maxSide above", WithMaxSide(5000), "maxSide"},
		{"sizeLimit small", WithSizeLimit(10), "sizeLimit"},
		{"quality invalid", WithQuality(101), "quality"},
		{"maxInputBytes small", WithMaxInputBytes(500), "maxInputBytes"},
		{"maxInputPixels zero", WithMaxInputPixels(0), "maxPixels"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := New(WithLogger(discardLogger()), tt.opt)
			if !errors.Is(err, ErrInvalidParameter) {
				t.Fatalf("error = %v, want ErrInvalidParameter", err)
			}
			errContains(t, err, tt.errMsg)
		})
	}
}

func TestWithLoggerNilKeepsDefault(t *testing.T) {
	s, err := New(WithLogger(nil))
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	if s.log == nil {
		t.Fatal("logger is nil, want default slog logger")
	}
}

func TestInputResolvesRawBytes(t *testing.T) {
	s := newTestService(t)
	info, err := s.Info(context.Background(), pngInput("scan1.png", solidPage(24, 32)))
	if err != nil {
		t.Fatalf("Info: %v", err)
	}
	if info.Width != 24 || info.Height != 32 || info.Format != "png" {
		t.Fatalf("info = %+v", info)
	}
	if info.ID != "scan1" {
		t.Fatalf("id = %s, want scan1 (sanitized name)", info.ID)
	}
}

func TestInputResolvesBase64(t *testing.T) {
	s := newTestService(t)
	encoded := base64.StdEncoding.EncodeToString(encodePNGBytes(solidPage(20, 10)))
	info, err := s.Info(context.Background(), Input{Base64: encoded, Name: "b64"})
	if err != nil {
		t.Fatalf("Info with base64: %v", err)
	}
	if info.Width != 20 || info.Height != 10 {
		t.Fatalf("info = %+v", info)
	}
}

func TestInputWithBothPayloadsRejected(t *testing.T) {
	s := newTestService(t)
	// Both fields set is ambiguous: the SDK must reject it instead of
	// silently preferring Data, which would hide transport-layer bugs.
	in := Input{
		Data:   encodePNGBytes(solidPage(30, 30)),
		Base64: base64.StdEncoding.EncodeToString(encodePNGBytes(solidPage(11, 11))),
	}
	_, err := s.Info(context.Background(), in)
	if !errors.Is(err, ErrInvalidParameter) {
		t.Fatalf("error = %v, want ErrInvalidParameter", err)
	}
	errContains(t, err, "exactly one")
}

// nilContext hands the operations a genuinely nil context. The indirection
// exists only so static analysis (SA1012) stays quiet about the deliberate
// negative test below.
func nilContext() context.Context { return nil }

// TestNilContextRejected pins the regression fix: a nil context is a
// caller bug and must fail with ErrInvalidParameter instead of panicking
// inside checkCtx.
func TestNilContextRejected(t *testing.T) {
	s := newTestService(t)
	src := pngInput("scan1", solidPage(10, 10))
	ctx := nilContext()

	if _, err := s.Info(ctx, src); !errors.Is(err, ErrInvalidParameter) {
		t.Fatalf("Info error = %v, want ErrInvalidParameter", err)
	}
	if _, err := s.View(ctx, src, 0); !errors.Is(err, ErrInvalidParameter) {
		t.Fatalf("View error = %v, want ErrInvalidParameter", err)
	}
	if _, err := s.Crop(ctx, src, Region{X: 0, Y: 0, W: 0.5, H: 0.5}, Options{}); !errors.Is(err, ErrInvalidParameter) {
		t.Fatalf("Crop error = %v, want ErrInvalidParameter", err)
	}
}

func TestEmptyInputRejected(t *testing.T) {
	s := newTestService(t)
	_, err := s.Info(context.Background(), Input{Name: "nothing"})
	if !errors.Is(err, ErrEmptyInput) {
		t.Fatalf("error = %v, want ErrEmptyInput", err)
	}
	_, err = s.Crop(context.Background(), Input{}, Region{}, Options{})
	if !errors.Is(err, ErrEmptyInput) {
		t.Fatalf("Crop error = %v, want ErrEmptyInput", err)
	}
}

func TestInputByteBudgetEnforced(t *testing.T) {
	s, err := New(WithLogger(discardLogger()), WithMaxInputBytes(1024))
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	// Noise compresses badly, so even a small image exceeds 1024 bytes.
	in := bytesInput("scan1", noisyPNG(100, 100))
	_, err = s.Info(context.Background(), in)
	if !errors.Is(err, ErrInputTooLarge) {
		t.Fatalf("raw bytes: error = %v, want ErrInputTooLarge", err)
	}
	_, err = s.View(context.Background(), bytesInput("x", makeGarbage(2000)), 0)
	if !errors.Is(err, ErrInputTooLarge) {
		t.Fatalf("oversized garbage: error = %v, want ErrInputTooLarge", err)
	}
}

// noisyPNG encodes a w x h deterministic noise image whose PNG form is
// several times larger than the byte budgets used in tests.
//
//nolint:gosec // test fixture: deterministic noise, narrowing conversion is safe
func noisyPNG(w, h int) []byte {
	img := image.NewNRGBA(image.Rect(0, 0, w, h))
	rnd := rand.New(rand.NewSource(1))
	for i := 0; i < len(img.Pix); i++ {
		img.Pix[i] = uint8(rnd.Intn(256))
	}
	return encodePNGBytes(img)
}

func TestBase64ByteBudgetAndMalformedRejected(t *testing.T) {
	s, err := New(WithLogger(discardLogger()), WithMaxInputBytes(1024))
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	// A large base64 string must be rejected before decoding allocates:
	// 1024 input bytes are ~1372 base64 characters.
	_, err = s.Info(context.Background(), Input{Base64: strings.Repeat("A", 4000)})
	if !errors.Is(err, ErrInputTooLarge) {
		t.Fatalf("oversized base64: error = %v, want ErrInputTooLarge", err)
	}
	// Malformed base64 fails cleanly.
	_, err = s.Info(context.Background(), Input{Base64: "%%%not-base64%%%"})
	if !errors.Is(err, ErrInvalidImage) {
		t.Fatalf("malformed base64: error = %v, want ErrInvalidImage", err)
	}
}

// makeGarbage returns n bytes of non-image filler.
func makeGarbage(n int) []byte {
	return bytes.Repeat([]byte("x"), n)
}

// TestAutoWindowRules ... (unchanged helpers keep their tests)
func TestAutoWindowRules(t *testing.T) {
	tests := []struct {
		shortEdge int
		want      int
	}{
		{100, 15},   // 3 would be below the floor
		{800, 25},   // 24 -> odded up
		{1600, 49},  // 48 -> odded up
		{5000, 121}, // capped
	}
	for _, tt := range tests {
		got := autoWindow(tt.shortEdge)
		if got != tt.want || got%2 != 1 {
			t.Fatalf("autoWindow(%d) = %d, want %d and odd", tt.shortEdge, got, tt.want)
		}
	}
}

func TestResolvedMaxSideAndEncode(t *testing.T) {
	s := newTestService(t)

	maxSide, notes, err := s.resolvedMaxSide(Options{})
	if err != nil || maxSide != defaultMaxSide || notes != nil {
		t.Fatalf("default: %d/%v/%v", maxSide, notes, err)
	}
	maxSide, notes, err = s.resolvedMaxSide(Options{MaxSide: 9999})
	if err != nil || maxSide != hardMaxSide || len(notes) != 1 {
		t.Fatalf("cap: %d/%v/%v", maxSide, notes, err)
	}
	if _, _, err := s.resolvedMaxSide(Options{MaxSide: -5}); !errors.Is(err, ErrInvalidParameter) {
		t.Fatalf("negative MaxSide: %v", err)
	}

	limit, quality, err := s.resolvedEncode(Options{})
	if err != nil || limit != defaultSizeLimit || quality != defaultQuality {
		t.Fatalf("default encode: %d/%d/%v", limit, quality, err)
	}
	if _, _, err := s.resolvedEncode(Options{SizeLimit: -1}); !errors.Is(err, ErrInvalidParameter) {
		t.Fatalf("negative SizeLimit: %v", err)
	}
	if _, _, err := s.resolvedEncode(Options{Quality: 120}); !errors.Is(err, ErrInvalidParameter) {
		t.Fatalf("invalid Quality: %v", err)
	}
}

func TestAdaptiveMaxGuard(t *testing.T) {
	if err := adaptiveMaxGuard(100, 10_000_000); err != nil {
		t.Fatalf("guard rejected small input: %v", err)
	}
	err := adaptiveMaxGuard(20_000_000, 10_000_000)
	if !errors.Is(err, ErrSourceTooLarge) {
		t.Fatalf("error = %v, want ErrSourceTooLarge", err)
	}
	errContains(t, err, "Crop")
}

func TestCanceledContextAbortsOperations(t *testing.T) {
	s := newTestService(t)
	src := pngInput("scan1", solidPage(100, 100))
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	if _, err := s.Info(ctx, src); !errors.Is(err, context.Canceled) {
		t.Fatalf("Info error = %v, want context.Canceled", err)
	}
	if _, err := s.Crop(ctx, src, Region{X: 0, Y: 0, W: 0.5, H: 0.5}, Options{}); !errors.Is(err, context.Canceled) {
		t.Fatalf("Crop error = %v, want context.Canceled", err)
	}
	if _, err := s.View(ctx, src, 0); !errors.Is(err, context.Canceled) {
		t.Fatalf("View error = %v, want context.Canceled", err)
	}
}

func TestServiceIsStatelessAndSafeForReuse(t *testing.T) {
	s := newTestService(t)
	src := pngInput("scan1", solidPage(100, 100))
	ctx := context.Background()

	// Two identical calls must produce identical results — the SDK holds
	// no state between calls, results live entirely in the host.
	first, err := s.Grayscale(ctx, src)
	if err != nil {
		t.Fatalf("Grayscale 1: %v", err)
	}
	second, err := s.Grayscale(ctx, src)
	if err != nil {
		t.Fatalf("Grayscale 2: %v", err)
	}
	if first.ID != second.ID || first.Size != second.Size || !bytes.Equal(first.Data, second.Data) {
		t.Fatalf("repeated call diverged: %s/%d vs %s/%d", first.ID, first.Size, second.ID, second.Size)
	}
}
