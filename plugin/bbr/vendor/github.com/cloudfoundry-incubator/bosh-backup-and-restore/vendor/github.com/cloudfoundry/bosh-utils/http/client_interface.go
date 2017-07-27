package http

import (
	"net/http"
)

type Client interface {
	Do(req *http.Request) (resp *http.Response, err error)
}
