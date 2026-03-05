// Package fanout limits the number of imports per file, preventing excessive
// coupling to external packages.
package fanout

import (
	"strings"

	"github.com/nicegoodthings/goarch/config"
	"golang.org/x/tools/go/analysis"
)

var Analyzer = &analysis.Analyzer{
	Name:     "fanout",
	Doc:      "limits the number of non-stdlib imports per file",
	Requires: []*analysis.Analyzer{config.Analyzer},
	Run:      run,
}

func run(pass *analysis.Pass) (interface{}, error) {
	cfg := pass.ResultOf[config.Analyzer].(*config.Config)
	if cfg.Rules.FanOut == nil || cfg.Rules.FanOut.MaxImports <= 0 {
		return nil, nil
	}

	max := cfg.Rules.FanOut.MaxImports

	for _, file := range pass.Files {
		count := 0
		for _, imp := range file.Imports {
			importPath := strings.Trim(imp.Path.Value, `"`)
			if !isStdlib(importPath) {
				count++
			}
		}
		if count > max {
			pass.Reportf(file.Package,
				"file has %d non-stdlib imports (max %d)", count, max)
		}
	}
	return nil, nil
}

func isStdlib(importPath string) bool {
	first := importPath
	if i := strings.Index(importPath, "/"); i >= 0 {
		first = importPath[:i]
	}
	return !strings.Contains(first, ".")
}
