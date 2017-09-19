package mockbosh

import "github.com/pivotal-cf-experimental/cf-webmock/mockhttp"

type deleteDeployMock struct {
	*mockhttp.MockHttp
}

func DeleteDeployment(deploymentName string) *deleteDeployMock {
	return &deleteDeployMock{
		MockHttp: mockhttp.NewMockedHttpRequest("DELETE", "/deployments/"+deploymentName),
	}
}

func (d *deleteDeployMock) RedirectsToTask(taskID int) *mockhttp.MockHttp {
	return d.RedirectsTo(taskURL(taskID))
}
