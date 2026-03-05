package example

type MyError struct { // want `\[errguard\] error type MyError should be defined in one of: allowed`
	Msg string
}

func (e *MyError) Error() string { return e.Msg }

type NotAnError struct { // OK — no Error() method
	Msg string
}
