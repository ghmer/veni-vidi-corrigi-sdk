// Package geometry contains coordinate and transformation math for raster
// images: normalized regions, fit/scale arithmetic, lossless rotation and
// flip permutations, and split grids. All functions are pure and free of I/O.
package geometry

import (
	"errors"
	"fmt"
	"image"
	"math"
)

// ErrInvalidRegion is returned for regions that cannot describe a non-empty
// rectangle on an image.
var ErrInvalidRegion = errors.New("invalid region")

// ErrInvalidParameter is returned when numeric arguments are out of range.
var ErrInvalidParameter = errors.New("invalid parameter")

// Region describes a rectangle in normalized coordinates (0.0-1.0) relative
// to the full source image. X/Y is the upper-left corner, W/H the relative
// width and height.
type Region struct {
	X, Y float64
	W, H float64
}

// Clamp sanitizes r for use on an image of any size. Non-finite values and
// empty regions are rejected; everything else is clamped into the unit
// square. The second return value reports whether clamping changed the
// region, so callers can surface it as a warning note.
func Clamp(r Region) (Region, bool, error) {
	if math.IsNaN(r.X) || math.IsInf(r.X, 0) ||
		math.IsNaN(r.Y) || math.IsInf(r.Y, 0) ||
		math.IsNaN(r.W) || math.IsInf(r.W, 0) ||
		math.IsNaN(r.H) || math.IsInf(r.H, 0) {
		return Region{}, false, fmt.Errorf("%w: non-finite values in %s", ErrInvalidRegion, r)
	}
	if r.W <= 0 || r.H <= 0 {
		return Region{}, false, fmt.Errorf("%w: width/height must be positive: %s", ErrInvalidRegion, r)
	}
	out := Region{
		X: math.Min(math.Max(r.X, 0), 1),
		Y: math.Min(math.Max(r.Y, 0), 1),
	}
	out.W = math.Min(math.Max(r.W, 0), 1-out.X)
	out.H = math.Min(math.Max(r.H, 0), 1-out.Y)
	// Clamping both corners can still yield an empty region (e.g. X=1).
	if out.W <= 0 || out.H <= 0 {
		return Region{}, false, fmt.Errorf("%w: outside the image after clamping: %s", ErrInvalidRegion, r)
	}
	changed := out.X != r.X || out.Y != r.Y || out.W != r.W || out.H != r.H
	return out, changed, nil
}

// PixelRect returns the pixel rectangle the region covers on a w x h image.
// x0/y0 use the floor and x1/y1 the ceiling of the exact float coordinates,
// so the rectangle is a superset of the exact float area.
func (r Region) PixelRect(w, h int) image.Rectangle {
	x0 := clampInt(int(math.Floor(r.X*float64(w))), w)
	y0 := clampInt(int(math.Floor(r.Y*float64(h))), h)
	x1 := clampInt(int(math.Ceil((r.X+r.W)*float64(w))), w)
	y1 := clampInt(int(math.Ceil((r.Y+r.H)*float64(h))), h)
	if x1 <= x0 {
		x1 = clampInt(x0+1, w)
	}
	if y1 <= y0 {
		y1 = clampInt(y0+1, h)
	}
	return image.Rect(x0, y0, x1, y1)
}

// String renders the region with three decimals for notes and error texts.
func (r Region) String() string {
	return fmt.Sprintf("X=%.3f Y=%.3f W=%.3f H=%.3f", r.X, r.Y, r.W, r.H)
}

// clampVal keeps v inside [0, hi]; every caller clamps against zero here.
func clampInt(v, hi int) int {
	return min(max(v, 0), hi)
}
