package api_agent

import (
	"fmt"
	"github.com/starkandwayne/shield/db"
)

func FetchListTargets(plugin, unused string) (*[]db.AnnotatedTarget, error) {

	// Data to be returned of proper type
	data := &[]db.AnnotatedTarget{}

	// Make uri based on options
	uri := fmt.Sprintf("/v1/targets")
	joiner := "?"
	if plugin != "" {
		uri = fmt.Sprintf("%s%splugin=%s", uri, joiner, plugin)
		joiner = "&"
	}
	if unused != "" {
		uri = fmt.Sprintf("%s%sunused=%s", uri, joiner, unused)
		joiner = "&"
	}

	// Call generic API request
	err := makeApiCall(data, `GET`, uri, nil)
	return data, err
}
