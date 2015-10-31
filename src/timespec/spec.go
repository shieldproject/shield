package timespec

import (
	"fmt"
	"time"
)

type Interval uint

const (
	Daily Interval = iota
	Weekly
	Monthly
)

const ns = 1000 * 1000 * 1000

type Spec struct {
	Interval   Interval
	TimeOfDay  int
	DayOfWeek  time.Weekday
	DayOfMonth int
	Week       int
}

func roundM(t time.Time) time.Time {
	return t.Add(time.Duration(-1 * (ns*t.Second() + t.Nanosecond())))
}
func offsetM(t time.Time, m int) time.Time {
	return t.Add(time.Duration(ns * 60 * m))
}
func nthWeek(t time.Time) int {
	return int(t.Day()/7) + 1
}

func (s *Spec) Next(t time.Time) (time.Time, error) {
	t = roundM(t)
	midnight := offsetM(t, -1*(t.Hour()*60+t.Minute()))

	if s.Interval == Daily {
		target := offsetM(midnight, s.TimeOfDay)
		if target.After(t) {
			return target, nil
		}
		return offsetM(target, 1440), nil

	} else if s.Interval == Weekly {
		target := offsetM(midnight, s.TimeOfDay)
		for target.Weekday() != s.DayOfWeek {
			target = offsetM(target, 1440)
		}
		return target, nil

	} else if s.Interval == Monthly && s.Week != 0 {
		if s.Week < 1 || s.Week > 5 {
			return t, fmt.Errorf("Cannot calculate the %dth week in a month", s.Week)
		}
		target := offsetM(midnight, s.TimeOfDay)
		for target.Weekday() != s.DayOfWeek || target.Before(t) {
			target = offsetM(target, 1440)
		}
		for nthWeek(target) != s.Week {
			target = offsetM(target, 1440*7)
		}
		return target, nil

	} else if s.Interval == Monthly && s.DayOfMonth != 0 {
		if s.DayOfMonth < 1 || s.DayOfMonth > 31 {
			return t, fmt.Errorf("Cannot calculate the %dth week in a month", s.Week)
		}
		target := offsetM(midnight, s.TimeOfDay)
		for target.Day() != s.DayOfMonth || target.Before(t) {
			target = offsetM(target, 1440)
		}
		return target, nil
	}

	return t, fmt.Errorf("unhandled Interval for Spec")
}
