package api_agent

import (
	"fmt"
)

// FIXME - Placeholder until plugin's schema is defined
type AnnotatedPlugin struct {
	Name string `json:"name"`
}

func FetchListPlugins() (*[]AnnotatedPlugin, error) {

	// Data to be returned of proper type
	data := &[]AnnotatedPlugin{}

	// Make uri based on options
	uri := fmt.Sprintf("v1/plugins")

	// Call generic API request
	err := makeApiCall(data, `GET`, uri, nil)
	return data, err
}

func GetPlugin(uuid string) (*AnnotatedPlugin, error) {
	// Data to be returned of proper type
	data := &AnnotatedPlugin{}

	// Make uri based on options
	uri := fmt.Sprintf("v1/plugin/%s", uuid)

	// Call generic API request
	err := makeApiCall(data, `GET`, uri, nil)
	return data, err
}
