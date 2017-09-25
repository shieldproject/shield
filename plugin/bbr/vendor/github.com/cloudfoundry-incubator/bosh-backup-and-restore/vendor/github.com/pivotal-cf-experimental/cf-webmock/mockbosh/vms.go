package mockbosh

import (
	"fmt"

	"github.com/pivotal-cf-experimental/cf-webmock/mockhttp"
)

type vmsForDeploymentMock struct {
	*mockhttp.MockHttp
	deploymentName string
}

func VMsForDeployment(deploymentName string) *vmsForDeploymentMock {
	mock := &vmsForDeploymentMock{
		MockHttp:       mockhttp.NewMockedHttpRequest("GET", fmt.Sprintf("/deployments/%s/vms?format=full", deploymentName)),
		deploymentName: deploymentName,
	}
	return mock
}

func (t *vmsForDeploymentMock) RedirectsToTask(taskID int) *mockhttp.MockHttp {
	return t.RedirectsTo(taskURL(taskID))
}
