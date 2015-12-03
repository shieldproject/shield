package api_agent

import (
	"encoding/json"
	"fmt"
	"github.com/spf13/viper"
	"io"
	"io/ioutil"
	"net/http"

	"github.com/starkandwayne/goutils/log"
)

func ShieldURI(p string, args ...interface{}) *URL {
	path := fmt.Sprintf(p, args...)
	scheme := "http"
	if viper.GetBool("ShieldSSL") {
		scheme = "https"
	}

	u, err := ParseURL(fmt.Sprintf("%s://%s:%s%s",
		scheme,
		viper.GetString("ShieldServer"),
		viper.GetString("ShieldPort"),
		path))
	if err != nil {
		panic(err)
	}
	return u
}

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
		log.Errorf("failed to successfully communicate with host: %s", err)
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		return ResponseError{Status: resp.StatusCode, FullResponse: resp}
	}

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
