package geometry

import (
	"errors"
	"image"
	"math"
	"testing"
)

// letterImg renders "abc / def" as an image so rotation tests can assert the
// exact pixel permutation with distinct colors.
func letterImg() *image.NRGBA {
	grid := [][]uint8{
		{1, 2, 3}, // a b c
		{4, 5, 6}, // d e f
	}
	img := image.NewNRGBA(image.Rect(0, 0, 3, 2))
	for y, row := range grid {
		for x, v := range row {
			i := img.PixOffset(x, y)
			img.Pix[i], img.Pix[i+1], img.Pix[i+2], img.Pix[i+3] = v, v, v, 255
		}
	}
	return img
}

// pixelAt reads the gray value of the pixel at (x, y).
func pixelAt(img *image.NRGBA, x, y int) uint8 {
	i := img.PixOffset(x, y)
	return img.Pix[i]
}

// regionEq compares regions with a small tolerance because clamping does
// float arithmetic (1-0.8 is not exactly 0.2).
func regionEq(a, b Region) bool {
	const eps = 1e-9
	return math.Abs(a.X-b.X) < eps && math.Abs(a.Y-b.Y) < eps &&
		math.Abs(a.W-b.W) < eps && math.Abs(a.H-b.H) < eps
}

func TestClamp(t *testing.T) {
	tests := []struct {
		name    string
		in      Region
		want    Region
		changed bool
		wantErr bool
	}{
		{name: "valid", in: Region{X: 0.1, Y: 0.2, W: 0.5, H: 0.5}, want: Region{X: 0.1, Y: 0.2, W: 0.5, H: 0.5}},
		{name: "negative origin", in: Region{X: -0.1, Y: -0.2, W: 0.3, H: 0.3}, want: Region{X: 0, Y: 0, W: 0.3, H: 0.3}, changed: true},
		{name: "overflow right", in: Region{X: 0.8, Y: 0.1, W: 0.5, H: 0.5}, want: Region{X: 0.8, Y: 0.1, W: 0.2, H: 0.5}, changed: true},
		{name: "overflow width", in: Region{X: 0, Y: 0, W: 2, H: 0.5}, want: Region{X: 0, Y: 0, W: 1, H: 0.5}, changed: true},
		{name: "empty", in: Region{X: 0.5, Y: 0.5, W: 0, H: 0.1}, wantErr: true},
		{name: "outside after clamp", in: Region{X: 1, Y: 0, W: 0.5, H: 0.5}, wantErr: true},
		{name: "nan", in: Region{X: 0.1, Y: float64(math.NaN()), W: 0.5, H: 0.5}, wantErr: true},
		{name: "inf", in: Region{X: 0.1, Y: 0, W: math.Inf(1), H: 0.5}, wantErr: true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, changed, err := Clamp(tt.in)
			if (err != nil) != tt.wantErr {
				t.Fatalf("Clamp(%s) error = %v, wantErr %v", tt.in, err, tt.wantErr)
			}
			if tt.wantErr {
				return
			}
			if !regionEq(got, tt.want) {
				t.Fatalf("Clamp(%s) = %s, want %s", tt.in, got, tt.want)
			}
			if changed != tt.changed {
				t.Fatalf("changed = %v, want %v", changed, tt.changed)
			}
		})
	}
}

func TestPixelRect(t *testing.T) {
	tests := []struct {
		name string
		r    Region
		w, h int
		want image.Rectangle
	}{
		{name: "upper left quarter", r: Region{X: 0, Y: 0, W: 0.5, H: 0.5}, w: 100, h: 100, want: image.Rect(0, 0, 50, 50)},
		{name: "ceil superset", r: Region{X: 0.001, Y: 0.25, W: 0.5, H: 0.5}, w: 100, h: 100, want: image.Rect(0, 25, 51, 75)},
		{name: "full", r: Region{X: 0, Y: 0, W: 1, H: 1}, w: 7, h: 9, want: image.Rect(0, 0, 7, 9)},
		{name: "wide region clamps at image edge", r: Region{X: 0.9, Y: 0.9, W: 0.5, H: 0.5}, w: 10, h: 10, want: image.Rect(9, 9, 10, 10)},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.r.PixelRect(tt.w, tt.h)
			if !got.Eq(tt.want) {
				t.Fatalf("PixelRect = %s, want %s", got, tt.want)
			}
		})
	}
}

