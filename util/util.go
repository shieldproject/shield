package util

import (
	"fmt"
	"regexp"
	"strconv"
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

func ParseRetain(s string) int {
	if m := regexp.MustCompile("^([0-9]+)\\s*([dDwW]?)$").FindStringSubmatch(s); m != nil {
		n, _ := strconv.ParseInt(m[1], 10, 64)
		switch m[2] {
		case "d", "D", "":
			return int(n)
		case "w", "W":
			return int(n) * 7
		}
	}
	return -1
}
