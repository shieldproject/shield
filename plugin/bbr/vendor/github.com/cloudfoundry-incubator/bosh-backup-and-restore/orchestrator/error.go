package orchestrator

import (
	"bytes"

	"fmt"

	"github.com/pkg/errors"
)

type customError struct {
	error
}

type LockError customError
type BackupError customError
type UnlockError customError
type CleanupError customError

func NewLockError(errorMessage string) LockError {
	return LockError{errors.New(errorMessage)}
}

func NewBackupError(errorMessage string) BackupError {
	return BackupError{errors.New(errorMessage)}
}

func NewPostBackupUnlockError(errorMessage string) UnlockError {
	return UnlockError{errors.New(errorMessage)}
}

func NewCleanupError(errorMessage string) CleanupError {
	return CleanupError{errors.New(errorMessage)}
}

func ConvertErrors(errs []error) error {
	if len(errs) == 0 {
		return nil
	}
	return Error(errs)
}

type Error []error

func (e Error) Error() string {
	return e.PrettyError(false)
}

func (e Error) PrettyError(includeStacktrace bool) string {
	if e.IsNil() {
		return ""
	}
	var buffer *bytes.Buffer = bytes.NewBufferString("")

	fmt.Fprintf(buffer, "%d error%s occurred:\n", len(e), e.getPostFix())
	for index, err := range e {
		fmt.Fprintf(buffer, "error %d:\n", index+1)
		if includeStacktrace {
			fmt.Fprintf(buffer, "%+v\n", err)
		} else {
			fmt.Fprintf(buffer, "%+v\n", err.Error())
		}
	}
	return buffer.String()
}

func (e Error) getPostFix() string {
	errorPostfix := ""
	if len(e) > 1 {
		errorPostfix = "s"
	}
	return errorPostfix
}

func (e Error) IsCleanup() bool {
	if len(e) == 1 {
		_, ok := e[0].(CleanupError)
		return ok
	}

	return false
}

func (err Error) IsPostBackup() bool {
	foundPostBackupError := false

	for _, e := range err {
		switch e.(type) {
		case UnlockError:
			foundPostBackupError = true
		case CleanupError:
			continue
		default:
			return false
		}
	}

	return foundPostBackupError
}

func (e Error) IsFatal() bool {
	return !e.IsNil() && !e.IsCleanup() && !e.IsPostBackup()
}

func (e Error) IsNil() bool {
	return len(e) == 0
}

func (e Error) Join(otherError Error) Error {
	return append(e, otherError...)
}

func ProcessError(errs Error) (int, string, string) {
	exitCode := 0

	for _, err := range errs {
		switch err.(type) {
		case LockError:
			exitCode = exitCode | 1<<2
		case UnlockError:
			exitCode = exitCode | 1<<3
		case CleanupError:
			exitCode = exitCode | 1<<4
		default:
			exitCode = exitCode | 1
		}
	}

	return exitCode, errs.Error(), errs.PrettyError(true)
}
