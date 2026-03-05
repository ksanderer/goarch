// Package errguard restricts where custom error types can be defined.
// Error types (types implementing the error interface) should be centralized
// in designated packages to prevent scattered error definitions.
package errguard

import (
	"go/ast"
	"strings"

	"github.com/ksanderer/goarch/config"
	"golang.org/x/tools/go/analysis"
	"golang.org/x/tools/go/analysis/passes/inspect"
	"golang.org/x/tools/go/ast/inspector"
)

var Analyzer = &analysis.Analyzer{
	Name:     "errguard",
	Doc:      "restricts where custom error types can be defined",
	Requires: []*analysis.Analyzer{config.Analyzer, inspect.Analyzer},
	Run:      run,
}

func run(pass *analysis.Pass) (interface{}, error) {
	cfg := pass.ResultOf[config.Analyzer].(*config.Config)
	if cfg.Rules.ErrGuard == nil || len(cfg.Rules.ErrGuard.AllowedPackages) == 0 {
		return nil, nil
	}

	pkg := pass.Pkg.Path()

	// Skip allowed packages.
	for _, allowed := range cfg.Rules.ErrGuard.AllowedPackages {
		if pkg == allowed || strings.HasSuffix(pkg, "/"+allowed) {
			return nil, nil
		}
	}

	insp := pass.ResultOf[inspect.Analyzer].(*inspector.Inspector)

	// Collect type names that have an Error() string method.
	errorTypes := make(map[string]bool)
	funcFilter := []ast.Node{(*ast.FuncDecl)(nil)}
	insp.Preorder(funcFilter, func(n ast.Node) {
		fn := n.(*ast.FuncDecl)
		if fn.Recv == nil || len(fn.Recv.List) == 0 {
			return
		}
		if fn.Name.Name != "Error" {
			return
		}
		// Check signature: Error() string
		if fn.Type.Params != nil && fn.Type.Params.NumFields() > 0 {
			return
		}
		if fn.Type.Results == nil || fn.Type.Results.NumFields() != 1 {
			return
		}
		retType, ok := fn.Type.Results.List[0].Type.(*ast.Ident)
		if !ok || retType.Name != "string" {
			return
		}
		typeName := receiverTypeName(fn.Recv.List[0].Type)
		if typeName != "" {
			errorTypes[typeName] = true
		}
	})

	if len(errorTypes) == 0 {
		return nil, nil
	}

	// Find the type declarations for error types and report them.
	typeFilter := []ast.Node{(*ast.TypeSpec)(nil)}
	insp.Preorder(typeFilter, func(n ast.Node) {
		ts := n.(*ast.TypeSpec)
		if errorTypes[ts.Name.Name] {
			pass.Reportf(ts.Pos(),
				"[errguard] error type %s should be defined in one of: %s",
				ts.Name.Name, strings.Join(cfg.Rules.ErrGuard.AllowedPackages, ", "))
		}
	})

	return nil, nil
}

func receiverTypeName(expr ast.Expr) string {
	switch t := expr.(type) {
	case *ast.Ident:
		return t.Name
	case *ast.StarExpr:
		return receiverTypeName(t.X)
	}
	return ""
}
