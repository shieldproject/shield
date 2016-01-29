package timespec

import (
	"fmt"
	"time"
)

type Interval uint

const (
	Hourly Interval = iota
	Daily
	Weekly
	Monthly
)

const ns = 1000 * 1000 * 1000

type Spec struct {
	Interval   Interval
	TimeOfDay  int
	TimeOfHour int
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

func ord(n int) string {
	switch {
	case n%100 >= 11 && n%100 <= 13:
		return "th"
	case n%10 == 1:
		return "st"
	case n%10 == 2:
		return "nd"
	case n%10 == 3:
		return "rd"
	}
	return "th"
}

func weekday(d time.Weekday) string {
	switch d {
	case time.Sunday:
		return "sunday"
	case time.Monday:
		return "monday"
	case time.Tuesday:
		return "tuesday"
	case time.Wednesday:
		return "wednesday"
	case time.Thursday:
		return "thursday"
	case time.Friday:
		return "friday"
	case time.Saturday:
		return "saturday"
	}

	return "unknown-weekday"
}

func (s *Spec) String() string {
	t := fmt.Sprintf("%d:%02d", s.TimeOfDay/60, s.TimeOfDay%60)

	if s.Interval == Hourly && s.TimeOfHour < 60 {
		return fmt.Sprintf("hourly at %d after", s.TimeOfHour)
	}

	if s.Interval == Daily {
		return fmt.Sprintf("daily at %s", t)

	} else if s.Interval == Weekly {
		return fmt.Sprintf("%ss at %s", weekday(s.DayOfWeek), t)

	} else if s.Interval == Monthly && s.Week != 0 {
		return fmt.Sprintf("%d%s %s at %s", s.Week, ord(s.Week), weekday(s.DayOfWeek), t)

	} else if s.Interval == Monthly && s.DayOfMonth != 0 {
		return fmt.Sprintf("monthly at %s on %d%s", t, s.DayOfMonth, ord(s.DayOfMonth))
	}

	return "<unknown interval>"
}

func (s *Spec) Next(t time.Time) (time.Time, error) {
	t = roundM(t)
	midnight := offsetM(t, -1*(t.Hour()*60+t.Minute()))

	if s.Interval == Hourly && s.TimeOfHour < 60 {
		target := offsetM(t, s.TimeOfHour-t.Minute())
		if target.After(t) {
			return target, nil
		}
		return offsetM(target, 60), nil
	}

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
