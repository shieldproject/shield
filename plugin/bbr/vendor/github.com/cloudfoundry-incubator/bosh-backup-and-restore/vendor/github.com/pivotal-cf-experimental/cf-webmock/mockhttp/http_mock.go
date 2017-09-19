package mockhttp

import (
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"net/http"
	"strings"

	. "github.com/onsi/gomega"
)

type MockHttp struct {
	expectedMethod              string
	expectedUrl                 string
	expectedContentType         string
	expectedBody                string
	expectedAuthorizationHeader string
	expectedFormValues          map[string]string

	responseRedirectToUrl string
	responseStatus        int
	responseBody          string

	responseCallback func(rawbody []byte)

	authenticated bool
}

func (i *MockHttp) Fails(message string) *MockHttp {
	i.responseBody = message
	i.responseStatus = http.StatusInternalServerError
	return i
}

func (i *MockHttp) RedirectsTo(uri string) *MockHttp {
	i.responseStatus = http.StatusFound
	i.responseRedirectToUrl = uri
	return i
}
func (i *MockHttp) NotFound() *MockHttp {
	i.responseBody = ""
	i.responseStatus = http.StatusNotFound
	return i
}
func (i *MockHttp) RespondsWithUnauthorized(body string) *MockHttp {
	i.responseBody = body
	i.responseStatus = http.StatusUnauthorized
	return i
}

func (i *MockHttp) RespondsWithJson(obj interface{}) *MockHttp {
	data, err := json.Marshal(obj)
	Expect(err).NotTo(HaveOccurred())
	i.RespondsWith(string(data))
	return i
}

func (i *MockHttp) RespondsWith(body string) *MockHttp {
	i.responseStatus = http.StatusOK
	i.responseBody = body
	return i
}

func (i *MockHttp) SkipAuthentication() *MockHttp {
	i.authenticated = false
	return i
}
func (i *MockHttp) RespondsEmpty() *MockHttp {
	i.responseBody = ""
	i.responseStatus = http.StatusNoContent
	return i
}

func (i *MockHttp) WithContentType(contentType string) *MockHttp {
	i.expectedContentType = contentType
	return i
}

func (i *MockHttp) WithBody(body string) *MockHttp {
	i.expectedBody = body
	return i
}

func (i *MockHttp) WithAuthorizationHeader(auth string) *MockHttp {
	i.expectedAuthorizationHeader = auth
	return i
}

func NewMockedHttpRequest(method, url string) *MockHttp {
	return &MockHttp{expectedMethod: method, expectedUrl: url, authenticated: true}
}

func (i *MockHttp) Verify(req *http.Request, d *Server) {
	Expect(req.Method+" "+req.URL.String()).To(Equal(i.expectedMethod+" "+i.expectedUrl), fmt.Sprintf("Received:\n\t%s\nCompleted mocks:\n%s\nPending mocks:\n%s\n", req.Method+" "+req.URL.String(), strings.Join(d.completedMocks(), "\n"), strings.Join(d.pendingMocks(), "\n")))
	if i.authenticated {
		if d.expectedAuthorizationHeader != "" {
			Expect(req.Header.Get("Authorization")).To(Equal(d.expectedAuthorizationHeader))
		}
		if i.expectedAuthorizationHeader != "" {
			Expect(req.Header.Get("Authorization")).To(Equal(i.expectedAuthorizationHeader))
		}
	}

	if i.expectedContentType != "" {
		Expect(req.Header.Get("Content-Type")).To(Equal(i.expectedContentType))
	}

	rawBody, err := ioutil.ReadAll(req.Body)
	Expect(err).NotTo(HaveOccurred())

	if i.expectedBody != "" {
		Expect(string(rawBody)).To(Equal(i.expectedBody))
	}
	if i.responseCallback != nil {
		i.responseCallback(rawBody)
	}
	if i.expectedFormValues != nil {
		for key, value := range i.expectedFormValues {
			Expect(req.FormValue(key)).To(Equal(value))
		}
	}

	Expect(req.Method+" "+req.URL.String()).To(Equal(i.expectedMethod+" "+i.expectedUrl), fmt.Sprintf("Received:\n\t%s\nCompleted mocks:\n%s\nPending mocks:\n%s\n", req.Method+" "+req.URL.String(), strings.Join(d.completedMocks(), "\n"), strings.Join(d.pendingMocks(), "\n")))
}

func (i *MockHttp) Respond(writer http.ResponseWriter, logger *log.Logger) {
	if len(i.responseRedirectToUrl) != 0 {
		logger.Printf("Redirecting to %s\n", i.responseRedirectToUrl)
		writer.Header().Set("Location", i.responseRedirectToUrl)
	}
	logger.Printf("Reponding with code(%d)\n%s\n", i.responseStatus, i.responseBody)
	writer.WriteHeader(i.responseStatus)
	io.WriteString(writer, i.responseBody)
}

func (i *MockHttp) For(comment string) *MockHttp {
	return i
}

func (i *MockHttp) Url() string {
	return i.expectedMethod + " " + i.expectedUrl
}

func (i *MockHttp) SetResponseCallback(b func(rawbody []byte)) {
	i.responseCallback = b
}
