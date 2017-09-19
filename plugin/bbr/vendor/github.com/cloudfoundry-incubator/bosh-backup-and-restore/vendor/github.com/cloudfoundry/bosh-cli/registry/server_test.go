package registry_test

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"strings"
	"time"

	. "github.com/cloudfoundry/bosh-cli/registry"
	boshlog "github.com/cloudfoundry/bosh-utils/logger"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("Server", func() {
	var (
		server                   Server
		registryURL              string
		incorrectAuthRegistryURL string
		client                   helperClient
	)

	retryStartingServer := func() (Server, error) {
		var err error
		var server Server
		logger := boshlog.NewLogger(boshlog.LevelNone)
		serverFactory := NewServerManager(logger)

		attempts := 0
		for attempts < 3 {
			server, err = serverFactory.Start("fake-user", "fake-password", "localhost", 6901)
			if err == nil {
				return server, nil
			}

			attempts++
			time.Sleep(1 * time.Second)
		}

		return nil, err
	}

	BeforeEach(func() {
		registryHost := "localhost:6901"
		registryURL = fmt.Sprintf("http://fake-user:fake-password@%s", registryHost)
		incorrectAuthRegistryURL = fmt.Sprintf("http://incorrect-user:incorrect-password@%s", registryHost)

		var err error
		server, err = retryStartingServer() // wait for previous test to close socket if still open
		Expect(err).ToNot(HaveOccurred())

		transport := &http.Transport{DisableKeepAlives: true}
		httpClient := http.Client{Transport: transport}

		client = newHelperClient(httpClient)
	})

	AfterEach(func() {
		server.Stop()
	})

	Describe("making a request with an unknown path", func() {
		It("returns 404", func() {
			_, _, statusCode := client.DoPut(registryURL+"/instances/1/something-else", "fake-agent-settings")
			Expect(statusCode).To(Equal(404))
		})
	})

	Describe("PUT instances/:instance_id/settings", func() {
		Context("when username and password are incorrect", func() {
			It("returns 401", func() {
				_, responseHeader, statusCode := client.DoPut(incorrectAuthRegistryURL+"/instances/1/settings", "fake-agent-settings")
				Expect(statusCode).To(Equal(401))
				Expect(responseHeader.Get("WWW-Authenticate")).To(Equal(`Basic realm="Bosh Registry"`))
			})
		})

		Context("when the settings do not yet exist", func() {
			It("creates the settings", func() {
				_, _, statusCode := client.DoPut(registryURL+"/instances/1/settings", "fake-agent-settings")
				Expect(statusCode).To(Equal(201))

				httpBody, statusCode := client.DoGet(registryURL + "/instances/1/settings")
				Expect(statusCode).To(Equal(200))
				var response SettingsResponse
				err := json.Unmarshal(httpBody, &response)
				Expect(err).ToNot(HaveOccurred())
				Expect(response.Settings).To(Equal("fake-agent-settings"))
				Expect(response.Status).To(Equal("ok"))
			})
		})

		Context("when the settings already exist", func() {
			It("updates the settings", func() {
				_, _, statusCode := client.DoPut(registryURL+"/instances/1/settings", "fake-agent-settings")
				Expect(statusCode).To(Equal(201))

				_, _, statusCode = client.DoPut(registryURL+"/instances/1/settings", "fake-agent-settings-updated")
				Expect(statusCode).To(Equal(200))

				httpBody, statusCode := client.DoGet(registryURL + "/instances/1/settings")
				Expect(statusCode).To(Equal(200))

				var response SettingsResponse
				err := json.Unmarshal(httpBody, &response)
				Expect(err).ToNot(HaveOccurred())
				Expect(response.Settings).To(Equal("fake-agent-settings-updated"))
				Expect(response.Status).To(Equal("ok"))
			})
		})
	})

	Describe("DELETE instances/:instance_id/settings", func() {
		Context("when username and password are incorrect", func() {
			It("returns 401", func() {
				responseHeader, statusCode := client.DoDelete(incorrectAuthRegistryURL + "/instances/1/settings")
				Expect(statusCode).To(Equal(401))
				Expect(responseHeader.Get("WWW-Authenticate")).To(Equal(`Basic realm="Bosh Registry"`))
			})
		})

		Context("when the settings exist", func() {
			It("deletes the settings", func() {
				_, _, statusCode := client.DoPut(registryURL+"/instances/1/settings", "fake-agent-settings")
				Expect(statusCode).To(Equal(201))

				_, statusCode = client.DoDelete(registryURL + "/instances/1/settings")
				Expect(statusCode).To(Equal(200))

				responseJSON, statusCode := client.DoGet(registryURL + "/instances/1/settings")
				Expect(statusCode).To(Equal(404))
				var settingsResponse SettingsResponse
				err := json.Unmarshal(responseJSON, &settingsResponse)
				Expect(err).ToNot(HaveOccurred())
				Expect(settingsResponse.Status).To(Equal("not_found"))
			})
		})

		Context("when the settings do not exist", func() {
			It("returns 200", func() {
				_, statusCode := client.DoDelete(registryURL + "/instances/1/settings")
				Expect(statusCode).To(Equal(200))

				_, statusCode = client.DoGet(registryURL + "/instances/1/settings")
				Expect(statusCode).To(Equal(404))
			})
		})
	})

	Describe("GET instances/:instance_id/settings", func() {
		Context("when settings do not exist", func() {
			It("returns 404", func() {
				_, statusCode := client.DoGet(registryURL + "/instances/1/settings")
				Expect(statusCode).To(Equal(404))
			})
		})

		Context("when settings exist", func() {
			BeforeEach(func() {
				_, _, statusCode := client.DoPut(registryURL+"/instances/1/settings", "fake-agent-settings")
				Expect(statusCode).To(Equal(201))
			})

			Context("when username and password are incorrect", func() {
				It("does not return 401, because GETs do not require authentication", func() {
					httpBody, statusCode := client.DoGet(incorrectAuthRegistryURL + "/instances/1/settings")
					Expect(statusCode).To(Equal(200))
					var response SettingsResponse
					err := json.Unmarshal(httpBody, &response)
					Expect(err).ToNot(HaveOccurred())
					Expect(response.Settings).To(Equal("fake-agent-settings"))
					Expect(response.Status).To(Equal("ok"))
				})
			})
		})
	})
})

