package imageutil

import (
	"context"
	"errors"
	"math"
	"testing"
)

func TestGrayscaleThenInvertRoundtrip(t *testing.T) {
	s := newTestService(t)
	red := pngInput("red", solidPageRGB(100, 100, 255, 0, 0))
	black := pngInput("black", solidPageRGB(50, 50, 0, 0, 0))

	gray, err := s.Grayscale(context.Background(), red)
	if err != nil {
		t.Fatalf("Grayscale: %v", err)
	}
	img := decodeResult(t, gray)
	r, g, b, _ := img.At(50, 50).RGBA()
	if int(r>>8) != int(g>>8) || int(g>>8) != int(b>>8) {
		t.Fatalf("grayscale pixel %d/%d/%d not neutral", r>>8, g>>8, b>>8)
	}

	inv, err := s.Invert(context.Background(), black)
	if err != nil {
		t.Fatalf("Invert: %v", err)
	}
	invImg := decodeResult(t, inv)
	r, _, _, _ = invImg.At(25, 25).RGBA()
	if int(r>>8) != 255 {
		t.Fatalf("inverted black = %d, want 255", r>>8)
	}
}

func TestEnhanceValidationErrors(t *testing.T) {
	s := newTestService(t)
	src := pngInput("page", solidPage(100, 100))
	ctx := context.Background()

	bad := []struct {
		name string
		run  func() error
	}{
		{"autoc contrast clip 6", func() error {
			_, err := s.AutoContrast(ctx, src, 6)
			return err
		}},
		{"contrast 150", func() error {
			_, err := s.AdjustContrast(ctx, src, 150)
			return err
		}},
		{"brightness -150", func() error {
			_, err := s.AdjustBrightness(ctx, src, -150)
			return err
		}},
		{"gamma 0", func() error {
			_, err := s.Gamma(ctx, src, 0)
			return err
		}},
		{"gamma nan", func() error {
			_, err := s.Gamma(ctx, src, math.NaN())
			return err
		}},
		{"sharpen sigma 0", func() error {
			_, err := s.Sharpen(ctx, src, 0, 1)
			return err
		}},
		{"sharpen amount -1", func() error {
			_, err := s.Sharpen(ctx, src, 1, -1)
			return err
		}},
		{"denoise radius 0", func() error {
			_, err := s.Denoise(ctx, src, 0)
			return err
		}},
		{"threshold unknown method", func() error {
			_, err := s.Threshold(ctx, src, "median", 0)
			return err
		}},
		{"threshold even window", func() error {
			_, err := s.Threshold(ctx, src, ThreshAdaptive, 30)
			return err
		}},
		{"threshold huge window", func() error {
			_, err := s.Threshold(ctx, src, ThreshAdaptive, 600)
			return err
		}},
	}
	for _, tt := range bad {
		t.Run(tt.name, func(t *testing.T) {
			if err := tt.run(); !errors.Is(err, ErrInvalidParameter) {
				t.Fatalf("error = %v, want ErrInvalidParameter", err)
			}
		})
	}
}

func TestThresholdGlobalSeparatesInkPaper(t *testing.T) {
	s := newTestService(t)
	page := solidPage(200, 200)
	paintInk(page, imageRect(40, 40, 100, 100))
	src := pngInput("scan1", page)

	res, err := s.Threshold(context.Background(), src, ThreshGlobal, 0)
	if err != nil {
		t.Fatalf("Threshold: %v", err)
	}
	img := decodeResult(t, res)
	if lumAt(img, 10, 10) != 255 {
		t.Fatalf("paper pixel = %d, want 255", lumAt(img, 10, 10))
	}
	if lumAt(img, 50, 50) != 0 {
		t.Fatalf("ink pixel = %d, want 0", lumAt(img, 50, 50))
	}
	if !containsNote(res, "otsu threshold") {
		t.Fatalf("notes = %v, want threshold note", res.Notes)
	}
}

func TestThresholdAdaptiveAutomaticWindow(t *testing.T) {
	s := newTestService(t)
	page := solidPage(200, 200)
	paintInk(page, imageRect(40, 40, 100, 100))
	src := pngInput("scan1", page)

	res, err := s.Threshold(context.Background(), src, ThreshAdaptive, 0)
	if err != nil {
		t.Fatalf("Threshold: %v", err)
	}
	if !containsNote(res, "(automatic)") || !containsNote(res, "sauvola window") {
		t.Fatalf("notes = %v, want automatic window note", res.Notes)
	}
	if lumAt(decodeResult(t, res), 41, 41) != 0 {
		// Sample at the ink boundary: deep inside a thick uniform ink block
		// Sauvola's local statistics degenerate (mean = std = 0), so paper
		// is the locally "correct" result there.
		t.Fatal("ink at the stroke boundary must survive adaptive thresholding")
	}
}

func TestThresholdAdaptiveExplicitWindow(t *testing.T) {
	s := newTestService(t)
	src := pngInput("scan1", solidPage(200, 200))

	res, err := s.Threshold(context.Background(), src, ThreshAdaptive, 31)
	if err != nil {
		t.Fatalf("Threshold: %v", err)
	}
	if !containsNote(res, "sauvola window 31") || containsNote(res, "(automatic)") {
		t.Fatalf("notes = %v, want explicit window note", res.Notes)
	}
}

func TestThresholdGlobalNotesIgnoredWindow(t *testing.T) {
	s := newTestService(t)
	src := pngInput("scan1", solidPage(200, 200))

	res, err := s.Threshold(context.Background(), src, ThreshGlobal, 31)
	if err != nil {
		t.Fatalf("Threshold: %v", err)
	}
	if !containsNote(res, "ignored") {
		t.Fatalf("notes = %v, want ignored-window note", res.Notes)
	}
}
