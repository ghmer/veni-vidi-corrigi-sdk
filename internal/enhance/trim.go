package enhance

import (
	"image"
	"sort"

	"github.com/ghmer/veni-vidi-corrigi-sdk/internal/pix"
)

// BorderMargins counts how many pixel rows/columns from each edge contain
// only background (the typical white paper margin or black scan frame). A
// row/column counts as background when at most ~1% of its pixels deviate
// from the corner-derived background color by more than tolerance. Scanning
// never passes the middle of the page.
//
// The margins are meant for facades that crop them away; zero margins mean
// "no uniform border detected".
func BorderMargins(img image.Image, tolerance int) (top, right, bottom, left int) {
	src := pix.ToNRGBA(img)
	w, h := src.Rect.Dx(), src.Rect.Dy()
	if w < 1 || h < 1 {
		return 0, 0, 0, 0
	}
	tol := clampVal(tolerance, 255)
	bg := backgroundFromCorners(src, w, h)

	deviates := func(x, y int) bool {
		i := (y*w + x) * 4
		if src.Pix[i+3] < 128 {
			return false
		}
		rd := absDiff(int(src.Pix[i]), int(bg[0]))
		gd := absDiff(int(src.Pix[i+1]), int(bg[1]))
		bd := absDiff(int(src.Pix[i+2]), int(bg[2]))
		return max(rd, gd, bd) > tol
	}
	rowBG := func(y int) bool {
		allowed := max(1, w/100)
		count := 0
		for x := range w {
			if deviates(x, y) {
				count++
				if count > allowed {
					return false
				}
			}
		}
		return true
	}
	colBG := func(x int) bool {
		allowed := max(1, h/100)
		count := 0
		for y := range h {
			if deviates(x, y) {
				count++
				if count > allowed {
					return false
				}
			}
		}
		return true
	}

	for top < h/2 && rowBG(top) {
		top++
	}
	for bottom < h/2 && rowBG(h-1-bottom) {
		bottom++
	}
	for left < w/2 && colBG(left) {
		left++
	}
	for right < w/2 && colBG(w-1-right) {
		right++
	}
	return top, right, bottom, left
}

// backgroundFromCorners derives the page background color as the per-channel
// median of the four image corners — robust against single dirty corners.
func backgroundFromCorners(src *image.NRGBA, w, h int) [3]uint8 {
	corners := [][4]uint8{
		pixelAt(src, 0, 0),
		pixelAt(src, w-1, 0),
		pixelAt(src, 0, h-1),
		pixelAt(src, w-1, h-1),
	}
	var bg [3]uint8
	for ch := range 3 {
		vals := []uint8{corners[0][ch], corners[1][ch], corners[2][ch], corners[3][ch]}
		sort.SliceStable(vals, func(i, j int) bool { return vals[i] < vals[j] })
		bg[ch] = vals[1]
	}
	return bg
}

func pixelAt(src *image.NRGBA, x, y int) [4]uint8 {
	i := (y*src.Rect.Dx() + x) * 4
	return [4]uint8{src.Pix[i], src.Pix[i+1], src.Pix[i+2], src.Pix[i+3]}
}

func absDiff(a, b int) int {
	if a > b {
		return a - b
	}
	return b - a
}
