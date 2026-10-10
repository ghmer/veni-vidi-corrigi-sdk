package enhance

import (
	"testing"
)

func TestMedianRemovesIsolatedSpeckles(t *testing.T) {
	src := solidImg(9, 9, grayColor(255))
	i := (4*9 + 4) * 4
	for ch := range 3 {
		src.Pix[i+ch] = 0 // one black speck on white paper
	}
	out, err := MedianFilter(src, 1)
	if err != nil {
		t.Fatalf("MedianFilter: %v", err)
	}
	if got := valueAt(out, 4, 4); got != 255 {
		t.Fatalf("speck value after median = %d, want 255", got)
	}
	if got := valueAt(out, 0, 0); got != 255 {
		t.Fatalf("background changed: %d", got)
	}
}

func TestMedianKeepsThickInkInteriors(t *testing.T) {
	src := solidImg(9, 9, grayColor(255))
	for y := range 9 {
		for x := 2; x <= 6; x++ {
			i := (y*9 + x) * 4
			for ch := range 3 {
				src.Pix[i+ch] = 0
			}
		}
	}
	out, err := MedianFilter(src, 1)
	if err != nil {
		t.Fatalf("MedianFilter: %v", err)
	}
	if got := valueAt(out, 4, 4); got != 0 {
		t.Fatalf("ink interior eroded: %d, want 0", got)
	}
}

func TestMedianClampsAtBorders(t *testing.T) {
	src := solidImg(5, 5, grayColor(128))
	out, err := MedianFilter(src, 2)
	if err != nil {
		t.Fatalf("MedianFilter: %v", err)
	}
	if got := valueAt(out, 0, 0); got != 128 {
		t.Fatalf("corner value = %d, want 128", got)
	}
}

func TestMedianErrorsOnBadRadius(t *testing.T) {
	for _, r := range []int{0, 4, -1} {
		if _, err := MedianFilter(solidImg(9, 9, grayColor(255)), r); err == nil {
			t.Fatalf("MedianFilter accepted radius %d", r)
		}
	}
}
