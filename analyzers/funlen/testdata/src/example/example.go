package example

func short() { // OK — 3 lines
	_ = 1
	_ = 2
}

func long() { // want `\[funlen\] function long is 12 lines long \(max 10\)`
	_ = 1
	_ = 2
	_ = 3
	_ = 4
	_ = 5
	_ = 6
	_ = 7
	_ = 8
	_ = 9
	_ = 10
	_ = 11
}
