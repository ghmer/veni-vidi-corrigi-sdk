package pix

import (
	"image"
	"image/color"
	"testing"
)

func TestGrayPlaneWeights(t *testing.T) {
	cases := []struct {
		name string
		c    color.RGBA
		want uint8
	}{
		{"white", color.RGBA{255, 255, 255, 255}, 255},
		{"black", color.RGBA{0, 0, 0, 255}, 0},
		{"red", color.RGBA{255, 0, 0, 255}, 76},
		{"green", color.RGBA{0, 255, 0, 255}, 150},
		{"blue", color.RGBA{0, 0, 255, 255}, 29},
	}
	for _, tt := range cases {
		t.Run(tt.name, func(t *testing.T) {
			img := image.NewRGBA(image.Rect(0, 0, 2, 2))
			for i := range img.Pix {
				img.Pix[i] = 255
			}
			colorToFill := color.RGBA{tt.c.R, tt.c.G, tt.c.B, 255}
			for i := 0; i < len(img.Pix); i += 4 {
				img.Pix[i] = colorToFill.R
				img.Pix[i+1] = colorToFill.G
				img.Pix[i+2] = colorToFill.B
				img.Pix[i+3] = 255
			}
			gray, w, h := GrayPlane(img)
			if w != 2 || h != 2 || len(gray) != 4 {
				t.Fatalf("dims = %dx%d len=%d", w, h, len(gray))
			}
			for i, v := range gray {
				if v != tt.want {
					t.Fatalf("gray[%d] = %d, want %d", i, v, tt.want)
				}
			}
		})
	}
}

func TestGrayPlaneTransparentBecomesPaper(t *testing.T) {
	img := image.NewNRGBA(image.Rect(0, 0, 1, 1)) // all zero = fully transparent black
	gray, _, _ := GrayPlane(img)
	if gray[0] != 255 {
		t.Fatalf("transparent pixel = %d, want 255", gray[0])
	}
}

func TestGrayPlaneSemiTransparentBecomesPaper(t *testing.T) {
	// Alpha byte 100 is 16-bit 25700 (< 0x8000): the pixel is invisible on
	// paper and must stay paper even though its premultiplied channels are
	// dark — the old byte-scale threshold misclassified it as visible ink.
	img := image.NewNRGBA(image.Rect(0, 0, 2, 1))
	img.Pix[0], img.Pix[1], img.Pix[2], img.Pix[3] = 0, 0, 0, 100
	img.Pix[4], img.Pix[5], img.Pix[6], img.Pix[7] = 0, 0, 0, 128 // half: visible
	gray, w, _ := GrayPlane(img)
	if w != 2 {
		t.Fatalf("w = %d, want 2", w)
	}
	if gray[0] != 255 {
		t.Fatalf("alpha 100/255 pixel = %d, want 255 (paper)", gray[0])
	}
	if gray[1] == 255 {
		t.Fatalf("alpha 128/255 pixel = %d, want visible ink", gray[1])
	}
}

func TestGrayPlaneRectOffset(t *testing.T) {
	// 5x5 image, inspect the 2x2 sub-rectangle at (3,3).
	canvas := image.NewNRGBA(image.Rect(0, 0, 5, 5))
	sub := canvas.SubImage(image.Rect(3, 3, 5, 5))
	gray, w, h := GrayPlane(sub)
	if w != 2 || h != 2 || len(gray) != 4 {
		t.Fatalf("dims = %dx%d len=%d, want 2x2", w, h, len(gray))
	}
}

func TestToNRGBANormalizesBounds(t *testing.T) {
	canvas := image.NewNRGBA(image.Rect(0, 0, 4, 4))
	sub := canvas.SubImage(image.Rect(2, 2, 4, 4))
	got := ToNRGBA(sub)
	if got.Bounds().Min != (image.Point{0, 0}) {
		t.Fatalf("bounds min = %v, want 0,0", got.Bounds().Min)
	}
	if got.Bounds().Dx() != 2 || got.Bounds().Dy() != 2 {
		t.Fatalf("dims = %dx%d, want 2x2", got.Bounds().Dx(), got.Bounds().Dy())
	}
}
