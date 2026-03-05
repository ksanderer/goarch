// Package argcount limits the number of parameters per function,
// encouraging the use of option structs for complex APIs.
package argcount

import (
	"go/ast"

	"github.com/ksanderer/goarch/config"
	"golang.org/x/tools/go/analysis"
	"golang.org/x/tools/go/analysis/passes/inspect"
	"golang.org/x/tools/go/ast/inspector"
)

var Analyzer = &analysis.Analyzer{
	Name:     "argcount",
	Doc:      "limits the number of function parameters",
	Requires: []*analysis.Analyzer{config.Analyzer, inspect.Analyzer},
	Run:      run,
}

func run(pass *analysis.Pass) (interface{}, error) {
	cfg := pass.ResultOf[config.Analyzer].(*config.Config)
	if cfg.Rules.ArgCount == nil || cfg.Rules.ArgCount.MaxArgs <= 0 {
		return nil, nil
	}

	max := cfg.Rules.ArgCount.MaxArgs

	insp := pass.ResultOf[inspect.Analyzer].(*inspector.Inspector)
	nodeFilter := []ast.Node{(*ast.FuncDecl)(nil)}

	insp.Preorder(nodeFilter, func(n ast.Node) {
		fn := n.(*ast.FuncDecl)
		if fn.Type.Params == nil {
			return
		}
		// Count individual params — grouped params like "a, b int" count as 2.
		count := 0
		for _, field := range fn.Type.Params.List {
			if len(field.Names) == 0 {
				count++ // unnamed param (e.g. interface method)
			} else {
				count += len(field.Names)
			}
		}
		if count > max {
			pass.Reportf(fn.Pos(),
				"[argcount] function %s has %d parameters (max %d)", fn.Name.Name, count, max)
		}
	})

	return nil, nil
}
