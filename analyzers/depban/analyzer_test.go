package depban

import (
	"os"
	"path/filepath"
	"testing"
)

func TestParseGoMod(t *testing.T) {
	dir := t.TempDir()
	gomod := filepath.Join(dir, "go.mod")
	content := `module example

go 1.22

require (
	github.com/bad/lib v1.0.0
	github.com/ok/lib1 v1.0.0
	github.com/ok/lib2 v1.0.0
	github.com/indirect/dep v2.0.0 // indirect
)

require github.com/single/dep v0.1.0
`
	os.WriteFile(gomod, []byte(content), 0644)

	modules, err := parseGoMod(gomod)
	if err != nil {
		t.Fatal(err)
	}

	expected := map[string]bool{
		"github.com/bad/lib":    true,
		"github.com/ok/lib1":    true,
		"github.com/ok/lib2":    true,
		"github.com/single/dep": true,
	}

	if len(modules) != len(expected) {
		t.Errorf("expected %d modules, got %d: %v", len(expected), len(modules), modules)
	}
	for _, m := range modules {
		if !expected[m] {
			t.Errorf("unexpected module: %s", m)
		}
	}
}

func TestParseGoMod_SkipsIndirect(t *testing.T) {
	dir := t.TempDir()
	gomod := filepath.Join(dir, "go.mod")
	content := `module example

go 1.22

require (
	github.com/direct v1.0.0
	github.com/indirect v1.0.0 // indirect
)
`
	os.WriteFile(gomod, []byte(content), 0644)

	modules, err := parseGoMod(gomod)
	if err != nil {
		t.Fatal(err)
	}

	if len(modules) != 1 || modules[0] != "github.com/direct" {
		t.Errorf("expected [github.com/direct], got %v", modules)
	}
}

func TestFindGoMod(t *testing.T) {
	dir := t.TempDir()
	// Resolve symlinks (macOS /var → /private/var)
	dir, _ = filepath.EvalSymlinks(dir)

	gomod := filepath.Join(dir, "go.mod")
	os.WriteFile(gomod, []byte("module test\n"), 0644)

	sub := filepath.Join(dir, "sub", "deep")
	os.MkdirAll(sub, 0755)

	orig, _ := os.Getwd()
	defer os.Chdir(orig)
	os.Chdir(sub)

	found := findGoMod()
	if found != gomod {
		t.Errorf("expected %s, got %s", gomod, found)
	}
}
