package api_agent

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/url"
)

type URL struct {
	base *url.URL
}

func ParseURL(s string) (*URL, error) {
	u, err := url.Parse(s)
	if err != nil {
		return nil, err
	}

	return &URL{base: u}, nil
}

func (u *URL) AddParameter(key string, value interface{}) error {
	q := u.base.Query()
	switch value.(type) {
	case string:
		q.Add(key, value.(string))
	case bool:
		if value.(bool) {
			q.Add(key, "t")
		} else {
			q.Add(key, "f")
		}
	default:
		q.Add(key, fmt.Sprintf("%v", value))
	}
	u.base.RawQuery = q.Encode()
	return nil
}

func (u *URL) String() string {
	return u.base.String()
}

func (u *URL) Request(out interface{}, req *http.Request) error {
	client := &http.Client{}
	r, err := client.Do(req)
	if err != nil {
		return err
	}
	defer r.Body.Close()

	body, err := ioutil.ReadAll(r.Body)
	if err != nil {
		return err
	}

	err = json.Unmarshal(body, out)
	if err != nil {
		return err
	}

	return nil
}

func (u *URL) Get(out interface{}) error {
	r, err := http.NewRequest("GET", u.String(), nil)
	if err != nil {
		return err
	}
	return u.Request(out, r)
}

func (u *URL) Delete(out interface{}) error {
	r, err := http.NewRequest("DELETE", u.String(), nil)
	if err != nil {
		return err
	}
	return u.Request(out, r)
}

func (u *URL) Post(out interface{}, data string) error {
	r, err := http.NewRequest("POST", u.String(),
		bytes.NewBufferString(data))
	if err != nil {
		return err
	}
	r.Header.Set("Content-Type", "application/json")
	return u.Request(out, r)
}

func (u *URL) Put(out interface{}, data string) error {
	r, err := http.NewRequest("PUT", u.String(),
		bytes.NewBufferString(data))
	if err != nil {
		return err
	}
	r.Header.Set("Content-Type", "application/json")
	return u.Request(out, r)
}
