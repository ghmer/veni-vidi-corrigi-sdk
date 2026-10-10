package imageutil

import (
	"context"
	"encoding/base64"
	"fmt"
	"image"
	"slices"

	"github.com/disintegration/imaging"

	"github.com/ghmer/veni-vidi-corrigi-sdk/internal/codec"
)

// source bundles everything operations need to know about one input after
// decoding.
type source struct {
	// id is the sanitized identifier fragment, the seed for result IDs.
	id string
	// w, h are the pixel dimensions after EXIF orientation was applied.
	w, h        int
	format      string
	orientation int
	img         image.Image
}

// inputBytes resolves the raw payload of an Input and enforces the input
// byte budget. Setting both payload fields is rejected as ambiguous rather
// than silently preferring one; base64 payloads are rejected before
// decoding, so oversized strings cannot blow up memory during the decode
// itself.
func (s *Service) inputBytes(in Input) ([]byte, error) {
	if len(in.Data) > 0 && in.Base64 != "" {
		return nil, fmt.Errorf("%w: Input carries both Data and Base64; set exactly one", ErrInvalidParameter)
	}
	if len(in.Data) > 0 {
		if err := s.checkInputBytes(int64(len(in.Data))); err != nil {
			return nil, err
		}
		return in.Data, nil
	}
	if in.Base64 != "" {
		maxB64 := s.set.maxInputBytes/3*4 + 4
		if int64(len(in.Base64)) > maxB64 {
			return nil, fmt.Errorf("%w: base64 payload of %d characters exceeds the budget of %d",
				ErrInputTooLarge, len(in.Base64), maxB64)
		}
		data, err := decodeBase64(in.Base64)
		if err != nil {
			return nil, fmt.Errorf("%w: %w", ErrInvalidImage, err)
		}
		if err := s.checkInputBytes(int64(len(data))); err != nil {
			return nil, err
		}
		return data, nil
	}
	return nil, ErrEmptyInput
}

// checkInputBytes guards against oversized payloads before any decoding.
func (s *Service) checkInputBytes(n int64) error {
	if n > s.set.maxInputBytes {
		return fmt.Errorf("%w: %d bytes exceeds the budget of %d bytes",
			ErrInputTooLarge, n, s.set.maxInputBytes)
	}
	return nil
}

// decodeBase64 accepts the padded and unpadded standard alphabets; hosts
// commonly emit either.
func decodeBase64(s string) ([]byte, error) {
	if data, err := base64.StdEncoding.DecodeString(s); err == nil {
		return data, nil
	}
	return base64.RawStdEncoding.DecodeString(s)
}

// load decodes the source input. It applies the input byte and pixel
// budgets and enforces the context at every step.
func (s *Service) load(ctx context.Context, src Input) (source, error) {
	if err := checkCtx(ctx); err != nil {
		return source{}, err
	}
	data, err := s.inputBytes(src)
	if err != nil {
		return source{}, err
	}
	if err := checkCtx(ctx); err != nil {
		return source{}, err
	}
	dec, err := codec.Decode(data, s.set.maxPixels)
	if err != nil {
		return source{}, err
	}
	bounds := dec.Img.Bounds()
	srcData := source{
		id:          seedID(src.Name),
		w:           bounds.Dx(),
		h:           bounds.Dy(),
		format:      dec.Format,
		orientation: dec.Orientation,
		img:         dec.Img,
	}
	switch dec.Orientation {
	case 0:
		s.log.WarnContext(ctx, "EXIF orientation unreadable, decoded as stored", "src", srcData.id)
	case 1:
	default:
		s.log.DebugContext(ctx, "EXIF orientation applied",
			"src", srcData.id, "orientation", dec.Orientation)
	}
	s.log.DebugContext(ctx, "source decoded",
		"src", srcData.id, "format", srcData.format,
		"width", srcData.w, "height", srcData.h)
	return srcData, nil
}

// boundedResize is the single resampling gateway: it rejects outputs whose
// pixel count exceeds the decode budget BEFORE the target buffer (and its
// resampling pass) is allocated. Scaling a valid 50 MP input by 4 would
// otherwise allocate ~800 MP and exhaust memory.
func (s *Service) boundedResize(img image.Image, tw, th int) (image.Image, error) {
	if tw < 1 || th < 1 {
		return nil, fmt.Errorf("%w: output dimensions %dx%d must be positive", ErrInvalidParameter, tw, th)
	}
	if int64(tw)*int64(th) > s.set.maxPixels {
		return nil, fmt.Errorf(
			"%w: output %dx%d = %d pixels exceeds the budget of %d pixels; lower the resize factor or MaxSide",
			ErrSourceTooLarge, tw, th, int64(tw)*int64(th), s.set.maxPixels)
	}
	return imaging.Resize(img, tw, th, imaging.Lanczos), nil
}

// save encodes the transformed image and returns the describing Result.
// The result stays in memory; nothing is written to disk.
func (s *Service) save(ctx context.Context, src source, op string, img image.Image, opts Options, notes []string) (Result, error) {
	return s.saveIndexed(ctx, src, op, 1, img, opts, notes)
}

// saveIndexed builds the result with an explicit running index: Split uses
// 1..N for its tiles, all other operations use 1.
func (s *Service) saveIndexed(ctx context.Context, src source, op string, index int, img image.Image, opts Options, notes []string) (Result, error) {
	if err := checkCtx(ctx); err != nil {
		return Result{}, err
	}
	if index < 1 || index > maxIDIndex {
		return Result{}, fmt.Errorf("%w: result index %d must lie in [1, %d]", ErrInvalidParameter, index, maxIDIndex)
	}
	limit, quality, err := s.resolvedEncode(opts)
	if err != nil {
		return Result{}, err
	}
	id := resultID(src.id, op, index)
	enc, err := codec.Encode(img, limit, quality)
	if err != nil {
		return Result{}, err
	}
	bounds := img.Bounds()
	allNotes := notes
	if len(enc.Notes) > 0 {
		allNotes = append(slices.Clone(notes), enc.Notes...)
	}
	result := Result{
		ID:     id,
		Width:  bounds.Dx(),
		Height: bounds.Dy(),
		Format: enc.Format,
		Size:   int64(len(enc.Data)),
		Data:   enc.Data,
		Notes:  allNotes,
	}
	s.log.InfoContext(ctx, "operation complete",
		"op", op, "src", src.id, "result", id,
		"format", enc.Format, "width", result.Width, "height", result.Height,
		"bytes", result.Size, "notes", allNotes)
	return result, nil
}

// appendNote adds one note to the running note slice, skipping empties.
func appendNote(notes []string, note string) []string {
	if note == "" {
		return notes
	}
	return append(notes, note)
}

// exifNotes collects the shared EXIF note for simple operations.
func exifNotes(src source) []string {
	var notes []string
	if note := codec.DescribeOrientation(src.orientation); note != "" {
		notes = appendNote(notes, note)
	}
	return notes
}
