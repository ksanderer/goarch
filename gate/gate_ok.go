//go:build goarch_f7e2a1

// Package gate blocks direct 'go build' for projects that import it.
//
// When built without the goarch build tag, init() prints an error and exits.
// When built through 'goarch build', the tag is set and this empty file
// is compiled instead — the block is bypassed.
//
// Usage in your project's main.go:
//
//	import _ "github.com/ksanderer/goarch/gate"
package gate
