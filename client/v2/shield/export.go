package shield

import (
	"io"
	"net/http"
)

func (c *Client) Export(task string) (io.Reader, error) {
	var req *http.Request
	var err error

	req, err = http.NewRequest("GET", "/v2/export?task="+task, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Accept", "application/json")
	req.Header.Set("Content-type", "application/json")

	res, err := c.curl(req)
	if err != nil {
		return nil, err
	}

	return res.Body, nil
}
