package api

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
)

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

func getV2Error(r *http.Response) error {
	body, err := ioutil.ReadAll(r.Body)
	if err != nil {
		return err
	}

	m := map[string]interface{}{}
	if err = json.Unmarshal(body, &m); err != nil {
		return err
	}

	errorString := m["error"].(string)

	if r.StatusCode == 401 {
		err = NewErrUnauthorized(errorString)
	} else if r.StatusCode == 403 {
		err = NewErrForbidden(errorString)
	} else {
		err = fmt.Errorf(errorString)
	}
	return err
}
