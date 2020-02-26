package shield

import (
	"io"
	"net/http"
)

func (c *Client) Import(in io.Reader) error {
	var req *http.Request
	var err error

	req, err = http.NewRequest("POST", "/v2/import", in)
	if err != nil {
		return err
	}
	req.Header.Set("Accept", "application/json")
	req.Header.Set("Content-type", "application/json")

	return c.request(req, nil)
}
