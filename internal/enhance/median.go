package enhance

import (
	"fmt"
	"image"

	"github.com/ghmer/veni-vidi-corrigi-sdk/internal/pix"
)

// MedianFilter applies a square median filter of the given radius (1-3 px)
// to the RGB channels. The median preserves stroke edges that box or
// gaussian filters would smear, which makes it the right denoiser for JPEG
// artifacts and sensor grain around handwritten text.
func MedianFilter(img image.Image, radius int) (*image.NRGBA, error) {
	if radius < 1 || radius > 3 {
		return nil, fmt.Errorf("%w: median radius %d must lie in [1, 3]", ErrInvalidParameter, radius)
	}
	src := pix.ToNRGBA(img)
	w, h := src.Rect.Dx(), src.Rect.Dy()
	out := image.NewNRGBA(image.Rect(0, 0, w, h))
	side := 2*radius + 1
	bufR := make([]uint8, side*side)
	bufG := make([]uint8, side*side)
	bufB := make([]uint8, side*side)

	for y := range h {
		for x := range w {
			n := 0
			for yy := y - radius; yy <= y+radius; yy++ {
				cy := clampVal(yy, h-1)
				for xx := x - radius; xx <= x+radius; xx++ {
					cx := clampVal(xx, w-1)
					si := (cy*w + cx) * 4
					bufR[n], bufG[n], bufB[n] = src.Pix[si], src.Pix[si+1], src.Pix[si+2]
					n++
				}
			}
			di := (y*w + x) * 4
			out.Pix[di+0] = medianValue(bufR, n)
			out.Pix[di+1] = medianValue(bufG, n)
			out.Pix[di+2] = medianValue(bufB, n)
			out.Pix[di+3] = src.Pix[di+3]
		}
	}
	return out, nil
}

// medianValue returns the lower median of the first n buffered values;
// insertion sort is the right tool for the tiny, nearly-sorted windows.
func medianValue(buf []uint8, n int) uint8 {
	for i := 1; i < n; i++ {
		v := buf[i]
		j := i - 1
		for j >= 0 && buf[j] > v {
			buf[j+1] = buf[j]
			j--
		}
		buf[j+1] = v
	}
	return buf[(n-1)/2]
}

func clampVal(v, hi int) int {
	return min(max(v, 0), hi)
}
