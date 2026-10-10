package geometry

import (
	"fmt"
	"image"
	"math"
)

// GridRects splits a w x h image into cols x rows tiles. Each tile is
// enlarged by the normalized overlap on both sides (except at the image
// borders), so content directly at the cut edge remains visible inside the
// neighbouring tile as well.
//
// With n tiles per axis and overlap o, the effective step between tiles is
// (1-o)/n and each tile spans (1-o)/n + o, which guarantees full coverage
// with exactly the requested overlap between neighbours.
func GridRects(w, h, cols, rows int, overlap float64) ([]image.Rectangle, error) {
	if w < 1 || h < 1 {
		return nil, fmt.Errorf("%w: image dimensions %dx%d must be positive", ErrInvalidParameter, w, h)
	}
	if cols < 1 || rows < 1 {
		return nil, fmt.Errorf("%w: grid %dx%d must have at least one tile per axis", ErrInvalidParameter, cols, rows)
	}
	if math.IsNaN(overlap) || overlap < 0 || overlap >= 1 {
		return nil, fmt.Errorf("%w: overlap %.3f must lie in [0, 1)", ErrInvalidParameter, overlap)
	}
	rects := make([]image.Rectangle, 0, cols*rows)
	for row := range rows {
		for col := range cols {
			x0, x1 := tileAxis(w, cols, overlap, col)
			y0, y1 := tileAxis(h, rows, overlap, row)
			rects = append(rects, image.Rect(x0, y0, x1, y1))
		}
	}
	return rects, nil
}

// tileAxis computes the [start, end) pixel range of tile index (0-based) on
// an axis of length size with n tiles.
func tileAxis(size, n int, overlap float64, index int) (int, int) {
	step := (1 - overlap) / float64(n)
	span := step + overlap
	x0 := int(math.Floor(step * float64(index) * float64(size)))
	x1 := int(math.Ceil((step*float64(index) + span) * float64(size)))
	return clampInt(x0, size), clampInt(x1, size)
}
