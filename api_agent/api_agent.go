package api_agent

import (
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"

	"github.com/starkandwayne/goutils/log"
)

func getServerDetails() (string, string, string) {
	return "http", "localhost", "8080"
}

func makeApiCall(data interface{}, action, uri string, postbody io.Reader) error {

	// Get this from config file, overridden via options
	schema, host, port := getServerDetails()

	url := fmt.Sprintf("%s://%s:%s/%s", schema, host, port, uri)

	req, err := http.NewRequest(action, url, postbody)
	httpClient := &http.Client{}

	resp, err := httpClient.Do(req)
	if err != nil {
		log.Errorf("failed to successfully communicate with host: %s", err)
		return err
	}
	defer resp.Body.Close()

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		log.Errorf("failed to read body: %s", err)
		return err
	}

	if err := json.Unmarshal(body, data); err != nil {
		log.Errorf("failed to unmarshal JSON from response: %s", err)
		return err
	}

	return nil

}
