// Package match provides wildcard pattern matching for package paths.
//
// Supported patterns:
//   - "foo/bar"    — exact match or suffix match (matches "example.com/foo/bar")
//   - "foo/bar/*"  — matches foo/bar and any direct child (foo/bar/baz but not foo/bar/baz/qux)
//   - "foo/bar/**" — matches foo/bar and any descendant (foo/bar/baz/qux)
package match

import "strings"

// Pattern checks whether s matches the given pattern.
func Pattern(s, pattern string) bool {
	if strings.HasSuffix(pattern, "/**") {
		prefix := strings.TrimSuffix(pattern, "/**")
		return s == prefix || strings.HasPrefix(s, prefix+"/")
	}
	if strings.HasSuffix(pattern, "/*") {
		prefix := strings.TrimSuffix(pattern, "/*")
		if s == prefix || strings.HasSuffix(s, "/"+prefix) {
			return true
		}
		// Full prefix match
		if strings.HasPrefix(s, prefix+"/") {
			rest := s[len(prefix)+1:]
			return !strings.Contains(rest, "/")
		}
		// Suffix prefix match: e.g. pattern "gateway/cmd/*" matching "example.com/gateway/cmd/keygen"
		sfx := "/" + prefix + "/"
		if idx := strings.Index(s, sfx); idx >= 0 {
			rest := s[idx+len(sfx):]
			return !strings.Contains(rest, "/")
		}
		return false
	}
	return s == pattern || strings.HasSuffix(s, "/"+pattern)
}
