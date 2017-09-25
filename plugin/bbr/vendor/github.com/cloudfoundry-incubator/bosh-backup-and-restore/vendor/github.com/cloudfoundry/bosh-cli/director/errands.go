package director

import (
	"encoding/json"
	"fmt"
	"net/http"

	bosherr "github.com/cloudfoundry/bosh-utils/errors"
	"io"
	"strings"
)

type Errand struct {
	Name string // e.g. "acceptance-tests"
}

type ErrandResult struct {
	ExitCode int

	Stdout string
	Stderr string

	LogsBlobstoreID string
	LogsSHA1        string
}

type ErrandRunResp struct {
	ExitCode int `json:"exit_code"`

	Stdout string
	Stderr string

	Logs struct {
		BlobstoreID string `json:"blobstore_id"`
		SHA1        string `json:"sha1"`
	} `json:"logs"`
}

func (d DeploymentImpl) Errands() ([]Errand, error) {
	return d.client.Errands(d.name)
}

func (d DeploymentImpl) RunErrand(name string, keepAlive bool, whenChanged bool) ([]ErrandResult, error) {
	resp, err := d.client.RunErrand(d.name, name, keepAlive, whenChanged)
	if err != nil {
		return []ErrandResult{}, err
	}

	var result []ErrandResult

	for _, value := range resp {
		errandResult := ErrandResult{
			ExitCode: value.ExitCode,

			Stdout: value.Stdout,
			Stderr: value.Stderr,

			LogsBlobstoreID: value.Logs.BlobstoreID,
			LogsSHA1:        value.Logs.SHA1,
		}
		result = append(result, errandResult)
	}

	return result, nil
}

func (c Client) Errands(deploymentName string) ([]Errand, error) {
	var errands []Errand

	if len(deploymentName) == 0 {
		return errands, bosherr.Error("Expected non-empty deployment name")
	}

	path := fmt.Sprintf("/deployments/%s/errands", deploymentName)

	err := c.clientRequest.Get(path, &errands)
	if err != nil {
		return errands, bosherr.WrapErrorf(err, "Finding errands")
	}

	return errands, nil
}

func (c Client) RunErrand(deploymentName, name string, keepAlive bool, whenChanged bool) ([]ErrandRunResp, error) {
	var resp []ErrandRunResp

	if len(deploymentName) == 0 {
		return resp, bosherr.Error("Expected non-empty deployment name")
	}

	if len(name) == 0 {
		return resp, bosherr.Error("Expected non-empty errand name")
	}

	path := fmt.Sprintf("/deployments/%s/errands/%s/runs", deploymentName, name)

	body := map[string]bool{"keep-alive": keepAlive, "when-changed": whenChanged}

	reqBody, err := json.Marshal(body)
	if err != nil {
		return resp, bosherr.WrapErrorf(err, "Marshaling request body")
	}

	setHeaders := func(req *http.Request) {
		req.Header.Add("Content-Type", "application/json")
	}

	resultBytes, err := c.taskClientRequest.PostResult(path, reqBody, setHeaders)
	if err != nil {
		return resp, bosherr.WrapErrorf(err, "Running errand '%s'", name)
	}

	dec := json.NewDecoder(strings.NewReader(string(resultBytes)))

	for {
		var errandRunResponse ErrandRunResp
		if err := dec.Decode(&errandRunResponse); err == io.EOF {
			break
		} else if err != nil {
			return nil, bosherr.WrapErrorf(err, "Unmarshaling errand result")
		}
		resp = append(resp, errandRunResponse)
	}

	return resp, nil
}
