package example // want `\[fanout\] file has 2 non-stdlib imports \(max 1\)`

import (
	"fmt"

	"github.com/fake/dep1"
	"github.com/fake/dep2"
)

var _ = fmt.Sprintf
var _ = dep1.X
var _ = dep2.X
