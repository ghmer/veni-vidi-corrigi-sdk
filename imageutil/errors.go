package imageutil

import (
	"errors"

	"github.com/ghmer/veni-vidi-corrigi-sdk/internal/codec"
	"github.com/ghmer/veni-vidi-corrigi-sdk/internal/geometry"
)

// Public error surface. The values are aliases of the domain errors, so
// errors.Is works no matter which layer produced the failure; each alias
// documents the situation callers will encounter it in.
var (
	// ErrInputTooLarge is returned when an input payload exceeds the
	// configured byte budget before it is decoded.
	ErrInputTooLarge = errors.New("input payload too large")
	// ErrEmptyInput is returned when an Input carries neither bytes nor a
	// base64 payload.
	ErrEmptyInput = errors.New("empty input image")
	// ErrSizeLimitExceeded is returned when neither the PNG encoding nor
	// the JPEG fallback ladder fits the configured SizeLimit.
	ErrSizeLimitExceeded = codec.ErrSizeLimitExceeded
	// ErrUnsupportedFormat is returned when an input contains neither
	// JPEG nor PNG pixel data.
	ErrUnsupportedFormat = codec.ErrUnsupportedFormat
	// ErrSourceTooLarge is returned when an input or output image exceeds
	// the pixel budget of an operation.
	ErrSourceTooLarge = codec.ErrSourceTooLarge
	// ErrInvalidImage is returned when image data is malformed.
	ErrInvalidImage = codec.ErrInvalidImage
	// ErrInvalidRegion is returned for regions that cannot describe a
	// non-empty rectangle.
	ErrInvalidRegion = geometry.ErrInvalidRegion
	// ErrInvalidParameter is returned for out-of-range operation
	// parameters (factors, window sizes, quality, ...).
	ErrInvalidParameter = errors.New("invalid parameter")
)
