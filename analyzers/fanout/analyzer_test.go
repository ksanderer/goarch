package fanout_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/ksanderer/goarch/analyzers/fanout"
	"github.com/ksanderer/goarch/config"
	"golang.org/x/tools/go/analysis/analysistest"
)

func TestFanOut(t *testing.T) {
	config.ResetForTesting()
	dir, _ := filepath.Abs("testdata")
	os.Chdir(dir)
	analysistest.Run(t, dir, fanout.Analyzer, "example")
}
