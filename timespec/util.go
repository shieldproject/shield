package timespec

import (
	"time"
)

func hhmm12(hours, minutes uint, am bool) int {
	if am && hours == 12 { /* 12am is 00:00 */
		hours = 0
	} else if !am && hours > 12 { /* 12pm is 12:00 */
		hours += 12
	}
	return hhmm24(hours, minutes)
}

func hhmm24(hours, minutes uint) int {
	return int(hours*60 + minutes)
}

func hourly(minutes int, cardinality float32) *Spec {
	if cardinality != 0 {
		//Bounds checking on cardinality to ensure it is positive and reduced
		if float32(minutes) > (cardinality*60) || cardinality < 0 || cardinality > 23 {
			return nil
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
