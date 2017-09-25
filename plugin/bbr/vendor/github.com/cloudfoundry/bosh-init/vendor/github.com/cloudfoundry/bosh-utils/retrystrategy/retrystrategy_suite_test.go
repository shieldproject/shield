package retrystrategy_test

import (
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	"testing"
)

func TestRetrystrategy(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Retrystrategy Suite")
}

type simpleRetryable struct {
	attemptOutputs []attemptOutput
	Attempts       int
}

type attemptOutput struct {
	IsRetryable bool
	AttemptErr  error
}

func newSimpleRetryable(attemptOutputs []attemptOutput) *simpleRetryable {
	return &simpleRetryable{
		attemptOutputs: attemptOutputs,
	}
}

func (r *simpleRetryable) Attempt() (bool, error) {
	r.Attempts++

	if len(r.attemptOutputs) > 0 {
		attemptOutput := r.attemptOutputs[0]
		r.attemptOutputs = r.attemptOutputs[1:]
		return attemptOutput.IsRetryable, attemptOutput.AttemptErr
	}

	return true, nil
}
