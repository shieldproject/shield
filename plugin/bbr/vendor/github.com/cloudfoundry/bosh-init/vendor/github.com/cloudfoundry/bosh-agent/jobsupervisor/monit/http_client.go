package monit

import (
	"encoding/xml"
	"io/ioutil"
	"net/http"
	"net/url"
	"path"
	"strings"

	"golang.org/x/net/html/charset"

	bosherr "github.com/cloudfoundry/bosh-utils/errors"
	boshhttp "github.com/cloudfoundry/bosh-utils/http"
	boshlog "github.com/cloudfoundry/bosh-utils/logger"
)

type httpClient struct {
	startClient     boshhttp.Client
	stopClient      boshhttp.Client
	unmonitorClient boshhttp.Client
	statusClient    boshhttp.Client
	host            string
	username        string
	password        string
	logger          boshlog.Logger
}

// NewHTTPClient creates a new monit client
//
// status & start use the shortClient
// unmonitor & stop use the longClient
func NewHTTPClient(
	host, username, password string,
	shortClient boshhttp.Client,
	longClient boshhttp.Client,
	logger boshlog.Logger,
) Client {
	return httpClient{
		host:            host,
		username:        username,
		password:        password,
		startClient:     shortClient,
		stopClient:      longClient,
		unmonitorClient: longClient,
		statusClient:    shortClient,
		logger:          logger,
	}
}

func (c httpClient) ServicesInGroup(name string) (services []string, err error) {
	status, err := c.status()
	if err != nil {
		return nil, bosherr.WrapError(err, "Getting status from Monit")
	}

	serviceGroup, found := status.ServiceGroups.Get(name)
	if !found {
		return []string{}, nil
	}

	return serviceGroup.Services, nil
}

func (c httpClient) StartService(serviceName string) error {
	response, err := c.makeRequest(c.startClient, c.monitURL(serviceName), "POST", "action=start")
	if err != nil {
		return bosherr.WrapError(err, "Sending start request to monit")
	}

	defer func() {
		if err := response.Body.Close(); err != nil {
			c.logger.Warn("http-client", "Failed to close monit start POST response body: %s", err.Error())
		}
	}()

	err = c.validateResponse(response)
	if err != nil {
		return bosherr.WrapErrorf(err, "Starting Monit service %s", serviceName)
	}

	return nil
}

func (c httpClient) StopService(serviceName string) error {
	response, err := c.makeRequest(c.stopClient, c.monitURL(serviceName), "POST", "action=stop")
	if err != nil {
		return bosherr.WrapError(err, "Sending stop request to monit")
	}

	defer func() {
		if err := response.Body.Close(); err != nil {
			c.logger.Warn("http-client", "Failed to close monit stop POST response body: %s", err.Error())
		}
	}()

	err = c.validateResponse(response)
	if err != nil {
		return bosherr.WrapErrorf(err, "Stopping Monit service %s", serviceName)
	}

	return nil
}

func (c httpClient) UnmonitorService(serviceName string) error {
	response, err := c.makeRequest(c.unmonitorClient, c.monitURL(serviceName), "POST", "action=unmonitor")
	if err != nil {
		return bosherr.WrapError(err, "Sending unmonitor request to monit")
	}

	defer func() {
		if err := response.Body.Close(); err != nil {
			c.logger.Warn("http-client", "Failed to close monit unmonitor POST response body: %s", err.Error())
		}
	}()

	err = c.validateResponse(response)
	if err != nil {
		return bosherr.WrapErrorf(err, "Unmonitoring Monit service %s", serviceName)
	}

	return nil
}

func (c httpClient) Status() (Status, error) {
	return c.status()
}

func (c httpClient) status() (status, error) {
	c.logger.Debug("http-client", "status function called")
	url := c.monitURL("/_status2")
	url.RawQuery = "format=xml"

	response, err := c.makeRequest(c.statusClient, url, "GET", "")
	if err != nil {
		return status{}, bosherr.WrapError(err, "Sending status request to monit")
	}

	defer func() {
		if err := response.Body.Close(); err != nil {
			c.logger.Warn("http-client", "Failed to close monit status GET response body: %s", err.Error())
		}
	}()

	err = c.validateResponse(response)
	if err != nil {
		return status{}, bosherr.WrapError(err, "Getting monit status")
	}

	decoder := xml.NewDecoder(response.Body)
	decoder.CharsetReader = charset.NewReaderLabel

	var st status

	err = decoder.Decode(&st)
	if err != nil {
		return status{}, bosherr.WrapError(err, "Unmarshalling Monit status")
	}

	return st, nil
}

func (c httpClient) monitURL(thing string) url.URL {
	return url.URL{
		Scheme: "http",
		Host:   c.host,
		Path:   path.Join("/", thing),
	}
}

func (c httpClient) validateResponse(response *http.Response) error {
	if response.StatusCode == http.StatusOK {
		return nil
	}

	body, err := ioutil.ReadAll(response.Body)
	if err != nil {
		return bosherr.WrapError(err, "Reading body of failed Monit response")
	}

	c.logger.Debug("http-client", "Request failed with %s: %s", response.Status, string(body))

	return bosherr.Errorf("Request failed with %s: %s", response.Status, string(body))
}

func (c httpClient) makeRequest(client boshhttp.Client, target url.URL, method, requestBody string) (*http.Response, error) {
	c.logger.Debug("http-client", "Monit request: url='%s' body='%s'", target.String(), requestBody)

	request, err := http.NewRequest(method, target.String(), strings.NewReader(requestBody))
	if err != nil {
		return nil, err
	}

	request.SetBasicAuth(c.username, c.password)

	request.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	return client.Do(request)
}
