package imageutil

// Geometry domain: cutting regions out of the original at full sharpness,
// scaling, lossless rotation and dividing pages.

import (
	"context"
	"fmt"
	"image"
	"math"

	"github.com/disintegration/imaging"

	"github.com/ghmer/veni-vidi-corrigi-sdk/internal/codec"
	"github.com/ghmer/veni-vidi-corrigi-sdk/internal/enhance"
	"github.com/ghmer/veni-vidi-corrigi-sdk/internal/geometry"
)

// minCropEdge is the target longest edge below which Crop upscales a region
// for readability (capped at the service's maxUpscale).
const minCropEdge = 800

// defaultSplitOverlap keeps content at cut edges visible inside the
// neighbouring tile as well.
const defaultSplitOverlap = 0.08

// maxSplitAxis caps tiles per axis to keep the tile count sane.
const maxSplitAxis = 6

// trimTolerance is the max channel difference that still counts as page
// background when searching for white margins.
const trimTolerance = 12

// trimMinArea keeps over-eager trimming from eating real content: if less
// than this fraction of the page would remain, the page is kept as-is.
const trimMinArea = 0.04

// Crop cuts a normalized region out of the ORIGINAL image at full sharpness
// and scales the excerpt so its longest edge lands between minCropEdge
// (800, upscale for readability) and the effective MaxSide. This is the
// central zoom operation that makes handwriting readable.
func (s *Service) Crop(ctx context.Context, src Input, region Region, opts Options) (Result, error) {
	if err := checkCtx(ctx); err != nil {
		return Result{}, err
	}
	srcData, err := s.load(ctx, src)
	if err != nil {
		return Result{}, err
	}
	maxSide, optsNotes, err := s.resolvedMaxSide(opts)
	if err != nil {
		return Result{}, err
	}
	notes := append([]string{}, optsNotes...)
	if note := codec.DescribeOrientation(srcData.orientation); note != "" {
		notes = appendNote(notes, note)
	}
	clamped, changed, err := geometry.Clamp(region)
	if err != nil {
		return Result{}, err
	}
	if changed {
		notes = appendNote(notes, "region clamped: "+clamped.String())
	}
	rect := clamped.PixelRect(srcData.w, srcData.h)
	cropped := imaging.Crop(srcData.img, rect)
	tw, th, scale, fitNotes, err := geometry.FitLongest(rect.Dx(), rect.Dy(), minCropEdge, maxSide, s.set.maxUpscale)
	if err != nil {
		return Result{}, err
	}
	notes = append(notes, fitNotes...)
	var out image.Image = cropped
	if scale != 1 {
		out, err = s.boundedResize(cropped, tw, th)
		if err != nil {
			return Result{}, err
		}
	}
	return s.save(ctx, srcData, "crop", out, opts, notes)
}

// Scale resizes the whole source by the given factor (0.5 up to 4x). For
// page excerpts Crop is usually the better tool; Scale exists for manual
// touch-ups of already extracted excerpts. Outputs that would exceed the
// pixel budget are rejected before any buffer is allocated.
func (s *Service) Scale(ctx context.Context, src Input, factor float64) (Result, error) {
	if err := checkCtx(ctx); err != nil {
		return Result{}, err
	}
	if math.IsNaN(factor) || math.IsInf(factor, 0) || factor < 0.5 || factor > s.set.maxUpscale {
		return Result{}, fmt.Errorf("%w: factor %.3g must lie in [0.5, %.1f]", ErrInvalidParameter, factor, s.set.maxUpscale)
	}
	srcData, err := s.load(ctx, src)
	if err != nil {
		return Result{}, err
	}
	tw := max(1, int(math.Round(float64(srcData.w)*factor)))
	th := max(1, int(math.Round(float64(srcData.h)*factor)))
	out, err := s.boundedResize(srcData.img, tw, th)
	if err != nil {
		return Result{}, err
	}
	notes := []string{fmt.Sprintf("scaled %dx%d -> %dx%d (factor %.3g)", srcData.w, srcData.h, tw, th, factor)}
	return s.save(ctx, srcData, "scale", out, Options{}, notes)
}

// Rotate turns the source by exactly 90, 180 or 270 degrees using exact
// pixel permutations (lossless, no interpolation). Pages scanned sideways
// must be fixed before any reading attempt.
func (s *Service) Rotate(ctx context.Context, src Input, deg int) (Result, error) {
	if err := checkCtx(ctx); err != nil {
		return Result{}, err
	}
	srcData, err := s.load(ctx, src)
	if err != nil {
		return Result{}, err
	}
	var out image.Image
	var op string
	switch deg {
	case 90:
		out = geometry.Rotate90(srcData.img)
		op = "rot90"
	case 180:
		out = geometry.Rotate180(srcData.img)
		op = "rot180"
	case 270:
		out = geometry.Rotate270(srcData.img)
		op = "rot270"
	default:
		return Result{}, fmt.Errorf(
			"%w: rotation %d must be 90, 180 or 270 (free angles are not supported; scanned pages rarely need them)",
			ErrInvalidParameter, deg)
	}
	notes := []string{}
	if note := codec.DescribeOrientation(srcData.orientation); note != "" {
		notes = appendNote(notes, note)
	}
	return s.save(ctx, srcData, op, out, Options{}, notes)
}

