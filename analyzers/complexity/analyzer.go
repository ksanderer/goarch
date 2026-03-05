// Package complexity computes cyclomatic complexity per function
// and reports functions that exceed the configured threshold.
package complexity

import (
	"go/ast"
	"go/token"

	"github.com/ksanderer/goarch/config"
	"golang.org/x/tools/go/analysis"
	"golang.org/x/tools/go/analysis/passes/inspect"
	"golang.org/x/tools/go/ast/inspector"
)

var Analyzer = &analysis.Analyzer{
	Name:     "complexity",
	Doc:      "limits cyclomatic complexity per function",
	Requires: []*analysis.Analyzer{config.Analyzer, inspect.Analyzer},
	Run:      run,
}

func run(pass *analysis.Pass) (interface{}, error) {
	cfg := pass.ResultOf[config.Analyzer].(*config.Config)
	if cfg.Rules.Complexity == nil || cfg.Rules.Complexity.MaxComplexity <= 0 {
		return nil, nil
	}

	max := cfg.Rules.Complexity.MaxComplexity

	insp := pass.ResultOf[inspect.Analyzer].(*inspector.Inspector)
	nodeFilter := []ast.Node{(*ast.FuncDecl)(nil)}

	insp.Preorder(nodeFilter, func(n ast.Node) {
		fn := n.(*ast.FuncDecl)
		if fn.Body == nil {
			return
		}
		c := 1 // base complexity
		ast.Inspect(fn.Body, func(n ast.Node) bool {
			switch n := n.(type) {
			case *ast.IfStmt:
				c++
			case *ast.ForStmt:
				c++
			case *ast.RangeStmt:
				c++
			case *ast.CaseClause:
				if n.List != nil { // skip default clause
					c++
				}
			case *ast.CommClause:
				if n.Comm != nil { // skip default clause
					c++
				}
			case *ast.BinaryExpr:
				if n.Op == token.LAND || n.Op == token.LOR {
					c++
				}
			}
			return true
		})
		if c > max {
			pass.Reportf(fn.Pos(),
				"[complexity] function %s has cyclomatic complexity %d (max %d)", fn.Name.Name, c, max)
		}
	})

	return nil, nil
}
