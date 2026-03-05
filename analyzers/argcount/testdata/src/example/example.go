package example

func ok(a, b, c int) {} // 3 params — OK

func tooMany(a, b, c, d int) {} // want `\[argcount\] function tooMany has 4 parameters \(max 3\)`

func grouped(a, b int, c string, d bool) {} // want `\[argcount\] function grouped has 4 parameters \(max 3\)`
