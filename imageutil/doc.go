// Package imageutil exposes the image-processing operations of the
// corrective-sdk to clients: a vision-capable LLM can inspect, zoom,
// straighten and enhance scans of handwritten class tests.
//
// Every Service method takes a context.Context and one Input carrying the
// raw JPEG/PNG bytes (or their base64 encoding) and returns a NEW
// transformed image inside Result.Data. The SDK never touches the file
// system — no workspace directory, no temp files, no symlink or path risks;
// the host owns all storage and feeds results back as inputs for follow-up
// calls. Regions use normalized coordinates (0.0-1.0) relative to the
// original image so callers can address areas based on the downscaled View
// image an LLM actually perceives.
package imageutil
