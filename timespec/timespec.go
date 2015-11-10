package timespec

import (
	"time"
)

func Next(s string) (time.Time, error) {
	t := time.Now()
	spec, err := Parse(s)
	if err != nil {
		return t, err
	}
	t, err = spec.Next(t)
	if err != nil {
		return t, err
	}
	return t, nil
}
