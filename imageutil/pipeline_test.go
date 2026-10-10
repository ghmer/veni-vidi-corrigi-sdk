package imageutil

import (
	"context"
	"errors"
	"image"
	"testing"
)

// faintPencil builds a page with 5 light-gray "pencil" rows on bright paper
// — the classic hard case for EnhanceForReading.
func faintPencil() *image.NRGBA {
	page := solidPage(200, 150)
	dx := page.Bounds().Dx()
	for y := 60; y < 65; y++ {
		for x := range dx {
			i := (y*dx + x) * 4
			page.Pix[i], page.Pix[i+1], page.Pix[i+2] = 200, 200, 200
		}
	}
	return page
}

func TestEnhanceForReadingBinarizesFaintInk(t *testing.T) {
	s := newTestService(t)
	src := pngInput("scan1", faintPencil())

	res, err := s.EnhanceForReading(context.Background(), src, 600, true)
	if err != nil {
		t.Fatalf("EnhanceForReading: %v", err)
	}
	// 200x150 with target edge 600 -> factor 3 -> 600x450.
	if res.Width != 600 || res.Height != 450 {
		t.Fatalf("dims = %dx%d, want 600x450", res.Width, res.Height)
	}
	for _, want := range []string{"grayscale", "autocontrast", "unsharp", "sauvola window", "resized 200x150 -> 600x450"} {
		if !containsNote(res, want) {
			t.Fatalf("notes = %v, missing %q", res.Notes, want)
		}
	}
	img := decodeResult(t, res)
	// Sample well inside the stroke (scaled rows 180..195) and the paper.
	// The Lanczos upscale of the binary result introduces tiny ringing at
	// stroke edges, so the interior check allows a small dark band.
	if lum := lumAt(img, 300, 186); lum > 32 {
		t.Fatalf("pencil stroke = %d, want <= 32", lum)
	}
	if lumAt(img, 300, 30) != 255 {
		t.Fatalf("paper = %d, want 255", lumAt(img, 300, 30))
	}
}

func TestEnhanceForReadingWithoutBinarization(t *testing.T) {
	s := newTestService(t)
	src := pngInput("scan1", faintPencil())

	res, err := s.EnhanceForReading(context.Background(), src, 600, false)
	if err != nil {
		t.Fatalf("EnhanceForReading: %v", err)
	}
	if containsNote(res, "sauvola") {
		t.Fatalf("notes = %v, want no binarization step", res.Notes)
	}
}

func TestEnhanceForReadingValidatesTarget(t *testing.T) {
	s := newTestService(t)
	src := pngInput("scan1", faintPencil())

	if _, err := s.EnhanceForReading(context.Background(), src, 100, true); !errors.Is(err, ErrInvalidParameter) {
		t.Fatalf("tiny target: %v", err)
	}
	if _, err := s.EnhanceForReading(context.Background(), src, -1, true); !errors.Is(err, ErrInvalidParameter) {
		t.Fatalf("negative target: %v", err)
	}
	res, err := s.EnhanceForReading(context.Background(), src, 9000, false)
	if err != nil {
		t.Fatalf("capped target: %v", err)
	}
	if !containsNote(res, "capped") {
		t.Fatalf("notes = %v, want cap note", res.Notes)
	}
}
