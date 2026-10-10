package imageutil

import "encoding/base64"

// Input carries the source image of one operation call: the raw JPEG/PNG
// bytes in Data, or their base64 encoding in Base64 for JSON-based
// transports that hand the image over as a string. Exactly one payload
// field may be set — an Input with both is rejected as ambiguous.
// Name optionally seeds result identifiers ("scan1" yields IDs like
// "scan1-crop01"); an empty name falls back to "img".
type Input struct {
	// Data holds the raw JPEG or PNG bytes.
	Data []byte
	// Base64 holds the base64-encoded image (standard alphabet, padding
	// optional); used when Data is empty.
	Base64 string
	// Name is the optional source identifier; it only labels results and is
	// never resolved against the file system.
	Name string
}

// Result describes one transformed image: the encoding bytes to display,
// store or feed into follow-up calls, plus the metadata an LLM needs to
// reason about the operation.
type Result struct {
	// ID is the deterministic result identifier (<name>-<op><index>); it
	// also serves as the Input.Name for chained follow-up calls.
	ID string `json:"id"`
	// Width and Height are the pixel dimensions of the transformed image.
	// For Info, which reports metadata only, they are the EXIF-corrected
	// canonical dimensions of the source (from v0.1.1 on): the stored
	// values transposed for orientations 5-8, matching what every
	// decoding operation reports.
	Width  int `json:"width"`
	Height int `json:"height"`
	// Orientation is the numeric EXIF orientation of the source as
	// reported by Info: 1..8 for a readable JPEG EXIF block, 1 for PNG
	// (PNG carries no EXIF) and 0 when a JPEG EXIF block exists but is
	// unreadable. Operation results leave it unset at 0; their pixels
	// are already EXIF-corrected.
	Orientation int `json:"orientation,omitempty"`
	// Format is the canonical encoding ("png" or "jpeg").
	Format string `json:"format"`
	// Size is len(Data).
	Size int64 `json:"size"`
	// Data holds the encoded result bytes (absent on Info, which reports
	// metadata only). Go's JSON encoder serializes []byte as base64.
	Data []byte `json:"data,omitempty"`
	// Notes carries warnings worth surfacing to an LLM (clamping, upscale
	// caps, JPEG fallbacks, EXIF orientation).
	Notes []string `json:"notes,omitempty"`
}

// Base64 returns the result bytes base64-encoded, for transports that
// require strings instead of binary fields.
func (r Result) Base64() string { return base64.StdEncoding.EncodeToString(r.Data) }

// Options tighten the encoding side of a single operation. Zero fields fall
// back to the Service defaults.
type Options struct {
	// MaxSide caps the longest output edge in pixels. The service default
	// is 1600 (model-friendly); anything above 4000 is capped with a note.
	MaxSide int
	// SizeLimit caps the output size in bytes as a hard limit: when the
	// PNG exceeds it, a JPEG fallback (requested quality, then a fixed
	// lower-quality ladder) must fit or the operation fails with
	// ErrSizeLimitExceeded. The service default is 4 MiB.
	SizeLimit int64
	// Quality is the JPEG fallback quality (1..100); 0 uses the service
	// default of 90.
	Quality int
}

// ThreshMethod selects the binarization algorithm for Threshold.
type ThreshMethod string

const (
	// ThreshGlobal is Otsu's method — the choice for flat scans with even
	// lighting.
	ThreshGlobal ThreshMethod = "global"
	// ThreshAdaptive is Sauvola's method — the choice for photos with
	// lighting gradients and paper texture.
	ThreshAdaptive ThreshMethod = "adaptive"
)

// SplitMode selects how Split divides a page.
type SplitMode string

const (
	// SplitTopBottom splits into two horizontal halves (upper, lower).
	SplitTopBottom SplitMode = "top-bottom"
	// SplitLeftRight splits into two vertical halves (left, right).
	SplitLeftRight SplitMode = "left-right"
	// SplitGrid2x2 splits into four quadrants.
	SplitGrid2x2 SplitMode = "grid2x2"
	// SplitTiles splits into the cols x rows grid passed next to the mode.
	SplitTiles SplitMode = "tiles"
)
