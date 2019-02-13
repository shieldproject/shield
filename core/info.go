package core

type Info struct {
	Version string `json:"version,omitempty"`
	IP      string `json:"ip,omitempty"`
	Env     string `json:"env,omitempty"`
	Color   string `json:"color,omitempty"`
	MOTD    string `json:"motd,omitempty"`

	API int `json:"api"`
}

func (core *Core) checkInfo(auth bool) Info {
	info := Info{
		IP:    core.ip,
		MOTD:  core.motd,
		Color: core.color,
		Env:   core.env,
		API:   2,
	}
	if auth {
		info.Version = Version
	}
	return info
}
