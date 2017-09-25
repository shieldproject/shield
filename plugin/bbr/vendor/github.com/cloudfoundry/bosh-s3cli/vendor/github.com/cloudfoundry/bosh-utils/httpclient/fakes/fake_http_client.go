package fakes

import (
	"io/ioutil"
	"net/http"
	"strings"
)

type FakeHTTPClient struct {
	PostInputs  []postInput
	postOutputs []output

	GetInputs  []getInput
	getOutputs []output

	DeleteInputs  []deleteInput
	deleteOutputs []output

	PutInputs  []putInput
	putOutputs []output
}

type postInput struct {
	Payload  []byte
	Endpoint string
}

type putInput struct {
	Payload  []byte
	Endpoint string
}

type deleteInput struct {
	Endpoint string
}

type getInput struct {
	Endpoint string
}

type output struct {
	response *http.Response
	err      error
}

func NewFakeHTTPClient() *FakeHTTPClient {
	return &FakeHTTPClient{
		postOutputs:   []output{},
		putOutputs:    []output{},
		getOutputs:    []output{},
		deleteOutputs: []output{},
	}
}

func (c *FakeHTTPClient) Post(endpoint string, payload []byte) (*http.Response, error) {
	c.PostInputs = append(c.PostInputs, postInput{
		Payload:  payload,
		Endpoint: endpoint,
	})

	postReturn := c.postOutputs[0]
	c.postOutputs = c.postOutputs[1:]

	return postReturn.response, postReturn.err
}

func (c *FakeHTTPClient) PostCustomized(endpoint string, payload []byte, f func(*http.Request)) (*http.Response, error) {
	c.PostInputs = append(c.PostInputs, postInput{
		Payload:  payload,
		Endpoint: endpoint,
	})

	postReturn := c.postOutputs[0]
	c.postOutputs = c.postOutputs[1:]

	return postReturn.response, postReturn.err
}

func (c *FakeHTTPClient) Get(endpoint string) (*http.Response, error) {
	c.GetInputs = append(c.GetInputs, getInput{
		Endpoint: endpoint,
	})

	getReturn := c.getOutputs[0]
	c.getOutputs = c.getOutputs[1:]

	return getReturn.response, getReturn.err
}

func (c *FakeHTTPClient) GetCustomized(endpoint string, f func(*http.Request)) (*http.Response, error) {
	c.GetInputs = append(c.GetInputs, getInput{
		Endpoint: endpoint,
	})

	getReturn := c.getOutputs[0]
	c.getOutputs = c.getOutputs[1:]

	return getReturn.response, getReturn.err
}

func (c *FakeHTTPClient) Put(endpoint string, payload []byte) (*http.Response, error) {
	c.PutInputs = append(c.PutInputs, putInput{
		Payload:  payload,
		Endpoint: endpoint,
	})

	putReturn := c.putOutputs[0]
	c.putOutputs = c.putOutputs[1:]

	return putReturn.response, putReturn.err
}

func (c *FakeHTTPClient) PutCustomized(endpoint string, payload []byte, f func(*http.Request)) (*http.Response, error) {
	c.PutInputs = append(c.PutInputs, putInput{
		Payload:  payload,
		Endpoint: endpoint,
	})

	putReturn := c.putOutputs[0]
	c.putOutputs = c.putOutputs[1:]

	return putReturn.response, putReturn.err
}

func (c *FakeHTTPClient) Delete(endpoint string) (*http.Response, error) {
	c.DeleteInputs = append(c.DeleteInputs, deleteInput{
		Endpoint: endpoint,
	})

	deleteReturn := c.deleteOutputs[0]
	c.deleteOutputs = c.deleteOutputs[1:]

	return deleteReturn.response, deleteReturn.err
}

func (c *FakeHTTPClient) DeleteCustomized(endpoint string, f func(*http.Request)) (*http.Response, error) {
	c.DeleteInputs = append(c.DeleteInputs, deleteInput{
		Endpoint: endpoint,
	})

	deleteReturn := c.deleteOutputs[0]
	c.deleteOutputs = c.deleteOutputs[1:]

	return deleteReturn.response, deleteReturn.err
}

func (c *FakeHTTPClient) SetPostBehavior(body string, statusCode int, err error) {
	postResponse := &http.Response{
		Body:       ioutil.NopCloser(strings.NewReader(body)),
		StatusCode: statusCode,
	}
	c.postOutputs = append(c.postOutputs, output{
		response: postResponse,
		err:      err,
	})
}

func (c *FakeHTTPClient) SetPutBehavior(body string, statusCode int, err error) {
	putResponse := &http.Response{
		Body:       ioutil.NopCloser(strings.NewReader(body)),
		StatusCode: statusCode,
	}
	c.putOutputs = append(c.putOutputs, output{
		response: putResponse,
		err:      err,
	})
}

func (c *FakeHTTPClient) SetGetBehavior(body string, statusCode int, err error) {
	getResponse := &http.Response{
		Body:       ioutil.NopCloser(strings.NewReader(body)),
		StatusCode: statusCode,
	}

	c.getOutputs = append(c.getOutputs, output{
		response: getResponse,
		err:      err,
	})
}

func (c *FakeHTTPClient) SetDeleteBehavior(body string, statusCode int, err error) {
	deleteResponse := &http.Response{
		Body:       ioutil.NopCloser(strings.NewReader(body)),
		StatusCode: statusCode,
	}

	c.deleteOutputs = append(c.deleteOutputs, output{
		response: deleteResponse,
		err:      err,
	})
}
