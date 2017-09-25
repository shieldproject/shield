package fakes

import (
	"io"
)

type FakeClient struct {
	GetPath     string
	GetContents io.ReadCloser
	GetErr      error

	PutPath          string
	PutContents      string
	PutContentLength int64
	PutErr           error
}

func NewFakeClient() *FakeClient {
	return &FakeClient{}
}

func (c *FakeClient) Get(path string) (io.ReadCloser, error) {
	c.GetPath = path

	return c.GetContents, c.GetErr
}

func (c *FakeClient) Put(path string, content io.ReadCloser, contentLength int64) error {
	c.PutPath = path
	contentBytes := make([]byte, contentLength)
	content.Read(contentBytes)
	defer content.Close()
	c.PutContents = string(contentBytes)
	c.PutContentLength = contentLength

	return c.PutErr
}
