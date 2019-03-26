package vault

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"strings"
)

func (c *Client) request(method, url string, data interface{}) (*http.Request, error) {
	if data == nil {
		return http.NewRequest(method, url, nil)
	}
	b, err := json.Marshal(data)
	if err != nil {
		return nil, err
	}

	return http.NewRequest(method, url, bytes.NewBuffer(b))
}

func (c *Client) Do(method, url string, data interface{}) (*http.Response, error) {
	if !strings.HasPrefix(url, "/") {
		url = "/v1/secret/" + url
	}
	url = c.URL + url

	req, err := c.request(method, url, data)
	if err != nil {
		return nil, err
	}

	req.Header.Add("X-Vault-Token", c.Token)
	return c.HTTP.Do(req)
}

func (c *Client) Get(path string, out interface{}) (bool, error) {
	res, err := c.Do("GET", path, nil)
	if err != nil {
		return false, err
	}
	defer res.Body.Close()

	if res.StatusCode == 404 {
		return false, nil
	}
	if res.StatusCode == 204 {
		return true, nil
	}

	if res.StatusCode == 200 {
		if out == nil {
			return true, nil
		}

		b, err := ioutil.ReadAll(res.Body)
		if err != nil {
			return true, err
		}

		return true, json.Unmarshal(b, out)
	}

	/* everything else is an eror */
	return false, fmt.Errorf("API %s", res.Status)
}

func (c *Client) post(method, path string, in, out interface{}) error {
	res, err := c.Do(method, path, in)
	if err != nil {
		return err
	}
	defer res.Body.Close()

	if res.StatusCode == 204 {
		return nil
	}
	if res.StatusCode == 200 {
		if out == nil {
			return nil
		}

		b, err := ioutil.ReadAll(res.Body)
		if err != nil {
			return err
		}

		return json.Unmarshal(b, &out)
	}

	return fmt.Errorf("API %s", res.Status)
}

func (c *Client) Post(path string, in, out interface{}) error {
	return c.post("POST", path, in, out)
}

func (c *Client) Put(path string, in, out interface{}) error {
	return c.post("PUT", path, in, out)
}

func (c *Client) Delete(path string) error {
	res, err := c.Do("DELETE", path, nil)
	if err != nil {
		return err
	}
	defer res.Body.Close()

	if res.StatusCode == 204 {
		return nil
	}

	return fmt.Errorf("API %s", res.Status)
}
