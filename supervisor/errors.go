package supervisor

import (
	"encoding/json"
	"fmt"
	"strings"
)

func Sentencify(words []string) string {
	switch len(words) {
	case 0:
		return ""
	case 1:
		return words[0]
	default:
		head := words[0 : len(words)-1]
		return strings.Join(head, ", ") + " and " + words[len(words)-1]
	}
}

type JSONError interface {
	JSON() string
}

type ClientError struct {
	Error string `json:"error"`
}

func ClientErrorf(format string, v ...interface{}) ClientError {
	return ClientError{fmt.Sprintf(format, v...)}
}

func (e ClientError) JSON() string {
	b, err := json.Marshal(e)
	if err != nil {
		return fmt.Sprintf(`{"error":"failed to unmarshal JSON: %s"}`, err)
	}
	return string(b)
}

type Validator func(name string, value interface{}) error

type InvalidParametersError struct {
	Errors map[string]string
}

func InvalidParameters(names ...string) InvalidParametersError {
	return InvalidParametersError{
		Errors: make(map[string]string),
	}
}

func (e *InvalidParametersError) Validate(name string, value interface{}, fn Validator) {
	err := fn(name, value)
	if err != nil {
		e.Errors[name] = err.Error()
	}
}

func (e *InvalidParametersError) IsValid() bool {
	return len(e.Errors) > 0
}

func (e InvalidParametersError) Error() string {
	keys := make([]string, len(e.Errors))
	for k, _ := range e.Errors {
		keys = append(keys, k)
	}
	return fmt.Sprintf("%s are invaid parameters", Sentencify(keys))
}

func (e InvalidParametersError) JSON() string {
	b, err := json.Marshal(e)
	if err != nil {
		return fmt.Sprintf(`{"error":"failed to unmarshal JSON: %s"}`, err)
	}
	return string(b)
}

type MissingParametersError struct {
	Missing []string `json:"missing"`
}

func MissingParameters(names ...string) MissingParametersError {
	return MissingParametersError{
		Missing: names,
	}
}

func (e *MissingParametersError) Check(name string, value string) {
	if value == "" {
		e.Missing = append(e.Missing, name)
	}
}

func (e MissingParametersError) IsValid() bool {
	return len(e.Missing) > 0
}

func (e MissingParametersError) Error() string {
	return fmt.Sprintf("Missing %s parameters", Sentencify(e.Missing))
}

func (e MissingParametersError) JSON() string {
	b, err := json.Marshal(e)
	if err != nil {
		return fmt.Sprintf(`{"error":"failed to unmarshal JSON: %s"}`, err)
	}
	return string(b)
}
