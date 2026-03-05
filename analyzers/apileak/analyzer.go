// Package apileak prevents internal types from appearing in the signatures
// of public-facing API packages.
//
// For example, if your API package exports a function returning an internal
// executor type, this analyzer flags it.
package apileak

import (
	"go/ast"
	"go/types"
	"strings"

	"github.com/ksanderer/goarch/config"
	"golang.org/x/tools/go/analysis"
	"golang.org/x/tools/go/analysis/passes/inspect"
	"golang.org/x/tools/go/ast/inspector"
)

var Analyzer = &analysis.Analyzer{
	Name:     "apileak",
	Doc:      "prevents internal types from leaking into public API signatures",
	Requires: []*analysis.Analyzer{config.Analyzer, inspect.Analyzer},
	Run:      run,
}

func run(pass *analysis.Pass) (interface{}, error) {
	cfg := pass.ResultOf[config.Analyzer].(*config.Config)
	if cfg.Rules.APILeak == nil {
		return nil, nil
	}

	rules := cfg.Rules.APILeak
	pkg := pass.Pkg.Path()

	// Only check public packages.
	isPublicPkg := false
	for _, pub := range rules.PublicPackages {
		if pkg == pub || strings.HasSuffix(pkg, "/"+pub) {
			isPublicPkg = true
			break
		}
	}
	if !isPublicPkg {
		return nil, nil
	}

	insp := pass.ResultOf[inspect.Analyzer].(*inspector.Inspector)
	nodeFilter := []ast.Node{(*ast.FuncDecl)(nil)}

	insp.Preorder(nodeFilter, func(n ast.Node) {
		fn := n.(*ast.FuncDecl)
		if !fn.Name.IsExported() {
			return
		}

		// Check parameters.
		if fn.Type.Params != nil {
			for _, param := range fn.Type.Params.List {
				checkType(pass, param.Type, rules.BannedTypesInPublic)
			}
		}

		// Check return types.
		if fn.Type.Results != nil {
			for _, result := range fn.Type.Results.List {
				checkType(pass, result.Type, rules.BannedTypesInPublic)
			}
		}
	})

	return nil, nil
}

func checkType(pass *analysis.Pass, expr ast.Expr, banned []string) {
	t := pass.TypesInfo.TypeOf(expr)
	if t == nil {
		return
	}

	checkTypesType(pass, expr, t, banned)
}

func checkTypesType(pass *analysis.Pass, expr ast.Expr, t types.Type, banned []string) {
	switch tt := t.(type) {
	case *types.Named:
		if tt.Obj().Pkg() == nil {
			return // builtin
		}
		fullPath := tt.Obj().Pkg().Path() + "." + tt.Obj().Name()
		for _, pattern := range banned {
			if matchBanned(fullPath, pattern) {
				pass.Reportf(expr.Pos(),
					"[apileak] public API must not expose internal type %s (matches ban %q)",
					fullPath, pattern)
				return
			}
		}
	case *types.Pointer:
		checkTypesType(pass, expr, tt.Elem(), banned)
	case *types.Slice:
		checkTypesType(pass, expr, tt.Elem(), banned)
	case *types.Map:
		checkTypesType(pass, expr, tt.Key(), banned)
		checkTypesType(pass, expr, tt.Elem(), banned)
	}
}

func matchBanned(fullPath, pattern string) bool {
	// Support "internal/executor.*" style patterns.
	if strings.HasSuffix(pattern, ".*") {
		prefix := strings.TrimSuffix(pattern, ".*")
		return strings.Contains(fullPath, prefix)
	}
	return strings.Contains(fullPath, pattern)
}
