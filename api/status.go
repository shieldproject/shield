package api

type Status struct {
	Name    string `json:"name"`
	Version string `json:"version"`
}

func GetStatus() (Status, error) {
	uri := ShieldURI("/v1/status")

	var data Status
	return data, uri.Get(&data)
}
