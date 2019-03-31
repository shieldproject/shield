package shield

import (
	"bytes"
	"encoding/json"
	"io/ioutil"
	"net/http"
)

func (c *Client) CheckTimespec(spec string) (bool, string, error) {
	b, err := json.Marshal(struct {
		Timespec string `json:"timespec"`
	}{
		Timespec: spec,
	})
	if err != nil {
		return false, "", err
	}

	req, err := http.NewRequest("POST", "/v2/ui/check/timespec", bytes.NewBuffer(b))
	if err != nil {
		return false, "", err
	}

	req.Header.Set("Accept", "application/json")
	req.Header.Set("Content-type", "application/json")

	res, err := c.curl(req)
	if err != nil {
		return false, "", err
	}
	defer res.Body.Close()

	if res.StatusCode != 200 {
		return false, "", nil
	}

	var out struct {
		OK string `json:"ok"`
	}
	b, err = ioutil.ReadAll(res.Body)
	if err != nil {
		return false, "", err
	}

	err = json.Unmarshal(b, &out)
	if err != nil {
		return false, "", err
	}

	return true, out.OK, nil
}
