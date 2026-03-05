package tagguard_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/ksanderer/goarch/analyzers/tagguard"
	"github.com/ksanderer/goarch/config"
	"golang.org/x/tools/go/analysis/analysistest"
)

func TestTagGuard(t *testing.T) {
	config.ResetForTesting()
	dir, _ := filepath.Abs("testdata")
	os.Chdir(dir)
	analysistest.Run(t, dir, tagguard.Analyzer, "example")
}
