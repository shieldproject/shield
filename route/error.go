package route

import (
	"fmt"
)

type Error struct {
	Diagnostic string   `json:"diagnostic,omitempty"`
	Message    string   `json:"error,omitempty"`
	Missing    []string `json:"missing,omitempty"`

	code int
	e    error
}

func (e Error) Error() string {
	return e.e.Error()
}

func (e *Error) ProvideDiagnostic() {
	if e.e != nil {
		e.Diagnostic = fmt.Sprintf("server-side error: %s", e.e)
	} else {
		e.Diagnostic = "no further diagnostic information available"
	}
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

func Forbidden(e error, msg string, args ...interface{}) Error {
	return Errorf(403, e, msg, args...)
}

func Unauthorized(e error, msg string, args ...interface{}) Error {
	return Errorf(401, e, msg, args...)
}

func Errorf(code int, e error, msg string, args ...interface{}) Error {
	return Error{
		code:    code,
		e:       e,
		Message: fmt.Sprintf(msg, args...),
	}
}
