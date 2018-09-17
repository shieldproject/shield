package timespec

import (
	"fmt"
	"time"
)

type Interval uint

const (
	Minutely Interval = iota
	Hourly
	Daily
	Weekly
	Monthly
)

const ns = 1000 * 1000 * 1000

type Spec struct {
	Error error

	Interval    Interval
	TimeOfDay   int
	TimeOfHour  int
	DayOfWeek   time.Weekday
	DayOfMonth  int
	Week        int
	Cardinality float32
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

	if s.Interval == Minutely {
		if s.Cardinality == 1.0 {
			return "every minute"
		}
		if s.TimeOfDay == 0 {
			return fmt.Sprintf("every %d minutes", int(s.Cardinality))
		}

		m := s.TimeOfDay
		ampm := "am"
		if m > 12*60 {
			ampm = "pm"
			m -= 12 * 60
		}
		if m < 60 {
			m += 12 * 60
		}
		return fmt.Sprintf("every %d minutes from %d:%02d%s", int(s.Cardinality), m/60, m%60, ampm)
	}

	if s.Interval == Hourly && s.TimeOfHour < 60 {
		if s.Cardinality == 0 {
			return fmt.Sprintf("hourly at %d after", s.TimeOfHour)
		}
		if s.Cardinality == 0.25 {
			return fmt.Sprintf("every quarter hour from %s", t)
		}
		if s.Cardinality == 0.5 {
			return fmt.Sprintf("every half hour from %s", t)
		}
		return fmt.Sprintf("every %d hours from %s", int(s.Cardinality), t)
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

	if s.Interval == Minutely {
		target := offsetM(midnight, s.TimeOfDay)
		for i := 0; i < 1440; i++ { //Incrementing 1440 minutes in the worst case
			if target.After(t) {
				return target, nil
			}
			target = offsetM(target, int(s.Cardinality))
		}
		return t, fmt.Errorf("Cannot calculate the %0.2fth minute", s.Cardinality)
	}

	if s.Interval == Hourly && s.TimeOfHour < 60 {
		if s.Cardinality != 0 {
			//Check bounds on cardinality for a one day period
			if s.Cardinality < 0 || s.Cardinality > 23 {
				return t, fmt.Errorf("Invalid Cardinality: Cannot calculate the %0.2fth hour in a day", s.Cardinality)
			}
			//Check Cardinality is Valid (i.e. integer, half, or quarter)
			if s.Cardinality != float32(int(s.Cardinality)) {
				if s.Cardinality != 0.5 && s.Cardinality != 0.25 {
					return t, fmt.Errorf("Invalid Cardinality: Cannot calculate the %0.2fth hour in a day", s.Cardinality)
				}
			}
			//Ensure "FROM" is reduced to its simplest form
			if float32(s.TimeOfDay) >= (s.Cardinality * 60) {
				s.TimeOfDay = s.TimeOfDay % int(s.Cardinality*60)
				return t, fmt.Errorf("Invalid FROM time: did you mean %d:%02d", s.TimeOfDay/60, s.TimeOfDay%60)
			}
			target := offsetM(midnight, s.TimeOfDay)
			for i := 0; i < 97; i++ { //Incrementing 96 1/4hours in the worst case
				if target.After(t) {
					return target, nil
				}
				target = offsetM(target, int(60*s.Cardinality))
			}
			return t, fmt.Errorf("Cannot calculate the %0.2fth hour in a day", s.Cardinality)
		}

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
		if target.Before(t) || target.Equal(t) {
			target = offsetM(target, 7*1440)
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
