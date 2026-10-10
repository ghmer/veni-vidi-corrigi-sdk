package enhance

import (
	"fmt"
	"image"
)

// Histogram builds the 256-bucket histogram of a gray plane.
func Histogram(gray []uint8) []int {
	hist := make([]int, 256)
	for _, v := range gray {
		hist[v]++
	}
	return hist
}

// Otsu returns the threshold t that maximizes the between-class variance of
// a 256-bucket histogram. For degenerate (uniform) histograms the loop never
// finds a separating variance and returns 0, which maps every pixel to
// paper. The result is meant for Binarize, where values strictly below t
// become ink.
func Otsu(hist []int) (int, error) {
	if len(hist) != 256 {
		return 0, fmt.Errorf("%w: histogram needs 256 buckets, got %d", ErrInvalidParameter, len(hist))
	}
	total, sumAll := 0, 0
	for value, count := range hist {
		total += count
		sumAll += value * count
	}
	if total == 0 {
		return 0, nil
	}
	var sumB float64
	wB, thresh := 0, 0
	maxVariance := 0.0
	for t := range hist {
		wB += hist[t]
		if wB == 0 {
			continue
		}
		wF := total - wB
		if wF == 0 {
			break
		}
		sumB += float64(t * hist[t])
		meanB := sumB / float64(wB)
		meanF := (float64(sumAll) - sumB) / float64(wF)
		variance := float64(wB) * float64(wF) * (meanB - meanF) * (meanB - meanF)
		if variance > maxVariance {
			maxVariance = variance
			thresh = t
		}
	}
	return thresh, nil
}

// Binarize renders a gray plane as black ink (0) on white paper (255):
// values at or below the threshold become ink. Otsu's returned threshold is
// the darkest class representative, so the class itself must stay ink.
func Binarize(gray []uint8, w, h, threshold int) *image.Gray {
	out := image.NewGray(image.Rect(0, 0, w, h))
	t := uint8(min(max(threshold, 0), 255))
	for idx, v := range gray {
		if v <= t {
			out.Pix[idx] = 0
		} else {
			out.Pix[idx] = 255
		}
	}
	return out
}
