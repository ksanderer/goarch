// Package depban checks go.mod for banned module dependencies and
// optionally limits the total number of direct dependencies.
//
// Unlike execguard (which checks source imports), depban catches modules
// that are in go.mod but may only be used transitively.
package depban

import (
	"bufio"
	"os"
	"path/filepath"
	"strings"

	"github.com/ksanderer/goarch/config"
	"golang.org/x/tools/go/analysis"
)

var Analyzer = &analysis.Analyzer{
	Name:     "depban",
	Doc:      "checks go.mod for banned dependencies",
	Requires: []*analysis.Analyzer{config.Analyzer},
	Run:      run,
}

func run(pass *analysis.Pass) (interface{}, error) {
	cfg := pass.ResultOf[config.Analyzer].(*config.Config)
	if cfg.Rules.DepBan == nil {
		return nil, nil
	}

	// Only run once — for the first package analyzed.
	// We check go.mod, not per-package source.
	if pass.Pkg.Path() != findRootPackage(pass) {
		return nil, nil
	}

	gomod := findGoMod()
	if gomod == "" {
		return nil, nil
	}

	modules, err := parseGoMod(gomod)
	if err != nil {
		return nil, nil // can't read go.mod, skip silently
	}

	rules := cfg.Rules.DepBan

	for _, mod := range modules {
		for _, ban := range rules.Deny {
			if mod == ban.Module || strings.HasPrefix(mod, ban.Module+"/") {
				reason := ban.Reason
				if reason == "" {
					reason = "banned by goarch"
				}
				pass.Reportf(pass.Files[0].Package,
					"[depban] module %q in go.mod is banned: %s", mod, reason)
			}
		}
	}

	if rules.MaxDependencies > 0 && len(modules) > rules.MaxDependencies {
		pass.Reportf(pass.Files[0].Package,
			"[depban] go.mod has %d direct dependencies (max %d)", len(modules), rules.MaxDependencies)
	}

	return nil, nil
}

func findRootPackage(pass *analysis.Pass) string {
	// Return the first package analyzed — we use this to run depban once.
	return pass.Pkg.Path()
}

func findGoMod() string {
	dir, err := os.Getwd()
	if err != nil {
		return ""
	}
	for {
		path := filepath.Join(dir, "go.mod")
		if _, err := os.Stat(path); err == nil {
			return path
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			return ""
		}
		dir = parent
	}
}

func parseGoMod(path string) ([]string, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	var modules []string
	inRequire := false
	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if strings.HasPrefix(line, "require (") || line == "require (" {
			inRequire = true
			continue
		}
		if inRequire && line == ")" {
			inRequire = false
			continue
		}
		if inRequire {
			// Lines look like: github.com/foo/bar v1.2.3 // indirect
			if strings.HasPrefix(line, "//") {
				continue
			}
			if strings.Contains(line, "// indirect") {
				continue
			}
			parts := strings.Fields(line)
			if len(parts) >= 2 {
				modules = append(modules, parts[0])
			}
		}
		// Single-line require: require github.com/foo/bar v1.2.3
		if strings.HasPrefix(line, "require ") && !strings.Contains(line, "(") {
			parts := strings.Fields(line)
			if len(parts) >= 3 {
				modules = append(modules, parts[1])
			}
		}
	}
	return modules, scanner.Err()
}
