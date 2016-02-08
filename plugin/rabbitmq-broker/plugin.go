// The `rabbitmq-broker` plugin for SHIELD implements backup + restore functionality
// for Pivotal's cf-rabbitmq-release RabbitMQ Service Broker BOSH release. It is specific
// to Pivotal's implementation, which can be found at https://github.com/pivotal-cf/cf-rabbitmq-release
//
// However, since this plugin utilizes the RabbitMQ Management API directly, it may be
// compatible with other RabbitMQ deployments.
//
// PLUGIN FEATURES
//
// This plugin implements functionality suitable for use with the following SHIELD
// Job components:
//
//    Target: yes
//    Store:  no
//
// PLUGIN CONFIGURATION
//
// The endpoint configuration passed to this plugin is used to identify what rabbitmq
// cluster to back up, and how to connect to it. Your endpoint JSON should look
// something like this:
//
//    {
//        "rmq_url":"url-to-rabbitmq-management-domain",
//        "rmq_username":"basic-auth-user-for-above-domain",
//        "rmq_password":"basic-auth-passwd-for-above-domain",
//        "skip_ssl_validation":false
//     }
//
// BACKUP DETAILS
//
// The `rabbitmq-broker` plugin connects to the RabbitMQ management API to perform
// a dump of the current configuration of the RabbitMQ cluster. This backs up vhosts,
// queues, users, passwords, roles, but may not necessarily back up the data inside
// the queue. It uses the `GET /api/definitions` URL from the RabbitMQ cluster. Details
// on what that does can be found here:
// https://cdn.rawgit.com/rabbitmq/rabbitmq-management/rabbitmq_v3_6_0/priv/www/api/index.html
//
// RESTORE DETAILS
//
// The `rabbitmq-broker` plugin connects to the RabbitMQ management API and restores
// the configuration dump into RabbitMQ, using the `PUT /api/definitions` URL. Any existing
// data is not replaced, but merged with the new configurations, making this a safe plugin
// to restore without service interruption.
//
// DEPENDENCIES
//
// This plugin relies on the `rabbitmq_management`. Please make sure it is deployed
// to your RabbbitMQ cluster.
//
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
