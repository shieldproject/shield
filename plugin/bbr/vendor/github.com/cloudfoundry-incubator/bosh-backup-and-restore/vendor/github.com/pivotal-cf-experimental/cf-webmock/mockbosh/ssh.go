package mockbosh

import (
	"encoding/json"
	"fmt"

	. "github.com/onsi/gomega"
	"github.com/pivotal-cf-experimental/cf-webmock/mockhttp"
)

type sshMock struct {
	*mockhttp.MockHttp
	instanceGroup string
	callback      func(string, string)
}

func StartSSHSession(deploymentName string) *sshMock {
	mock := &sshMock{
		MockHttp: mockhttp.NewMockedHttpRequest("POST", fmt.Sprintf("/deployments/%s/ssh", deploymentName)),
	}
	mock.SetResponseCallback(mock.verifyRequest)
	return mock
}

var CleanupSSHSession = StartSSHSession

func (mock *sshMock) RedirectsToTask(taskID int) *mockhttp.MockHttp {
	return mock.RedirectsTo(taskURL(taskID))
}

func (mock *sshMock) ForInstanceGroup(instanceGroup string) *sshMock {
	mock.instanceGroup = instanceGroup
	return mock
}
func (mock *sshMock) SetSSHResponseCallback(callback func(string, string)) *sshMock {
	mock.callback = callback
	return mock
}

func (mock *sshMock) verifyRequest(body []byte) {
	response := map[string]interface{}{}
	Expect(json.Unmarshal(body, &response)).To(Succeed())
	params := response["params"].(map[string]interface{})
	target := response["target"].(map[string]interface{})

	if mock.instanceGroup != "" {
		Expect(target["job"]).To(Equal(mock.instanceGroup))
	}

	if mock.callback != nil {
		mock.callback(params["user"].(string), params["public_key"].(string))
	}
}
