package director

import (
	"net/http"

	"encoding/json"
	bosherr "github.com/cloudfoundry/bosh-utils/errors"
)

type CloudConfigDiffResponse struct {
	Diff [][]interface{} `json:"diff"`
}

type CloudConfig struct {
	Properties string
}

type CloudConfigDiff struct {
	Diff [][]interface{}
}

func NewCloudConfigDiff(diff [][]interface{}) CloudConfigDiff {
	return CloudConfigDiff{
		Diff: diff,
	}
}

func (d DirectorImpl) LatestCloudConfig() (CloudConfig, error) {
	resps, err := d.client.CloudConfigs()
	if err != nil {
		return CloudConfig{}, err
	}

	if len(resps) == 0 {
		return CloudConfig{}, bosherr.Error("No cloud config")
	}

	return resps[0], nil
}

func (d DirectorImpl) UpdateCloudConfig(manifest []byte) error {
	return d.client.UpdateCloudConfig(manifest)
}

func (c Client) CloudConfigs() ([]CloudConfig, error) {
	var resps []CloudConfig

	err := c.clientRequest.Get("/cloud_configs?limit=1", &resps)
	if err != nil {
		return resps, bosherr.WrapErrorf(err, "Finding cloud configs")
	}

	return resps, nil
}

func (c Client) UpdateCloudConfig(manifest []byte) error {
	path := "/cloud_configs"

	setHeaders := func(req *http.Request) {
		req.Header.Add("Content-Type", "text/yaml")
	}

	_, _, err := c.clientRequest.RawPost(path, manifest, setHeaders)
	if err != nil {
		return bosherr.WrapErrorf(err, "Updating cloud config")
	}

	return nil
}

func (d DirectorImpl) DiffCloudConfig(manifest []byte) (CloudConfigDiff, error) {
	resp, err := d.client.DiffCloudConfig(manifest)
	if err != nil {
		return CloudConfigDiff{}, err
	}

	return NewCloudConfigDiff(resp.Diff), nil
}

func (c Client) DiffCloudConfig(manifest []byte) (CloudConfigDiffResponse, error) {
	setHeaders := func(req *http.Request) {
		req.Header.Add("Content-Type", "text/yaml")
	}

	var resp CloudConfigDiffResponse

	respBody, response, err := c.clientRequest.RawPost("/cloud_configs/diff", manifest, setHeaders)
	if err != nil {
		if response != nil && response.StatusCode == http.StatusNotFound {
			// return empty diff, just for compatibility with directors which don't have the endpoint
			return resp, nil
		} else {
			return resp, bosherr.WrapErrorf(err, "Fetching diff result")
		}
	}

	err = json.Unmarshal(respBody, &resp)
	if err != nil {
		return resp, bosherr.WrapError(err, "Unmarshaling Director response")
	}

	return resp, nil
}
