package http

import (
	"io"
	"io/ioutil"
)

func ReadAndClose(body io.ReadCloser) ([]byte, error) {
	defer body.Close()
	return ioutil.ReadAll(body)
}
