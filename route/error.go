package route

import (
	"fmt"
)

type Error struct {
	Message string `json:"error"`

	code int
	e    error
}

func (e Error) Error() string {
	return e.Error()
}

func Bad(e error, msg string, args ...interface{}) Error {
	return Errorf(400, e, msg, args...)
}

func Oops(e error, msg string, args ...interface{}) Error {
	return Errorf(500, e, msg, args...)
}

func NotFound(e error, msg string, args ...interface{}) Error {
	return Errorf(404, e, msg, args...)
}

func Errorf(code int, e error, msg string, args ...interface{}) Error {
	return Error{
		code:    code,
		e:       e,
		Message: fmt.Sprintf(msg, args...),
	}
}
