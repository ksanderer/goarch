package allowed

type AppError struct {
	Code int
	Msg  string
}

func (e *AppError) Error() string { return e.Msg }
