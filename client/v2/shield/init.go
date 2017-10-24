package shield

func (c *Client) Initialize(master string) error {
	in := struct {
		Master string `json:"master"`
	}{
		Master: master,
	}
	return c.post("/v2/init", in, nil)
}
