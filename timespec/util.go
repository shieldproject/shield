package timespec

import (
	"fmt"
	"time"
)

func hhmm12(hours, minutes uint, am bool) int {
	if am && hours == 12 { /* 12am is 00:00 */
		hours = 0
	} else if !am && hours < 12 { /* 12pm is 12:00 */
		hours += 12
	}
	return hhmm24(hours, minutes)
}

func hhmm24(hours, minutes uint) int {
	return int(hours*60 + minutes)
}

func minutely(offset int, minutes int) *Spec {
	if minutes < 1 {
		return &Spec{
			Error: fmt.Errorf("Schedules cannot be created for a higher frequency than once per minute"),
		}
	}
	if minutes > 1440 {
		return &Spec{
			Error: fmt.Errorf("Minutely schedules cannot be created for a lower frequency than 1440 minutes (1 day)"),
		}
	}
	if offset > minutes {
		m := offset
		ampm := "am"
		if m > 12*60 {
			ampm = "pm"
			m -= 12 * 60
		}
		if m < 60 {
			m += 12 * 60
		}
		return &Spec{
			Error: fmt.Errorf("A schedule to run every %d minute(s) must start before %d:%02d%s", minutes, m/60, m%60, ampm),
		}
	}

	return &Spec{
		Interval:    Minutely,
		TimeOfDay:   offset,
		Cardinality: float32(minutes),
	}
}

func hourly(minutes int, cardinality float32) *Spec {
	if cardinality != 0 {
		//Bounds checking on cardinality to ensure it is positive and reduced
		if float32(minutes) >= (cardinality*60) || cardinality < 0 || cardinality > 23 {
			m := cardinality * 60
			ampm := "am"
			if m >= 12*60 {
				ampm = "pm"
				m -= 12 * 60
			}
			if m < 60 {
				m += 12 * 60
			}
			return &Spec{
				Error: fmt.Errorf("A schedule to run every %0.1f hour(s) must start before %d:%02d%s", cardinality, int(m/60), int(m)%60, ampm),
			}
		}
		return &Spec{
			Interval:    Hourly,
			TimeOfDay:   minutes,
			Cardinality: cardinality,
		}
	}
	return &Spec{
		Interval:    Hourly,
		TimeOfHour:  minutes,
		Cardinality: cardinality,
	}
}

func daily(minutes int) *Spec {
	return &Spec{
		Interval:  Daily,
		TimeOfDay: minutes,
	}
}
func weekly(minutes int, weekday time.Weekday) *Spec {
	return &Spec{
		Interval:  Weekly,
		TimeOfDay: minutes,
		DayOfWeek: weekday,
	}
}
func mday(minutes int, day uint) *Spec {
	return &Spec{
		Interval:   Monthly,
		TimeOfDay:  minutes,
		DayOfMonth: int(day),
	}
}
func mweek(minutes int, weekday time.Weekday, week uint) *Spec {
	return &Spec{
		Interval:  Monthly,
		TimeOfDay: minutes,
		DayOfWeek: weekday,
		Week:      int(week),
	}
}
