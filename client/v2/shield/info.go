package shield

type Info struct {
	Env     string `json:"env"`
	API     int    `json:"api"`
	Version string `json:"version,omitempty"`
}

func (c *Client) Info() (*Info, error) {
	nfo := &Info{}
	return nfo, c.get("/v2/info", nfo)
}
