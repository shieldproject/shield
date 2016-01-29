package main

import (
	"crypto/tls"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"os"

	"github.com/starkandwayne/shield/plugin"
)

func main() {
	p := RabbitMQBrokerPlugin{
		Name:    "Pivotal RabbitMQ Broker Backup Plugin",
		Author:  "Stark & Wayne",
		Version: "0.0.1",
		Features: plugin.PluginFeatures{
			Target: "yes",
			Store:  "no",
		},
	}

	plugin.Run(p)
}

type RabbitMQBrokerPlugin plugin.PluginInfo

type RabbitMQEndpoint struct {
	Username          string
	Password          string
	URL               string
	SkipSSLValidation bool
}

func (p RabbitMQBrokerPlugin) Meta() plugin.PluginInfo {
	return plugin.PluginInfo(p)
}

func (p RabbitMQBrokerPlugin) Backup(endpoint plugin.ShieldEndpoint) error {
	rmq, err := getRabbitMQEndpoint(endpoint)
	if err != nil {
		return err
	}

	resp, err := makeRequest("GET", fmt.Sprintf("%s/api/definitions", rmq.URL), nil, rmq.Username, rmq.Password, rmq.SkipSSLValidation)
	if err != nil {
		return err
	}

	defer resp.Body.Close()
	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return err
	}

	fmt.Fprintf(os.Stdout, "%s\n", body)

	return nil
}

func (p RabbitMQBrokerPlugin) Restore(endpoint plugin.ShieldEndpoint) error {
	rmq, err := getRabbitMQEndpoint(endpoint)
	if err != nil {
		return err
	}
	_, err = makeRequest("POST", fmt.Sprintf("%s/api/definitions", rmq.URL), os.Stdin, rmq.Username, rmq.Password, rmq.SkipSSLValidation)
	if err != nil {
		return err
	}

	return nil
}

func (p RabbitMQBrokerPlugin) Store(endpoint plugin.ShieldEndpoint) (string, error) {
	return "", plugin.UNIMPLEMENTED
}

func (p RabbitMQBrokerPlugin) Retrieve(endpoint plugin.ShieldEndpoint, file string) error {
	return plugin.UNIMPLEMENTED
}

func (p RabbitMQBrokerPlugin) Purge(endpoint plugin.ShieldEndpoint, file string) error {
	return plugin.UNIMPLEMENTED
}

func getRabbitMQEndpoint(endpoint plugin.ShieldEndpoint) (RabbitMQEndpoint, error) {
	url, err := endpoint.StringValue("rmq_url")
	if err != nil {
		return RabbitMQEndpoint{}, err
	}

	user, err := endpoint.StringValue("rmq_username")
	if err != nil {
		return RabbitMQEndpoint{}, err
	}

	passwd, err := endpoint.StringValue("rmq_password")
	if err != nil {
		return RabbitMQEndpoint{}, err
	}

	sslValidate, err := endpoint.BooleanValue("skip_ssl_validation")
	if err != nil {
		return RabbitMQEndpoint{}, err
	}

	return RabbitMQEndpoint{
		Username:          user,
		Password:          passwd,
		URL:               url,
		SkipSSLValidation: sslValidate,
	}, nil
}

func makeRequest(method string, url string, body io.Reader, username string, password string, skipSSLValidation bool) (*http.Response, error) {
	req, err := http.NewRequest(method, url, body)
	if err != nil {
		return nil, err
	}
	req.SetBasicAuth(username, password)
	req.Header.Add("Content-Type", "application/json")

	httpClient := http.Client{}
	httpClient.Transport = &http.Transport{TLSClientConfig: &tls.Config{InsecureSkipVerify: skipSSLValidation}}

	resp, err := httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	if resp.StatusCode >= 300 {
		plugin.DEBUG("%#v", resp)
		return nil, fmt.Errorf("Got '%d' response while retrieving RMQ definitions", resp.StatusCode)
	}

	return resp, nil
}
