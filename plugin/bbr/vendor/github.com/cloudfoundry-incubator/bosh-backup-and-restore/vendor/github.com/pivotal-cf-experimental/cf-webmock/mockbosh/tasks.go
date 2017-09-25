package mockbosh

import (
	"fmt"

	"github.com/pivotal-cf-experimental/cf-webmock/mockhttp"
)

type tasksMock struct {
	*mockhttp.MockHttp
}

func Tasks(deploymentName string) *tasksMock {
	mock := &tasksMock{MockHttp: mockhttp.NewMockedHttpRequest("GET", fmt.Sprintf("/tasks?deployment=%s", deploymentName))}
	return mock
}

func (t *tasksMock) RespondsWithNoTasks() *mockhttp.MockHttp {
	return t.RespondsWithJson([]interface{}{})
}

func (t *tasksMock) RespondsWithATaskContainingState(provisioningTaskState string, description string) *mockhttp.MockHttp {
	return t.RespondsWithJson([]interface{}{
		map[string]string{
			"state":       provisioningTaskState,
			"description": description,
		},
	})
}

func (t *tasksMock) RespondsWithATask(task interface{}) *mockhttp.MockHttp {
	return t.RespondsWithJson([]interface{}{task})
}

func taskURL(taskID int) string {
	return fmt.Sprintf("/tasks/%d", taskID)
}
