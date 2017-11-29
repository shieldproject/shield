package s3

import (
	"bytes"
	"fmt"
	"net/http"
)

func (c *Client) url(path string) string {
	if path == "" || path[0:1] != "/" {
		path = "/" + path
	}
	scheme := c.Protocol
	if scheme == "" {
		scheme = "https"
	}

	return fmt.Sprintf("%s://%s.%s%s", scheme, c.bucket(), c.domain(), path)
}

func (c *Client) request(method, path string, payload []byte) (*http.Response, error) {
	in := bytes.NewBuffer(payload)
	req, err := http.NewRequest(method, c.url(path), in)
	if err != nil {
		return nil, err
	}

	/* sign the request */
	req.ContentLength = int64(len(payload))
	req.Header.Set("Authorization", c.signature(req, payload))

	res, err := c.ua.Do(req)
	if err != nil {
		return nil, err
	}
	return res, nil
}

func (c *Client) post(path string, payload []byte) (*http.Response, error) {
	return c.request("POST", path, payload)
}

func (c *Client) put(path string, payload []byte) (*http.Response, error) {
	return c.request("PUT", path, payload)
}

func (c *Client) get(path string) (*http.Response, error) {
	return c.request("GET", path, nil)
}

func (c *Client) delete(path string) (*http.Response, error) {
	return c.request("DELETE", path, nil)
}
