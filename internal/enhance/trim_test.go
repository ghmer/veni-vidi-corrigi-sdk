package enhance

import (
	"image"
	"testing"
)

// framedInk builds a white page with an ink rectangle inset from all four
// edges, the scan equivalent of white margins around written content.
func framedInk() *image.NRGBA {
	img := solidImg(20, 16, grayColor(255))
	for y := 4; y <= 11; y++ {
		for x := 6; x <= 13; x++ {
			i := (y*20 + x) * 4
			img.Pix[i], img.Pix[i+1], img.Pix[i+2] = 0, 0, 0
		}
	}
	return img
}

func TestBorderMarginsAroundInk(t *testing.T) {
	top, right, bottom, left := BorderMargins(framedInk(), 0)
	if top != 4 || right != 6 || bottom != 4 || left != 6 {
		t.Fatalf("margins = top %d right %d bottom %d left %d, want 4/6/4/6", top, right, bottom, left)
	}
}

func TestBorderMarginsTolerantToNoise(t *testing.T) {
	// Same page with a single stray dark pixel in the top margin: the
	// ~1% row rule must still detect the margin.
	img := framedInk()
	i := (1*20 + 3) * 4
	img.Pix[i], img.Pix[i+1], img.Pix[i+2] = 0, 0, 0
	top, _, _, left := BorderMargins(img, 30)
	if top != 4 || left != 6 {
		t.Fatalf("noisy margins = top %d left %d, want 4/6", top, left)
	}
}

func TestBorderMarginsNoTrimmableContent(t *testing.T) {
	// A diagonal gradient fills every row and column with deviating pixels,
	// so no margin exists.
	img := solidImg(16, 16, grayColor(0))
	for y := range 16 {
		for x := range 16 {
			v := uint8((x*17 + y*29) % 256)
			i := (y*16 + x) * 4
			img.Pix[i], img.Pix[i+1], img.Pix[i+2] = v, v, v
		}
	}
	top, right, bottom, left := BorderMargins(img, 10)
	if top != 0 || right != 0 || bottom != 0 || left != 0 {
		t.Fatalf("margins = %d/%d/%d/%d, want all zero", top, right, bottom, left)
	}
}

func TestBorderMarginsCapsAtHalf(t *testing.T) {
	// A blank page must not produce margins that erase the whole content
	// area; each margin stops at half the axis length.
	top, right, bottom, left := BorderMargins(solidImg(10, 8, grayColor(255)), 0)
	if top+bottom > 8 || left+right > 10 {
		t.Fatalf("margins overrun: %d+%d/%d+%d", top, bottom, left, right)
	}
}
