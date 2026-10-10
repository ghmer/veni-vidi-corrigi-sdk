package imageutil

import (
	"context"
	"log/slog"
	"os"
)

// Service implements the image utility operations. Every method receives
// its source image as bytes (Input with raw Data or Base64) and returns a
// NEW transformed image inside Result.Data — nothing is ever read from or
// written to the file system. Originals are never modified and the host
// owns all storage decisions.
//
// Concurrency: a Service holds no per-operation state, so every method is
// safe for concurrent use; operations on the same input are independent
// because inputs are passed by value.
type Service struct {
	set settings
	log *slog.Logger
}

// WithMaxInputPixels overrides the decode budget (width*height); useful for
// tests and stricter deployments.
func WithMaxInputPixels(n int64) Option {
	return func(s *settings) { s.maxPixels = n }
}

// WithMaxBinPixels overrides the pixel budget for adaptive thresholding.
func WithMaxBinPixels(n int64) Option {
	return func(s *settings) { s.binPixels = n }
}

// WithMaxInputBytes overrides the byte budget applied to every input
// payload before decoding.
func WithMaxInputBytes(n int64) Option {
	return func(s *settings) { s.maxInputBytes = n }
}

// New applies the functional options, validates the settings and returns
// the in-memory service.
func New(opts ...Option) (*Service, error) {
	set := settings{
		logger:        nil,
		maxSide:       defaultMaxSide,
		sizeLimit:     defaultSizeLimit,
		quality:       defaultQuality,
		maxUpscale:    defaultUpscale,
		maxPixels:     maxInputMegapixels * 1_000_000,
		binPixels:     maxBinMegapixels * 1_000_000,
		maxInputBytes: defaultMaxInputBytes,
	}
	for _, opt := range opts {
		opt(&set)
	}
	if err := set.validate(); err != nil {
		return nil, err
	}
	s := &Service{set: set, log: set.logger}
	s.log.InfoContext(context.Background(), "imageutil service started",
		"maxSide", set.maxSide, "sizeLimit", set.sizeLimit, "quality", set.quality,
		"maxInputBytes", set.maxInputBytes)
	return s, nil
}

// stderrWriter is the default log destination, kept as a function so tests
// could redirect it without touching the Service.
func stderrWriter() *os.File { return os.Stderr }
