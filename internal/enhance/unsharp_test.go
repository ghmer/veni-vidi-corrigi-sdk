package enhance

import (
	"image"
	"testing"
)

// stepEdge returns a wide image with a vertical step edge: left columns at
// 100, right columns at 150.
func stepEdge() *image.NRGBA {
	img := image.NewNRGBA(image.Rect(0, 0, 32, 16))
	for y := range 16 {
		for x := range 32 {
			v := uint8(100)
			if x >= 16 {
				v = 150
			}
			i := (y*32 + x) * 4
			img.Pix[i], img.Pix[i+1], img.Pix[i+2], img.Pix[i+3] = v, v, v, 255
		}
	}
	return img
}

func TestUnsharpMaskOvershootsAtEdges(t *testing.T) {
	out, err := UnsharpMask(stepEdge(), 1.0, 2.0)
	if err != nil {
		t.Fatalf("UnsharpMask: %v", err)
	}
	if got := valueAt(out, 17, 8); got <= 150 {
		t.Fatalf("bright side value = %d, want > 150 (overshoot)", got)
	}
	if got := valueAt(out, 14, 8); got >= 100 {
		t.Fatalf("dark side value = %d, want < 100 (undershoot)", got)
	}
	// Far away from the edge the original values must survive.
	if valueAt(out, 2, 8) != 100 || valueAt(out, 29, 8) != 150 {
		t.Fatalf("flat areas must be unchanged: %d / %d", valueAt(out, 2, 8), valueAt(out, 29, 8))
	}
}

func TestUnsharpMaskZeroAmountIsIdentity(t *testing.T) {
	src := rampImg(8, 255)
	out, err := UnsharpMask(src, 1.0, 0)
	if err != nil {
		t.Fatalf("UnsharpMask: %v", err)
	}
	if valueAt(out, 4, 100) != 100 {
		t.Fatalf("amount 0 changed value: %d", valueAt(out, 4, 100))
	}
}

func TestUnsharpMaskErrors(t *testing.T) {
	if _, err := UnsharpMask(stepEdge(), 0, 1); err == nil {
		t.Fatal("accepted sigma 0")
	}
	if _, err := UnsharpMask(stepEdge(), 1, -1); err == nil {
		t.Fatal("accepted negative amount")
	}
}
