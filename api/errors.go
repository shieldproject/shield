package api

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"strings"
)

//ErrBadRequest is returned from the API if a request returns a 400 status code
type ErrBadRequest struct {
	message string
}

//NewErrBadRequest returns a new instance of ErrBadRequest, like fmt.Errorf
func NewErrBadRequest(format string, args ...interface{}) error {
	return ErrBadRequest{
		message: fmt.Sprintf(format, args...),
	}
}

func (e ErrBadRequest) Error() string {
	return e.message
}

//ErrUnauthorized is returned from the API if a request returns a 401 status code.
type ErrUnauthorized struct {
	message string
}

//NewErrUnauthorized returns a new instance of ErrUnauthorized, like fmt.Errorf
func NewErrUnauthorized(format string, args ...interface{}) error {
	return ErrUnauthorized{
		message: fmt.Sprintf(format, args...),
	}
}

func (e ErrUnauthorized) Error() string {
	return e.message
}

//ErrForbidden is returned from the API if a request returns a 403 status code.
type ErrForbidden struct {
	message string
}

//NewErrForbidden returns a new instance of ErrForbidden, like fmt.Errorf
func NewErrForbidden(format string, args ...interface{}) error {
	return ErrForbidden{
		message: fmt.Sprintf(format, args...),
	}
}

func (e ErrForbidden) Error() string {
	return e.message
}

func getV1Error(r *http.Response) error {
	body, err := ioutil.ReadAll(r.Body)
	if err != nil {
		return err
	}

	errorString := string(body)

	if r.StatusCode == 401 || r.StatusCode == 403 {
		err = NewErrUnauthorized(errorString)
	} else {
		err = fmt.Errorf(errorString)
	}
	return err
}

func getAPIError(r *http.Response) error {
	body, err := ioutil.ReadAll(r.Body)
	if err != nil {
		return err
	}

	var errorString string

	m := map[string]interface{}{}
	if err = json.Unmarshal(body, &m); err != nil { //v1 api doesn't return JSON
		errorString = string(body)
	} else {
		errorString = getJSONErrorString(m)
	}

	switch r.StatusCode {
	case 400:
		err = NewErrBadRequest(errorString)
	case 401:
		err = NewErrUnauthorized(errorString)
	case 403:
		err = NewErrForbidden(errorString)
	default:
		err = fmt.Errorf(errorString)
	}

	return err
}

func getJSONErrorString(m map[string]interface{}) string {
	errorString, hasErrorKey := m["error"].(string)
	if hasErrorKey {
		return errorString
	}
	missingKeys, hasMissingKey := m["missing"].([]interface{})
	var missingStrings []string
	for _, key := range missingKeys {
		missingStrings = append(missingStrings, key.(string))
	}
	if hasMissingKey {
		return fmt.Sprintf("missing keys: `%s'", strings.Join(missingStrings, "', `"))
	}

	j, err := json.Marshal(&m)
	if err != nil {
		panic("Couldn't marshal given map into JSON")
	}

	return string(j)
}
