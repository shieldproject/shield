package util

import (
	"fmt"
)

func StringifyKeys(things interface{}) interface{} {
	switch things.(type) {
	case map[interface{}]interface{}:
		m := make(map[string]interface{})
		for k, v := range things.(map[interface{}]interface{}) {
			m[fmt.Sprintf("%s", k)] = StringifyKeys(v)
		}
		return m

	case []interface{}:
		l := make([]interface{}, 0)
		for _, thing := range things.([]interface{}) {
			l = append(l, StringifyKeys(thing))
		}
		return l

	default:
		return things
	}
}
