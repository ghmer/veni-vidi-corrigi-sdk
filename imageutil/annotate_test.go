package imageutil

import (
	"context"
	"errors"
	"image/color"
	"testing"
)

func TestAnnotateRectDrawsBorderAndLabel(t *testing.T) {
	s := newTestService(t)
	src := pngInput("scan1", solidPage(800, 600))

	res, err := s.AnnotateRect(context.Background(), src, Region{X: 0.25, Y: 0.25, W: 0.5, H: 0.5}, "Task 2")
	if err != nil {
		t.Fatalf("AnnotateRect: %v", err)
	}
	// Page 800x600 stays within MaxSide 1600, so the rectangle sits at
	// pixels (200,150)-(600,450). The left border must be red.
	img := decodeResult(t, res)
	if !isRedish(img.At(201, 300)) {
		t.Fatalf("border pixel = %v, want red", img.At(201, 300))
	}
	if !isRedish(img.At(400, 152)) {
		t.Fatalf("top border pixel = %v, want red", img.At(400, 152))
	}

	// The label box (top-left corner) must contain red background and white
	// glyph pixels.
	var red, white bool
	for y := 150; y < 168; y++ {
		for x := 200; x < 246; x++ {
			c := img.At(x, y)
			red = red || isRedish(c)
			white = white || isWhiteish(c)
		}
	}
	if !red || !white {
		t.Fatalf("label box incomplete: red=%v white=%v", red, white)
	}
	if !containsNote(res, "annotation rectangle") {
		t.Fatalf("notes = %v", res.Notes)
	}
}

func TestAnnotateRectClampsRegion(t *testing.T) {
	s := newTestService(t)
	src := pngInput("scan1", solidPage(400, 300))

	res, err := s.AnnotateRect(context.Background(), src, Region{X: 0.9, Y: 0.1, W: 0.5, H: 0.5}, "")
	if err != nil {
		t.Fatalf("AnnotateRect: %v", err)
	}
	if !containsNote(res, "region clamped") {
		t.Fatalf("notes = %v", res.Notes)
	}
}

func TestAnnotateRectRejectsEmptyRegion(t *testing.T) {
	s := newTestService(t)
	src := pngInput("scan1", solidPage(100, 100))

	_, err := s.AnnotateRect(context.Background(), src, Region{X: 1, Y: 0, W: 0.2, H: 0.2}, "")
	if !errors.Is(err, ErrInvalidRegion) {
		t.Fatalf("error = %v, want ErrInvalidRegion", err)
	}
}

func isRedish(c color.Color) bool {
	r, g, b, _ := c.RGBA()
	return int(r>>8) > 200 && int(g>>8) < 64 && int(b>>8) < 64
}

func isWhiteish(c color.Color) bool {
	r, g, b, _ := c.RGBA()
	return int(r>>8) > 200 && int(g>>8) > 200 && int(b>>8) > 200
}
