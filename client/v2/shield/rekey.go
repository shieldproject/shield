package shield

func (c *Client) Rekey(oldmaster, newmaster string) error {
	in := struct {
		OldMaster string `json:"current"`
		NewMaster string `json:"new"`
	}{
		OldMaster: oldmaster,
		NewMaster: newmaster,
	}
	return c.post("/v2/rekey", in, nil)
}
