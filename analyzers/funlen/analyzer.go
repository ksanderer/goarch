// Package funlen limits the number of lines per function body,
// preventing functions that are too long to understand at a glance.
package funlen

import (
	"go/ast"

	"github.com/ksanderer/goarch/config"
	"golang.org/x/tools/go/analysis"
	"golang.org/x/tools/go/analysis/passes/inspect"
	"golang.org/x/tools/go/ast/inspector"
)

var Analyzer = &analysis.Analyzer{
	Name:     "funlen",
	Doc:      "limits function body length",
	Requires: []*analysis.Analyzer{config.Analyzer, inspect.Analyzer},
	Run:      run,
}

func run(pass *analysis.Pass) (interface{}, error) {
	cfg := pass.ResultOf[config.Analyzer].(*config.Config)
	if cfg.Rules.FunLen == nil || cfg.Rules.FunLen.MaxLines <= 0 {
		return nil, nil
	}

	max := cfg.Rules.FunLen.MaxLines
	fset := pass.Fset

	insp := pass.ResultOf[inspect.Analyzer].(*inspector.Inspector)
	nodeFilter := []ast.Node{(*ast.FuncDecl)(nil)}

	insp.Preorder(nodeFilter, func(n ast.Node) {
		fn := n.(*ast.FuncDecl)
		if fn.Body == nil {
			return
		}
		start := fset.Position(fn.Body.Lbrace)
		end := fset.Position(fn.Body.Rbrace)
		lines := end.Line - start.Line
		if lines > max {
			pass.Reportf(fn.Pos(),
				"[funlen] function %s is %d lines long (max %d)", fn.Name.Name, lines, max)
		}
	})

	return nil, nil
}
