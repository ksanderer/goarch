package errguard_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/ksanderer/goarch/analyzers/errguard"
	"github.com/ksanderer/goarch/config"
	"golang.org/x/tools/go/analysis/analysistest"
)

func TestErrGuard(t *testing.T) {
	config.ResetForTesting()
	dir, _ := filepath.Abs("testdata")
	os.Chdir(dir)
	analysistest.Run(t, dir, errguard.Analyzer, "example", "allowed")
}