type helperClient struct {
	httpClient http.Client
}

func newHelperClient(httpClient http.Client) helperClient {
	return helperClient{
		httpClient: httpClient,
	}
}

func (c helperClient) DoDelete(endpoint string) (http.Header, int) {
	request, err := http.NewRequest("DELETE", endpoint, strings.NewReader(""))
	Expect(err).ToNot(HaveOccurred())
	httpResponse, err := c.httpClient.Do(request)
	Expect(err).ToNot(HaveOccurred())

	return httpResponse.Header, httpResponse.StatusCode
}

func (c helperClient) DoPut(endpoint string, body string) (string, http.Header, int) {
	putPayload := strings.NewReader(body)

	request, err := http.NewRequest("PUT", endpoint, putPayload)
	Expect(err).ToNot(HaveOccurred())

	httpResponse, err := c.httpClient.Do(request)
	Expect(err).ToNot(HaveOccurred())

	defer httpResponse.Body.Close()

	httpBody, err := ioutil.ReadAll(httpResponse.Body)
	Expect(err).ToNot(HaveOccurred())

	return string(httpBody), httpResponse.Header, httpResponse.StatusCode
}

func (c helperClient) DoGet(endpoint string) ([]byte, int) {
	httpResponse, err := c.httpClient.Get(endpoint)
	Expect(err).ToNot(HaveOccurred())

	defer httpResponse.Body.Close()

	httpBody, err := ioutil.ReadAll(httpResponse.Body)
	Expect(err).ToNot(HaveOccurred())

	return httpBody, httpResponse.StatusCode
}
