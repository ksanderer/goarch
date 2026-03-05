//go:build !goarch_f7e2a1

// This file intentionally produces compile errors when built without the
// goarch build tag. Each "var _x _" line triggers a "cannot use _ as value
// or type" error, but the //line directives replace the compiler's file:line
// prefix with a human-readable banner. The result is a multi-line error
// message that clearly explains what went wrong and how to fix it.
//
// This is the only reliable way to inject a custom error message at compile
// time in Go — there is no #error pragma. The duplication across lines is
// intentional: each line produces one compiler error, and together they form
// a readable block in the terminal output.
package gate

//line [goarch] ─────────────────────────────────────────────────────────:1
var _a _
//line [goarch]  Direct 'go build' is not allowed.                       :1
var _b _
//line [goarch]                                                          :1
var _c _
//line [goarch]  Use:  go tool goarch build ./cmd/api                    :1
var _d _
//line [goarch]        go tool goarch run   ./cmd/api                    :1
var _e _
//line [goarch]        go tool goarch test  ./...                        :1
var _f _
//line [goarch]                                                          :1
var _g _
//line [goarch]  goarch validates architecture rules before building.    :1
var _h _
//line [goarch] ─────────────────────────────────────────────────────────:1
var _i _
