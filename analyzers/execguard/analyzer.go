// Package execguard bans specific package imports (or specific methods from
// packages) outside of explicitly allowed packages.
//
// Use cases: restrict os/exec to subprocess, os.Getenv to config, etc.
package execguard

import (
	"go/ast"
	"strings"

	"github.com/ksanderer/goarch/config"
	"golang.org/x/tools/go/analysis"
	"golang.org/x/tools/go/analysis/passes/inspect"
	"golang.org/x/tools/go/ast/inspector"
)

var Analyzer = &analysis.Analyzer{
	Name:     "execguard",
	Doc:      "bans package/method usage outside allowed packages",
	Requires: []*analysis.Analyzer{config.Analyzer, inspect.Analyzer},
	Run:      run,
}

func run(pass *analysis.Pass) (interface{}, error) {
	cfg := pass.ResultOf[config.Analyzer].(*config.Config)
	if cfg.Rules.ExecGuard == nil {
		return nil, nil
	}

	pkg := pass.Pkg.Path()
	rules := cfg.Rules.ExecGuard.Banned

	// Phase 1: Check import-level bans (entire package banned).
	for _, file := range pass.Files {
		for _, imp := range file.Imports {
			importPath := strings.Trim(imp.Path.Value, `"`)
			for _, ban := range rules {
				if len(ban.Methods) > 0 {
					continue // method-level ban handled in phase 2
				}
				if importPath == ban.Pkg && !isExcepted(pkg, ban.Except) {
					reason := ban.Reason
					if reason == "" {
						reason = "banned by goarch"
					}
					pass.Reportf(imp.Pos(), "[execguard] import %q is banned: %s", importPath, reason)
				}
			}
		}
	}

	// Phase 2: Check method-level bans using AST inspection.
	methodBans := filterMethodBans(rules, pkg)
	if len(methodBans) == 0 {
		return nil, nil
	}

	insp := pass.ResultOf[inspect.Analyzer].(*inspector.Inspector)
	nodeFilter := []ast.Node{(*ast.SelectorExpr)(nil)}
	insp.Preorder(nodeFilter, func(n ast.Node) {
		sel, ok := n.(*ast.SelectorExpr)
		if !ok {
			return
		}
		ident, ok := sel.X.(*ast.Ident)
		if !ok {
			return
		}
		methodName := sel.Sel.Name
		for _, ban := range methodBans {
			pkgAlias := lastElement(ban.Pkg)
			if ident.Name == pkgAlias && containsMethod(ban.Methods, methodName) {
				reason := ban.Reason
				if reason == "" {
					reason = "banned by goarch"
				}
				pass.Reportf(sel.Pos(), "[execguard] %s.%s is banned: %s", ban.Pkg, methodName, reason)
			}
		}
	})

	return nil, nil
}

func isExcepted(pkg string, exceptions []string) bool {
	// Strip .test suffix for test packages (e.g. "github.com/.../backend.test")
	cleanPkg := strings.TrimSuffix(pkg, ".test")
	for _, exc := range exceptions {
		if cleanPkg == exc || strings.HasSuffix(cleanPkg, "/"+exc) {
			return true
		}
	}
	return false
}

func filterMethodBans(rules []config.BannedImport, pkg string) []config.BannedImport {
	var result []config.BannedImport
	for _, ban := range rules {
		if len(ban.Methods) > 0 && !isExcepted(pkg, ban.Except) {
			result = append(result, ban)
		}
	}
	return result
}

func containsMethod(methods []string, name string) bool {
	for _, m := range methods {
		if m == name {
			return true
		}
	}
	return false
}

func lastElement(pkg string) string {
	if i := strings.LastIndex(pkg, "/"); i >= 0 {
		return pkg[i+1:]
	}
	return pkg
}
