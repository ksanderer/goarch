// Command goarch is an architecture-enforcing build proxy for Go projects.
//
// Usage:
//
//	goarch build ./cmd/api        # validate → go build
//	goarch run ./cmd/api          # validate → go run
//	goarch test ./...             # validate → go test
//	goarch check ./...            # validate only (no build)
package main

import (
	"bufio"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
	"unicode"

	"github.com/ksanderer/goarch/analyzers/apileak"
	"github.com/ksanderer/goarch/analyzers/argcount"
	"github.com/ksanderer/goarch/analyzers/authguard"
	"github.com/ksanderer/goarch/analyzers/complexity"
	"github.com/ksanderer/goarch/analyzers/depban"
	"github.com/ksanderer/goarch/analyzers/errguard"
	"github.com/ksanderer/goarch/analyzers/execguard"
	"github.com/ksanderer/goarch/analyzers/fanout"
	"github.com/ksanderer/goarch/analyzers/funlen"
	"github.com/ksanderer/goarch/analyzers/layerguard"
	"github.com/ksanderer/goarch/analyzers/methodcount"
	"github.com/ksanderer/goarch/analyzers/secretguard"
	"github.com/ksanderer/goarch/analyzers/tagguard"
	"github.com/ksanderer/goarch/config"
	"github.com/ksanderer/goarch/docs"
	"golang.org/x/tools/go/analysis"
	"golang.org/x/tools/go/analysis/multichecker"
	"gopkg.in/yaml.v3"
)

// buildTag is the secret tag injected into go build/run/test.
// Intentionally non-obvious so agents can't guess it and bypass validation.
const buildTag = "goarch_f7e2a1"

var analyzers = []*analysis.Analyzer{
	layerguard.Analyzer,
	execguard.Analyzer,
	secretguard.Analyzer,
	fanout.Analyzer,
	methodcount.Analyzer,
	apileak.Analyzer,
	funlen.Analyzer,
	argcount.Analyzer,
	complexity.Analyzer,
	depban.Analyzer,
	tagguard.Analyzer,
	errguard.Analyzer,
	authguard.Analyzer,
}

func main() {
	if len(os.Args) < 2 {
		printUsage()
		os.Exit(1)
	}

	switch os.Args[1] {
	case "build", "run", "test":
		proxyCommand(os.Args[1], os.Args[2:])
	case "check":
		runCheck(os.Args[2:])
	case "explain":
		explainRule(os.Args[2:])
	case "rules":
		listRules()
	case "help", "-h", "--help":
		printUsage()
	default:
		// Fallback: treat as multichecker args (backward compat).
		multichecker.Main(analyzers...)
	}
}

func runCheck(args []string) {
	// Run go vet with ourselves as the vettool + the build tag to bypass gate.
	self, _ := os.Executable()
	args, tags := mergeBuildTags(args)
	vetArgs := []string{"vet", "-vettool", self, "-tags", tags}
	vetArgs = append(vetArgs, args...)

	goCmd := exec.Command("go", vetArgs...)
	goCmd.Stdout = os.Stdout
	goCmd.Stderr = os.Stderr

	if err := goCmd.Run(); err != nil {
		if exitErr, ok := err.(*exec.ExitError); ok {
			os.Exit(exitErr.ExitCode())
		}
		os.Exit(1)
	}
}

func proxyCommand(cmd string, args []string) {
	fmt.Fprintf(os.Stderr, "goarch: validating architecture...\n")

	args, tags := mergeBuildTags(args)

	// Always validate the entire project — architecture rules apply globally.
	// Uses go vet -vettool with the build tag to bypass gate during analysis.
	self, _ := os.Executable()
	vetArgs := []string{"vet", "-vettool", self, "-tags", tags, "./..."}
	check := exec.Command("go", vetArgs...)
	check.Stdout = os.Stdout

	// Pipe stderr through a dedup filter — go vet runs analysis on both
	// main and test packages, producing duplicate violation messages.
	stderrPipe, _ := check.StderrPipe()
	check.Start()

	seen := &sync.Map{}
	scanner := bufio.NewScanner(stderrPipe)
	for scanner.Scan() {
		line := scanner.Text()
		if _, loaded := seen.LoadOrStore(line, true); !loaded {
			fmt.Fprintln(os.Stderr, line)
		}
	}

	if err := check.Wait(); err != nil {
		fmt.Fprintf(os.Stderr, "\ngoarch: BUILD BLOCKED — fix architecture violations above.\n")
		fmt.Fprintf(os.Stderr, "goarch: Run 'go tool goarch explain <rule>' for details on any rule.\n")
		ruleNames := make([]string, len(analyzers))
		for i, a := range analyzers {
			ruleNames[i] = a.Name
		}
		fmt.Fprintf(os.Stderr, "goarch: Rules: %s\n", strings.Join(ruleNames, ", "))
		os.Exit(1)
	}

	fmt.Fprintf(os.Stderr, "goarch: architecture OK\n")

	// Run external tools if configured.
	runExternalTools()

	// Proxy to go build/run/test with the secret tag plus any caller tags.
	goArgs := []string{cmd, "-tags", tags}
	goArgs = append(goArgs, args...)

	fmt.Fprintf(os.Stderr, "goarch: go %s\n", strings.Join(goArgs, " "))

	goCmd := exec.Command("go", goArgs...)
	goCmd.Stdout = os.Stdout
	goCmd.Stderr = os.Stderr
	goCmd.Stdin = os.Stdin

	if err := goCmd.Run(); err != nil {
		if exitErr, ok := err.(*exec.ExitError); ok {
			os.Exit(exitErr.ExitCode())
		}
		os.Exit(1)
	}
}

