package supervisor

import (
	"strings"
	"encoding/json"
	"fmt"
)

func Sentencify(words []string) string {
	switch len(words) {
	case 0:
		return ""
	case 1:
		return words[0]
	default:
		head := words[0:len(words)-1]
		return strings.Join(head, ", ") + " and " + words[len(words)-1]
	}
}

type JSONError interface {
	JSON() string
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

func (e MissingParametersError) Valid() bool {
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
