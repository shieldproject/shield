package api

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/http/httputil"
	"net/url"
	"os"
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

func (u *URL) MaybeAddParameter(key string, value interface{}) error {
	if s, ok := value.(string); ok {
		if s != "" {
			return u.AddParameter(key, value)
		}
		return nil
	}

	if yn, ok := value.(YesNo); ok {
		if yn.On {
			return u.AddParameter(key, yn.Yes)
		}
		return nil
	}

	return u.AddParameter(key, value)
}

func (u *URL) String() string {
	return u.base.String()
}

func httpTracingEnabled() bool {
	shieldTrace := os.Getenv("SHIELD_TRACE")
	return shieldTrace != "" && shieldTrace != "0" && shieldTrace != "false" && shieldTrace != "no"
}
func debugRequest(req *http.Request) {
	if httpTracingEnabled() {
		r, _ := httputil.DumpRequest(req, true)
		fmt.Fprintf(os.Stderr, "Request:\n%s\n---------------------------\n", r)
	}
}
func debugResponse(res *http.Response) {
	if httpTracingEnabled() {
		r, _ := httputil.DumpResponse(res, true)
		fmt.Fprintf(os.Stderr, "Response:\n%s\n---------------------------\n", r)
	}
}

func (u *URL) Request(out interface{}, req *http.Request) error {
	r, err := makeRequest(req)
	if err != nil {
		return err
	}

	if r.StatusCode == 200 {
		if out != nil {
			body, err := ioutil.ReadAll(r.Body)
			if err != nil {
				return err
			}

			err = json.Unmarshal(body, out)
			if err != nil {
				return err
			}
		}
	} else {
		return getAPIError(r)
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

func (u *URL) Patch(out interface{}, data string) error {
	r, err := http.NewRequest("PATCH", u.String(),
		bytes.NewBufferString(data))
	if err != nil {
		return err
	}
	r.Header.Set("Content-Type", "application/json")
	return u.Request(out, r)
}

func makeRequest(req *http.Request) (*http.Response, error) {
	if os.Getenv("SHIELD_API_TOKEN") != "" {
		req.Header.Set("X-Shield-Token", os.Getenv("SHIELD_API_TOKEN"))
	}

	if curBackend.Token != "" {
		if curBackend.APIVersion == 1 {
			req.Header.Set("Authorization", curBackend.Token)
		} else {
			req.Header.Set("X-Shield-Session", curBackend.Token)
		}
	}

	debugRequest(req)
	r, err := curClient.Do(req)
	debugResponse(r)
	if err != nil {
		return nil, err
	}

	return r, err
}
