package imageutil

import "github.com/ghmer/veni-vidi-corrigi-sdk/internal/geometry"

// Region describes a rectangle in normalized coordinates (0.0-1.0) relative
// to the full size of the ORIGINAL image: X/Y is the upper-left corner, W/H
// width and height. Regions can therefore be estimated from the downscaled
// View image, independent of the actual pixel resolution.
//
// Values are tolerated and clamped into the unit square automatically
// (recorded in Result.Notes); only empty regions and non-finite values are
// rejected.
type Region = geometry.Region
