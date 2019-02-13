package shield

func (c *Client) Rekey(oldmaster, newmaster string, rotateFixed bool) (string, error) {
	in := struct {
		OldMaster   string `json:"current"`
		NewMaster   string `json:"new"`
		RotateFixed bool   `json:"rotate_fixed_key"`
	}{
		OldMaster:   oldmaster,
		NewMaster:   newmaster,
		RotateFixed: rotateFixed,
	}

	out := struct {
		FixedKey string `json:"fixed_key"`
	}{}

	err := c.post("/v2/rekey", in, &out)

	return out.FixedKey, err
}
