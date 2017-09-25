// +build windows

package jobsupervisor

import (
	"fmt"
	"time"
)

func init() {
	// Change constant for unit test process labels to prevent collisions with
	// bosh deployed Job Supervisor
	serviceDescription = fmt.Sprintf("vcap_test_%d", time.Now().UnixNano())
}

func GetServiceDescription() string { return serviceDescription }

func SetPipeExePath(s string) (previous string) {
	previous = pipeExePath
	pipeExePath = s
	return previous
}

func GetPipeExePath() string {
	return pipeExePath
}
