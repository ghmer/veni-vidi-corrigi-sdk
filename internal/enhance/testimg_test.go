package enhance

import (
	"image"
	"image/color"
)

// solidImg returns a w x h image filled with the given color.
func solidImg(w, h int, c color.NRGBA) *image.NRGBA {
	img := image.NewNRGBA(image.Rect(0, 0, w, h))
	for i := 0; i < len(img.Pix); i += 4 {
		img.Pix[i], img.Pix[i+1], img.Pix[i+2], img.Pix[i+3] = c.R, c.G, c.B, c.A
	}
	return img
}

// rampImg returns a w x (maxY+1) image whose pixel value in row y equals y,
// the standard fixture for histogram tests.
func rampImg(w, maxY int) *image.NRGBA {
	img := image.NewNRGBA(image.Rect(0, 0, w, maxY+1))
	for y := 0; y <= maxY; y++ {
		for x := range w {
			i := (y*w + x) * 4
			img.Pix[i], img.Pix[i+1], img.Pix[i+2], img.Pix[i+3] =
				uint8(y), uint8(y), uint8(y), 255
		}
	}
	return img
}

// valueAt reads the red channel — equivalently the gray value for the
// achromatic fixtures used throughout these tests.
func valueAt(img *image.NRGBA, x, y int) uint8 {
	return img.Pix[(y*img.Rect.Dx()+x)*4]
}

// alphaAt reads the alpha channel.
func alphaAt(img *image.NRGBA, x, y int) uint8 {
	return img.Pix[(y*img.Rect.Dx()+x)*4+3]
}