func mergeBuildTags(args []string) ([]string, string) {
	tags := []string{buildTag}
	filtered := make([]string, 0, len(args))
	for i := 0; i < len(args); i++ {
		arg := args[i]
		switch {
		case arg == "-tags" && i+1 < len(args):
			tags = appendBuildTags(tags, args[i+1])
			i++
		case strings.HasPrefix(arg, "-tags="):
			tags = appendBuildTags(tags, strings.TrimPrefix(arg, "-tags="))
		default:
			filtered = append(filtered, arg)
		}
	}
	return filtered, strings.Join(uniqueStrings(tags), ",")
}

func appendBuildTags(tags []string, raw string) []string {
	for _, tag := range strings.FieldsFunc(raw, func(r rune) bool {
		return r == ',' || unicode.IsSpace(r)
	}) {
		if tag != "" {
			tags = append(tags, tag)
		}
	}
	return tags
}

func uniqueStrings(values []string) []string {
	seen := make(map[string]struct{}, len(values))
	out := make([]string, 0, len(values))
	for _, value := range values {
		if _, ok := seen[value]; ok {
			continue
		}
		seen[value] = struct{}{}
		out = append(out, value)
	}
	return out
}

func explainRule(args []string) {
	if len(args) == 0 {
		listRules()
		return
	}
	rule := docs.Get(args[0])
	if rule == nil {
		fmt.Fprintf(os.Stderr, "goarch: unknown rule %q\n\n", args[0])
		listRules()
		os.Exit(1)
	}
	fmt.Println(rule.Long)
}

func listRules() {
	fmt.Println("Available rules:")
	fmt.Println()
	for _, r := range docs.All() {
		fmt.Printf("  %-14s %s\n", r.ID, r.Short)
	}
	fmt.Println()
	fmt.Println("Run 'go tool goarch explain <rule>' for detailed documentation.")
}

func runExternalTools() {
	cfg := loadConfigFile()
	if cfg == nil || len(cfg.Rules.External) == 0 {
		return
	}
	for _, tool := range cfg.Rules.External {
		fmt.Fprintf(os.Stderr, "goarch: running %s...\n", tool.Name)
		parts := strings.Fields(tool.Cmd)
		cmd := exec.Command(parts[0], parts[1:]...)
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr
		if err := cmd.Run(); err != nil {
			fmt.Fprintf(os.Stderr, "\ngoarch: BUILD BLOCKED — %s failed.\n", tool.Name)
			os.Exit(1)
		}
		fmt.Fprintf(os.Stderr, "goarch: %s OK\n", tool.Name)
	}
}

func loadConfigFile() *config.Config {
	dir, err := os.Getwd()
	if err != nil {
		return nil
	}
	for {
		path := filepath.Join(dir, ".goarch.yml")
		data, err := os.ReadFile(path)
		if err == nil {
			var cfg config.Config
			if yaml.Unmarshal(data, &cfg) == nil {
				return &cfg
			}
			return nil
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			return nil
		}
		dir = parent
	}
}

func printUsage() {
	fmt.Fprintf(os.Stderr, `goarch — architecture-enforcing build proxy for Go

Usage:
  goarch build [go build flags] [packages]   Validate then build
  goarch run   [go run flags]   [packages]   Validate then run
  goarch test  [go test flags]  [packages]   Validate then test
  goarch check [packages]                    Validate only
  goarch explain <rule>                      Show rule documentation
  goarch rules                               List all rules

Examples:
  go tool goarch build -o bin/api ./cmd/api
  go tool goarch check ./...
  go tool goarch explain secretguard
`)
}
