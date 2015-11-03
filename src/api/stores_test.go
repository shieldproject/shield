package api_test

import (
	. "api"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	// sql drivers
	_ "github.com/mattn/go-sqlite3"

	"db"

	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
)

var _ = Describe("/v1/stores API", func() {
	var orm *db.ORM

	BeforeEach(func() {
		var err error
		orm, err = setupORM(
			`INSERT INTO stores (uuid, name, summary, plugin, endpoint) VALUES
				("66be7c43-6c57-4391-8ea9-e770d6ab5e9e",
				 "redis-shared",
				 "Shared Redis services for CF",
				 "redis",
				 "<<redis-configuration>>")`,

			`INSERT INTO stores (uuid, name, summary, plugin, endpoint) VALUES
				("05c3d005-f968-452f-bd59-bee8e79ab982",
				 "s3",
				 "Amazon S3 Blobstore",
				 "s3",
				 "<<s3-configuration>>")`,
		)
		Ω(err).ShouldNot(HaveOccurred())
	})

	It("should retrieve all stores", func() {
		handler := StoreAPI{Data: orm}
		req, _ := http.NewRequest("GET", "/v1/stores", nil)
		w := httptest.NewRecorder()

		handler.ServeHTTP(w, req)
		Ω(w.Body.String()).Should(MatchJSON(`[
				{
					"uuid"     : "66be7c43-6c57-4391-8ea9-e770d6ab5e9e",
					"name"     : "redis-shared",
					"summary"  : "Shared Redis services for CF",
					"plugin"   : "redis",
					"endpoint" : "<<redis-configuration>>"
				},
				{
					"uuid"     : "05c3d005-f968-452f-bd59-bee8e79ab982",
					"name"     : "s3",
					"summary"  : "Amazon S3 Blobstore",
					"plugin"   : "s3",
					"endpoint" : "<<s3-configuration>>"
				}
			]`))
		Ω(w.Code).Should(Equal(200))
	})

	It("can create new stores", func() {
		handler := StoreAPI{Data: orm}
		req, _ := http.NewRequest("POST", "/v1/stores",
			strings.NewReader(`{
					"name"     : "New Store",
					"summary"  : "A new one",
					"plugin"   : "s3",
					"endpoint" : "[ENDPOINT]"
				}`))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()

		handler.ServeHTTP(w, req)
		Ω(w.Code).Should(Equal(200))
		Ω(w.Body.String()).Should(MatchRegexp(`{"ok":"created","uuid":"[a-z0-9-]+"}`))
	})

	It("requires the `name' and `when' keys in POST'ed data", func() {
		handler := StoreAPI{Data: orm}
		req, _ := http.NewRequest("POST", "/v1/stores", strings.NewReader(`{}`))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()

		handler.ServeHTTP(w, req)
		Ω(w.Code).Should(Equal(400))
	})

	It("can update existing store", func() {
		handler := StoreAPI{Data: orm}
		req, _ := http.NewRequest("PUT", "/v1/store/66be7c43-6c57-4391-8ea9-e770d6ab5e9e",
			strings.NewReader(`{
					"name"     : "Renamed",
					"summary"  : "UPDATED!",
					"plugin"   : "redis",
					"endpoint" : "{NEW-ENDPOINT}"
				}`))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()

		handler.ServeHTTP(w, req)
		Ω(w.Code).Should(Equal(200))
		Ω(w.Body.String()).Should(MatchJSON(`{"ok":"updated","uuid":"66be7c43-6c57-4391-8ea9-e770d6ab5e9e"}`))

		req, _ = http.NewRequest("GET", "/v1/stores", nil)
		w = httptest.NewRecorder()

		handler.ServeHTTP(w, req)
		Ω(w.Body.String()).Should(MatchJSON(`[
				{
					"uuid"     : "66be7c43-6c57-4391-8ea9-e770d6ab5e9e",
					"name"     : "Renamed",
					"summary"  : "UPDATED!",
					"plugin"   : "redis",
					"endpoint" : "{NEW-ENDPOINT}"
				},
				{
					"uuid"     : "05c3d005-f968-452f-bd59-bee8e79ab982",
					"name"     : "s3",
					"summary"  : "Amazon S3 Blobstore",
					"plugin"   : "s3",
					"endpoint" : "<<s3-configuration>>"
				}
			]`))
		Ω(w.Code).Should(Equal(200))
	})

	It("ignores other HTTP methods", func() {
		handler := StoreAPI{Data: orm}
		for _, method := range []string{"PUT", "DELETE", "PATCH", "OPTIONS", "TRACE"} {
			notImplemented(handler, method, "/v1/stores", nil)
		}

		for _, method := range []string{"GET", "HEAD", "POST", "PATCH", "OPTIONS", "TRACE"} {
			notImplemented(handler, method, "/v1/stores/sub/requests", nil)
			notImplemented(handler, method, "/v1/store/sub/requests", nil)
			notImplemented(handler, method, "/v1/store/5981f34c-ef58-4e3b-a91e-428480c68100", nil)
		}
	})

	It("ignores malformed UUIDs", func() {
		handler := StoreAPI{Data: orm}
		for _, id := range []string{"malformed-uuid-01234", "", "(abcdef-01234-56-789)"} {
			notImplemented(handler, "GET", fmt.Sprintf("/v1/store/%s", id), nil)
			notImplemented(handler, "PUT", fmt.Sprintf("/v1/store/%s", id), nil)
		}
	})

	/* FIXME: handle ?unused=[tf] query string... */

	/* FIXME: write tests for DELETE /v1/store/:uuid */
	/*        (incl. test for delete of an in-use store) */
})
