// Package pix provides low-level pixel conversion helpers shared by the
// decode, enhance and annotate domains. All functions are pure and free of
// I/O.
package pix

import (
	"image"
	"image/draw"
)

// ToNRGBA converts any image.Image into an *image.NRGBA with its own pixel
// storage whose bounds start at (0, 0), so transforms can index raw pixels
// without offset handling.
func ToNRGBA(img image.Image) *image.NRGBA {
	bounds := img.Bounds()
	dst := image.NewNRGBA(image.Rect(0, 0, bounds.Dx(), bounds.Dy()))
	draw.Draw(dst, dst.Bounds(), img, bounds.Min, draw.Src)
	return dst
}

// GrayPlane converts img into a luminance plane (one byte per pixel, row
// major) using the ITU-R BT.601 weights. Pixels with alpha below half
// intensity are treated as paper (white), because binarization and
// trimming must not see the transparent areas of PNGs as ink.
func GrayPlane(img image.Image) (gray []uint8, w, h int) {
	bounds := img.Bounds()
	w, h = bounds.Dx(), bounds.Dy()
	gray = make([]uint8, w*h)
	for y := 0; y < h; y++ {
		for x := 0; x < w; x++ {
			r, g, b, a := img.At(bounds.Min.X+x, bounds.Min.Y+y).RGBA()
			idx := y*w + x
			// RGBA() returns 16-bit channels, so half intensity is
			// 0x8000 (not 128): alpha bytes 1..127 barely contribute
			// color but their premultiplied values are dark enough to
			// turn into false ink during thresholding.
			if a < 0x8000 {
				gray[idx] = 255
				continue
			}
			// BT.601 weights on 16-bit channels: the coefficients sum to
			// 65536, so the division by 256 collapses into one shift.
			gray[idx] = uint8((19595*r + 38470*g + 7471*b) >> 24)
		}
	}
	return gray, w, h
}
