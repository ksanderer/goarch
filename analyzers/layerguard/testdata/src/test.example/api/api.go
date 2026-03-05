package api

import (
	"test.example/secret" // want `\[layerguard\] import "test.example/secret" is not allowed in this package`
)

var _ = secret.Hidden
