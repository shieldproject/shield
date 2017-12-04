package agent

import (
	"bytes"
	"crypto/tls"
	"crypto/x509"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/jhunt/go-log"
)

func (agent *Agent) Ping() {
	if agent.Registration.URL == "" {
		log.Infof("no registration.url provided; skipping agent auto-registration")
		return
	}
	if agent.Registration.Interval <= 0 {
		log.Errorf("invalid registration.interval %d supplied; skipping agent auto-registration", agent.Registration.Interval)
		return
	}

	pool := x509.NewCertPool()
	if agent.Registration.ShieldCACert != "" {
		if ok := pool.AppendCertsFromPEM([]byte(agent.Registration.ShieldCACert)); !ok {
			log.Errorf("Invalid or malformed CA Certificate")
			return
		}
	}

	client := &http.Client{
		Transport: &http.Transport{
			TLSClientConfig: &tls.Config{
				InsecureSkipVerify: agent.Registration.SkipVerify,
				RootCAs:            pool,
			},
			Proxy:             http.ProxyFromEnvironment,
			DisableKeepAlives: true,
		},
		Timeout: 30 * time.Second,
	}

	ping := func() {
		log.Debugf("pinging shield core")
		var params = struct {
			Name string `json:"name"`
			Port int    `json:"port"`
		}{
			Name: agent.Name,
			Port: agent.Port,
		}
		b, err := json.Marshal(params)
		if err != nil {
			log.Errorf("failed to marshal /v2/agents request parameters to JSON: %s", err)
			return
		}

		log.Debugf("attempting to pre-register with %s/v2/agents as %s", agent.Registration.URL, string(b))
		req, err := http.NewRequest("POST", agent.Registration.URL+"/v2/agents", bytes.NewBuffer(b))
		if err != nil {
			log.Errorf("failed to issue POST %s/v2/agents: %s", agent.Registration.URL, err)
			return
		}

		res, err := client.Do(req)
		if err != nil {
			fmt.Errorf("failed to issue POST /v2/agents: %s", err)
			return
		}

		log.Debugf("POST /v2/agents returned [%s]", res.Status)
	}

	ch := time.Tick(time.Duration(agent.Registration.Interval) * time.Second)
	ping()
	for _ = range ch {
		ping()
	}
}
