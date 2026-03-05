// Package secretguard ensures struct fields with sensitive names (password,
// token, secret, etc.) use a designated wrapper type instead of raw strings.
package secretguard

import (
	"go/ast"
	"go/types"
	"strings"

	"github.com/nicegoodthings/goarch/config"
	"golang.org/x/tools/go/analysis"
	"golang.org/x/tools/go/analysis/passes/inspect"
	"golang.org/x/tools/go/ast/inspector"
)

var Analyzer = &analysis.Analyzer{
	Name:     "secretguard",
	Doc:      "ensures sensitive fields use a Secret wrapper type",
	Requires: []*analysis.Analyzer{config.Analyzer, inspect.Analyzer},
	Run:      run,
}

func run(pass *analysis.Pass) (interface{}, error) {
	cfg := pass.ResultOf[config.Analyzer].(*config.Config)
	if cfg.Rules.SecretGuard == nil {
		return nil, nil
	}

	requiredType := cfg.Rules.SecretGuard.Type
	patterns := cfg.Rules.SecretGuard.FieldPatterns
	if requiredType == "" || len(patterns) == 0 {
		return nil, nil
	}

	insp := pass.ResultOf[inspect.Analyzer].(*inspector.Inspector)
	nodeFilter := []ast.Node{(*ast.StructType)(nil)}

	insp.Preorder(nodeFilter, func(n ast.Node) {
		st := n.(*ast.StructType)
		for _, field := range st.Fields.List {
			for _, name := range field.Names {
				if matchesSensitive(name.Name, patterns) {
					fieldType := pass.TypesInfo.TypeOf(field.Type)
					if fieldType != nil && !isSecretType(fieldType, requiredType) {
						pass.Reportf(name.Pos(),
							"sensitive field %q should use type %s, got %s",
							name.Name, requiredType, fieldType.String())
					}
				}
			}
		}
	})

	return nil, nil
}

func matchesSensitive(name string, patterns []string) bool {
	lower := strings.ToLower(name)
	for _, p := range patterns {
		if strings.Contains(lower, strings.ToLower(p)) {
			return true
		}
	}
	return false
}

func isSecretType(t types.Type, requiredType string) bool {
	named, ok := t.(*types.Named)
	if !ok {
		return false
	}
	fullName := named.Obj().Pkg().Path() + "." + named.Obj().Name()
	// Match either full path or just the type name.
	return fullName == requiredType ||
		strings.HasSuffix(fullName, "/"+requiredType) ||
		named.Obj().Name() == lastElement(requiredType)
}

func lastElement(s string) string {
	if i := strings.LastIndex(s, "."); i >= 0 {
		return s[i+1:]
	}
	if i := strings.LastIndex(s, "/"); i >= 0 {
		return s[i+1:]
	}
	return s
}
