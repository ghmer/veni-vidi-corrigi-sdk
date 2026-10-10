package imageutil

import (
	"context"
	"errors"
	"image"
	"math"
	"strings"
	"testing"
)

func TestCropRegionMappingAndUpscale(t *testing.T) {
	s := newTestService(t)
	page := solidPage(1600, 1600)
	inkSquare(page, Region{X: 0, Y: 0, W: 0.25, H: 0.25})
	src := pngInput("scan1", page)

	// The ink square occupies the upper-left quarter; cropping exactly that
	// region must keep the ink and scale the 400x400 excerpt up to 800.
	res, err := s.Crop(context.Background(), src, Region{X: 0, Y: 0, W: 0.25, H: 0.25}, Options{})
	if err != nil {
		t.Fatalf("Crop: %v", err)
	}
	if res.Width != 800 || res.Height != 800 {
		t.Fatalf("dims = %dx%d, want 800x800 (2x upscale)", res.Width, res.Height)
	}
	errContains(t, errors.New(joinNotes(res)), "resized 400x400 -> 800x800")
	if !hasInk(t, res) {
		t.Fatal("ink disappeared from the crop")
	}
}

func TestCropClampsOverflow(t *testing.T) {
	s := newTestService(t)
	src := pngInput("scan1", solidPage(2000, 2000))

	res, err := s.Crop(context.Background(), src, Region{X: 0.9, Y: 0.1, W: 0.5, H: 0.5}, Options{})
	if err != nil {
		t.Fatalf("Crop: %v", err)
	}
	errContains(t, errors.New(joinNotes(res)), "region clamped")
}

func TestCropRejectsEmptyRegion(t *testing.T) {
	s := newTestService(t)
	src := pngInput("scan1", solidPage(100, 100))

	_, err := s.Crop(context.Background(), src, Region{X: 0, Y: 0, W: 0, H: 0.5}, Options{})
	if !errors.Is(err, ErrInvalidRegion) {
		t.Fatalf("error = %v, want ErrInvalidRegion", err)
	}
	_, err = s.Crop(context.Background(), src, Region{X: 1.0, Y: 0, W: 0.2, H: 0.2}, Options{})
	if !errors.Is(err, ErrInvalidRegion) {
		t.Fatalf("outside-after-clamp: error = %v, want ErrInvalidRegion", err)
	}
}

// joinNotes flattens Result notes for substring checks.
func joinNotes(r Result) string {
	return strings.Join(r.Notes, "\n")
}

// containsNote reports whether any result note contains text.
func containsNote(r Result, text string) bool {
	return strings.Contains(joinNotes(r), text)
}

func TestCropUpscaleCapNote(t *testing.T) {
	s := newTestService(t)
	src := pngInput("tiny", solidPage(100, 100))

	res, err := s.Crop(context.Background(), src, Region{X: 0, Y: 0, W: 0.1, H: 0.1}, Options{})
	if err != nil {
		t.Fatalf("Crop: %v", err)
	}
	switch {
	case res.Width != 40 || res.Height != 40:
		t.Fatalf("dims = %dx%d, want 40x40 (factor capped at 4)", res.Width, res.Height)
	case !containsNote(res, "capped"):
		t.Fatalf("notes = %v, want upscale cap note", res.Notes)
	}
}

func TestCropRespectsOptionsMaxSide(t *testing.T) {
	s := newTestService(t)
	src := pngInput("scan1", solidPage(3000, 3000))

	res, err := s.Crop(context.Background(), src, Region{X: 0, Y: 0, W: 1, H: 1}, Options{MaxSide: 1200})
	if err != nil {
		t.Fatalf("Crop: %v", err)
	}
	if res.Width != 1200 || res.Height != 1200 {
		t.Fatalf("dims = %dx%d, want 1200x1200", res.Width, res.Height)
	}
	if _, err = s.Crop(context.Background(), src, Region{X: 0, Y: 0, W: 1, H: 1}, Options{MaxSide: 1200, SizeLimit: -1}); !errors.Is(err, ErrInvalidParameter) {
		t.Fatalf("negative SizeLimit: %v", err)
	}
}

