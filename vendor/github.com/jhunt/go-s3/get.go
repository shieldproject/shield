package s3

import (
	"io"
	"io/ioutil"
)

func (c *Client) Get(key string) (io.Reader, error) {
	res, err := c.get(key)
	if err != nil {
		return nil, err
	}

	if res.StatusCode != 200 {
		b, err := ioutil.ReadAll(res.Body)
		if err != nil {
			return nil, err
		}
		return nil, ResponseError(b)
	}

	return res.Body, nil
}
