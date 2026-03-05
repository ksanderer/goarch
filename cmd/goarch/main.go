// Command goarch runs architectural rule checks configured via .goarch.yml.
//
// Usage:
//
//	goarch ./...                              # run all enabled rules
//	go vet -vettool=$(which goarch) ./...     # run as go vet plugin
package main

import (
	"github.com/nicegoodthings/goarch/analyzers/apileak"
	"github.com/nicegoodthings/goarch/analyzers/execguard"
	"github.com/nicegoodthings/goarch/analyzers/fanout"
	"github.com/nicegoodthings/goarch/analyzers/layerguard"
	"github.com/nicegoodthings/goarch/analyzers/methodcount"
	"github.com/nicegoodthings/goarch/analyzers/secretguard"
	"golang.org/x/tools/go/analysis/multichecker"
)

func main() {
	multichecker.Main(
		layerguard.Analyzer,
		execguard.Analyzer,
		secretguard.Analyzer,
		fanout.Analyzer,
		methodcount.Analyzer,
		apileak.Analyzer,
	)
}
