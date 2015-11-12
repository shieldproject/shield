package api_agent

import (
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
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
		fmt.Println("ERROR: Failed to successfully communicate with host", err)
		return err
	}
	defer resp.Body.Close()

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		fmt.Println("ERROR: Failed to read body", err)
		return err
	}

	if err := json.Unmarshal(body, data); err != nil {
		fmt.Println("ERROR: Error unmarshalling json from response:\n\n", err)
		return err
	}

	return nil

}
