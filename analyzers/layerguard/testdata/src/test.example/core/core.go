package core

import (
	"test.example/util"
	"test.example/other" // want `\[layerguard\] import "test.example/other" is not allowed in this package`
)

var _ = util.X
var _ = other.Y