func TestFitLongest(t *testing.T) {
	tests := []struct {
		name         string
		w, h         int
		minEdge      int
		maxSide      int
		maxUpscale   float64
		wantW, wantH int
		wantScale    float64
		wantNoteNum  int
	}{
		{name: "keep inside window", w: 1000, h: 1500, minEdge: 800, maxSide: 1600, maxUpscale: 4, wantW: 1000, wantH: 1500, wantScale: 1},
		{name: "downscale", w: 2480, h: 3508, minEdge: 800, maxSide: 1600, maxUpscale: 4, wantW: 1131, wantH: 1600, wantScale: 1600.0 / 3508.0, wantNoteNum: 1},
		{name: "upscale", w: 400, h: 300, minEdge: 800, maxSide: 1600, maxUpscale: 4, wantW: 800, wantH: 600, wantScale: 800.0 / 400.0, wantNoteNum: 1},
		{name: "upscale capped", w: 100, h: 50, minEdge: 800, maxSide: 1600, maxUpscale: 4, wantW: 400, wantH: 200, wantScale: 4, wantNoteNum: 2},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tw, th, scale, notes, err := FitLongest(tt.w, tt.h, tt.minEdge, tt.maxSide, tt.maxUpscale)
			if err != nil {
				t.Fatalf("FitLongest: %v", err)
			}
			if tw != tt.wantW || th != tt.wantH {
				t.Fatalf("dims = %dx%d, want %dx%d", tw, th, tt.wantW, tt.wantH)
			}
			if math.Abs(scale-tt.wantScale) > 1e-9 {
				t.Fatalf("scale = %v, want %v", scale, tt.wantScale)
			}
			if len(notes) != tt.wantNoteNum {
				t.Fatalf("notes = %v, want %d note(s)", notes, tt.wantNoteNum)
			}
		})
	}
}

func TestFitDownNeverUpscales(t *testing.T) {
	tw, th, scale, notes, err := FitDown(500, 300, 1600)
	if err != nil {
		t.Fatalf("FitDown: %v", err)
	}
	if tw != 500 || th != 300 || scale != 1 || notes != nil {
		t.Fatalf("FitDown changed an already-fitting image: %dx%d scale=%v notes=%v", tw, th, scale, notes)
	}
}

func TestFitRejectsInvalidDims(t *testing.T) {
	if _, _, _, _, err := FitLongest(0, 100, 800, 1600, 4); err == nil {
		t.Fatal("FitLongest accepted zero width")
	}
	if _, _, _, _, err := FitDown(100, -1, 1600); err == nil {
		t.Fatal("FitDown accepted negative height")
	}
}

func TestRotate90Clockwise(t *testing.T) {
	src := letterImg() // [[1 2 3] / [4 5 6]]
	got := Rotate90(src)
	if got.Bounds().Dx() != 2 || got.Bounds().Dy() != 3 {
		t.Fatalf("dims = %dx%d, want 2x3", got.Bounds().Dx(), got.Bounds().Dy())
	}
	want := [][]uint8{
		{4, 1},
		{5, 2},
		{6, 3},
	}
	assertGrid(t, got, want)
}

func TestRotate270CounterClockwise(t *testing.T) {
	got := Rotate270(letterImg())
	want := [][]uint8{
		{3, 6},
		{2, 5},
		{1, 4},
	}
	assertGrid(t, got, want)
}

func TestRotate180(t *testing.T) {
	got := Rotate180(letterImg())
	want := [][]uint8{
		{6, 5, 4},
		{3, 2, 1},
	}
	assertGrid(t, got, want)
}

func TestFlipsAndDiagonals(t *testing.T) {
	src := letterImg()
	assertGrid(t, FlipH(src), [][]uint8{{3, 2, 1}, {6, 5, 4}})
	assertGrid(t, FlipV(src), [][]uint8{{4, 5, 6}, {1, 2, 3}})
	assertGrid(t, Transpose(src), [][]uint8{{1, 4}, {2, 5}, {3, 6}})
	assertGrid(t, Transverse(src), [][]uint8{{6, 3}, {5, 2}, {4, 1}})
}

