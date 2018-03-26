package main

type NotImplementedError struct {
	msg string
}

func (e *NotImplementedError) Error() string{
	return e.msg
}

func NewNotImplementedError (msg string) *NotImplementedError {
	return &NotImplementedError{msg:msg}
}

func IsNotImplemented(e error) bool {
	_, ok := e.(*NotImplementedError)
	return ok
}