func TestCropJPEGExifApplied(t *testing.T) {
	s := newTestService(t)
	src := Input{Data: jpegWithExif(t, solidPage(3, 2), 6), Name: "phone"}

	res, err := s.Crop(context.Background(), src, Region{X: 0, Y: 0, W: 1, H: 1}, Options{})
	if err != nil {
		t.Fatalf("Crop: %v", err)
	}
	// Orientation 6 rotates the 3x2 photo to 2x3; the Crop upscale (factor
	// cap 4 for tiny excerpts) then enlarges the rotated page to 8x12.
	if res.Width != 8 || res.Height != 12 {
		t.Fatalf("dims = %dx%d, want 8x12 (rotated then upscaled)", res.Width, res.Height)
	}
	if !containsNote(res, "orientation 6 applied") {
		t.Fatalf("notes = %v, want orientation note", res.Notes)
	}
}

func TestResultIDsAreDeterministic(t *testing.T) {
	s := newTestService(t)
	src := pngInput("scan1", solidPage(4000, 4000))
	ctx := context.Background()

	// The SDK is stateless: identical calls produce identical identifiers
	// and the host decides which results to keep.
	first, err := s.Crop(ctx, src, Region{X: 0, Y: 0, W: 0.5, H: 0.5}, Options{})
	if err != nil {
		t.Fatalf("Crop 1: %v", err)
	}
	second, err := s.Crop(ctx, src, Region{X: 0, Y: 0.5, W: 0.5, H: 0.5}, Options{})
	if err != nil {
		t.Fatalf("Crop 2: %v", err)
	}
	if first.ID != "scan1-crop01" || second.ID != "scan1-crop01" {
		t.Fatalf("ids = %s, %s, want deterministic scan1-crop01", first.ID, second.ID)
	}

	// An unnamed input receives the default fragment.
	anon, err := s.View(ctx, Input{Data: src.Data}, 0)
	if err != nil {
		t.Fatalf("View: %v", err)
	}
	if anon.ID != "img-view01" {
		t.Fatalf("id = %s, want img-view01", anon.ID)
	}

	// Names are sanitized: no path traversal or separators survive.
	evil, err := s.Rotate(ctx, Input{Data: src.Data, Name: "../..//evil:name.png"}, 90)
	if err != nil {
		t.Fatalf("Rotate: %v", err)
	}
	if strings.ContainsAny(evil.ID, "/\\:") || strings.Contains(evil.ID, "..") {
		t.Fatalf("id = %s, want a sanitized fragment", evil.ID)
	}
}

func TestScaleValidatesFactor(t *testing.T) {
	s := newTestService(t)
	src := pngInput("scan1", solidPage(100, 100))

	for _, f := range []float64{-1, 0.2, 5, 4.5} {
		if _, err := s.Scale(context.Background(), src, f); !errors.Is(err, ErrInvalidParameter) {
			t.Fatalf("Scale(%v) error = %v, want ErrInvalidParameter", f, err)
		}
	}
}

func TestScaleDoublesDimensions(t *testing.T) {
	s := newTestService(t)
	src := pngInput("scan1", solidPage(120, 80))

	res, err := s.Scale(context.Background(), src, 2)
	if err != nil {
		t.Fatalf("Scale: %v", err)
	}
	if res.Width != 240 || res.Height != 160 {
		t.Fatalf("dims = %dx%d, want 240x160", res.Width, res.Height)
	}
}

