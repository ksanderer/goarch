package execguard_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/ksanderer/goarch/analyzers/execguard"
	"github.com/ksanderer/goarch/config"
	"golang.org/x/tools/go/analysis/analysistest"
)

func TestExecGuard(t *testing.T) {
	config.ResetForTesting()
	dir, _ := filepath.Abs("testdata")
	os.Chdir(dir)
	analysistest.Run(t, dir, execguard.Analyzer, "example", "allowed")
}
