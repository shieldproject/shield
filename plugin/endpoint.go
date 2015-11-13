package plugin

/*

ShieldEndpoints are used for store + targets. This code genericizes them and makes it easy for you to pull out arbitrary values from them. The plugin framework will feed your action methods with the appropriate endpoint, and you can pull whatever data out that you need.

*/

import (
	"encoding/json"
	"fmt"
	"reflect"
)

type ShieldEndpoint map[string]interface{}

func getEndpoint(j string) (ShieldEndpoint, error) {
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

func (endpoint ShieldEndpoint) BooleanValue(key string) (bool, error) {
	_, ok := endpoint[key]
	if !ok {
		return false, EndpointMissingRequiredDataError{Key: key}
	}

	if reflect.TypeOf(endpoint[key]).Kind() != reflect.Bool {
		return false, EndpointDataTypeMismatchError{Key: key, DesiredType: "boolean"}
	}

	return endpoint[key].(bool), nil
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
