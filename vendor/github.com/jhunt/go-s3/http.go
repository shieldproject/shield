package s3

import (
	"bytes"
	"fmt"
	"net/http"
	"regexp"
)

func (c *Client) url(path string) string {
	if path == "" || path[0:1] != "/" {
		path = "/" + path
	}
	scheme := c.Protocol
	if scheme == "" {
		scheme = "https"
	}

	if c.Bucket == "" {
		return fmt.Sprintf("%s://%s%s", scheme, c.domain(), path)
	}

	if c.UsePathBuckets {
		return fmt.Sprintf("%s://%s/%s%s", scheme, c.domain(), c.Bucket, path)
	} else {
		return fmt.Sprintf("%s://%s.%s%s", scheme, c.Bucket, c.domain(), path)
	}
}

func (c *Client) request(method, path string, payload []byte, headers *http.Header) (*http.Response, error) {
	in := bytes.NewBuffer(payload)
	req, err := http.NewRequest(method, c.url(path), in)
	if err != nil {
		return nil, err
	}

	/* copy in any headers */
	if headers != nil {
		for header, values := range *headers {
			for _, value := range values {
				req.Header.Add(header, value)
			}
		}
	}

	/* sign the request */
	req.ContentLength = int64(len(payload))
	req.Header.Set("Authorization", c.signature(req, payload))

	/* stupid continuation tokens sometimes have literal +'s in them */
	req.URL.RawQuery = regexp.MustCompile(`\+`).ReplaceAllString(req.URL.RawQuery, "%2B")

	/* optional debugging */
	if err := c.traceRequest(req); err != nil {
		return nil, err
	}

	/* submit the request */
	res, err := c.ua.Do(req)
	if err != nil {
		return nil, err
	}

	/* optional debugging */
	if err := c.traceResponse(res); err != nil {
		return nil, err
	}
	return res, nil
}

func (c *Client) post(path string, payload []byte, headers *http.Header) (*http.Response, error) {
	return c.request("POST", path, payload, headers)
}

func (c *Client) put(path string, payload []byte, headers *http.Header) (*http.Response, error) {
	return c.request("PUT", path, payload, headers)
}

func (c *Client) get(path string, headers *http.Header) (*http.Response, error) {
	return c.request("GET", path, nil, headers)
}

func (c *Client) delete(path string, headers *http.Header) (*http.Response, error) {
	return c.request("DELETE", path, nil, headers)
}
