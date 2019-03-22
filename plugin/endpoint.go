package plugin

import (
	"encoding/json"
	"fmt"
	"reflect"
	"regexp"
)

type ShieldEndpoint map[string]interface{}

func getEndpoint(j string) (ShieldEndpoint, error) {
	if j == "" {
		return nil, fmt.Errorf("Missing required --endpoint flag")
	}
	endpoint := make(ShieldEndpoint)
	err := json.Unmarshal([]byte(j), &endpoint)
	if err != nil {
		return nil, JSONError{Err: fmt.Sprintf("Error trying parse --endpoint value as JSON: %s", err.Error())}
	}

	return endpoint, nil
}

func (endpoint ShieldEndpoint) StringValue(key string) (string, error) {
	_, ok := endpoint[key]
	if !ok {
		return "", EndpointMissingRequiredDataError{Key: key}
	}

	if reflect.TypeOf(endpoint[key]).Kind() != reflect.String {
		return "", EndpointDataTypeMismatchError{Key: key, DesiredType: "string"}
	}

	return endpoint[key].(string), nil
}

func (endpoint ShieldEndpoint) StringValueDefault(key string, def string) (string, error) {
	s, err := endpoint.StringValue(key)
	if err == nil {
		if s == "" {
			return def, nil
		}
		return s, nil
	}
	if _, ok := err.(EndpointMissingRequiredDataError); ok {
		return def, nil
	}
	return "", err
}

func (endpoint ShieldEndpoint) FloatValue(key string) (float64, error) {
	_, ok := endpoint[key]
	if !ok {
		return 0, EndpointMissingRequiredDataError{Key: key}
	}

	if reflect.TypeOf(endpoint[key]).Kind() != reflect.Float64 {
		return 0, EndpointDataTypeMismatchError{Key: key, DesiredType: "numeric"}
	}

	return endpoint[key].(float64), nil
}

func (endpoint ShieldEndpoint) FloatValueDefault(key string, def float64) (float64, error) {
	f, err := endpoint.FloatValue(key)
	if err == nil {
		return f, nil
	}
	if _, ok := err.(EndpointMissingRequiredDataError); ok {
		return def, nil
	}
	return 0.0, err
}

func (endpoint ShieldEndpoint) BooleanValue(key string) (bool, error) {
	_, ok := endpoint[key]
	if !ok {
		return false, EndpointMissingRequiredDataError{Key: key}
	}

	if s, ok := endpoint[key].(string); ok {
		if regexp.MustCompile(`(?i)^(1|y|yes|t|true|on)$`).MatchString(s) {
			return true, nil
		}
		if regexp.MustCompile(`(?i)^(0|n|no|f|false|off|)$`).MatchString(s) {
			return false, nil
		}
	}

	if v, ok := endpoint[key].(float64); ok {
		if uint(v) == 1 {
			return true, nil
		}
		if uint(v) == 0 {
			return false, nil
		}
	}

	if v, ok := endpoint[key].(int); ok {
		if v == 1 {
			return true, nil
		}
		if v == 0 {
			return false, nil
		}
	}

	if reflect.TypeOf(endpoint[key]).Kind() != reflect.Bool {
		return false, EndpointDataTypeMismatchError{Key: key, DesiredType: "boolean"}
	}

	return endpoint[key].(bool), nil
}

func (endpoint ShieldEndpoint) BooleanValueDefault(key string, def bool) (bool, error) {
	tf, err := endpoint.BooleanValue(key)
	if err == nil {
		return tf, nil
	}
	if _, ok := err.(EndpointMissingRequiredDataError); ok {
		return def, nil
	}
	return false, err
}

func (endpoint ShieldEndpoint) ArrayValue(key string) ([]interface{}, error) {
	_, ok := endpoint[key]
	if !ok {
		return nil, EndpointMissingRequiredDataError{Key: key}
	}

	if reflect.TypeOf(endpoint[key]).Kind() != reflect.Slice {
		return nil, EndpointDataTypeMismatchError{Key: key, DesiredType: "array"}
	}

	return endpoint[key].([]interface{}), nil
}

func (endpoint ShieldEndpoint) MapValue(key string) (map[string]interface{}, error) {
	_, ok := endpoint[key]
	if !ok {
		return nil, EndpointMissingRequiredDataError{Key: key}
	}

	if reflect.TypeOf(endpoint[key]).Kind() != reflect.Map {
		return nil, EndpointDataTypeMismatchError{Key: key, DesiredType: "map"}
	}

	return endpoint[key].(map[string]interface{}), nil
}
