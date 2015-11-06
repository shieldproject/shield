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
		return nil, err
	}

	return endpoint, nil
}

func (endpoint ShieldEndpoint) StringValue(key string) (string, error) {
	_, ok := endpoint[key]
	if !ok {
		return "", fmt.Errorf("No '%s' key specified in the endpoint json", key)
	}

	if reflect.TypeOf(endpoint[key]).Kind() != reflect.String {
		return "", fmt.Errorf("'%s' key in endpoint json is a %s, not a string", key, reflect.TypeOf(endpoint[key]).Name())
	}

	return endpoint[key].(string), nil
}

func (endpoint ShieldEndpoint) FloatValue(key string) (float64, error) {
	_, ok := endpoint[key]
	if !ok {
		return 0, fmt.Errorf("No '%s' key specified in the endpoint json", key)
	}

	if reflect.TypeOf(endpoint[key]).Kind() != reflect.Float64 {
		return 0, fmt.Errorf("'%s' key in endpoint json is a %s, not a numeric", key, reflect.TypeOf(endpoint[key]).Name())
	}

	return endpoint[key].(float64), nil
}

func (endpoint ShieldEndpoint) BooleanValue(key string) (bool, error) {
	_, ok := endpoint[key]
	if !ok {
		return false, fmt.Errorf("No '%s' key specified in the endpoint json", key)
	}

	if reflect.TypeOf(endpoint[key]).Kind() != reflect.Bool {
		return false, fmt.Errorf("'%s' key in endpoint json is a %s, not a boolean", key, reflect.TypeOf(endpoint[key]).Name())
	}

	return endpoint[key].(bool), nil
}

func (endpoint ShieldEndpoint) ArrayValue(key string) ([]interface{}, error) {
	_, ok := endpoint[key]
	if !ok {
		return nil, fmt.Errorf("No '%s' key specified in the endpoint json", key)
	}

	if reflect.TypeOf(endpoint[key]).Kind() != reflect.Slice {
		return nil, fmt.Errorf("'%s' key in endpoint json is a %s, not an array", key, reflect.TypeOf(endpoint[key]).Name())
	}

	return endpoint[key].([]interface{}), nil
}

func (endpoint ShieldEndpoint) MapValue(key string) (map[string]interface{}, error) {
	_, ok := endpoint[key]
	if !ok {
		return nil, fmt.Errorf("No '%s' key specified in the endpoint json", key)
	}

	if reflect.TypeOf(endpoint[key]).Kind() != reflect.Map {
		return nil, fmt.Errorf("'%s' key in endpoint json is a %s, not a map", key, reflect.TypeOf(endpoint[key]).Name())
	}

	return endpoint[key].(map[string]interface{}), nil
}
