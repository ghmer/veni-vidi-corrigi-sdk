package imageutil

// Inspection domain: reading input metadata and producing the model-sized
// view image an LLM orients itself with. Discovering available sources is
// the host's job — the SDK is stateless and sees only the inputs it is
// handed.

import (
	"context"
	"fmt"

	"github.com/ghmer/veni-vidi-corrigi-sdk/internal/codec"
	"github.com/ghmer/veni-vidi-corrigi-sdk/internal/geometry"
)

// Info reports the basic data of an input without producing a new image:
// pixel dimensions, detected format, EXIF orientation and usage hints for
// the next useful call. Detection happens against the content bytes.
func (s *Service) Info(ctx context.Context, src Input) (Result, error) {
	if err := checkCtx(ctx); err != nil {
		return Result{}, err
	}
	data, err := s.inputBytes(src)
	if err != nil {
		return Result{}, err
	}
	if err := checkCtx(ctx); err != nil {
		return Result{}, err
	}
	width, height, format, err := codec.DecodeConfig(data)
	if err != nil {
		return Result{}, err
	}
	notes := []string{}
	if note := codec.DescribeOrientation(codec.OrientationOf(data)); note != "" {
		notes = append(notes, note)
	}
	if max(width, height) > s.set.maxSide {
		notes = appendNote(notes, fmt.Sprintf(
			"longest edge %d exceeds MaxSide %d: use View for the page overview or Crop for excerpts",
			max(width, height), s.set.maxSide))
	}
	id := seedID(src.Name)
	s.log.InfoContext(ctx, "source inspected",
		"src", id, "format", format, "width", width, "height", height)
	return Result{
		ID:     id,
		Width:  width,
		Height: height,
		Format: format,
		Size:   int64(len(data)),
		Notes:  notes,
	}, nil
}

// viewTarget normalizes the effective View edge.
func viewTarget(maxSide int) (int, []string, error) {
	if maxSide < 0 {
		return 0, nil, fmt.Errorf("%w: maxSide %d must not be negative", ErrInvalidParameter, maxSide)
	}
	if maxSide == 0 {
		return defaultMaxSide, nil, nil
	}
	if maxSide > hardMaxSide {
		return hardMaxSide, []string{fmt.Sprintf("maxSide %d capped to %d", maxSide, hardMaxSide)}, nil
	}
	return maxSide, nil, nil
}

// View produces the model-sized reference image of a source: the longest
// edge is reduced to maxSide (0 = service default, 1600 px). This is the
// image an LLM should look at first; Crop regions use normalized
// coordinates 0..1 against the FULL original, independent of this view.
func (s *Service) View(ctx context.Context, src Input, maxSide int) (Result, error) {
	if err := checkCtx(ctx); err != nil {
		return Result{}, err
	}
	srcData, err := s.load(ctx, src)
	if err != nil {
		return Result{}, err
	}
	target, notes, err := viewTarget(maxSide)
	if err != nil {
		return Result{}, err
	}
	tw, th, scale, fitNotes, err := geometry.FitDown(srcData.w, srcData.h, target)
	if err != nil {
		return Result{}, err
	}
	notes = append(notes, fitNotes...)
	out := srcData.img
	if scale != 1 {
		out, err = s.boundedResize(srcData.img, tw, th)
		if err != nil {
			return Result{}, err
		}
	} else {
		notes = appendNote(notes, fmt.Sprintf("longest edge %d already within %d", max(srcData.w, srcData.h), target))
	}
	return s.save(ctx, srcData, "view", out, Options{}, notes)
}
