package util

import (
	"regexp"
	"strconv"
)

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
