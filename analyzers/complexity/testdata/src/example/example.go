package example

func simple(x int) { // complexity 2 — OK
	if x > 0 {
		_ = x
	}
}

func complex(x, y int) { // want `\[complexity\] function complex has cyclomatic complexity 4 \(max 3\)`
	if x > 0 {
		_ = x
	}
	if y > 0 {
		_ = y
	}
	for i := range 10 {
		_ = i
	}
}
