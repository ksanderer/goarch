package match

import "testing"

func TestPattern(t *testing.T) {
	tests := []struct {
		s, pattern string
		want       bool
	}{
		// Exact match
		{"foo/bar", "foo/bar", true},
		{"foo/baz", "foo/bar", false},

		// Suffix match
		{"example.com/foo/bar", "foo/bar", true},
		{"example.com/foo/bar2", "foo/bar", false},

		// Single-level wildcard
		{"foo/bar", "foo/*", true},
		{"foo/bar/baz", "foo/*", false},
		{"foo", "foo/*", true},
		{"example.com/gateway/cmd/keygen", "gateway/cmd/*", true},
		{"example.com/gateway/cmd/keygen/sub", "gateway/cmd/*", false},

		// Deep wildcard
		{"foo/bar", "foo/**", true},
		{"foo/bar/baz", "foo/**", true},
		{"foo/bar/baz/qux", "foo/**", true},
		{"foo", "foo/**", true},
		{"foobar/x", "foo/**", false},

		// Full path wildcards
		{"example.com/gateway/cmd", "example.com/gateway/cmd/*", true},
		{"example.com/gateway/cmd/keygen", "example.com/gateway/cmd/*", true},
		{"example.com/gateway/cmd/keygen/sub", "example.com/gateway/cmd/*", false},
		{"example.com/gateway/cmd/keygen/sub", "example.com/gateway/cmd/**", true},
	}
	for _, tt := range tests {
		t.Run(tt.s+"_"+tt.pattern, func(t *testing.T) {
			got := Pattern(tt.s, tt.pattern)
			if got != tt.want {
				t.Errorf("Pattern(%q, %q) = %v, want %v", tt.s, tt.pattern, got, tt.want)
			}
		})
	}
}