// Transpose equals FlipH applied after Rotate90; Transverse equals FlipH
// applied after Rotate270 — both identities must hold exactly.
func TestDiagonalIdentities(t *testing.T) {
	src := letterImg()
	assertSamePixels(t, Transpose(src), FlipH(Rotate90(src)))
	assertSamePixels(t, Transverse(src), FlipH(Rotate270(src)))
}

// assertSamePixels requires identical dimensions and identical pixel values.
func assertSamePixels(t *testing.T, got, want *image.NRGBA) {
	t.Helper()
	if !got.Rect.Eq(want.Rect) {
		t.Fatalf("rect = %s, want %s", got.Rect, want.Rect)
	}
	for i := range want.Pix {
		if got.Pix[i] != want.Pix[i] {
			t.Fatalf("pixel %d: got %d, want %d", i, got.Pix[i], want.Pix[i])
		}
	}
}

func assertGrid(t *testing.T, img *image.NRGBA, want [][]uint8) {
	t.Helper()
	if len(want) == 0 || len(want[0]) == 0 {
		t.Fatal("empty want grid")
	}
	if img.Bounds().Dy() != len(want) || img.Bounds().Dx() != len(want[0]) {
		t.Fatalf("dims = %dx%d, want %dx%d", img.Bounds().Dx(), img.Bounds().Dy(), len(want[0]), len(want))
	}
	for y, row := range want {
		for x, v := range row {
			if got := pixelAt(img, x, y); got != v {
				t.Fatalf("pixel(%d,%d) = %d, want %d", x, y, got, v)
			}
		}
	}
}

func TestGridRectsCoverageAndOverlap(t *testing.T) {
	// 2 columns x 1 row with 8% overlap on a 100-wide image:
	// step = 0.46, span = 0.54 -> tiles [0..54] and [46..100].
	rects, err := GridRects(100, 100, 2, 1, 0.08)
	if err != nil {
		t.Fatalf("GridRects: %v", err)
	}
	if len(rects) != 2 {
		t.Fatalf("rects = %d, want 2", len(rects))
	}
	if !rects[0].Eq(image.Rect(0, 0, 54, 100)) {
		t.Fatalf("first tile = %s", rects[0])
	}
	if !rects[1].Eq(image.Rect(46, 0, 100, 100)) {
		t.Fatalf("second tile = %s", rects[1])
	}
	if rects[0].Max.X <= rects[1].Min.X {
		t.Fatal("tiles must overlap")
	}
}

func TestGridRectsOverlapZeroSplitsExactly(t *testing.T) {
	rects, err := GridRects(200, 100, 2, 1, 0)
	if err != nil {
		t.Fatalf("GridRects: %v", err)
	}
	if !rects[0].Eq(image.Rect(0, 0, 100, 100)) || !rects[1].Eq(image.Rect(100, 0, 200, 100)) {
		t.Fatalf("zero-overlap tiles = %s, %s", rects[0], rects[1])
	}
}

func TestGridRectsErrors(t *testing.T) {
	if _, err := GridRects(0, 100, 2, 1, 0); err == nil {
		t.Fatal("accepted zero width")
	}
	if _, err := GridRects(100, 100, 0, 1, 0); err == nil {
		t.Fatal("accepted zero columns")
	}
	if _, err := GridRects(100, 100, 2, 1, 1); err == nil {
		t.Fatal("accepted overlap of 1")
	}
	if _, err := GridRects(100, 100, 2, 1, -0.1); err == nil {
		t.Fatal("accepted negative overlap")
	}
}

func TestFitLongestRejectsNonFiniteMaxUpscale(t *testing.T) {
	for _, mu := range []float64{math.NaN(), math.Inf(1), math.Inf(-1)} {
		if _, _, _, _, err := FitLongest(100, 100, 800, 1600, mu); err == nil {
			t.Fatalf("FitLongest accepted maxUpscale %v", mu)
		} else if !errors.Is(err, ErrInvalidParameter) {
			t.Fatalf("error = %v, want ErrInvalidParameter", err)
		}
	}
}

func TestGridRectsRejectsNaNOverlap(t *testing.T) {
	if _, err := GridRects(100, 100, 2, 1, math.NaN()); err == nil {
		t.Fatal("accepted NaN overlap")
	}
}
