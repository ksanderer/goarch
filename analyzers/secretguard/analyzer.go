// Package secretguard ensures struct fields with sensitive names (password,
// token, secret, etc.) use a designated wrapper type instead of raw strings.
package secretguard

import (
	"go/ast"
	"go/types"
	"strings"
	"unicode"

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

	// Skip excepted packages.
	pkg := pass.Pkg.Path()
	for _, exc := range cfg.Rules.SecretGuard.ExceptPackages {
		if pkg == exc || strings.HasSuffix(pkg, "/"+exc) {
			return nil, nil
		}
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
							"[secretguard] sensitive field %q should use type %s, got %s",
							name.Name, requiredType, fieldType.String())
					}
				}
			}
		}
	})

	return nil, nil
}

func matchesSensitive(name string, patterns []string) bool {
	// Split camelCase/PascalCase into lowercase words:
	// "OpenRouterAPIKey" -> ["open", "router", "api", "key"]
	// "PromptTokens" -> ["prompt", "tokens"]
	words := splitCamelCase(name)
	joined := strings.ToLower(strings.Join(words, ""))

	for _, p := range patterns {
		pl := strings.ToLower(p)
		// Match if the concatenated words contain the pattern.
		// "apikey" matches "OpenRouterAPIKey" (open+router+api+key)
		// "token" does NOT match "PromptTokens" because we require
		// the pattern to align with word boundaries.
		if joined == pl || matchWordBoundary(words, pl) {
			return true
		}
	}
	return false
}

// matchWordBoundary checks if the pattern matches a contiguous sequence of
// complete words. "apikey" matches ["api","key"] but "token" does not match
// ["prompt","tokens"] (because "tokens" != "token").
func matchWordBoundary(words []string, pattern string) bool {
	for i := range words {
		concat := ""
		for j := i; j < len(words); j++ {
			concat += strings.ToLower(words[j])
			if concat == pattern {
				return true
			}
			if len(concat) > len(pattern) {
				break
			}
		}
	}
	return false
}

// splitCamelCase splits "OpenRouterAPIKey" into ["Open", "Router", "API", "Key"].
func splitCamelCase(s string) []string {
	var words []string
	var current []rune
	runes := []rune(s)

	for i := 0; i < len(runes); i++ {
		r := runes[i]
		if i > 0 && unicode.IsUpper(r) {
			// Start new word if: lowercase->uppercase, or end of uppercase run
			prev := runes[i-1]
			if unicode.IsLower(prev) || (unicode.IsUpper(prev) && i+1 < len(runes) && unicode.IsLower(runes[i+1])) {
				if len(current) > 0 {
					words = append(words, string(current))
					current = nil
				}
			}
		}
		current = append(current, r)
	}
	if len(current) > 0 {
		words = append(words, string(current))
	}
	return words
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
