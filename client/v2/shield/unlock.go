package shield

func (c *Client) Unlock(master string) error {
	in := struct {
		Master string `json:"master"`
	}{
		Master: master,
	}
	return c.post("/v2/unlock", in, nil)
}
