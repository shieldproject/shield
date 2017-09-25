package mockbosh

import (
	"gopkg.in/yaml.v2"

	. "github.com/onsi/gomega"
	"github.com/pivotal-cf-experimental/cf-webmock/mockhttp"
)

type deployMock struct {
	expectedManifest []byte
	*mockhttp.MockHttp
}

func Deploy() *deployMock {
	mock := &deployMock{MockHttp: mockhttp.NewMockedHttpRequest("POST", "/deployments")}
	mock.WithContentType("text/yaml")
	return mock
}

func (d *deployMock) RedirectsToTask(taskID int) *mockhttp.MockHttp {
	return d.RedirectsTo(taskURL(taskID))
}

func (d *deployMock) WithRawManifest(manifest []byte) *deployMock {
	d.WithBody(string(manifest))
	return d
}

func (d *deployMock) WithManifest(manifest interface{}) *deployMock {
	d.WithBody(toYaml(manifest))
	return d
}

func toYaml(obj interface{}) string {
	data, err := yaml.Marshal(obj)
	Expect(err).NotTo(HaveOccurred())
	return string(data)
}
