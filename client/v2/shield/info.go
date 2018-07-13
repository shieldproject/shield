package shield

type Info struct {
	Version string `json:"version,omitempty"`
	IP      string `json:"ip,omitempty"`
	Env     string `json:"env,omitempty"`
	Color   string `json:"color,omitempty"`
	MOTD    string `json:"motd,omitempty"`

	API int `json:"api"`
}

func (c *Client) Info() (*Info, error) {
	nfo := &Info{}
	return nfo, c.get("/v2/info", nfo)
}
