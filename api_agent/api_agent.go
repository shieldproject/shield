package api_agent

import (
	"encoding/json"
	"fmt"
	"github.com/spf13/viper"
	"io"
	"io/ioutil"
	"net/http"

	log "gopkg.in/inconshreveable/log15.v2"
)

func getServerDetails() (string, string, string) {
	if viper.GetBool("ShieldSSL") {
		return "https", viper.GetString("ShieldServer"), viper.GetString("ShieldPort")
	} else {
		return "http", viper.GetString("ShieldServer"), viper.GetString("ShieldPort")
	}
}

type ResponseError struct {
	Status       int
	FullResponse *http.Response
	//Notes  string
}

func (e ResponseError) Error() string {
	return fmt.Sprintf("Status: %d\n    %+v", e.Status, e.FullResponse)
}

func makeApiCall(data interface{}, action, uri string, postbody io.Reader) error {

	// Get this from config file, overridden via options
	schema, host, port := getServerDetails()

	url := fmt.Sprintf("%s://%s:%s/%s", schema, host, port, uri)

	if viper.GetBool("Verbose") {
		fmt.Println("API Call:", url)
	}

	req, err := http.NewRequest(action, url, postbody)
	httpClient := &http.Client{}

	resp, err := httpClient.Do(req)

	// FIXME - Make this a --debug flag
	//fmt.Println("Req : ", req, "\nResp: ", resp)

	if err != nil {
		fmt.Println("ERROR: Failed to successfully communicate with host", err)
		log.Error("Failed to successfully communicate with host: ", err) 
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		return ResponseError{Status: resp.StatusCode, FullResponse: resp}
	}

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
