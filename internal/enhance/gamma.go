package enhance

import (
	"fmt"
	"image"
	"math"

	"github.com/ghmer/veni-vidi-corrigi-sdk/internal/pix"
)

// Gamma applies the power-law v -> 255 * (v/255)^g to every RGB channel via
// a lookup table. Values > 1 darken midtones (useful for faint pencil),
// values < 1 brighten.
func Gamma(img image.Image, g float64) (*image.NRGBA, error) {
	if math.IsNaN(g) || g <= 0 {
		return nil, fmt.Errorf("%w: gamma %v must be positive", ErrInvalidParameter, g)
	}
	src := pix.ToNRGBA(img)
	w, h := src.Rect.Dx(), src.Rect.Dy()
	var lut [256]uint8
	for v := range lut {
		mapped := math.Round(255 * math.Pow(float64(v)/255, g))
		lut[v] = uint8(min(mapped, 255))
	}
	out := image.NewNRGBA(image.Rect(0, 0, w, h))
	for idx, srcIdx := 0, 0; idx < w*h; idx, srcIdx = idx+1, srcIdx+4 {
		out.Pix[srcIdx+0] = lut[src.Pix[srcIdx+0]]
		out.Pix[srcIdx+1] = lut[src.Pix[srcIdx+1]]
		out.Pix[srcIdx+2] = lut[src.Pix[srcIdx+2]]
		out.Pix[srcIdx+3] = src.Pix[srcIdx+3]
	}
	return out, nil
}
