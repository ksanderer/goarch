// Package authguard verifies that HTTP route registrations go through
// auth middleware, unless the route matches an exempt pattern.
//
// It works by analyzing chi router patterns: r.Route(), r.Get(), r.Post(), etc.
// Routes inside a group that calls Use(authMiddleware) are considered protected.
// Routes outside such groups are flagged unless they match exempt patterns.
package authguard

import (
	"go/ast"
	"strings"

	"github.com/ksanderer/goarch/config"
	"golang.org/x/tools/go/analysis"
	"golang.org/x/tools/go/analysis/passes/inspect"
	"golang.org/x/tools/go/ast/inspector"
)

var Analyzer = &analysis.Analyzer{
	Name:     "authguard",
	Doc:      "verifies endpoint auth middleware coverage",
	Requires: []*analysis.Analyzer{config.Analyzer, inspect.Analyzer},
	Run:      run,
}

var httpMethods = map[string]bool{
	"Get": true, "Post": true, "Put": true, "Delete": true,
	"Patch": true, "Head": true, "Options": true,
	"Handle": true, "HandleFunc": true, "Method": true, "MethodFunc": true,
}

func run(pass *analysis.Pass) (interface{}, error) {
	cfg := pass.ResultOf[config.Analyzer].(*config.Config)
	if cfg.Rules.AuthGuard == nil {
		return nil, nil
	}

	rules := cfg.Rules.AuthGuard
	pkg := pass.Pkg.Path()

	if pkg != rules.RouterPackage && !strings.HasSuffix(pkg, "/"+rules.RouterPackage) {
		return nil, nil
	}

	insp := pass.ResultOf[inspect.Analyzer].(*inspector.Inspector)

	// Walk function declarations and analyze router setup at the top level of each.
	funcFilter := []ast.Node{(*ast.FuncDecl)(nil)}
	insp.Preorder(funcFilter, func(n ast.Node) {
		fn := n.(*ast.FuncDecl)
		if fn.Body == nil {
			return
		}
		analyzeBlock(pass, fn.Body, "", rules)
	})

	return nil, nil
}

// analyzeBlock walks a block statement looking for Route() and HTTP method calls.
// It does NOT recurse into Route callback bodies — those are handled separately.
func analyzeBlock(pass *analysis.Pass, block *ast.BlockStmt, pathPrefix string, rules *config.AuthGuardConfig) {
	for _, stmt := range block.List {
		ast.Inspect(stmt, func(n ast.Node) bool {
			call, ok := n.(*ast.CallExpr)
			if !ok {
				return true
			}
			sel, ok := call.Fun.(*ast.SelectorExpr)
			if !ok {
				return true
			}

			// r.Route("/path", func(sub *Router) { ... })
			if sel.Sel.Name == "Route" && len(call.Args) >= 2 {
				path := extractStringLit(call.Args[0])
				if path == "" {
					return false // skip children
				}
				fullPath := pathPrefix + path
				if isExemptPath(fullPath, rules.ExemptPatterns) {
					return false
				}
				fnLit, ok := call.Args[1].(*ast.FuncLit)
				if !ok {
					return false
				}
				if callsUseWithAuth(fnLit.Body, rules.AuthMiddleware) {
					// Protected group — all routes inside are OK.
				} else {
					// Unprotected group — check sub-routes.
					checkUnprotectedRoutes(pass, fnLit.Body, fullPath, rules)
				}
				return false // don't recurse into Route callback
			}

			// r.Get("/path", handler) etc. — top-level in this block
			if httpMethods[sel.Sel.Name] && len(call.Args) >= 1 {
				path := extractStringLit(call.Args[0])
				if path == "" {
					return true
				}
				fullPath := pathPrefix + path
				if !isExemptPath(fullPath, rules.ExemptPatterns) {
					pass.Reportf(call.Pos(),
						"[authguard] route %s %q may not have auth middleware — add middleware or exempt in .goarch.yml",
						sel.Sel.Name, fullPath)
				}
				return false
			}

			return true
		})
	}
}

func checkUnprotectedRoutes(pass *analysis.Pass, body *ast.BlockStmt, prefix string, rules *config.AuthGuardConfig) {
	for _, stmt := range body.List {
		ast.Inspect(stmt, func(n ast.Node) bool {
			call, ok := n.(*ast.CallExpr)
			if !ok {
				return true
			}
			sel, ok := call.Fun.(*ast.SelectorExpr)
			if !ok {
				return true
			}
			if httpMethods[sel.Sel.Name] && len(call.Args) >= 1 {
				subPath := extractStringLit(call.Args[0])
				fullPath := prefix + subPath
				if !isExemptPath(fullPath, rules.ExemptPatterns) {
					pass.Reportf(call.Pos(),
						"[authguard] route %s %q has no auth middleware in its Route group",
						sel.Sel.Name, fullPath)
				}
			}
			return true
		})
	}
}

func callsUseWithAuth(body *ast.BlockStmt, authMiddleware string) bool {
	found := false
	ast.Inspect(body, func(n ast.Node) bool {
		if found {
			return false
		}
		call, ok := n.(*ast.CallExpr)
		if !ok {
			return true
		}
		sel, ok := call.Fun.(*ast.SelectorExpr)
		if !ok {
			return true
		}
		if sel.Sel.Name != "Use" {
			return true
		}
		for _, arg := range call.Args {
			src := exprToString(arg)
			if strings.Contains(src, authMiddleware) {
				found = true
				return false
			}
		}
		return true
	})
	return found
}

func extractStringLit(expr ast.Expr) string {
	lit, ok := expr.(*ast.BasicLit)
	if !ok {
		return ""
	}
	return strings.Trim(lit.Value, `"`)
}

func exprToString(expr ast.Expr) string {
	switch e := expr.(type) {
	case *ast.Ident:
		return e.Name
	case *ast.SelectorExpr:
		return exprToString(e.X) + "." + e.Sel.Name
	case *ast.CallExpr:
		return exprToString(e.Fun) + "()"
	}
	return ""
}

func isExemptPath(path string, patterns []string) bool {
	for _, p := range patterns {
		if p == path {
			return true
		}
		if strings.HasSuffix(p, "/*") {
			prefix := strings.TrimSuffix(p, "/*")
			if strings.HasPrefix(path, prefix) {
				return true
			}
		}
	}
	return false
}
