package complexity_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/ksanderer/goarch/analyzers/complexity"
	"github.com/ksanderer/goarch/config"
	"golang.org/x/tools/go/analysis/analysistest"
)

func TestComplexity(t *testing.T) {
	config.ResetForTesting()
	dir, _ := filepath.Abs("testdata")
	os.Chdir(dir)
	analysistest.Run(t, dir, complexity.Analyzer, "example")
}
