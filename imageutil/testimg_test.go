package imageutil

import (
	"bytes"
	"encoding/binary"
	"image"
	"image/jpeg"
	"image/png"
	"log/slog"
	"strings"
	"testing"
)

// discardLogger silences the service logs in tests.
func discardLogger() *slog.Logger {
	return slog.New(slog.DiscardHandler)
}

// newTestService builds a service with quiet logging and default settings.
func newTestService(t *testing.T) *Service {
	t.Helper()
	s, err := New(WithLogger(discardLogger()))
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	return s
}

func encodePNGBytes(img image.Image) []byte {
	var buf bytes.Buffer
	if err := png.Encode(&buf, img); err != nil {
		panic("png encode: " + err.Error()) // test fixture: cannot fail for valid images
	}
	return buf.Bytes()
}

func encodeJPEGBytes(img image.Image) []byte {
	var buf bytes.Buffer
	if err := jpeg.Encode(&buf, img, &jpeg.Options{Quality: 92}); err != nil {
		panic("jpeg encode: " + err.Error()) // test fixture: cannot fail for valid images
	}
	return buf.Bytes()
}

// pngInput wraps an image as raw PNG Input with the given identifier.
func pngInput(name string, img image.Image) Input {
	return Input{Data: encodePNGBytes(img), Name: name}
}

// bytesInput wraps raw bytes as Input — for corrupt or adversarial fixtures.
func bytesInput(name string, data []byte) Input {
	return Input{Data: data, Name: name}
}

// solidPage returns a w x h white page.
func solidPage(w, h int) *image.NRGBA {
	return solidPageRGB(w, h, 255, 255, 255)
}

// solidPageRGB returns a w x h page of one solid color.
func solidPageRGB(w, h int, r, g, b uint8) *image.NRGBA {
	img := image.NewNRGBA(image.Rect(0, 0, w, h))
	for i := 0; i < len(img.Pix); i += 4 {
		img.Pix[i], img.Pix[i+1], img.Pix[i+2], img.Pix[i+3] = r, g, b, 255
	}
	return img
}

// paintInk colors the rectangle pixels black on the page.
func paintInk(img *image.NRGBA, rect image.Rectangle) {
	dx := img.Bounds().Dx()
	for y := rect.Min.Y; y < rect.Max.Y; y++ {
		for x := rect.Min.X; x < rect.Max.X; x++ {
			i := (y*dx + x) * 4
			img.Pix[i], img.Pix[i+1], img.Pix[i+2] = 0, 0, 0
		}
	}
}

// imageRect mirrors image.Rect to keep test tables compact.
func imageRect(x0, y0, x1, y1 int) image.Rectangle {
	return image.Rect(x0, y0, x1, y1)
}

// lumAt computes the BT.601 luminance of a decoded pixel.
func lumAt(img image.Image, x, y int) int {
	r, g, b, _ := img.At(x, y).RGBA()
	return int((19595*r + 38470*g + 7471*b) >> 24)
}

// inkSquare paints a black square with normalized coordinates on the page —
// the handwriting stand-in.
func inkSquare(img *image.NRGBA, region Region) {
	paintInk(img, region.PixelRect(img.Bounds().Dx(), img.Bounds().Dy()))
}

// hasInk reports whether the result image contains dark ink pixels.
func hasInk(t *testing.T, r Result) bool {
	t.Helper()
	img := decodeResult(t, r)
	return imgMinLum(img) < 64
}

// decodeResult decodes the encoded bytes of a Result.
func decodeResult(t *testing.T, r Result) image.Image {
	t.Helper()
	imageData, _, err := image.Decode(bytes.NewReader(r.Data))
	if err != nil {
		t.Fatalf("decode result %s: %v", r.ID, err)
	}
	return imageData
}

func imgMinLum(img image.Image) int {
	bounds := img.Bounds()
	minimum := 255
	for y := bounds.Min.Y; y < bounds.Max.Y; y++ {
		for x := bounds.Min.X; x < bounds.Max.X; x++ {
			r, g, b, _ := img.At(x, y).RGBA()
			lum := int((19595*r + 38470*g + 7471*b) >> 24)
			minimum = min(minimum, lum)
		}
	}
	return minimum
}

// jpegWithExif wraps a JPEG payload + APP1 EXIF orientation segment (same
// fixture technique as internal/codec's tests but local to this package).
//
//nolint:gosec // test fixture: narrowing conversions on small controlled values
func jpegWithExif(t *testing.T, img image.Image, orient uint16) []byte {
	t.Helper()
	base := encodeJPEGBytes(img)
	tiff := []byte("II")
	tiff = binary.LittleEndian.AppendUint16(tiff, 42)
	tiff = binary.LittleEndian.AppendUint32(tiff, 8)
	tiff = binary.LittleEndian.AppendUint16(tiff, 1)
	tiff = binary.LittleEndian.AppendUint16(tiff, 0x0112)
	tiff = binary.LittleEndian.AppendUint16(tiff, 3)
	tiff = binary.LittleEndian.AppendUint32(tiff, 1)
	tiff = binary.LittleEndian.AppendUint16(tiff, orient)
	tiff = binary.LittleEndian.AppendUint16(tiff, 0)
	tiff = binary.LittleEndian.AppendUint32(tiff, 0)
	payload := append([]byte("Exif\x00\x00"), tiff...)
	out := append([]byte{0xFF, 0xD8}, 0xFF, 0xE1)
	out = binary.BigEndian.AppendUint16(out, uint16(len(payload)+2))
	out = append(out, payload...)
	return append(out, base[2:]...)
}

// errContains verifies that err exists and mentions text.
func errContains(t *testing.T, err error, text string) {
	t.Helper()
	if err == nil {
		t.Fatalf("expected error containing %q", text)
	}
	if !strings.Contains(err.Error(), text) {
		t.Fatalf("error %v does not contain %q", err, text)
	}
}
