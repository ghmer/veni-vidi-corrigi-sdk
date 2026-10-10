package codec

import (
	"errors"
	"image"
	"testing"
)

// rotatedL is the 3x2 "L" fixture: an L-shape drawn in the upper-left, so
// every orientation transform must move its dark corner predictably.
func rotatedL() *image.NRGBA {
	img := solid(3, 2, 255)
	put := func(x, y int) {
		i := (y*3 + x) * 4
		img.Pix[i], img.Pix[i+1], img.Pix[i+2] = 0, 0, 0
	}
	put(0, 0)
	put(0, 1)
	put(1, 1)
	return img
}

func TestDecodeAppliesExifOrientation(t *testing.T) {
	// Orientation 6 (rotate 90 CW) must swap the 3x2 dimensions to 2x3 and
	// move the vertical leg of the L to the horizontal leg.
	img := rotatedL()
	raw := jpegWithOrientation(t, img, 6)
	dec, err := Decode(raw, 1<<20)
	if err != nil {
		t.Fatalf("Decode: %v", err)
	}
	if dec.Orientation != 6 {
		t.Fatalf("orientation = %d, want 6", dec.Orientation)
	}
	if dec.Img.Bounds().Dx() != 2 || dec.Img.Bounds().Dy() != 3 {
		t.Fatalf("dims = %dx%d, want 2x3", dec.Img.Bounds().Dx(), dec.Img.Bounds().Dy())
	}
}

func TestDecodeWithoutExifStaysNormal(t *testing.T) {
	dec, err := Decode(encodePNG(t, rotatedL()), 1<<20)
	if err != nil {
		t.Fatalf("Decode: %v", err)
	}
	if dec.Format != FormatPNG || dec.Orientation != 1 {
		t.Fatalf("got %s orientation %d, want png/1", dec.Format, dec.Orientation)
	}
	if dec.Img.Bounds().Dx() != 3 || dec.Img.Bounds().Dy() != 2 {
		t.Fatalf("dims = %dx%d, want 3x2", dec.Img.Bounds().Dx(), dec.Img.Bounds().Dy())
	}
}

func TestDecodeJPEGWithoutExifIsOrientationOne(t *testing.T) {
	dec, err := Decode(encodeJPEG(t, rotatedL()), 1<<20)
	if err != nil {
		t.Fatalf("Decode: %v", err)
	}
	if dec.Orientation != 1 {
		t.Fatalf("orientation = %d, want 1", dec.Orientation)
	}
}

func TestDecodeRejectsOversizedSources(t *testing.T) {
	raw := encodePNG(t, solid(10, 10, 0))
	_, err := Decode(raw, 99)
	if !errors.Is(err, ErrSourceTooLarge) {
		t.Fatalf("error = %v, want ErrSourceTooLarge", err)
	}
}

func TestDecodeRejectsUnknownFormats(t *testing.T) {
	_, err := Decode([]byte("definitely not an image"), 1<<20)
	if !errors.Is(err, ErrUnsupportedFormat) {
		t.Fatalf("error = %v, want ErrUnsupportedFormat", err)
	}
}

func TestOrientationOfReadsNonJpegAsOne(t *testing.T) {
	if got := OrientationOf(encodePNG(t, rotatedL())); got != 1 {
		t.Fatalf("png orientation = %d, want 1", got)
	}
}

func TestCorrectedDimensions(t *testing.T) {
	for orient := 1; orient <= 8; orient++ {
		w, h := CorrectedDimensions(3, 2, orient)
		wantW, wantH := 3, 2
		if orient >= 5 {
			wantW, wantH = 2, 3
		}
		if w != wantW || h != wantH {
			t.Fatalf("orientation %d: %dx%d, want %dx%d", orient, w, h, wantW, wantH)
		}
	}
	if w, h := CorrectedDimensions(3, 2, 0); w != 3 || h != 2 {
		t.Fatalf("orientation 0: %dx%d, want stored 3x2", w, h)
	}
}
