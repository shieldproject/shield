package rabbitmq

import (
	"crypto/tls"
	"io"
	"io/ioutil"
	"net/http"

	fmt "github.com/jhunt/go-ansi"

	"github.com/shieldproject/shield/plugin"
)

func New() plugin.Plugin {
	return RabbitMQBrokerPlugin{
		Name:    "Pivotal RabbitMQ Broker Backup Plugin",
		Author:  "SHIELD Core Team",
		Version: "0.0.1",
		Fields: []plugin.Field{
			plugin.Field{
				Mode:     "target",
				Name:     "rmq_url",
				Type:     "string",
				Title:    "RabbitMQ URL",
				Help:     "The URL of your RabbitMQ management UI, usually run on port 15672.",
				Example:  "http://1.2.3.4:15672",
				Required: true,
			},
			plugin.Field{
				Mode:     "target",
				Name:     "rmq_username",
				Type:     "string",
				Title:    "RabbitMQ Username",
				Help:     "Username to use when authenticating to RabbitMQ.",
				Required: true,
			},
			plugin.Field{
				Mode:     "target",
				Name:     "rmq_password",
				Type:     "password",
				Title:    "RabbitMQ Password",
				Help:     "Password to use when authenticating to RabbitMQ.",
				Required: true,
			},
			plugin.Field{
				Mode:  "target",
				Name:  "skip_ssl_validation",
				Type:  "bool",
				Title: "Skip SSL Validation",
				Help:  "If your RabbitMQ installation has an invalid or expired SSL/TLS certificate, you can ignore those errors by disabling SSL validation.  This is not recommended from a security perspective, however.",
			},
		},
	}

}

func Run() {
	plugin.Run(New())
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

func (p RabbitMQBrokerPlugin) Validate(log io.Writer, endpoint plugin.ShieldEndpoint) error {
	var (
		s    string
		err  error
		fail bool
	)

	s, err = endpoint.StringValue("rmq_url")
	if err != nil {
		fmt.Fprintf(log, "@R{\u2717 rmq_url              %s}\n", err)
		fail = true
	} else {
		fmt.Fprintf(log, "@G{\u2713 rmq_url}              @C{%s}\n", s)
	}

	s, err = endpoint.StringValue("rmq_username")
	if err != nil {
		fmt.Fprintf(log, "@R{\u2717 rmq_username         %s}\n", err)
		fail = true
	} else {
		fmt.Fprintf(log, "@G{\u2713 rmq_username}         @C{%s}\n", plugin.Redact(s))
	}

	s, err = endpoint.StringValue("rmq_password")
	if err != nil {
		fmt.Fprintf(log, "@R{\u2717 rmq_password         %s}\n", err)
		fail = true
	} else {
		fmt.Fprintf(log, "@G{\u2713 rmq_password}         @C{%s}\n", plugin.Redact(s))
	}

	tf, err := endpoint.BooleanValueDefault("skip_ssl_validation", false)
	if err != nil {
		fmt.Fprintf(log, "@R{\u2717 skip_ssl_validation  %s}\n", err)
		fail = true
	} else {
		if tf {
			fmt.Fprintf(log, "@G{\u2713 skip_ssl_validation}  @C{yes}, SSL will @Y{NOT} be validated\n")
		} else {
			fmt.Fprintf(log, "@G{\u2713 skip_ssl_validation}  @C{no}, SSL @Y{WILL} be validated\n")
		}
	}

	if fail {
		return fmt.Errorf("rabbitmq-broker: invalid configuration")
	}
	return nil
}

func (p RabbitMQBrokerPlugin) Backup(out io.Writer, log io.Writer, endpoint plugin.ShieldEndpoint) error {
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

	fmt.Fprintf(out, "%s\n", body)

	return nil
}

func (p RabbitMQBrokerPlugin) Restore(in io.Reader, log io.Writer, endpoint plugin.ShieldEndpoint) error {
	rmq, err := getRabbitMQEndpoint(endpoint)
	if err != nil {
		return err
	}
	_, err = makeRequest("POST", fmt.Sprintf("%s/api/definitions", rmq.URL), in, rmq.Username, rmq.Password, rmq.SkipSSLValidation)
	if err != nil {
		return err
	}

	return nil
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
