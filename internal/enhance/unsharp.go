package enhance

import (
	"image"
	"math"

	"github.com/disintegration/imaging"

	"github.com/ghmer/veni-vidi-corrigi-sdk/internal/pix"
)

// UnsharpMask sharpens edges with out = orig + amount * (orig - blur(orig)).
// The gaussian blur uses sigma as standard deviation in pixels; moderate
// values (sigma 0.5-1.5, amount 0.5-2) re-trace washed-out pen strokes
// without amplifying paper texture.
func UnsharpMask(img image.Image, sigma, amount float64) (*image.NRGBA, error) {
	if sigma <= 0 || amount < 0 {
		return nil, ErrInvalidParameter
	}
	if amount == 0 {
		return pix.ToNRGBA(img), nil
	}
	src := pix.ToNRGBA(img)
	blur := pix.ToNRGBA(imaging.Blur(img, sigma))
	w, h := src.Rect.Dx(), src.Rect.Dy()
	out := image.NewNRGBA(image.Rect(0, 0, w, h))
	for idx, srcIdx := 0, 0; idx < w*h; idx, srcIdx = idx+1, srcIdx+4 {
		for ch := range 3 {
			orig := float64(src.Pix[srcIdx+ch])
			spread := orig - float64(blur.Pix[srcIdx+ch])
			sharpened := orig + amount*spread
			sharpened = max(sharpened, 0)
			sharpened = min(sharpened, 255)
			out.Pix[srcIdx+ch] = uint8(math.Round(sharpened))
		}
		out.Pix[srcIdx+3] = src.Pix[srcIdx+3]
	}
	return out, nil
}