// TestScaleOutputPixelBudget pins the finding fix: scaling a valid input by
// up to maxUpscale must never allocate beyond the pixel budget.
func TestScaleOutputPixelBudget(t *testing.T) {
	s, err := New(WithLogger(discardLogger()), WithMaxInputPixels(10_000))
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	src := pngInput("scan1", solidPage(100, 100)) // 10k pixels, decodes fine

	_, err = s.Scale(context.Background(), src, 4) // would be 160k pixels
	if !errors.Is(err, ErrSourceTooLarge) {
		t.Fatalf("error = %v, want ErrSourceTooLarge", err)
	}
	errContains(t, err, "output 400x400")
}

// TestCropOutputPixelBudget covers the boundedResize guard in the Crop
// upscale path: upscaling a small excerpt is fine, but the guard still
// applies when the budget is tight.
func TestCropOutputPixelBudget(t *testing.T) {
	s, err := New(WithLogger(discardLogger()), WithMaxInputPixels(10_000))
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	src := pngInput("scan1", solidPage(100, 100))

	_, err = s.Crop(context.Background(), src, Region{X: 0, Y: 0, W: 1, H: 1}, Options{MaxSide: 4000})
	if !errors.Is(err, ErrSourceTooLarge) {
		t.Fatalf("error = %v, want ErrSourceTooLarge", err)
	}
}

func TestRotateMapsAllAngles(t *testing.T) {
	s := newTestService(t)
	src := pngInput("wide", solidPage(60, 40))
	ctx := context.Background()

	cases := []struct {
		deg    int
		w, h   int
		opName string
	}{
		{90, 40, 60, "wide-rot9001"},
		{180, 60, 40, "wide-rot18001"},
		{270, 40, 60, "wide-rot27001"},
	}
	for _, c := range cases {
		res, err := s.Rotate(ctx, src, c.deg)
		if err != nil {
			t.Fatalf("Rotate(%d): %v", c.deg, err)
		}
		if res.Width != c.w || res.Height != c.h {
			t.Fatalf("Rotate(%d) dims = %dx%d, want %dx%d", c.deg, res.Width, res.Height, c.w, c.h)
		}
		if res.ID != c.opName {
			t.Fatalf("id = %s, want %s", res.ID, c.opName)
		}
	}
}

func TestRotateRejectsFreeAngles(t *testing.T) {
	s := newTestService(t)
	src := pngInput("page", solidPage(100, 100))

	for _, deg := range []int{0, 45, -90, 360} {
		_, err := s.Rotate(context.Background(), src, deg)
		if !errors.Is(err, ErrInvalidParameter) {
			t.Fatalf("Rotate(%d) error = %v, want ErrInvalidParameter", deg, err)
		}
	}
}

func TestSplitTopBottom(t *testing.T) {
	s := newTestService(t)
	page := solidPage(100, 100)
	paintInk(page, image.Rect(0, 0, 100, 50))
	src := pngInput("scan1", page)

	results, err := s.Split(context.Background(), src, SplitTopBottom, 0, 0, 0)
	if err != nil {
		t.Fatalf("Split: %v", err)
	}
	if len(results) != 2 {
		t.Fatalf("results = %d, want 2", len(results))
	}
	for i, res := range results {
		if res.Width != 100 || res.Height != 54 { // overlap 0.08 -> span 0.54
			t.Fatalf("tile %d dims = %dx%d, want 100x54", i, res.Width, res.Height)
		}
		if !containsNote(res, "default overlap") {
			t.Fatalf("tile %d notes = %v, want overlap default note", i, res.Notes)
		}
	}
	if results[0].ID != "scan1-split01" || results[1].ID != "scan1-split02" {
		t.Fatalf("ids = %s, %s", results[0].ID, results[1].ID)
	}
	if !hasInk(t, results[0]) || !hasInk(t, results[1]) {
		t.Fatal("overlap must keep context visible in both tiles")
	}
}

