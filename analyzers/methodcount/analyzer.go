// Package methodcount limits the number of exported methods per named type,
// preventing god-objects with oversized interfaces.
package methodcount

import (
	"go/ast"
	"go/token"

	"github.com/nicegoodthings/goarch/config"
	"golang.org/x/tools/go/analysis"
	"golang.org/x/tools/go/analysis/passes/inspect"
	"golang.org/x/tools/go/ast/inspector"
)

var Analyzer = &analysis.Analyzer{
	Name:     "methodcount",
	Doc:      "limits the number of exported methods per type",
	Requires: []*analysis.Analyzer{config.Analyzer, inspect.Analyzer},
	Run:      run,
}

func run(pass *analysis.Pass) (interface{}, error) {
	cfg := pass.ResultOf[config.Analyzer].(*config.Config)
	if cfg.Rules.MethodCount == nil || cfg.Rules.MethodCount.MaxPublicMethods <= 0 {
		return nil, nil
	}

	max := cfg.Rules.MethodCount.MaxPublicMethods

	insp := pass.ResultOf[inspect.Analyzer].(*inspector.Inspector)

	// Collect exported methods per receiver type name.
	counts := make(map[string]int)
	positions := make(map[string]token.Pos) // first method position per type

	nodeFilter := []ast.Node{(*ast.FuncDecl)(nil)}
	insp.Preorder(nodeFilter, func(n ast.Node) {
		fn := n.(*ast.FuncDecl)
		if fn.Recv == nil || len(fn.Recv.List) == 0 {
			return // not a method
		}
		if !fn.Name.IsExported() {
			return
		}
		typeName := receiverTypeName(fn.Recv.List[0].Type)
		if typeName == "" {
			return
		}
		counts[typeName]++
		if _, ok := positions[typeName]; !ok {
			positions[typeName] = fn.Pos()
		}
	})

	for typeName, count := range counts {
		if count > max {
			pass.Reportf(positions[typeName],
				"type %s has %d exported methods (max %d)", typeName, count, max)
		}
	}

	return nil, nil
}

func receiverTypeName(expr ast.Expr) string {
	switch t := expr.(type) {
	case *ast.Ident:
		return t.Name
	case *ast.StarExpr:
		return receiverTypeName(t.X)
	case *ast.IndexExpr: // generic type
		return receiverTypeName(t.X)
	case *ast.IndexListExpr: // generic type with multiple params
		return receiverTypeName(t.X)
	}
	return ""
}
