package s3

import (
	"encoding/xml"
	"fmt"
	"io/ioutil"
	"net/http"
)

func ResponseError(res *http.Response) error {
	b, err := ioutil.ReadAll(res.Body)
	if err != nil {
		return err
	}
	return ResponseErrorFrom(b)
}

func ResponseErrorFrom(b []byte) error {
	var payload struct {
		XMLName xml.Name `xml:"Error"`
		Code    string   `xml:"Code"`
		Message string   `xml:"Message"`
	}
	if err := xml.Unmarshal(b, &payload); err != nil {
		return fmt.Errorf("unable to parse response xml: %s", err)
	}

	return fmt.Errorf("%s (%s) [raw %s]", payload.Message, payload.Code, string(b))
}
