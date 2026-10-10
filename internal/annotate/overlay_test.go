package annotate

import (
	"image"
	"image/color"
	"testing"
)

func whitePage(w, h int) *image.NRGBA {
	img := image.NewNRGBA(image.Rect(0, 0, w, h))
	for i := 0; i < len(img.Pix); i += 4 {
		img.Pix[i], img.Pix[i+1], img.Pix[i+2], img.Pix[i+3] = 255, 255, 255, 255
	}
	return img
}

func isRed(c color.Color) bool {
	r, g, b, _ := c.RGBA()
	return int(r>>8) > 200 && int(g>>8) < 64 && int(b>>8) < 64
}

func isWhite(c color.Color) bool {
	r, g, b, _ := c.RGBA()
	return int(r>>8) > 200 && int(g>>8) > 200 && int(b>>8) > 200
}

func TestOverlayDrawsBorder(t *testing.T) {
	out := Overlay(whitePage(30, 20), image.Rect(5, 5, 20, 15), "")
	// Shortest side 10 -> thickness = max(2, 10/100=0) = 2: the border spans
	// x 5-6 at the left edge.
	if !isRed(out.At(5, 10)) || !isRed(out.At(6, 10)) {
		t.Fatalf("left border missing: %v", out.At(5, 10))
	}
	if isRed(out.At(7, 10)) {
		t.Fatal("border must be exactly two pixels wide here")
	}
	if isRed(out.At(10, 9)) {
		t.Fatal("interior must stay untouched")
	}
}

func TestOverlayDrawsLabelBox(t *testing.T) {
	out := Overlay(whitePage(40, 20), image.Rect(5, 5, 20, 15), "Hi")
	// Label box sits inside the rectangle's top-left corner.
	var red, white bool
	for y := 5; y < 22; y++ {
		for x := 5; x < 24; x++ {
			if isRed(out.At(x, y)) {
				red = true
			}
			if isWhite(out.At(x, y)) {
				white = true
			}
		}
	}
	if !red || !white {
		t.Fatalf("label box incomplete: red=%v white=%v", red, white)
	}
}

func TestOverlaySanitizesLabel(t *testing.T) {
	out := Overlay(whitePage(40, 20), image.Rect(2, 2, 18, 12), "A\x00\tBüö")
	// Only 'A' and 'B' survive; the draw call must not panic and the box
	// must exist.
	var red bool
	for y := 2; y < 19; y++ {
		for x := 2; x < 16; x++ {
			if isRed(out.At(x, y)) {
				red = true
			}
		}
	}
	if !red {
		t.Fatal("label box missing after sanitization")
	}
}

func TestOverlayIgnoresEmptyRect(t *testing.T) {
	src := whitePage(10, 10)
	out := Overlay(src, image.Rect(50, 50, 60, 60), "")
	for i := range src.Pix {
		if out.Pix[i] != src.Pix[i] {
			t.Fatalf("pixel %d changed on empty rect", i)
		}
	}
}

func TestBorderThicknessClamps(t *testing.T) {
	tests := []struct {
		rect      image.Rectangle
		wantRange [2]int
	}{
		{image.Rect(0, 0, 10, 10), [2]int{2, 6}},
		{image.Rect(0, 0, 1000, 1000), [2]int{2, 6}}, // 1000/100=10 -> capped
	}
	for _, tt := range tests {
		got := borderThickness(tt.rect)
		if got < tt.wantRange[0] || got > tt.wantRange[1] {
			t.Fatalf("borderThickness(%s) = %d, outside %v", tt.rect, got, tt.wantRange)
		}
	}
}
