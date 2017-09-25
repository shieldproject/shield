package fakes

import (
	"time"
)

type FakeAgentKiller struct {
}

func NewFakeAgentKiller() FakeAgentKiller {
	return FakeAgentKiller{}
}

func (a FakeAgentKiller) KillAgent(waitToKillAgentInterval time.Duration) {
	return
}
