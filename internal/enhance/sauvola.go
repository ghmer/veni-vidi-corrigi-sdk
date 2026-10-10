package enhance

import (
	"fmt"
	"image"
	"math"
)

// Sauvola binarizes a gray plane with locally adapted thresholds, ideal for
// photos where lighting gradients and paper texture would defeat a global
// threshold. For every pixel the threshold is
//
//	T = mean(1 + k (std/R - 1))
//
// over a square window, with the literature defaults k = 0.2 (sensitivity)
// and R = 128 (dynamic range). Border pixels use the clipped window.
func Sauvola(gray []uint8, w, h, window int) (*image.Gray, error) {
	if w < 1 || h < 1 {
		return nil, fmt.Errorf("%w: gray plane %dx%d is empty", ErrInvalidParameter, w, h)
	}
	if window < 3 || window%2 == 0 {
		return nil, fmt.Errorf("%w: window %d must be an odd number >= 3", ErrInvalidParameter, window)
	}
	const (
		sensitivity  = 0.2
		dynamicRange = 128.0
	)
	half := window / 2

	// Integral images give constant-time window statistics; float64 is
	// exact for the magnitudes involved (sums of squares stay far below
	// 2^53 for any image worth binarizing).
	sum := make([]float64, (w+1)*(h+1))
	sumSq := make([]float64, (w+1)*(h+1))
	for y := range h {
		rowSum, rowSumSq := 0.0, 0.0
		for x := range w {
			v := float64(gray[y*w+x])
			rowSum += v
			rowSumSq += v * v
			sum[(y+1)*(w+1)+x+1] = sum[y*(w+1)+x+1] + rowSum
			sumSq[(y+1)*(w+1)+x+1] = sumSq[y*(w+1)+x+1] + rowSumSq
		}
	}

	out := image.NewGray(image.Rect(0, 0, w, h))
	for y := range h {
		y0, y1 := max(0, y-half), min(h-1, y+half)
		for x := range w {
			x0, x1 := max(0, x-half), min(w-1, x+half)
			area := float64((x1 - x0 + 1) * (y1 - y0 + 1))
			mean, std := windowStats(sum, sumSq, w+1, x0, y0, x1, y1, area)
			threshold := mean * (1 + sensitivity*(std/dynamicRange-1))
			if float64(gray[y*w+x]) < threshold {
				out.Pix[y*w+x] = 0
			} else {
				out.Pix[y*w+x] = 255
			}
		}
	}
	return out, nil
}

// windowStats computes mean and standard deviation of the window
// [x0..x1] x [y0..y1] from integral images with stride stride.
func windowStats(sum, sumSq []float64, stride, x0, y0, x1, y1 int, area float64) (mean, std float64) {
	s := rectSum(sum, stride, x0, y0, x1, y1)
	q := rectSum(sumSq, stride, x0, y0, x1, y1)
	mean = s / area
	variance := q/area - mean*mean
	if variance < 0 {
		variance = 0
	}
	return mean, math.Sqrt(variance)
}

// rectSum reads the sum over the pixel rectangle from an integral image.
func rectSum(plane []float64, stride, x0, y0, x1, y1 int) float64 {
	return plane[(y1+1)*stride+x1+1] - plane[y0*stride+x1+1] - plane[(y1+1)*stride+x0] + plane[y0*stride+x0]
}
