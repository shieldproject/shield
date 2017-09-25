package http

//go:generate mockgen -source=agent_client_factory.go -package=mocks -destination=mocks/mocks.go

import (
	"time"

	"github.com/cloudfoundry/bosh-agent/agentclient"
	"github.com/cloudfoundry/bosh-utils/httpclient"
	boshlog "github.com/cloudfoundry/bosh-utils/logger"
)

type AgentClientFactory interface {
	NewAgentClient(directorID, mbusURL string) agentclient.AgentClient
}

type agentClientFactory struct {
	getTaskDelay time.Duration
	logger       boshlog.Logger
}

func NewAgentClientFactory(
	getTaskDelay time.Duration,
	logger boshlog.Logger,
) AgentClientFactory {
	return &agentClientFactory{
		getTaskDelay: getTaskDelay,
		logger:       logger,
	}
}

func (f *agentClientFactory) NewAgentClient(directorID, mbusURL string) agentclient.AgentClient {
	httpClient := httpclient.NewHTTPClient(httpclient.DefaultClient, f.logger)
	return NewAgentClient(mbusURL, directorID, f.getTaskDelay, 10, httpClient, f.logger)
}
