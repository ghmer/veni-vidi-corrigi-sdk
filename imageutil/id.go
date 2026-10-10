package imageutil

import (
	"fmt"
	"strings"
)

// Identifier derivation. The SDK is stateless: identifiers are pure labels
// built from the input name, the operation and a per-call running index.
// The host decides which results to keep and how to name them persistently.

const (
	// maxIDLen caps identifier length to keep result names readable.
	maxIDLen = 64
	// maxIDIndex bounds the running index; Split emits at most 12 tiles,
	// so 999 leaves generous headroom.
	maxIDIndex = 999
	// defaultName replaces empty or fully sanitized-away input names.
	defaultName = "img"
)

// SanitizeID maps an arbitrary source name to a safe identifier fragment:
// characters outside [A-Za-z0-9._-] become underscores, the length is
// capped.
func SanitizeID(name string) string {
	trimmed := stripExt(name)
	var b strings.Builder
	b.Grow(len(trimmed))
	for _, r := range trimmed {
		switch {
		case r >= 'a' && r <= 'z', r >= 'A' && r <= 'Z', r >= '0' && r <= '9', r == '-', r == '_', r == '.':
			b.WriteRune(r)
		default:
			b.WriteByte('_')
		}
	}
	out := b.String()
	if len(out) > maxIDLen {
		out = out[:maxIDLen]
	}
	return strings.Trim(out, "-_.")
}

// seedID picks the identifier fragment for a source: the sanitized input
// name, or the default when nothing survives sanitization.
func seedID(name string) string {
	if id := SanitizeID(name); id != "" {
		return id
	}
	return defaultName
}

// resultID composes the deterministic result identifier of the form
// <base>-<op><index>, e.g. "scan1-crop01".
func resultID(base, op string, index int) string {
	return fmt.Sprintf("%s-%s%02d", base, op, index)
}

// stripExt removes the file extension from name.
func stripExt(name string) string {
	lastDot := strings.LastIndexByte(name, '.')
	if lastDot <= 0 {
		return name
	}
	return name[:lastDot]
}
