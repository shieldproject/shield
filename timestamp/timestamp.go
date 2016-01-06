package timestamp

import (
	"fmt"
	"time"
)

type Timestamp struct {
	t time.Time
}

func (orig Timestamp) Add(d time.Duration) Timestamp {
	return Timestamp{t: orig.t.Add(d)}
}

func NewTimestamp(t time.Time) Timestamp {
	return Timestamp{t: t}
}

func (t Timestamp) MarshalJSON() ([]byte, error) {
	if t.t.IsZero() {
		return []byte("\"\""), nil
	}
	stamp := fmt.Sprintf("\"%s\"", t.t.Format("2006-01-02 15:04:05"))
	return []byte(stamp), nil
}

func (t *Timestamp) UnmarshalJSON(b []byte) error {
	if string(b) == "\"\"" {
		return nil
	}
	var err error
	t.t, err = time.Parse("2006-01-02 15:04:05", string(b[1:len(b)-1]))
	return err
}

func (t Timestamp) Format(layout string) string {
	return t.t.Format(layout)
}

func (t Timestamp) IsZero() bool {
	return t.t.IsZero()
}

func (t Timestamp) Time() time.Time {
	return t.t
}
