//go:build !goarch_f7e2a1

package gate

// go_build_is_blocked is intentionally undefined.
// Direct 'go build' is not allowed for projects using goarch.
//
// Use instead:
//   go tool goarch build ./cmd/api
//   go tool goarch run ./cmd/api
//   go tool goarch test ./...
var go_build_is_blocked = _use_go_tool_goarch_build_instead
