package config

import (
	"os"
	"path/filepath"
	"reflect"
	"sync"

	"golang.org/x/tools/go/analysis"
	"gopkg.in/yaml.v3"
)

const configFileName = ".goarch.yml"

var (
	cached   *Config
	loadOnce sync.Once
	loadErr  error
)

// Analyzer is a pseudo-analyzer that loads the config file once and shares
// the result with all real analyzers via pass.ResultOf.
var Analyzer = &analysis.Analyzer{
	Name:       "goarch_config",
	Doc:        "loads .goarch.yml configuration",
	Run:        run,
	ResultType: reflect.TypeOf((*Config)(nil)),
}

func run(pass *analysis.Pass) (interface{}, error) {
	loadOnce.Do(func() {
		cached, loadErr = findAndLoad()
	})
	if loadErr != nil {
		return nil, loadErr
	}
	return cached, nil
}

func findAndLoad() (*Config, error) {
	dir, err := os.Getwd()
	if err != nil {
		return nil, err
	}
	for {
		path := filepath.Join(dir, configFileName)
		data, err := os.ReadFile(path)
		if err == nil {
			return parse(data)
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			// No config found — return empty config (all rules disabled).
			return &Config{}, nil
		}
		dir = parent
	}
}

func parse(data []byte) (*Config, error) {
	var cfg Config
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return nil, err
	}
	return &cfg, nil
}
