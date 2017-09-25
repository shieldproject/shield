package action

import "time"

type Killer interface {
	KillAgent(waitToKillAgentInterval time.Duration)
}
