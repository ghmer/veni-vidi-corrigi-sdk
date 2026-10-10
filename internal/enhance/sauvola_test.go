package enhance

import (
	"testing"
)

// shadowPage builds an A4-ish test scene: the background brightness falls
// with the row (lighting gradient from shadows), and two full-width stroke
// rows simulate handwriting.
func shadowPage(w, h int) []uint8 {
	gray := make([]uint8, w*h)
	for y := range h {
		bg := uint8(60 + y) // 60 (dark shadow) .. 60+h-1
		for x := range w {
			gray[y*w+x] = bg
		}
	}
	for _, y := range []int{h / 2, h/2 + 1} {
		for x := range w {
			gray[y*w+x] = 20 // ink
		}
	}
	return gray
}

func TestSauvolaKeepsInkAcrossShadowGradient(t *testing.T) {
	const w, h = 40, 100
	page := shadowPage(w, h)
	out, err := Sauvola(page, w, h, 31)
	if err != nil {
		t.Fatalf("Sauvola: %v", err)
	}
	// Ink must survive in both the bright and the dark half.
	for _, p := range [][2]int{{10, h / 2}, {w - 5, h/2 + 1}, {5, h / 2}, {5, h/2 + 1}} {
		if got := out.Pix[p[1]*w+p[0]]; got != 0 {
			t.Fatalf("ink at (%d,%d) = %d, want 0", p[0], p[1], got)
		}
	}
	// Background must stay paper in both halves.
	for _, y := range []int{5, h - 5} {
		for _, x := range []int{3, w - 3} {
			if got := out.Pix[y*w+x]; got != 255 {
				t.Fatalf("background at (%d,%d) = %d, want 255", x, y, got)
			}
		}
	}
}

func TestSauvolaErrorsOnBadWindow(t *testing.T) {
	for _, win := range []int{1, 2, 4, 0, -3} {
		if _, err := Sauvola(make([]uint8, 400), 20, 20, win); err == nil {
			t.Fatalf("Sauvola accepted window %d", win)
		}
	}
	if _, err := Sauvola(nil, 0, 0, 31); err == nil {
		t.Fatal("Sauvola accepted empty plane")
	}
}
