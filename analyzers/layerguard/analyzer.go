// Package layerguard enforces dependency allowlists and denylists per package.
//
// For each package path matching a configured layer, it checks that all imports
// satisfy the allow/deny rules. When deny_all_others is true, only explicitly
// allowed imports (and stdlib) are permitted.
package layerguard

import (
	"strings"

	"github.com/nicegoodthings/goarch/config"
	"golang.org/x/tools/go/analysis"
)

var Analyzer = &analysis.Analyzer{
	Name:     "layerguard",
	Doc:      "enforces per-package dependency allowlists/denylists",
	Requires: []*analysis.Analyzer{config.Analyzer},
	Run:      run,
}

func run(pass *analysis.Pass) (interface{}, error) {
	cfg := pass.ResultOf[config.Analyzer].(*config.Config)
	if cfg.Rules.LayerGuard == nil {
		return nil, nil
	}

	pkg := pass.Pkg.Path()

	for pattern, rule := range cfg.Rules.LayerGuard.Layers {
		if !matchPattern(pkg, pattern) {
			continue
		}
		checkLayer(pass, rule)
		break // first match wins
	}
	return nil, nil
}

func checkLayer(pass *analysis.Pass, rule config.LayerRule) {
	for _, file := range pass.Files {
		for _, imp := range file.Imports {
			importPath := strings.Trim(imp.Path.Value, `"`)

			if isStdlib(importPath) {
				continue
			}

			if isDenied(importPath, rule) {
				pass.Reportf(imp.Pos(), "[layerguard] import %q is not allowed in this package", importPath)
			}
		}
	}
}

func isDenied(importPath string, rule config.LayerRule) bool {
	// Check explicit deny list first.
	for _, pattern := range rule.Deny {
		if matchPattern(importPath, pattern) {
			return true
		}
	}

	// If deny_all_others, check that import is in allow list.
	if rule.DenyAllOthers {
		for _, pattern := range rule.Allow {
			if matchPattern(importPath, pattern) {
				return false
			}
		}
		return true // not in allow list = denied
	}

	return false
}

func matchPattern(s, pattern string) bool {
	// Support trailing wildcard: "transport/*" matches "transport/rest"
	if strings.HasSuffix(pattern, "/*") {
		prefix := strings.TrimSuffix(pattern, "/*")
		return strings.HasPrefix(s, prefix+"/") || s == prefix
	}
	// Support ** for deep match
	if strings.HasSuffix(pattern, "/**") {
		prefix := strings.TrimSuffix(pattern, "/**")
		return strings.HasPrefix(s, prefix+"/") || s == prefix
	}
	// Exact or suffix match (for relative package paths like "internal/subprocess")
	return s == pattern || strings.HasSuffix(s, "/"+pattern)
}

func isStdlib(importPath string) bool {
	// Stdlib packages don't contain a dot in the first path element.
	first := importPath
	if i := strings.Index(importPath, "/"); i >= 0 {
		first = importPath[:i]
	}
	return !strings.Contains(first, ".")
}

