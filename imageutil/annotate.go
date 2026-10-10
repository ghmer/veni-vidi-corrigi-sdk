package imageutil

import (
	"context"
	"fmt"

	"github.com/disintegration/imaging"

	"github.com/ghmer/veni-vidi-corrigi-sdk/internal/annotate"
	"github.com/ghmer/veni-vidi-corrigi-sdk/internal/codec"
	"github.com/ghmer/veni-vidi-corrigi-sdk/internal/geometry"
)

// AnnotateRect draws a visible rectangle (plus optional label) around the
// region on a model-sized copy of the source (longest edge = service
// MaxSide). LLMs use it to verify their estimated regions against the page
// context, which sharply reduces misplaced crops. The region is normalized
// 0..1 like everywhere else.
func (s *Service) AnnotateRect(ctx context.Context, src Input, region Region, label string) (Result, error) {
	if err := checkCtx(ctx); err != nil {
		return Result{}, err
	}
	srcData, err := s.load(ctx, src)
	if err != nil {
		return Result{}, err
	}
	clamped, changed, err := geometry.Clamp(region)
	if err != nil {
		return Result{}, err
	}
	notes := []string{}
	if changed {
		notes = appendNote(notes, "region clamped: "+clamped.String())
	}
	if note := codec.DescribeOrientation(srcData.orientation); note != "" {
		notes = appendNote(notes, note)
	}
	maxSide, _, err := s.resolvedMaxSide(Options{})
	if err != nil {
		return Result{}, err
	}
	tw, th, scale, fitNotes, err := geometry.FitDown(srcData.w, srcData.h, maxSide)
	if err != nil {
		return Result{}, err
	}
	notes = append(notes, fitNotes...)
	base := srcData.img
	if scale != 1 {
		base = imaging.Resize(srcData.img, tw, th, imaging.Lanczos)
	}
	rect := clamped.PixelRect(tw, th)
	notes = appendNote(notes, fmt.Sprintf("annotation rectangle %s", rect))
	out := annotate.Overlay(base, rect, label)
	return s.save(ctx, srcData, "anno", out, Options{}, notes)
}
