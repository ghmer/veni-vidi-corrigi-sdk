package imageutil

// Enhancement domain: making handwriting readable — grayscale, contrast,
// gamma, binarization, sharpening and denoising.

import (
	"context"
	"fmt"
	"image"
	"math"

	"github.com/disintegration/imaging"

	"github.com/ghmer/veni-vidi-corrigi-sdk/internal/enhance"
	"github.com/ghmer/veni-vidi-corrigi-sdk/internal/pix"
)

// adjustRange bounds manual contrast/brightness percent values.
const adjustRange = 100.0

// maxGamma bounds the gamma exponent against useless extremes.
const maxGamma = 10.0

// maxSharpenSigma and maxSharpenAmount bound the unsharp mask parameters.
const (
	maxSharpenSigma  = 5.0
	maxSharpenAmount = 5.0
)

// Grayscale converts the source to grayscale: fewer distractions, smaller
// results, pure stroke contrast. Pencil and blue ink both turn dark and
// readable.
func (s *Service) Grayscale(ctx context.Context, src Input) (Result, error) {
	if err := checkCtx(ctx); err != nil {
		return Result{}, err
	}
	srcData, err := s.load(ctx, src)
	if err != nil {
		return Result{}, err
	}
	out := imaging.Grayscale(srcData.img)
	return s.save(ctx, srcData, "gray", out, Options{}, exifNotes(srcData))
}

// Invert flips to a negative; useful for rare dark-background captures.
func (s *Service) Invert(ctx context.Context, src Input) (Result, error) {
	if err := checkCtx(ctx); err != nil {
		return Result{}, err
	}
	srcData, err := s.load(ctx, src)
	if err != nil {
		return Result{}, err
	}
	out := imaging.Invert(srcData.img)
	return s.save(ctx, srcData, "inv", out, Options{}, exifNotes(srcData))
}

// AutoContrast stretches the luminance histogram so clipPercent percent of
// the darkest and brightest pixels saturate. Deterministic and robust —
// usually the first thing to try on faint handwriting or color casts.
func (s *Service) AutoContrast(ctx context.Context, src Input, clipPercent float64) (Result, error) {
	if err := checkCtx(ctx); err != nil {
		return Result{}, err
	}
	if math.IsNaN(clipPercent) || clipPercent < 0 || clipPercent > 5 {
		return Result{}, fmt.Errorf("%w: clipPercent %.3f must lie in [0, 5]", ErrInvalidParameter, clipPercent)
	}
	srcData, err := s.load(ctx, src)
	if err != nil {
		return Result{}, err
	}
	notes := append(exifNotes(srcData), fmt.Sprintf("autocontrast clip %.2f%%", clipPercent))
	out := enhance.AutoContrast(srcData.img, clipPercent)
	return s.save(ctx, srcData, "ac", out, Options{}, notes)
}

// AdjustContrast applies a manual contrast change (-100..100) as percentages;
// prefer AutoContrast when possible, the LLM cannot estimate manual values
// well.
func (s *Service) AdjustContrast(ctx context.Context, src Input, amount float64) (Result, error) {
	if err := checkCtx(ctx); err != nil {
		return Result{}, err
	}
	if math.IsNaN(amount) || math.Abs(amount) > adjustRange {
		return Result{}, fmt.Errorf("%w: amount %.3g must lie in [-100, 100]", ErrInvalidParameter, amount)
	}
	srcData, err := s.load(ctx, src)
	if err != nil {
		return Result{}, err
	}
	out := imaging.AdjustContrast(srcData.img, amount)
	notes := append(exifNotes(srcData), fmt.Sprintf("contrast %+.1f", amount))
	return s.save(ctx, srcData, "contrast", out, Options{}, notes)
}

// AdjustBrightness applies a manual brightness change (-100..100).
func (s *Service) AdjustBrightness(ctx context.Context, src Input, amount float64) (Result, error) {
	if err := checkCtx(ctx); err != nil {
		return Result{}, err
	}
	if math.IsNaN(amount) || math.Abs(amount) > adjustRange {
		return Result{}, fmt.Errorf("%w: amount %.3g must lie in [-100, 100]", ErrInvalidParameter, amount)
	}
	srcData, err := s.load(ctx, src)
	if err != nil {
		return Result{}, err
	}
	out := imaging.AdjustBrightness(srcData.img, amount)
	notes := append(exifNotes(srcData), fmt.Sprintf("brightness %+.1f", amount))
	return s.save(ctx, srcData, "bright", out, Options{}, notes)
}

// Gamma applies the power-law v -> 255*(v/255)^g: values above 1 darken
// midtones (good for faint pencil), values below 1 brighten.
func (s *Service) Gamma(ctx context.Context, src Input, g float64) (Result, error) {
	if err := checkCtx(ctx); err != nil {
		return Result{}, err
	}
	if badGamma(g) {
		return Result{}, fmt.Errorf("%w: gamma %.3g must lie in (0, %.1f]", ErrInvalidParameter, g, maxGamma)
	}
	srcData, err := s.load(ctx, src)
	if err != nil {
		return Result{}, err
	}
	out, err := enhance.Gamma(srcData.img, g)
	if err != nil {
		// Defensive path: the facade has already validated g.
		return Result{}, err
	}
	notes := append(exifNotes(srcData), fmt.Sprintf("gamma %.2f", g))
	return s.save(ctx, srcData, "gamma", out, Options{}, notes)
}

