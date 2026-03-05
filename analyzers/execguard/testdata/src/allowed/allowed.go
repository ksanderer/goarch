package allowed

import (
	"os"
	"os/exec"
)

func ok() {
	_ = exec.Command("ls") // OK — excepted package
	_ = os.Getenv("HOME")  // OK — excepted package
}
