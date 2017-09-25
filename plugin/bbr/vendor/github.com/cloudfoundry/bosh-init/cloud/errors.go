package cloud

import (
	"fmt"
)

const (
	VMNotFoundError       = "Bosh::Clouds::VMNotFound"
	DiskNotFoundError     = "Bosh::Clouds::DiskNotFound"
	StemcellNotFoundError = "Bosh::Clouds::StemcellNotFound"
	NotImplementedError   = "Bosh::Clouds::NotImplemented"
)

type Error interface {
	error
	Method() string
	Type() string
	Message() string
	OkToRetry() bool
}

type cpiError struct {
	method   string
	cmdError CmdError
}

func NewCPIError(method string, cmdError CmdError) Error {
	return cpiError{
		method:   method,
		cmdError: cmdError,
	}
}

func (e cpiError) Error() string {
	return fmt.Sprintf("CPI '%s' method responded with error: %s", e.method, e.cmdError)
}

func (e cpiError) Method() string {
	return e.method
}

func (e cpiError) Type() string {
	return e.cmdError.Type
}

func (e cpiError) Message() string {
	return e.cmdError.Message
}

func (e cpiError) OkToRetry() bool {
	return e.cmdError.OkToRetry
}
