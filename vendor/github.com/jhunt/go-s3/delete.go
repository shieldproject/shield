package s3

func (c *Client) Delete(path string) error {
	res, err := c.delete(path, nil)
	if err != nil {
		return err
	}

	if res.StatusCode != 204 {
		return ResponseError(res)
	}

	return nil
}
