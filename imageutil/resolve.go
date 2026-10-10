package imageutil

import (
	"context"
	"fmt"
)

// checkCtx validates the context before use: nil is rejected as an invalid
// parameter instead of panicking, and a canceled or expired context is
// surfaced wrapped so callers can distinguish cancellation from operation
// failures with errors.Is.
func checkCtx(ctx context.Context) error {
	if ctx == nil {
		return fmt.Errorf("%w: context must not be nil", ErrInvalidParameter)
	}
	if err := ctx.Err(); err != nil {
		return fmt.Errorf("context: %w", err)
	}
	return nil
}

// resolvedMaxSide normalizes the per-call MaxSide: 0 selects the service
// default, values above the hard cap are reduced with a note, negatives are
// rejected.
func (s *Service) resolvedMaxSide(opts Options) (int, []string, error) {
	if opts.MaxSide == 0 {
		return s.set.maxSide, nil, nil
	}
	if opts.MaxSide < 0 {
		return 0, nil, fmt.Errorf("%w: MaxSide %d must not be negative", ErrInvalidParameter, opts.MaxSide)
	}
	if opts.MaxSide > hardMaxSide {
		return hardMaxSide, []string{fmt.Sprintf("MaxSide %d capped to %d", opts.MaxSide, hardMaxSide)}, nil
	}
	return opts.MaxSide, nil, nil
}

// resolvedEncode normalizes SizeLimit and Quality for the encoder.
func (s *Service) resolvedEncode(opts Options) (int64, int, error) {
	limit := opts.SizeLimit
	if limit == 0 {
		limit = s.set.sizeLimit
	} else if limit < 0 {
		return 0, 0, fmt.Errorf("%w: SizeLimit %d must not be negative", ErrInvalidParameter, opts.SizeLimit)
	}
	quality := opts.Quality
	if quality == 0 {
		quality = s.set.quality
	} else if quality < 1 || quality > 100 {
		return 0, 0, fmt.Errorf("%w: Quality %d must lie in [1, 100] or be 0", ErrInvalidParameter, quality)
	}
	return limit, quality, nil
}

// autoWindow picks an odd Sauvola window of roughly 3% of the shorter edge
// (bounded to [15, 121]), matching the guidance in IMAGE_UTILITIES.md.
func autoWindow(shortEdge int) int {
	window := shortEdge * 3 / 100
	window = max(window, 15)
	window = min(window, 121)
	if window%2 == 0 {
		window++
	}
	return window
}

// adaptiveMaxGuard rejects adaptive thresholding on inputs that are too
// large for region-based processing.
func adaptiveMaxGuard(pixels int, budget int64) error {
	if int64(pixels) > budget {
		return fmt.Errorf("%w: adaptive thresholding over %d pixels exceeds the budget of %d; binarize a Crop instead",
			ErrSourceTooLarge, pixels, budget)
	}
	return nil
}
