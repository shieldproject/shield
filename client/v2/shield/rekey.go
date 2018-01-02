package shield

func (c *Client) Rekey(oldmaster, newmaster string, rekeyDR bool) (string, error) {
	in := struct {
		OldMaster string `json:"current"`
		NewMaster string `json:"new"`
		RekeyDR   bool   `json:"rekey_dr"`
	}{
		OldMaster: oldmaster,
		NewMaster: newmaster,
		RekeyDR:   rekeyDR,
	}

	out := struct {
		DisasterKey string `json:"disaster_key"`
	}{
		DisasterKey: "",
	}

	err := c.post("/v2/rekey", in, &out)

	return out.DisasterKey, err
}
