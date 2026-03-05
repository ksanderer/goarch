package example

import (
	"os"
	"os/exec" // want `\[execguard\] import "os/exec" is banned: use subprocess`
)

func bad() {
	_ = exec.Command("ls")
	_ = os.Getenv("HOME") // want `\[execguard\] os.Getenv is banned: use config`
}
