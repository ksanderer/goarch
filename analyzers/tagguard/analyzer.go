// Package tagguard validates struct tag naming conventions per package.
// It checks that JSON tags follow a consistent naming style (snake_case or camelCase)
// and optionally requires all exported fields to have JSON tags.
package tagguard

import (
	"go/ast"
	"reflect"
	"strings"
	"unicode"

	"github.com/ksanderer/goarch/config"
	"golang.org/x/tools/go/analysis"
	"golang.org/x/tools/go/analysis/passes/inspect"
	"golang.org/x/tools/go/ast/inspector"
)

var Analyzer = &analysis.Analyzer{
	Name:     "tagguard",
	Doc:      "validates struct tag naming conventions",
	Requires: []*analysis.Analyzer{config.Analyzer, inspect.Analyzer},
	Run:      run,
}

func run(pass *analysis.Pass) (interface{}, error) {
	cfg := pass.ResultOf[config.Analyzer].(*config.Config)
	if cfg.Rules.TagGuard == nil || len(cfg.Rules.TagGuard.Packages) == 0 {
		return nil, nil
	}

	pkg := pass.Pkg.Path()

	// Find matching rule for this package.
	var rule *config.TagRule
	for pattern, r := range cfg.Rules.TagGuard.Packages {
		if pkg == pattern || strings.HasSuffix(pkg, "/"+pattern) {
			r := r
			rule = &r
			break
		}
	}
	if rule == nil {
		return nil, nil
	}

	insp := pass.ResultOf[inspect.Analyzer].(*inspector.Inspector)
	nodeFilter := []ast.Node{(*ast.StructType)(nil)}

	insp.Preorder(nodeFilter, func(n ast.Node) {
		st := n.(*ast.StructType)
		for _, field := range st.Fields.List {
			for _, name := range field.Names {
				if !name.IsExported() {
					continue
				}
				tag := extractJSONTag(field)
				if tag == "" {
					if rule.RequireJSONTags {
						pass.Reportf(name.Pos(),
							"[tagguard] exported field %s has no json tag", name.Name)
					}
					continue
				}
				if tag == "-" {
					continue // explicitly omitted
				}
				if rule.JSONNaming != "" && !matchesNaming(tag, rule.JSONNaming) {
					pass.Reportf(name.Pos(),
						"[tagguard] json tag %q on field %s should be %s",
						tag, name.Name, rule.JSONNaming)
				}
			}
		}
	})

	return nil, nil
}

func extractJSONTag(field *ast.Field) string {
	if field.Tag == nil {
		return ""
	}
	raw := strings.Trim(field.Tag.Value, "`")
	tag := reflect.StructTag(raw).Get("json")
	if tag == "" {
		return ""
	}
	// Strip options like ",omitempty"
	if i := strings.Index(tag, ","); i >= 0 {
		tag = tag[:i]
	}
	return tag
}

func matchesNaming(tag string, convention string) bool {
	switch convention {
	case "snake_case":
		return isSnakeCase(tag)
	case "camelCase":
		return isCamelCase(tag)
	}
	return true
}

func isSnakeCase(s string) bool {
	for _, r := range s {
		if unicode.IsUpper(r) {
			return false
		}
		if r != '_' && !unicode.IsLetter(r) && !unicode.IsDigit(r) {
			return false
		}
	}
	return true
}

func isCamelCase(s string) bool {
	if len(s) == 0 {
		return true
	}
	// camelCase starts with lowercase
	runes := []rune(s)
	if unicode.IsUpper(runes[0]) {
		return false
	}
	// No underscores
	return !strings.Contains(s, "_")
}
