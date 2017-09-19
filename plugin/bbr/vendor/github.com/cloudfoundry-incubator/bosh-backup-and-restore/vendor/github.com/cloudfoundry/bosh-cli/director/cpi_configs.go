package director

import (
	"net/http"

	bosherr "github.com/cloudfoundry/bosh-utils/errors"
)

type CPIConfig struct {
	Properties string
}

func (d DirectorImpl) LatestCPIConfig() (CPIConfig, error) {
	resps, err := d.client.CPIConfigs()
	if err != nil {
		return CPIConfig{}, err
	}

	if len(resps) == 0 {
		return CPIConfig{}, bosherr.Error("No CPI config")
	}

	return resps[0], nil
}

func (d DirectorImpl) UpdateCPIConfig(manifest []byte) error {
	return d.client.UpdateCPIConfig(manifest)
}

func (c Client) CPIConfigs() ([]CPIConfig, error) {
	var resps []CPIConfig

	err := c.clientRequest.Get("/cpi_configs?limit=1", &resps)
	if err != nil {
		return resps, bosherr.WrapErrorf(err, "Finding CPI configs")
	}

	return resps, nil
}

func (c Client) UpdateCPIConfig(manifest []byte) error {
	path := "/cpi_configs"

	setHeaders := func(req *http.Request) {
		req.Header.Add("Content-Type", "text/yaml")
	}

	_, _, err := c.clientRequest.RawPost(path, manifest, setHeaders)
	if err != nil {
		return bosherr.WrapErrorf(err, "Updating CPI config")
	}

	return nil
}
