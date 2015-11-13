package plugin

import (
	"fmt"
)

/*

Hi Jaime! Here's where we define exit codes that the plugins will use, so that all plugins can behave in a consistent manner

*/

const SUCCESS = 0
const USAGE = 1
const UNSUPPORTED_ACTION = 2
const EXEC_FAILURE = 3
const PLUGIN_FAILURE = 4
const JSON_FAILURE = 10
const RESTORE_KEY_REQUIRED = 11
const ENDPOINT_MISSING_KEY = 12
const ENDPOINT_BAD_DATA = 13

type UnsupportedActionError struct {
	Action string
}

func (e UnsupportedActionError) Error() string {
	return fmt.Sprintf("The '%s' command is currently unsupported by this plugin", e.Action)
}

var UNIMPLEMENTED = UnsupportedActionError{}

type EndpointMissingRequiredDataError struct {
	Key string
}

func (e EndpointMissingRequiredDataError) Error() string {
	return fmt.Sprintf("No '%s' key specified in the endpoint json", e.Key)
}

type EndpointDataTypeMismatchError struct {
	Key         string
	DesiredType string
}

func (e EndpointDataTypeMismatchError) Error() string {
	return fmt.Sprintf("'%s' key in endpoint json is not of type '%s'", e.Key, e.DesiredType)
}

type ExecFailure struct {
	Err string
}

func (e ExecFailure) Error() string {
	return e.Err
}

type JSONError struct {
	Err string
}

func (e JSONError) Error() string {
	return e.Err
}

type MissingRestoreKeyError struct{}

func (e MissingRestoreKeyError) Error() string {
	return "retrieving requires --key, but it was not provided"
}
