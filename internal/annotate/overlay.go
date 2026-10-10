// Package annotate draws visible region overlays onto page images so an
// LLM can verify that a planned Crop region actually covers the intended
// excerpt.
package annotate

import (
	"image"
	"image/color"
	"image/draw"
	"strings"
	"unicode"

	"golang.org/x/image/font"
	"golang.org/x/image/font/basicfont"
	"golang.org/x/image/math/fixed"

	"github.com/ghmer/veni-vidi-corrigi-sdk/internal/pix"
)

// rectColor is the high-visibility overlay color; red reads well on paper
// white and survives most downsizing.
var rectColor = color.NRGBA{255, 0, 0, 255}

// labelColor is the text color inside the label box.
var labelColor = color.NRGBA{255, 255, 255, 255}

// maxLabelLength caps the drawn label text.
const maxLabelLength = 64

// Overlay draws a rectangle outline plus an optional label onto a copy of
// img (a zero-based NRGBA). The border thickness scales with the image so
// it stays visible after downscaling. The label is rendered in the basic
// 7x13 bitmap font inside a filled box at the rectangle corner.
func Overlay(img image.Image, rect image.Rectangle, label string) *image.NRGBA {
	base := pix.ToNRGBA(img)
	bounds := base.Bounds()
	rect = rect.Intersect(bounds)
	if rect.Empty() {
		return base
	}
	thickness := borderThickness(rect)
	drawRectBorder(base, rect, thickness)
	if label != "" {
		drawLabel(base, rect, label)
	}
	return base
}

// borderThickness scales the border with the shorter rectangle side,
// clamped to [2, 6] px.
func borderThickness(rect image.Rectangle) int {
	shortest := min(rect.Dx(), rect.Dy())
	thickness := shortest / 100
	thickness = max(thickness, 2)
	return min(thickness, 6)
}

// drawRectBorder strokes the border of rect in rectColor.
func drawRectBorder(dst *image.NRGBA, rect image.Rectangle, thickness int) {
	top := image.Rect(rect.Min.X, rect.Min.Y, rect.Max.X, rect.Min.Y+thickness)
	bottom := image.Rect(rect.Min.X, rect.Max.Y-thickness, rect.Max.X, rect.Max.Y)
	left := image.Rect(rect.Min.X, rect.Min.Y, rect.Min.X+thickness, rect.Max.Y)
	right := image.Rect(rect.Max.X-thickness, rect.Min.Y, rect.Max.X, rect.Max.Y)
	for _, r := range []image.Rectangle{top, bottom, left, right} {
		draw.Draw(dst, r, image.NewUniform(rectColor), image.Point{}, draw.Src)
	}
}

// drawLabel paints a filled label box with white text inside the top-left
// corner of the rectangle.
func drawLabel(dst *image.NRGBA, rect image.Rectangle, label string) {
	text := sanitizeLabel(label)
	if text == "" {
		return
	}
	const (
		glyphWidth  = 7
		glyphHeight = 13
		padding     = 2
	)
	boxW := len(text)*glyphWidth + 2*padding
	boxH := glyphHeight + 2*padding
	box := image.Rect(rect.Min.X, rect.Min.Y, rect.Min.X+boxW, rect.Min.Y+boxH)
	box = box.Intersect(dst.Bounds())
	draw.Draw(dst, box, image.NewUniform(rectColor), image.Point{}, draw.Src)

	d := &font.Drawer{
		Dst:  dst,
		Src:  image.NewUniform(labelColor),
		Face: basicfont.Face7x13,
		Dot:  fixed.P(box.Min.X+padding, box.Min.Y+glyphHeight+padding-1),
	}
	d.DrawString(text)
}

// sanitizeLabel strips runes the basic font cannot render (non-ASCII,
// control characters) and caps the length.
func sanitizeLabel(label string) string {
	var b strings.Builder
	for _, r := range label {
		if r > unicode.MaxASCII || unicode.IsControl(r) {
			continue
		}
		b.WriteRune(r)
		if b.Len() >= maxLabelLength {
			break
		}
	}
	return b.String()
}
