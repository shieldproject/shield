package agentclient

import (
	"regexp"

	bosherr "github.com/cloudfoundry/bosh-utils/errors"
	boshretry "github.com/cloudfoundry/bosh-utils/retrystrategy"
)

type pingRetryable struct {
	agentClient AgentClient
}

func NewPingRetryable(agentClient AgentClient) boshretry.Retryable {
	return &pingRetryable{
		agentClient: agentClient,
	}
}

func (r *pingRetryable) Attempt() (bool, error) {
	_, err := r.agentClient.Ping()

	if err != nil {
		for {
			if innerErr, ok := err.(bosherr.ComplexError); ok {
				err = innerErr.Cause
			} else {
				break
			}
		}
		r, _ := regexp.Compile("x509: ")
		if r.MatchString(err.Error()) {
			return false, err
		}
	}

	return true, err
}
