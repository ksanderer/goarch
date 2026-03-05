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
	"fmt"
	"os"
	"os/exec"
	"strings"

	"github.com/nicegoodthings/goarch/analyzers/apileak"
	"github.com/nicegoodthings/goarch/analyzers/execguard"
	"github.com/nicegoodthings/goarch/analyzers/fanout"
	"github.com/nicegoodthings/goarch/analyzers/layerguard"
	"github.com/nicegoodthings/goarch/analyzers/methodcount"
	"github.com/nicegoodthings/goarch/analyzers/secretguard"
	"golang.org/x/tools/go/analysis"
	"golang.org/x/tools/go/analysis/multichecker"
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
		// Run analyzers only — rewrite args for multichecker.
		os.Args = append([]string{os.Args[0]}, os.Args[2:]...)
		multichecker.Main(analyzers...)
	case "help", "-h", "--help":
		printUsage()
	default:
		// Fallback: treat as multichecker args (backward compat).
		multichecker.Main(analyzers...)
	}
}

func proxyCommand(cmd string, args []string) {
	fmt.Fprintf(os.Stderr, "goarch: validating architecture...\n")

	// Always validate the entire project — architecture rules apply globally,
	// not just to the package being built.
	checkArgs := []string{os.Args[0], "check", "./..."}
	check := exec.Command(checkArgs[0], checkArgs[1:]...)
	check.Stdout = os.Stdout
	check.Stderr = os.Stderr
	check.Dir, _ = os.Getwd()

	if err := check.Run(); err != nil {
		fmt.Fprintf(os.Stderr, "\ngoarch: BUILD BLOCKED — fix architecture violations above.\n")
		os.Exit(1)
	}

	fmt.Fprintf(os.Stderr, "goarch: architecture OK\n")

	// Phase 2: Proxy to go build/run/test with the secret tag.
	goArgs := []string{cmd, "-tags", buildTag}
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

// flagsWithArgs are go build flags that take a following argument.
var flagsWithArgs = map[string]bool{
	"-o": true, "-p": true, "-asmflags": true, "-buildmode": true,
	"-buildvcs": true, "-compiler": true, "-gccgoflags": true,
	"-gcflags": true, "-ldflags": true, "-mod": true, "-modfile": true,
	"-overlay": true, "-pgo": true, "-pkgdir": true, "-tags": true,
	"-toolexec": true, "-cover": true, "-covermode": true,
	"-coverpkg": true, "-exec": true, "-timeout": true,
	"-run": true, "-bench": true, "-count": true, "-cpu": true,
	"-blockprofile": true, "-cpuprofile": true, "-memprofile": true,
	"-mutexprofile": true, "-trace": true, "-outputdir": true,
}

func extractTargets(args []string) []string {
	var targets []string
	skipNext := false
	for _, arg := range args {
		if skipNext {
			skipNext = false
			continue
		}
		if strings.HasPrefix(arg, "-") {
			// Check if this flag takes an argument.
			flag := arg
			if i := strings.Index(arg, "="); i >= 0 {
				flag = arg[:i] // -o=bin/api → -o
			} else if flagsWithArgs[flag] {
				skipNext = true
			}
			continue
		}
		// Package paths start with . or contain /
		if strings.HasPrefix(arg, ".") || strings.Contains(arg, "/") {
			targets = append(targets, arg)
		}
	}
	return targets
}

func printUsage() {
	fmt.Fprintf(os.Stderr, `goarch — architecture-enforcing build proxy for Go

Usage:
  goarch build [go build flags] [packages]   Validate then build
  goarch run   [go run flags]   [packages]   Validate then run
  goarch test  [go test flags]  [packages]   Validate then test
  goarch check [packages]                    Validate only

Examples:
  goarch build -o bin/api ./cmd/api
  goarch run ./cmd/api
  goarch test -v ./...
  goarch check ./...
`)
}
