// +build windows

package winsvc

import (
	"fmt"
	"time"

	"golang.org/x/sys/windows/svc"
)

// ServiceError records an error and the operation and service that caused it.
type ServiceError struct {
	Op   string
	Name string
	Err  error
}

func (e *ServiceError) Error() string {
	return "winsvc: " + e.Op + " (" + e.Name + "): " + e.Err.Error()
}

// TimeoutError records an error from a timeout, the timeout itself and the time
// that elapsed before the timeout was triggered.
type TimeoutError struct {
	Timeout time.Duration
	Elapsed time.Duration
}

func (e *TimeoutError) Error() string {
	return "operation exceeded timeout (" + e.Timeout.String() + ") after: " +
		e.Elapsed.String()
}

// TransitionError records an error that occurred waiting for a service to
// transition state and the service that caused it and it's state when the
// error occurred.
type TransitionError struct {
	Msg      string        // error message
	Name     string        // service name
	Status   svc.Status    // service status
	WaitHint time.Duration // calculated WaitHint
	Duration time.Duration // time elapsed waiting for the transition
}

func (e *TransitionError) Error() string {
	const format = "winsvc: %s: Service %s: State: %s Checkpoint: %d " +
		"WaitHint: %s TimeElapsed: %s"
	return fmt.Sprintf(format, e.Msg, e.Name, svcStateString(e.Status.State),
		e.Status.CheckPoint, e.WaitHint, e.Duration)
}

// StartError records an error that resulted from a service failing to run
// after a successful call to start.
type StartError struct {
	Name   string
	Status svc.Status
}

func (e *StartError) Error() string {
	// TODO (CEV): Improve error message
	return fmt.Sprintf("winsvc: start service: service not started. "+
		"Name: %s State: %s Checkpoint: %d WaitHint: %d", e.Name,
		svcStateString(e.Status.State), e.Status.CheckPoint, e.Status.WaitHint)
}
