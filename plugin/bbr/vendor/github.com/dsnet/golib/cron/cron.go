// Copyright 2017, Joe Tsai. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE.md file.

// Package cron provides functionality for parsing and running cron schedules.
package cron

import (
	"context"
	"errors"
	"strconv"
	"strings"
	"time"
)

var (
	monthNames = map[string]int{
		"JAN": 1, "FEB": 2, "MAR": 3, "APR": 4, "MAY": 5, "JUN": 6,
		"JUL": 7, "AUG": 8, "SEP": 9, "OCT": 10, "NOV": 11, "DEC": 12,
	}
	dayNames = map[string]int{
		"SUN": 0, "MON": 1, "TUE": 2, "WED": 3, "THU": 4, "FRI": 5, "SAT": 6,
	}
	scheduleMacros = map[string]string{
		"@yearly":   "0 0 1 1 *",
		"@annually": "0 0 1 1 *",
		"@monthly":  "0 0 1 * *",
		"@weekly":   "0 0 * * 0",
		"@daily":    "0 0 * * *",
		"@hourly":   "0 * * * *",
	}
)

// set64 represents a set of integers containing values in 0..63.
type set64 uint64

func (s *set64) set(i int)     { *s |= 1 << uint(i) }
func (s set64) has(i int) bool { return s&(1<<uint(i)) != 0 }

// Schedule represents a cron schedule.
type Schedule struct {
	str      string
	mins     set64 // 0-59
	hours    set64 // 0-23
	days     set64 // 1-31
	months   set64 // 1-12
	weekDays set64 // 0-6
}

// ParseSchedule parses a cron schedule, which is a space-separated list of
// five fields representing:
//	• minutes:       0-59
//	• hours:         0-23
//	• days of month: 1-31
//	• months:        1-12 or JAN-DEC
//	• days of week:  0-6 or SUN-SAT
//
// Each field may be a glob (e.g., "*"), representing the full range of values,
// or a comma-separated list, containing individual values (e.g., "JAN")
// or a dash-separated pair representing a range of values (e.g., "MON-FRI").
//
// The following macros are permitted:
//	• @yearly:   "0 0 1 1 *"
//	• @annually: "0 0 1 1 *"
//	• @monthly:  "0 0 1 * *"
//	• @weekly:   "0 0 * * 0"
//	• @daily:    "0 0 * * *"
//	• @hourly:   "0 * * * *"
//
// A given timestamp is in the schedule if the associated fields
// of the timestamp matches each field specified in the schedule.
//
// See https://wikipedia.org/wiki/cron
func ParseSchedule(s string) (Schedule, error) {
	s = strings.Join(strings.Fields(s), " ")
	sch := Schedule{str: s}
	if scheduleMacros[s] != "" {
		s = scheduleMacros[s]
	}
	var ok [5]bool
	if ss := strings.Fields(s); len(ss) == 5 {
		sch.mins, ok[0] = parseField(ss[0], 0, 59, nil)
		sch.hours, ok[1] = parseField(ss[1], 0, 23, nil)
		sch.days, ok[2] = parseField(ss[2], 1, 31, nil)
		sch.months, ok[3] = parseField(ss[3], 1, 12, monthNames)
		sch.weekDays, ok[4] = parseField(ss[4], 0, 6, dayNames)
	}
	if ok != [5]bool{true, true, true, true, true} {
		return Schedule{}, errors.New("cron: invalid schedule: " + s)
	}
	return sch, nil
}

func parseField(s string, min, max int, aliases map[string]int) (set64, bool) {
	var m set64
	for _, s := range strings.Split(s, ",") {
		var lo, hi int
		if i := strings.IndexByte(s, '-'); i >= 0 {
			lo = parseToken(s[:i], min, aliases)
			hi = parseToken(s[i+1:], max, aliases)
		} else {
			lo = parseToken(s, min, aliases)
			hi = parseToken(s, max, aliases)
		}
		if lo < min || max < hi || hi < lo {
			return m, false
		}
		for i := lo; i <= hi; i++ {
			m.set(i)
		}
	}
	return m, true
}

func parseToken(s string, wild int, aliases map[string]int) int {
	if n, ok := aliases[strings.ToUpper(s)]; ok {
		return n
	}
	if s == "*" {
		return wild
	}
	if n, err := strconv.Atoi(s); err == nil {
		return n
	}
	return -1
}

// NextAfter returns the next scheduled event, relative to the specified t,
// taking into account t.Location.
// This returns the zero value if unable to determine the next scheduled event.
func (s Schedule) NextAfter(t time.Time) time.Time {
	if s == (Schedule{}) {
		return time.Time{}
	}

	// Round-up to the nearest minute.
	t = t.Add(time.Minute).Truncate(time.Minute)

	// Increment min/hour first, then increment by days.
	// When incrementing by days, we do not need to verify the min/hour again.
	t100 := t.AddDate(100, 0, 0) // Sanity bounds of 100 years
	for !s.mins.has(t.Minute()) && t.Before(t100) {
		t = t.Add(time.Minute)
	}
	for !s.hours.has(t.Hour()) && t.Before(t100) {
		t = t.Add(time.Hour)
	}
	for !s.matchDate(t) && t.Before(t100) {
		t = t.AddDate(0, 0, 1)
	}

	// Check that the date truly matches.
	if !s.mins.has(t.Minute()) || !s.hours.has(t.Hour()) || !s.matchDate(t) {
		return time.Time{}
	}
	return t
}

func (s Schedule) matchDate(t time.Time) bool {
	return s.days.has(t.Day()) && s.months.has(int(t.Month())) && s.weekDays.has(int(t.Weekday()))
}

func (s Schedule) String() string {
	return s.str
}

// A Cron holds a channel that delivers events based on the cron schedule.
type Cron struct {
	C <-chan time.Time // The channel on which events are delivered

	cancel context.CancelFunc
}

// NewCron returns a new Cron containing a channel that sends the time
// at every moment specified by the Schedule.
// The timezone the cron job is operating in must be specified.
// Stop Cron to release associated resources.
func NewCron(sch Schedule, tz *time.Location) *Cron {
	if tz == nil {
		panic("cron: unspecified time.Location; consider using time.Local")
	}
	ch := make(chan time.Time, 1)
	ctx, cancel := context.WithCancel(context.Background())

	// Start monitor goroutine.
	go func() {
		timer := time.NewTimer(0)
		<-timer.C
		for {
			// Schedule the next firing.
			now := time.Now().In(tz)
			next := sch.NextAfter(now)
			if next.IsZero() {
				return
			}
			timer.Reset(next.Sub(now))

			// Wait until either stopped or triggered.
			select {
			case <-ctx.Done():
				timer.Stop()
				return
			case t := <-timer.C:
				// Best-effort at forwarding the signal.
				select {
				case ch <- t:
				default:
				}
			}
		}
	}()
	return &Cron{C: ch, cancel: cancel}
}

// Stop turns off the cron job. After Stop, no more events will be sent.
// Stop does not close the channel, to prevent a read from the channel
// succeeding incorrectly.
func (c *Cron) Stop() {
	c.cancel()
}
