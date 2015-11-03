package api

import (
	"net/http"
)

func Run(bind string) {
	http.Handle("/v1/ping", &PingAPI{})
	http.ListenAndServe(bind, nil)
}
