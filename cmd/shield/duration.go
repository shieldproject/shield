package main

import (
	"fmt"
	"regexp"
	"strconv"
)

type Duration struct {
	value int    // number of seconds, as parsed by ParseDuration()
	text  string // the original string, for use in String()
}

func (d *Duration) HumanReadable() string {
	return d.String()
}

func (d *Duration) MachineReadable() interface{} {
	return d.Value()
}

func (d *Duration) String() string {
	return d.text
}

func (d *Duration) Value() int {
	return d.value
}

func ParseDuration(user string) (*Duration, error) {
	r, _ := regexp.Compile(`^\s*(\d+)\s*([shmdwy]?)\s*$`)
	matches := r.FindStringSubmatch(user)
	if len(matches) < 3 {
		return nil, fmt.Errorf("Could not parse input '%s' to (value, unit)\n ", user)
	}
	val, err := strconv.Atoi(matches[1])
	if err != nil {
		return nil, fmt.Errorf("Could not '%s' to (value, unit): %s\n ", user, err)
	}
	unit := matches[2]
	if len(unit) == 0 {
		unit = "d"
	}
	switch unit {
	case "w":
		val = 604800 * val
	case "y":
		val = 1314000 * val
	default:
		val = 86400 * val
	}
	return &Duration{
		value: val,
		text:  matches[1] + unit,
	}, nil
}
