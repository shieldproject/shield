package director_test

import (
	"net/http"

	"github.com/onsi/gomega/ghttp"
)

func ConfigureTaskResult(firstHandler http.HandlerFunc, result string, server *ghttp.Server) {
	redirectHeader := http.Header{}
	redirectHeader.Add("Location", "/tasks/123")

	server.AppendHandlers(
		ghttp.CombineHandlers(
			firstHandler,
			ghttp.RespondWith(http.StatusFound, nil, redirectHeader),
		),
		ghttp.CombineHandlers(
			ghttp.VerifyRequest("GET", "/tasks/123"),
			ghttp.VerifyBasicAuth("username", "password"),
			ghttp.RespondWith(http.StatusOK, `{"id":123, "state":"done"}`),
		),
		ghttp.CombineHandlers(
			ghttp.VerifyRequest("GET", "/tasks/123"),
			ghttp.VerifyBasicAuth("username", "password"),
			ghttp.RespondWith(http.StatusOK, `{"id":123, "state":"done"}`),
		),
		ghttp.CombineHandlers(
			ghttp.VerifyRequest("GET", "/tasks/123/output", "type=event"),
			ghttp.VerifyBasicAuth("username", "password"),
			ghttp.RespondWith(http.StatusOK, ``),
		),
		ghttp.CombineHandlers(
			ghttp.VerifyRequest("GET", "/tasks/123/output", "type=result"),
			ghttp.VerifyBasicAuth("username", "password"),
			ghttp.RespondWith(http.StatusOK, result),
		),
	)
}

func AppendBadRequest(firstHandler http.HandlerFunc, server *ghttp.Server) {
	server.AppendHandlers(
		ghttp.CombineHandlers(
			firstHandler,
			ghttp.RespondWith(http.StatusBadRequest, ""),
		),
	)
}
