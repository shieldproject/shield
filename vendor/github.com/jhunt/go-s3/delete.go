package s3

import (
	"io/ioutil"
)

func (c *Client) Delete(path string) error {
	res, err := c.delete(path)
	if err != nil {
		return err
	}

	if res.StatusCode != 204 {
		b, err := ioutil.ReadAll(res.Body)
		if err != nil {
			return err
		}
		return ResponseError(b)
	}

	return nil
}