// Split divides a page according to the mode: top-bottom, left-right, the
// grid2x2 quadrant split, or an arbitrary tiles (cols x rows) grid. Tiles
// overlap by default (8%) so handwriting at a cut edge stays readable in
// the neighbouring tile. Splitting helps with long pages and densely
// written double columns.
func (s *Service) Split(ctx context.Context, src Input, mode SplitMode, cols, rows int, overlap float64) ([]Result, error) {
	if err := checkCtx(ctx); err != nil {
		return nil, err
	}
	srcData, err := s.load(ctx, src)
	if err != nil {
		return nil, err
	}
	gridCols, gridRows, err := splitGrid(mode, cols, rows)
	if err != nil {
		return nil, err
	}
	overlap, notes, err := splitOverlap(overlap, nil)
	if err != nil {
		return nil, err
	}
	maxSide, optsNotes, err := s.resolvedMaxSide(Options{})
	if err != nil {
		return nil, err
	}
	notes = append(notes, optsNotes...)
	if note := codec.DescribeOrientation(srcData.orientation); note != "" {
		notes = appendNote(notes, note)
	}
	rects, err := geometry.GridRects(srcData.w, srcData.h, gridCols, gridRows, overlap)
	if err != nil {
		return nil, err
	}
	results := make([]Result, 0, len(rects))
	for _, rect := range rects {
		if err := checkCtx(ctx); err != nil {
			return nil, err
		}
		tile := imaging.Crop(srcData.img, rect)
		tw, th, scale, fitNotes, err := geometry.FitDown(tile.Bounds().Dx(), tile.Bounds().Dy(), maxSide)
		if err != nil {
			return nil, err
		}
		tileNotes := appendClone(notes, fitNotes...)
		var out image.Image = tile
		if scale != 1 {
			out, err = s.boundedResize(tile, tw, th)
			if err != nil {
				return nil, err
			}
		}
		result, err := s.saveIndexed(ctx, srcData, "split", len(results)+1, out, Options{}, tileNotes)
		if err != nil {
			return nil, err
		}
		results = append(results, result)
	}
	return results, nil
}

// splitGrid maps the mode to a concrete grid and validates tile counts.
func splitGrid(mode SplitMode, cols, rows int) (int, int, error) {
	switch mode {
	case SplitTopBottom:
		return 1, 2, nil
	case SplitLeftRight:
		return 2, 1, nil
	case SplitGrid2x2:
		return 2, 2, nil
	case SplitTiles:
		if cols < 1 || cols > maxSplitAxis || rows < 1 || rows > maxSplitAxis || cols*rows > 12 {
			return 0, 0, fmt.Errorf(
				"%w: tiles grid %dx%d needs cols and rows in [1, %d] with at most 12 tiles",
				ErrInvalidParameter, cols, rows, maxSplitAxis)
		}
		return cols, rows, nil
	default:
		return 0, 0, fmt.Errorf("%w: split mode %q must be one of %s", ErrInvalidParameter, mode,
			[]string{string(SplitTopBottom), string(SplitLeftRight), string(SplitGrid2x2), string(SplitTiles)})
	}
}

// splitOverlap validates the overlap and applies the default for 0.
func splitOverlap(overlap float64, notes []string) (float64, []string, error) {
	if math.IsNaN(overlap) || overlap < 0 {
		return 0, nil, fmt.Errorf("%w: overlap %.3f must lie in [0, 0.25]", ErrInvalidParameter, overlap)
	}
	if overlap == 0 {
		return defaultSplitOverlap, appendNote(notes, fmt.Sprintf("using default overlap %.2f", defaultSplitOverlap)), nil
	}
	if overlap > 0.25 {
		return 0, nil, fmt.Errorf("%w: overlap %.3f must lie in [0, 0.25]", ErrInvalidParameter, overlap)
	}
	return overlap, notes, nil
}

// TrimBorders removes uniform white margins or black scan frames around the
// actual page content (photos of pages on a desk typically need this).
// When no uniform border is detected or trimming would erase nearly
// everything, the operation notes it and returns an unchanged copy.
func (s *Service) TrimBorders(ctx context.Context, src Input) (Result, error) {
	if err := checkCtx(ctx); err != nil {
		return Result{}, err
	}
	srcData, err := s.load(ctx, src)
	if err != nil {
		return Result{}, err
	}
	notes := exifNotes(srcData)
	top, right, bottom, left := enhance.BorderMargins(srcData.img, trimTolerance)
	rect := image.Rect(left, top, srcData.w-right, srcData.h-bottom)
	area := float64(rect.Dx()*rect.Dy()) / float64(srcData.w*srcData.h)
	switch {
	case top == 0 && right == 0 && bottom == 0 && left == 0:
		notes = appendNote(notes, "no uniform border detected")
		rect = srcData.img.Bounds()
	case area < trimMinArea:
		notes = appendNote(notes, fmt.Sprintf("trimming would remove %.0f%% of the page, kept as-is", (1-area)*100))
		rect = srcData.img.Bounds()
	default:
		notes = appendNote(notes, fmt.Sprintf("trimmed margins: top %d right %d bottom %d left %d", top, right, bottom, left))
	}
	out := imaging.Crop(srcData.img, rect)
	return s.save(ctx, srcData, "trim", out, Options{}, notes)
}

// appendClone copies the base notes before per-result additions.
func appendClone(base []string, extra ...string) []string {
	out := append([]string{}, base...)
	return append(out, extra...)
}
