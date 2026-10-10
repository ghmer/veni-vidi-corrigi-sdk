package imageutil

import (
	"fmt"
	"log/slog"
)

const (
	// defaultMaxSide is the output edge used when the caller passes 0 —
	// sized so results travel through vision APIs without downscaling.
	defaultMaxSide = 1600
	// hardMaxSide is the hard upper bound for output edges.
	hardMaxSide = 4000
	// hardMinSide guards against degenerate tiny outputs.
	hardMinSide = 200
	// defaultSizeLimit keeps results below typical vision API upload
	// limits (~4-5 MiB).
	defaultSizeLimit = 4 << 20
	// defaultQuality is the JPEG fallback quality.
	defaultQuality = 90
	// defaultUpscale caps how far small crops may be magnified; beyond
	// that no reading benefit exists and interpolation artifacts grow.
	defaultUpscale = 4.0
	// maxInputMegapixels rejects accidental whole-photo-stream inputs.
	maxInputMegapixels = 50
	// maxBinMegapixels bounds the window computation of adaptive
	// thresholding; binarization belongs to crops, not full pages.
	maxBinMegapixels = 12
	// defaultMaxInputBytes bounds every input payload before decoding, so
	// oversized or malformed data cannot exhaust memory even though the
	// bytes already sit in host memory.
	defaultMaxInputBytes = 128 << 20
)

// settings are the Service-level defaults; per-call Options may tighten the
// output side.
type settings struct {
	logger        *slog.Logger
	maxSide       int
	sizeLimit     int64
	quality       int
	maxUpscale    float64
	maxPixels     int64
	binPixels     int64
	maxInputBytes int64
}

// Option mutates the service settings before validation.
type Option func(*settings)

// WithLogger injects a structured logger (JSON logging is the default via
// slog.NewJSONHandler on stderr); nil restores the default.
func WithLogger(l *slog.Logger) Option {
	return func(s *settings) {
		if l != nil {
			s.logger = l
		}
	}
}

// WithMaxSide sets the default output edge for View/Crop/pipeline results.
func WithMaxSide(n int) Option {
	return func(s *settings) { s.maxSide = n }
}

// WithSizeLimit sets the output byte limit (PNG-to-JPEG fallback ladder).
func WithSizeLimit(n int64) Option {
	return func(s *settings) { s.sizeLimit = n }
}

// WithQuality sets the JPEG fallback quality.
func WithQuality(n int) Option {
	return func(s *settings) { s.quality = n }
}

// validate checks the settings after all options were applied.
func (s *settings) validate() error {
	if s.logger == nil {
		s.logger = slog.New(slog.NewJSONHandler(stderrWriter(), nil))
	}
	if s.maxSide < hardMinSide || s.maxSide > hardMaxSide {
		return fmt.Errorf("%w: maxSide %d must lie in [%d, %d]", ErrInvalidParameter, s.maxSide, hardMinSide, hardMaxSide)
	}
	if s.sizeLimit < 1024 {
		return fmt.Errorf("%w: sizeLimit %d must be at least 1024 bytes", ErrInvalidParameter, s.sizeLimit)
	}
	if s.quality < 1 || s.quality > 100 {
		return fmt.Errorf("%w: quality %d must lie in [1, 100]", ErrInvalidParameter, s.quality)
	}
	if s.maxUpscale < 1 || s.maxUpscale > 8 {
		return fmt.Errorf("%w: maxUpscale %.2f must lie in [1, 8]", ErrInvalidParameter, s.maxUpscale)
	}
	if s.maxPixels < 1 {
		return fmt.Errorf("%w: maxPixels %d must be positive", ErrInvalidParameter, s.maxPixels)
	}
	if s.binPixels < 1 {
		return fmt.Errorf("%w: binPixels %d must be positive", ErrInvalidParameter, s.binPixels)
	}
	if s.maxInputBytes < 1024 {
		return fmt.Errorf("%w: maxInputBytes %d must be at least 1024 bytes", ErrInvalidParameter, s.maxInputBytes)
	}
	return nil
}
