package unison

type panicError struct {
	panic any
}

func (panicError) Error() string {
	return "panicError"
}

type result[T any] struct {
	value T
	err   error
}
