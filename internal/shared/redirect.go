package shared

import "strings"

// SafeRedirect validates that a URL is safe to redirect to. It only allows
// paths starting with a single slash (no // which bypasses origin checks)
// and no scheme (no :// for external URLs). Returns "/" for unsafe values.
func SafeRedirect(ref string) string {
	if ref == "" {
		return "/"
	}
	// Reject external URLs (contains ://)
	if idx := strings.Index(ref, "://"); idx >= 0 {
		return "/"
	}
	// Reject protocol-relative URLs (starts with //)
	if strings.HasPrefix(ref, "//") {
		return "/"
	}
	// Only allow paths starting with /
	if !strings.HasPrefix(ref, "/") {
		return "/"
	}
	return ref
}
