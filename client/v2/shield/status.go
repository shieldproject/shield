package shield

type Status struct {
	SHIELD struct {
		Version string `json:"version"`
		IP      string `json:"ip"`
		FQDN    string `json:"fqdn"`
		Env     string `json:"env"`
		MOTD    string `json:"motd"`
	} `json:"shield"`

	Health struct {
		Core      string `json:"core"`
		StorageOK bool   `json:"storage_ok"`
		JobsOK    bool   `json:"jobs_ok"`
	} `json:"health"`

	Storage []struct {
		Name   string `json:"name"`
		Health bool   `json:"healthy"`
	} `json:"storage"`

	Jobs []struct {
		UUID    string `json:"uuid"`
		Target  string `json:"target"`
		Job     string `json:"job"`
		Healthy bool   `json:"healthy"`
	} `json:"jobs"`

	Stats struct {
		Jobs        int   `json:"jobs"`
		Systems     int   `json:"systems"`
		Archives    int   `json:"archives"`
		StorageUsed int64 `json:"storage"`
		DailyDelta  int   `json:"daily"`
	} `json:"stats"`
}

func (c *Client) GlobalStatus() (*Status, error) {
	var st *Status
	return st, c.get("/v2/health", &st)
}

func (c *Client) TenantStatus(tenant *Tenant) (*Status, error) {
	var st *Status
	return st, c.get("/v2/tenants/"+tenant.UUID+"/health", &st)
}
