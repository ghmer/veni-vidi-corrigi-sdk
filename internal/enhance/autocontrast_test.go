package enhance

import (
	"image/color"
	"testing"
)

func TestAutoContrastClipZeroStretchFullRange(t *testing.T) {
	src := rampImg(16, 255) // every value 0..255 present -> lo=0, hi=255
	out := AutoContrast(src, 0)
	// Ramp rows carry their own value, so row y shows value y.
	for _, p := range [][2]int{{0, 0}, {128, 128}, {255, 255}} {
		if got := int(valueAt(out, 8, p[0])); got != p[1] {
			t.Fatalf("row %d: value = %d, want %d", p[0], got, p[1])
		}
	}
}

func TestAutoContrastTrimsPercentiles(t *testing.T) {
	// 4096 pixels, clip 2% -> 82 pixels trimmed per side = 5.125 buckets of
	// 16 -> values 0..4 fully trimmed at the dark end, 251..255 at the top;
	// the mapping stretches [5, 250] onto [0, 255].
	src := rampImg(16, 255)
	out := AutoContrast(src, 2)
	cases := []struct {
		y    int
		want uint8
	}{
		{0, 0}, {4, 0}, {5, 0}, {128, 128}, {250, 255}, {253, 255}, {255, 255},
	}
	for _, c := range cases {
		if got := valueAt(out, 8, c.y); got != c.want {
			t.Fatalf("row %d: value = %d, want %d", c.y, got, c.want)
		}
	}
}

func TestAutoContrastUniformIsIdentity(t *testing.T) {
	src := solidImg(8, 8, color.NRGBA{200, 200, 200, 255})
	out := AutoContrast(src, 1)
	if valueAt(out, 3, 3) != 200 {
		t.Fatalf("uniform image changed: %d, want 200", valueAt(out, 3, 3))
	}
}

func TestAutoContrastKeepsTransparentPixels(t *testing.T) {
	src := rampImg(16, 255)
	i := (10*16 + 3) * 4
	src.Pix[i+3] = 0 // make one pixel fully transparent
	out := AutoContrast(src, 0)
	if alphaAt(out, 3, 10) != 0 {
		t.Fatalf("alpha = %d, want 0", alphaAt(out, 3, 10))
	}
	if valueAt(out, 3, 10) != 10 {
		t.Fatalf("transparent pixel color changed: %d", valueAt(out, 3, 10))
	}
}
