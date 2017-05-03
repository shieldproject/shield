package api

type Status struct {
	Name    string `json:"name"`
	Version string `json:"version"`
}

func GetStatus() (Status, error) {
	uri, err := ShieldURI("/v1/status")
	if err != nil {
		return Status{}, err
	}

	var data Status
	return data, uri.Get(&data)
}
