package core

import (
	"fmt"
	"regexp"
	"strconv"
)

var durationPattern *regexp.Regexp

func init() {
	durationPattern = regexp.MustCompile(`^(?i)(\d+(?:\.\d+)?)\s?([Ywdhms]?)$`)
}

type duration int

func (d duration) String() string {
	i := (int)(d)

	f := 60 * 60 * 24 * 365
	if i >= f {
		if i%f == 0 {
			return fmt.Sprintf("%dy", (int)(i/f))
		}
		return fmt.Sprintf("%0.2fy", (float32)(i/f))
	}

	f = 60 * 60 * 24
	if i >= f {
		if i%f == 0 {
			return fmt.Sprintf("%dd", (int)(i/f))
		}
		return fmt.Sprintf("%0.2fd", (float32)(i/f))
	}

	f = 60 * 60
	if i >= f {
		if i%f == 0 {
			return fmt.Sprintf("%dh", (int)(i/f))
		}
		return fmt.Sprintf("%0.2fh", (float32)(i/f))
	}

	f = 60
	if i >= f && i%f == 0 {
		return fmt.Sprintf("%dm", (int)(i/f))
	}

	return fmt.Sprintf("%ds", i)
}

func (d duration) parse(raw string) (int, error) {
	m := durationPattern.FindStringSubmatch(raw)
	if m == nil {
		return 0, fmt.Errorf("invalid duration '%s' (expecting something like '90d')", raw)
	}

	f, err := strconv.ParseFloat(m[1], 64)
	if err != nil {
		return 0, fmt.Errorf("invalid duration '%s' (%s)", raw, err)
	}

	switch m[2] {
	case "", "s", "S":
		return (int)(f), nil

	case "m", "M":
		return (int)(f * 60), nil

	case "h", "H":
		return (int)(f * 60 * 60), nil

	case "d", "D":
		return (int)(f * 60 * 60 * 24), nil

	case "w", "W":
		return (int)(f * 60 * 60 * 24 * 7), nil

	case "y", "Y":
		return (int)(f * 60 * 60 * 24 * 365), nil
	}

	return 0, fmt.Errorf("unrecognized duration unit '%s' (in '%s')", m[2], raw)
}

func (d *duration) UnmarshalYAML(unmarshal func(interface{}) error) error {
	var raw string

	err := unmarshal(&raw)
	if err != nil {
		return err
	}

	d.UnmarshalEnv(raw)
	return nil
}

func (d *duration) UnmarshalEnv(raw string) {
	i, err := d.parse(raw)
	if err != nil {
		panic(err)
	}

	*d = (duration)(i)
}
