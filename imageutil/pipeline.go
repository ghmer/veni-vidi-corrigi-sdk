package imageutil

import (
	"context"
	"fmt"
	"image"

	"github.com/disintegration/imaging"

	"github.com/ghmer/veni-vidi-corrigi-sdk/internal/enhance"
	"github.com/ghmer/veni-vidi-corrigi-sdk/internal/geometry"
	"github.com/ghmer/veni-vidi-corrigi-sdk/internal/pix"
)

// pipeline defaults: the EnhanceForReading chain is tuned for faint pencil
// strokes on uneven paper.
const (
	pipelineClipPercent = 0.4
	pipelineSigma       = 0.75
	pipelineAmount      = 1.0
	minPipelineTarget   = 300
)

// EnhanceForReading is the one-call rescue for hard-to-read excerpts:
// grayscale -> AutoContrast -> unsharp mask -> (optional Sauvola
// binarization) -> interpolation to targetMaxSide (0 = service default).
// One call instead of five, with a fixed order that protects quality.
func (s *Service) EnhanceForReading(ctx context.Context, src Input, targetMaxSide int, binarize bool) (Result, error) {
	if err := checkCtx(ctx); err != nil {
		return Result{}, err
	}
	srcData, err := s.load(ctx, src)
	if err != nil {
		return Result{}, err
	}
	target, notes, err := pipelineTarget(targetMaxSide)
	if err != nil {
		return Result{}, err
	}
	notes = append(notes, exifNotes(srcData)...)

	var img image.Image = imaging.Grayscale(srcData.img)
	notes = appendNote(notes, "grayscale")

	img = enhance.AutoContrast(img, pipelineClipPercent)
	notes = appendNote(notes, fmt.Sprintf("autocontrast clip %.1f%%", pipelineClipPercent))

	img, err = enhance.UnsharpMask(img, pipelineSigma, pipelineAmount)
	if err != nil {
		return Result{}, err
	}
	notes = appendNote(notes, fmt.Sprintf("unsharp sigma %.2f amount %.1f", pipelineSigma, pipelineAmount))

	if binarize {
		if err := adaptiveMaxGuard(srcData.w*srcData.h, s.set.binPixels); err != nil {
			return Result{}, err
		}
		gray, w, h := pix.GrayPlane(img)
		window := autoWindow(min(w, h))
		img, err = enhance.Sauvola(gray, w, h, window)
		if err != nil {
			return Result{}, err
		}
		notes = appendNote(notes, fmt.Sprintf("sauvola window %d (automatic)", window))
	}

	out := img
	tw, th, scale, fitNotes, err := geometry.FitLongest(srcData.w, srcData.h, target, target, s.set.maxUpscale)
	if err != nil {
		return Result{}, err
	}
	notes = append(notes, fitNotes...)
	if scale != 1 {
		out = imaging.Resize(img, tw, th, imaging.Lanczos)
	}
	return s.save(ctx, srcData, "enh", out, Options{}, notes)
}

// pipelineTarget normalizes the target edge with a lower bound, because
// tiny targets defeat the whole purpose.
func pipelineTarget(targetMaxSide int) (int, []string, error) {
	if targetMaxSide < 0 {
		return 0, nil, fmt.Errorf("%w: targetMaxSide %d must not be negative", ErrInvalidParameter, targetMaxSide)
	}
	if targetMaxSide == 0 {
		return defaultMaxSide, nil, nil
	}
	if targetMaxSide < minPipelineTarget {
		return 0, nil, fmt.Errorf("%w: targetMaxSide %d is below %d; reading needs resolution", ErrInvalidParameter, targetMaxSide, minPipelineTarget)
	}
	if targetMaxSide > hardMaxSide {
		return hardMaxSide, []string{fmt.Sprintf("targetMaxSide %d capped to %d", targetMaxSide, hardMaxSide)}, nil
	}
	return targetMaxSide, nil, nil
}
