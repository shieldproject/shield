package shield

import (
	"bytes"
	"io/ioutil"
	"net/http"
)

func (c *Client) Curl(method, path, body string) (int, string, error) {
	var req *http.Request
	var err error

	switch method {
	case "GET", "DELETE":
		req, err = http.NewRequest(method, path, nil)
		if err != nil {
			return 0, "", err
		}
		req.Header.Set("Accept", "application/json")

	case "POST", "PUT", "PATCH":
		req, err = http.NewRequest(method, path, bytes.NewBufferString(body))
		if err != nil {
			return 0, "", err
		}
		req.Header.Set("Accept", "application/json")
		req.Header.Set("Content-type", "application/json")
	}

	res, err := c.curl(req)
	if err != nil {
		return 0, "", err
	}

	b, err := ioutil.ReadAll(res.Body)
	if err != nil {
		return res.StatusCode, "", err
	}

	return res.StatusCode, string(b), nil
}
