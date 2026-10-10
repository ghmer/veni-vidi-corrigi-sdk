package geometry

import (
	"fmt"
	"math"
)

// FitLongest computes target dimensions for an image of w x h so that its
// longest edge lands between minEdge (small excerpts are upscaled for
// readability) and maxSide (large images are downscaled). Upscaling never
// exceeds maxUpscale; the resulting notes describe any resizing action.
// maxUpscale < 1 forbids upscaling; non-finite values are rejected, so the
// scale arithmetic can never poison the dimension rounding downstream.
func FitLongest(w, h, minEdge, maxSide int, maxUpscale float64) (tw, th int, scale float64, notes []string, err error) {
	if err := checkDims(w, h); err != nil {
		return 0, 0, 0, nil, err
	}
	if math.IsNaN(maxUpscale) || math.IsInf(maxUpscale, 0) {
		return 0, 0, 0, nil, fmt.Errorf("%w: maxUpscale %.3g must be a finite number", ErrInvalidParameter, maxUpscale)
	}
	longest := max(w, h)
	switch {
	case longest < minEdge:
		scale = min(float64(minEdge)/float64(longest), maxUpscale)
		if scale <= 1 {
			// maxUpscale smaller than 1 means the caller forbids upscaling.
			scale = 1
		} else if scale >= maxUpscale {
			notes = append(notes, fmt.Sprintf("upscale capped to %.1fx", maxUpscale))
		}
	case longest > maxSide:
		scale = float64(maxSide) / float64(longest)
	default:
		scale = 1
	}
	tw, th, notes = applyScale(w, h, scale, notes)
	return tw, th, scale, notes, nil
}

// FitDown computes target dimensions that keep the longest edge at or below
// maxSide without ever upscaling (view-image semantics). It returns notes
// "only" when the image actually shrinks; a "kept" note is the caller's job.
func FitDown(w, h, maxSide int) (tw, th int, scale float64, notes []string, err error) {
	if err := checkDims(w, h); err != nil {
		return 0, 0, 0, nil, err
	}
	if maxSide < 1 {
		return 0, 0, 0, nil, fmt.Errorf("%w: maxSide must be >= 1", ErrInvalidParameter)
	}
	scale = 1.0
	longest := max(w, h)
	if longest > maxSide {
		scale = float64(maxSide) / float64(longest)
	}
	tw, th, notes = applyScale(w, h, scale, notes)
	return tw, th, scale, notes, nil
}

// checkDims validates positive image dimensions.
func checkDims(w, h int) error {
	if w < 1 || h < 1 {
		return fmt.Errorf("%w: image dimensions %dx%d must be positive", ErrInvalidParameter, w, h)
	}
	return nil
}

// applyScale rounds the scaled dimensions, guards against zero-sized images
// and appends the resize note when the scale differs from 1.
func applyScale(w, h int, scale float64, notes []string) (int, int, []string) {
	if scale == 1 {
		return w, h, notes
	}
	tw := max(1, int(math.Round(float64(w)*scale)))
	th := max(1, int(math.Round(float64(h)*scale)))
	notes = append(notes, fmt.Sprintf("resized %dx%d -> %dx%d (factor %.3g)", w, h, tw, th, scale))
	return tw, th, notes
}
