package director

import (
	"time"
)

type TimeParser struct{}

func (p TimeParser) Parse(s string) (time.Time, error) {
	parsed, err := time.Parse("2006-01-02 15:04:05 -0700", s)
	if err != nil {
		parsed, err = time.Parse("2006-01-02 15:04:05 MST", s)
		if err != nil {
			return time.Time{}, err
		}
	}

	return parsed.UTC(), nil
}
