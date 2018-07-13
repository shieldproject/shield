package shield

type Status struct {
	Health  StatusHealth  `json:"health"`
	Storage StatusStorage `json:"storage"`
	Jobs    StatusJobs    `json:"jobs"`
	Stats   StatusStats   `json:"stats"`
}

type StatusHealth struct {
	Core      string `json:"core"`
	StorageOK bool   `json:"storage_ok"`
	JobsOK    bool   `json:"jobs_ok"`
}

type StatusStorage []struct {
	Name   string `json:"name"`
	Health bool   `json:"healthy"`
}

type StatusJobs []struct {
	UUID    string `json:"uuid"`
	Target  string `json:"target"`
	Job     string `json:"job"`
	Healthy bool   `json:"healthy"`
}

type StatusStats struct {
	Jobs        int   `json:"jobs"`
	Systems     int   `json:"systems"`
	Archives    int   `json:"archives"`
	StorageUsed int64 `json:"storage"`
	DailyDelta  int   `json:"daily"`
}

func (c *Client) GlobalStatus() (*Status, error) {
	var st *Status
	return st, c.get("/v2/health", &st)
}

func (c *Client) TenantStatus(tenant *Tenant) (*Status, error) {
	var st *Status
	return st, c.get("/v2/tenants/"+tenant.UUID+"/health", &st)
}
