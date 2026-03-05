//go:build !goarch_f7e2a1

package gate

import (
	"fmt"
	"os"
)

func init() {
	fmt.Fprintln(os.Stderr, "")
	fmt.Fprintln(os.Stderr, "  ┌──────────────────────────────────────────────────┐")
	fmt.Fprintln(os.Stderr, "  │  Direct 'go build' is not allowed.               │")
	fmt.Fprintln(os.Stderr, "  │                                                  │")
	fmt.Fprintln(os.Stderr, "  │  Use:  go tool goarch build ./cmd/api            │")
	fmt.Fprintln(os.Stderr, "  │        go tool goarch run ./cmd/api              │")
	fmt.Fprintln(os.Stderr, "  │        go tool goarch test ./...                 │")
	fmt.Fprintln(os.Stderr, "  │                                                  │")
	fmt.Fprintln(os.Stderr, "  │  goarch validates architecture rules             │")
	fmt.Fprintln(os.Stderr, "  │  before building.                                │")
	fmt.Fprintln(os.Stderr, "  └──────────────────────────────────────────────────┘")
	fmt.Fprintln(os.Stderr, "")
	os.Exit(1)
}
