package agentclient

import (
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
	return true, err
}
