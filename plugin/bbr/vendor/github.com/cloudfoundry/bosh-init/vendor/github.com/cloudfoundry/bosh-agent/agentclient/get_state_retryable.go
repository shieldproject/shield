package agentclient

import (
	bosherr "github.com/cloudfoundry/bosh-utils/errors"
	boshretry "github.com/cloudfoundry/bosh-utils/retrystrategy"
)

type getStateRetryable struct {
	agentClient AgentClient
}

func NewGetStateRetryable(agentClient AgentClient) boshretry.Retryable {
	return &getStateRetryable{
		agentClient: agentClient,
	}
}

func (r *getStateRetryable) Attempt() (bool, error) {
	stateResponse, err := r.agentClient.GetState()
	if err != nil {
		return false, err
	}

	if stateResponse.JobState == "running" {
		return true, nil
	}

	return true, bosherr.Errorf("Received non-running job state: '%s'", stateResponse.JobState)
}
