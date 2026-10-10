// Package enhance contains image-quality transforms for handwriting scans:
// histogram stretching, gamma, global and adaptive binarization, median
// denoising, unsharp masking and border trimming. All functions are pure
// and free of I/O; parameter validation that is meant for users happens in
// the facade, the checks here are defensive.
package enhance

import (
	"errors"
	"image"
	"math"

	"github.com/ghmer/veni-vidi-corrigi-sdk/internal/pix"
)

// ErrInvalidParameter is returned when defensive checks catch an argument
// the facade should have validated before.
var ErrInvalidParameter = errors.New("enhance: invalid parameter")

// AutoContrast stretches the luminance histogram so that clipPercent percent
// of the darkest and brightest pixels saturate to black and white. A single
// linear map derived from luminance cut points is applied to all RGB
// channels, which lifts pencil strokes out of paper color casts without
// destroying colored ink. Transparent pixels (alpha < 128) are treated as
// paper in the histogram and copied unchanged to the output.
func AutoContrast(img image.Image, clipPercent float64) *image.NRGBA {
	src := pix.ToNRGBA(img)
	w, h := src.Rect.Dx(), src.Rect.Dy()
	if w == 0 || h == 0 {
		return src
	}
	gray, gw, gh := pix.GrayPlane(src)
	total := gw * gh
	hist := make([]int, 256)
	for _, v := range gray {
		hist[v]++
	}
	cut := int(math.Round(clipPercent / 100 * float64(total)))
	cut = min(cut, total/2)
	lo, hi := trimHistogram(hist, cut)
	if hi <= lo {
		return src
	}
	lut := contrastLUT(lo, hi)
	out := image.NewNRGBA(image.Rect(0, 0, w, h))
	for idx, srcIdx := 0, 0; idx < w*h; idx, srcIdx = idx+1, srcIdx+4 {
		if src.Pix[srcIdx+3] < 128 {
			copy(out.Pix[srcIdx:srcIdx+4], src.Pix[srcIdx:srcIdx+4])
			continue
		}
		out.Pix[srcIdx+0] = lut[src.Pix[srcIdx+0]]
		out.Pix[srcIdx+1] = lut[src.Pix[srcIdx+1]]
		out.Pix[srcIdx+2] = lut[src.Pix[srcIdx+2]]
		out.Pix[srcIdx+3] = src.Pix[srcIdx+3]
	}
	return out
}

// trimHistogram walks whole histogram buckets in from both ends while their
// pixel count fits into the cut budget. This is the standard percentile trim:
// with cut = 0 it yields the lowest and highest value actually present, and
// a uniform image degenerates to lo == hi (no-op).
func trimHistogram(hist []int, cut int) (lo, hi int) {
	lo, hi = 0, 255
	acc := 0
	for lo < hi && acc+hist[lo] <= cut {
		acc += hist[lo]
		lo++
	}
	acc = 0
	for hi > lo && acc+hist[hi] <= cut {
		acc += hist[hi]
		hi--
	}
	return lo, hi
}

// contrastLUT builds the linear stretch mapping [lo, hi] onto [0, 255].
func contrastLUT(lo, hi int) [256]uint8 {
	var lut [256]uint8
	scale := 255.0 / float64(hi-lo)
	for v := range lut {
		stretched := float64(v-lo) * scale
		stretched = max(stretched, 0)
		stretched = min(stretched, 255)
		lut[v] = uint8(stretched)
	}
	return lut
}
