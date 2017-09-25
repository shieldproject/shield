package action

import (
	"os"
	"time"
)

type AgentKiller struct {
}

func NewAgentKiller() AgentKiller {
	return AgentKiller{}
}

func (a AgentKiller) KillAgent(waitToKillAgentInterval time.Duration) {
	time.Sleep(waitToKillAgentInterval)

	os.Exit(0)

	return
}
