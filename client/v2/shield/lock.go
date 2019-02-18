package shield

func (c *Client) Lock() error {
	return c.post("/v2/lock", nil, nil)
}
