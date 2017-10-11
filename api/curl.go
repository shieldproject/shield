package api

import (
	"bytes"
	"io"
	"net/http"
)

func Curl(method, url, body string) (interface{}, error) {
	var data interface{}

	u, err := ShieldURI("%s", url)
	if err != nil {
		return data, err
	}

	var b io.Reader
	if body != "" {
		b = bytes.NewBufferString(body)
	}

	r, err := http.NewRequest(method, u.String(), b)
	if err != nil {
		return data, err
	}

	r.Header.Set("Content-Type", "application/json")
	r.Header.Set("Accept", "application/json")

	return data, u.Request(&data, r)
}
