// Package generated identifies syntax that belongs to generated Go files.
package generated

import (
	"go/ast"
	"go/token"

	"golang.org/x/tools/go/analysis"
)

type Files map[*token.File]struct{}

func FromPass(pass *analysis.Pass) Files {
	files := make(Files)
	for _, file := range pass.Files {
		if ast.IsGenerated(file) {
			files[pass.Fset.File(file.Pos())] = struct{}{}
		}
	}
	return files
}

func (files Files) Contains(fset *token.FileSet, position token.Pos) bool {
	_, ok := files[fset.File(position)]
	return ok
}