// badGamma reports out-of-range gamma values.
func badGamma(g float64) bool {
	return math.IsNaN(g) || math.IsInf(g, 0) || g <= 0 || g > maxGamma
}

// Sharpen re-traces washed-out edges with an unsharp mask
// (out = orig + amount * (orig - blur(orig, sigma))). Useful for pencil
// strokes and slightly soft phone photos.
func (s *Service) Sharpen(ctx context.Context, src Input, sigma, amount float64) (Result, error) {
	if err := checkCtx(ctx); err != nil {
		return Result{}, err
	}
	if badSigma(sigma) || badAmount(amount) {
		return Result{}, fmt.Errorf("%w: sigma must lie in (0, %.1f] and amount in [0, %.1f]",
			ErrInvalidParameter, maxSharpenSigma, maxSharpenAmount)
	}
	srcData, err := s.load(ctx, src)
	if err != nil {
		return Result{}, err
	}
	out, err := enhance.UnsharpMask(srcData.img, sigma, amount)
	if err != nil {
		return Result{}, err
	}
	notes := append(exifNotes(srcData), fmt.Sprintf("unsharp sigma %.2f amount %.2f", sigma, amount))
	return s.save(ctx, srcData, "sharp", out, Options{}, notes)
}

func badSigma(sigma float64) bool {
	return math.IsNaN(sigma) || math.IsInf(sigma, 0) || sigma <= 0 || sigma > maxSharpenSigma
}

func badAmount(amount float64) bool {
	return math.IsNaN(amount) || amount < 0 || amount > maxSharpenAmount
}

// Denoise applies a median filter of radius 1-3 px to remove JPEG artifacts
// and sensor grain. The median keeps stroke edges that gaussian smoothers
// would smear. Apply to excerpts, not pages.
func (s *Service) Denoise(ctx context.Context, src Input, radius int) (Result, error) {
	if err := checkCtx(ctx); err != nil {
		return Result{}, err
	}
	if radius < 1 || radius > 3 {
		return Result{}, fmt.Errorf("%w: radius %d must lie in [1, 3]", ErrInvalidParameter, radius)
	}
	srcData, err := s.load(ctx, src)
	if err != nil {
		return Result{}, err
	}
	out, err := enhance.MedianFilter(srcData.img, radius)
	if err != nil {
		return Result{}, err
	}
	notes := append(exifNotes(srcData), fmt.Sprintf("median radius %d", radius))
	return s.save(ctx, srcData, "denoise", out, Options{}, notes)
}

// Threshold binarizes the source into ink-black on paper-white.
// ThreshGlobal (Otsu) suits flat scans; ThreshAdaptive (Sauvola) suits
// photos with lighting gradients. Binarization belongs to excerpts with an
// unambiguous ink/paper split — for full pages prefer Crops first.
//
// window controls the Sauvola neighborhood (odd, 3..501, in pixels);
// window = 0 picks an automatic window of ~3% of the shorter edge. It is
// ignored for the global method.
func (s *Service) Threshold(ctx context.Context, src Input, method ThreshMethod, window int) (Result, error) {
	if err := checkCtx(ctx); err != nil {
		return Result{}, err
	}
	srcData, err := s.load(ctx, src)
	if err != nil {
		return Result{}, err
	}
	notes := exifNotes(srcData)
	var out image.Image
	switch method {
	case ThreshGlobal:
		if window != 0 {
			notes = appendNote(notes, fmt.Sprintf("window parameter ignored for %q thresholding", ThreshGlobal))
		}
		out, notes, err = s.globalThreshold(srcData, notes)
	case ThreshAdaptive:
		out, notes, err = s.adaptiveThreshold(srcData, window, notes)
	default:
		return Result{}, fmt.Errorf(
			"%w: method %q must be %q or %q",
			ErrInvalidParameter, method, ThreshGlobal, ThreshAdaptive)
	}
	if err != nil {
		return Result{}, err
	}
	return s.save(ctx, srcData, "thresh", out, Options{}, notes)
}

// globalThreshold computes the Otsu threshold on the gray plane and
// binarizes with it.
func (s *Service) globalThreshold(srcData source, notes []string) (image.Image, []string, error) {
	gray, w, h := pix.GrayPlane(srcData.img)
	hist := enhance.Histogram(gray)
	threshold, err := enhance.Otsu(hist)
	if err != nil {
		return nil, notes, err
	}
	notes = appendNote(notes, fmt.Sprintf("otsu threshold %d", threshold))
	return enhance.Binarize(gray, w, h, threshold), notes, nil
}

// adaptiveThreshold computes the Sauvola binarization with the requested or
// automatic window.
func (s *Service) adaptiveThreshold(srcData source, window int, notes []string) (image.Image, []string, error) {
	if err := adaptiveMaxGuard(srcData.w*srcData.h, s.set.binPixels); err != nil {
		return nil, notes, err
	}
	switch {
	case window == 0:
		window = autoWindow(min(srcData.w, srcData.h))
		notes = appendNote(notes, fmt.Sprintf("sauvola window %d (automatic)", window))
	case window >= 3 && window <= 501 && window%2 == 1:
		notes = appendNote(notes, fmt.Sprintf("sauvola window %d", window))
	default:
		return nil, notes, fmt.Errorf("%w: window %d must be an odd number in [3, 501] or 0 for automatic", ErrInvalidParameter, window)
	}
	gray, w, h := pix.GrayPlane(srcData.img)
	bin, err := enhance.Sauvola(gray, w, h, window)
	if err != nil {
		return nil, notes, err
	}
	return bin, notes, nil
}
