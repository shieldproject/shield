package fakes

import (
	"github.com/cloudfoundry/bosh-agent/agentclient"
)

type FakeAgentClientFactory struct {
	CreateAgentClient agentclient.AgentClient
	CreateDirectorID  string
	CreateMbusURL     string
}

func NewFakeAgentClientFactory() *FakeAgentClientFactory {
	return &FakeAgentClientFactory{}
}

func (f *FakeAgentClientFactory) NewAgentClient(directorID, mbusURL string) agentclient.AgentClient {
	f.CreateDirectorID = directorID
	f.CreateMbusURL = mbusURL
	return f.CreateAgentClient
}
