package fakes

import (
	"net/http"

	boshhttp "github.com/cloudfoundry/bosh-utils/http"
)

type doInput struct {
	req *http.Request
}

type doOutput struct {
	resp *http.Response
	err  error
}

type FakeClient struct {
	StatusCode        int
	CallCount         int
	Error             error
	returnNilResponse bool
	RequestBodies     []string
	Requests          []*http.Request
	responseMessage   string

	doBehavior []doOutput
}

func NewFakeClient() (fakeClient *FakeClient) {
	fakeClient = &FakeClient{}
	return
}

func (c *FakeClient) SetMessage(message string) {
	c.responseMessage = message
}

func (c *FakeClient) SetNilResponse() {
	c.returnNilResponse = true
}

func (c *FakeClient) Do(req *http.Request) (*http.Response, error) {
	c.CallCount++

	if req.Body != nil {
		content, err := boshhttp.ReadAndClose(req.Body)
		if err != nil {
			return nil, err
		}
		c.RequestBodies = append(c.RequestBodies, string(content))
	}
	c.Requests = append(c.Requests, req)

	if len(c.doBehavior) > 0 {
		output := c.doBehavior[0]
		c.doBehavior = c.doBehavior[1:]
		return output.resp, output.err
	}

	var resp *http.Response
	if !c.returnNilResponse {
		resp = &http.Response{
			Body:       boshhttp.NewStringReadCloser(c.responseMessage),
			StatusCode: c.StatusCode,
		}
	}
	err := c.Error

	return resp, err
}

func (c *FakeClient) AddDoBehavior(resp *http.Response, err error) {
	c.doBehavior = append(c.doBehavior, doOutput{resp: resp, err: err})
}
