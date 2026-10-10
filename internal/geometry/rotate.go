package geometry

import (
	"image"

	"github.com/ghmer/veni-vidi-corrigi-sdk/internal/pix"
)

// Pixel orientation transforms. All functions accept any image.Image and
// return a zero-based *image.NRGBA; they are exact pixel permutations (no
// interpolation), so they are lossless.
//
// Source coordinates are (col, row) with col in [0,w) and row in [0,h);
// destination dimensions are (dw, dh).

// Rotate90 rotates the image 90 degrees clockwise: the left source column
// becomes the top destination row.
func Rotate90(img image.Image) *image.NRGBA {
	sw, sh := dims(img)
	return transform(img, sh, sw, func(dx, dy int) (int, int) {
		return dy, sh - 1 - dx
	})
}

// Rotate270 rotates the image 90 degrees counter-clockwise.
func Rotate270(img image.Image) *image.NRGBA {
	sw, sh := dims(img)
	return transform(img, sh, sw, func(dx, dy int) (int, int) {
		return sw - 1 - dy, dx
	})
}

// Rotate180 rotates the image by 180 degrees.
func Rotate180(img image.Image) *image.NRGBA {
	sw, sh := dims(img)
	return transform(img, sw, sh, func(dx, dy int) (int, int) {
		return sw - 1 - dx, sh - 1 - dy
	})
}

// FlipH mirrors the image horizontally (left-right).
func FlipH(img image.Image) *image.NRGBA {
	sw, sh := dims(img)
	return transform(img, sw, sh, func(dx, dy int) (int, int) {
		return sw - 1 - dx, dy
	})
}

// FlipV mirrors the image vertically (top-bottom).
func FlipV(img image.Image) *image.NRGBA {
	sw, sh := dims(img)
	return transform(img, sw, sh, func(dx, dy int) (int, int) {
		return dx, sh - 1 - dy
	})
}

// Transpose mirrors the image across its main diagonal.
func Transpose(img image.Image) *image.NRGBA {
	sw, sh := dims(img)
	return transform(img, sh, sw, func(dx, dy int) (int, int) {
		return dy, dx
	})
}

// Transverse mirrors the image across its anti-diagonal.
func Transverse(img image.Image) *image.NRGBA {
	sw, sh := dims(img)
	return transform(img, sh, sw, func(dx, dy int) (int, int) {
		return sw - 1 - dy, sh - 1 - dx
	})
}

// transform permutes the pixels of img into a dw x dh destination. srcXY
// maps each destination pixel to its source coordinate, which the permutation
// definitions above guarantee to be inside the source bounds.
func transform(img image.Image, dw, dh int, srcXY func(dx, dy int) (sx, sy int)) *image.NRGBA {
	src := pix.ToNRGBA(img)
	dst := image.NewNRGBA(image.Rect(0, 0, dw, dh))
	for dy := range dh {
		for dx := range dw {
			sx, sy := srcXY(dx, dy)
			copyPixel(dst, dx, dy, src, sx, sy)
		}
	}
	return dst
}

// copyPixel moves one NRGBA pixel from src(sx, sy) to dst(dx, dy); all
// images are zero-based, so the offset math is a linear index.
func copyPixel(dst *image.NRGBA, dx, dy int, src *image.NRGBA, sx, sy int) {
	si := sy*src.Stride + sx*4
	di := dy*dst.Stride + dx*4
	dst.Pix[di+0] = src.Pix[si+0]
	dst.Pix[di+1] = src.Pix[si+1]
	dst.Pix[di+2] = src.Pix[si+2]
	dst.Pix[di+3] = src.Pix[si+3]
}

// dims returns the normalized dimensions of any image.
func dims(img image.Image) (w, h int) {
	bounds := img.Bounds()
	return bounds.Dx(), bounds.Dy()
}