func TestSplitTilesGrid(t *testing.T) {
	s := newTestService(t)
	src := pngInput("scan1", solidPage(300, 200))

	results, err := s.Split(context.Background(), src, SplitTiles, 3, 2, 0)
	if err != nil {
		t.Fatalf("Split: %v", err)
	}
	if len(results) != 6 {
		t.Fatalf("results = %d, want 6", len(results))
	}
}

func TestSplitExplicitOverlap(t *testing.T) {
	s := newTestService(t)
	src := pngInput("scan1", solidPage(100, 100))

	// overlap 0.25: step 0.375, span 0.625 -> tiles y [0..63] / [37..100].
	results, err := s.Split(context.Background(), src, SplitTopBottom, 0, 0, 0.25)
	if err != nil {
		t.Fatalf("Split: %v", err)
	}
	if results[0].Height != 63 || results[1].Height != 63 {
		t.Fatalf("heights = %d/%d, want 63/63", results[0].Height, results[1].Height)
	}
}

func TestSplitValidates(t *testing.T) {
	s := newTestService(t)
	src := pngInput("scan1", solidPage(100, 100))
	ctx := context.Background()

	if _, err := s.Split(ctx, src, "diagonal", 0, 0, 0); !errors.Is(err, ErrInvalidParameter) {
		t.Fatalf("bad mode: %v", err)
	}
	if _, err := s.Split(ctx, src, SplitTiles, 7, 7, 0); !errors.Is(err, ErrInvalidParameter) {
		t.Fatalf("oversized grid: %v", err)
	}
	if _, err := s.Split(ctx, src, SplitTiles, 2, 2, -0.1); !errors.Is(err, ErrInvalidParameter) {
		t.Fatalf("negative overlap: %v", err)
	}
	if _, err := s.Split(ctx, src, SplitTopBottom, 0, 0, 0.5); !errors.Is(err, ErrInvalidParameter) {
		t.Fatalf("overlap over 0.25: %v", err)
	}
	if _, err := s.Split(ctx, src, SplitTopBottom, 0, 0, math.NaN()); !errors.Is(err, ErrInvalidParameter) {
		t.Fatalf("NaN overlap: %v", err)
	}
}

func TestTrimBordersRemovesFrame(t *testing.T) {
	s := newTestService(t)
	page := solidPage(60, 48)
	paintInk(page, image.Rect(10, 8, 50, 32))
	src := pngInput("scan1", page)

	res, err := s.TrimBorders(context.Background(), src)
	if err != nil {
		t.Fatalf("TrimBorders: %v", err)
	}
	if res.Width != 40 || res.Height != 24 {
		t.Fatalf("dims = %dx%d, want 40x24", res.Width, res.Height)
	}
	if !containsNote(res, "trimmed margins") {
		t.Fatalf("notes = %v", res.Notes)
	}
}

func TestTrimBordersBlankPageKeepsContent(t *testing.T) {
	s := newTestService(t)
	src := pngInput("scan1", solidPage(20, 16))

	res, err := s.TrimBorders(context.Background(), src)
	if err != nil {
		t.Fatalf("TrimBorders: %v", err)
	}
	if res.Width != 20 || res.Height != 16 {
		t.Fatalf("blank page shrank to %dx%d", res.Width, res.Height)
	}
	if !containsNote(res, "kept as-is") {
		t.Fatalf("notes = %v", res.Notes)
	}
}

func TestTrimBordersNoBorderDetected(t *testing.T) {
	s := newTestService(t)
	img := solidPage(16, 16)
	for y := range 16 {
		for x := range 16 {
			i := (y*16 + x) * 4
			v := uint8((x*17 + y*29) % 256)
			img.Pix[i], img.Pix[i+1], img.Pix[i+2] = v, v, v
		}
	}
	src := pngInput("scan1", img)

	res, err := s.TrimBorders(context.Background(), src)
	if err != nil {
		t.Fatalf("TrimBorders: %v", err)
	}
	if !containsNote(res, "no uniform border detected") {
		t.Fatalf("notes = %v", res.Notes)
	}
}
