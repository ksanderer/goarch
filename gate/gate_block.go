//go:build !goarch_f7e2a1

// This file deliberately fails compilation when built without goarch.
// The error message IS the variable name — it tells the developer what to do.
package gate

var _ = ERROR__use__go_tool_goarch_build__instead_of__go_build
