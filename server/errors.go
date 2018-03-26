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

// User error, corresponds with a bad request
type UserError struct {
	msg string
}

func (e *UserError) Error() string {
	return e.msg
}

func NewUserError(msg string) *UserError {
	return &UserError{msg:msg}
}

func WrapUserError(e error) *UserError {
	return &UserError{msg:e.Error()}
}

func IsUser(e error) bool {
	_, ok := e.(*UserError)
	return ok
}