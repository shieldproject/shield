// Copyright 2017, Joe Tsai. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE.md file.

package cron

import (
	"testing"
	"time"
)

func TestSchedule(t *testing.T) {
	tests := []struct {
		schedule string
		events   [][2]time.Time
		wantFail bool
	}{{
		schedule: "daily",
		wantFail: true,
	}, {
		schedule: " 1-1,1,1,1,1,1    1,2,3,4,5,6,7,8,9,10,11,12        13   1-12,JAN-DEC  0-6,MON-FRI    ",
	}, {
		schedule: "* * * * 7", // 7 does not represent SUN
		wantFail: true,
	}, {
		schedule: "423432432 * * * 0",
		wantFail: true,
	}, {
		schedule: "* * * * * *",
		wantFail: true,
	}, {
		schedule: "* * * * *", // Every minute
		events: [][2]time.Time{
			{time.Date(2000, 1, 1, 23, 59, 59, 999999999, time.UTC), time.Date(2000, 1, 2, 0, 0, 0, 0, time.UTC)},
			{time.Date(2000, 1, 2, 0, 0, 0, 0, time.UTC), time.Date(2000, 1, 2, 0, 1, 0, 0, time.UTC)},
			{time.Date(2000, 1, 2, 0, 1, 0, 0, time.UTC), time.Date(2000, 1, 2, 0, 2, 0, 0, time.UTC)},
			{time.Date(2000, 1, 1, 23, 59, 59, 999999999, time.Local), time.Date(2000, 1, 2, 0, 0, 0, 0, time.Local)},
			{time.Date(2000, 1, 2, 0, 0, 0, 0, time.Local), time.Date(2000, 1, 2, 0, 1, 0, 0, time.Local)},
			{time.Date(2000, 1, 2, 0, 1, 0, 0, time.Local), time.Date(2000, 1, 2, 0, 2, 0, 0, time.Local)},
		},
	}, {
		schedule: "@daily",
		events: [][2]time.Time{
			{time.Date(2000, 1, 1, 23, 59, 59, 999999999, time.UTC), time.Date(2000, 1, 2, 0, 0, 0, 0, time.UTC)},
			{time.Date(2000, 1, 2, 0, 0, 0, 0, time.UTC), time.Date(2000, 1, 3, 0, 0, 0, 0, time.UTC)},
			{time.Date(2000, 1, 1, 23, 59, 59, 999999999, time.Local), time.Date(2000, 1, 2, 0, 0, 0, 0, time.Local)},
			{time.Date(2000, 1, 2, 0, 0, 0, 0, time.Local), time.Date(2000, 1, 3, 0, 0, 0, 0, time.Local)},
		},
	}, {
		schedule: "@weekly",
		events: [][2]time.Time{
			{time.Date(2000, 1, 1, 23, 59, 59, 999999999, time.UTC), time.Date(2000, 1, 2, 0, 0, 0, 0, time.UTC)},
			{time.Date(2000, 1, 2, 0, 0, 0, 0, time.UTC), time.Date(2000, 1, 9, 0, 0, 0, 0, time.UTC)},
			{time.Date(2000, 1, 1, 23, 59, 59, 999999999, time.Local), time.Date(2000, 1, 2, 0, 0, 0, 0, time.Local)},
			{time.Date(2000, 1, 2, 0, 0, 0, 0, time.Local), time.Date(2000, 1, 9, 0, 0, 0, 0, time.Local)},
		},
	}, {
		schedule: "0,15,30,45 * * * *", // Every 15 minutes
		events: [][2]time.Time{
			{time.Date(2000, 1, 1, 23, 59, 59, 999999999, time.UTC), time.Date(2000, 1, 2, 0, 0, 0, 0, time.UTC)},
			{time.Date(2000, 1, 2, 0, 0, 0, 0, time.UTC), time.Date(2000, 1, 2, 0, 15, 0, 0, time.UTC)},
			{time.Date(2000, 1, 2, 0, 15, 0, 0, time.UTC), time.Date(2000, 1, 2, 0, 30, 0, 0, time.UTC)},
			{time.Date(2000, 1, 2, 0, 30, 0, 0, time.UTC), time.Date(2000, 1, 2, 0, 45, 0, 0, time.UTC)},
			{time.Date(2000, 1, 2, 0, 45, 0, 0, time.UTC), time.Date(2000, 1, 2, 1, 0, 0, 0, time.UTC)},
			{time.Date(2000, 1, 1, 23, 59, 59, 999999999, time.Local), time.Date(2000, 1, 2, 0, 0, 0, 0, time.Local)},
			{time.Date(2000, 1, 2, 0, 0, 0, 0, time.Local), time.Date(2000, 1, 2, 0, 15, 0, 0, time.Local)},
			{time.Date(2000, 1, 2, 0, 15, 0, 0, time.Local), time.Date(2000, 1, 2, 0, 30, 0, 0, time.Local)},
			{time.Date(2000, 1, 2, 0, 30, 0, 0, time.Local), time.Date(2000, 1, 2, 0, 45, 0, 0, time.Local)},
			{time.Date(2000, 1, 2, 0, 45, 0, 0, time.Local), time.Date(2000, 1, 2, 1, 0, 0, 0, time.Local)},
		},
	}, {
		schedule: "30 11-13 * MAR *", // Every 11th, 12th, and 13th hour in March
		events: [][2]time.Time{
			{time.Date(2000, 1, 1, 23, 59, 59, 999999999, time.UTC), time.Date(2000, 3, 1, 11, 30, 0, 0, time.UTC)},
			{time.Date(2000, 3, 1, 11, 30, 0, 0, time.UTC), time.Date(2000, 3, 1, 12, 30, 0, 0, time.UTC)},
			{time.Date(2000, 3, 1, 12, 30, 0, 0, time.UTC), time.Date(2000, 3, 1, 13, 30, 0, 0, time.UTC)},
			{time.Date(2000, 3, 1, 13, 30, 0, 0, time.UTC), time.Date(2000, 3, 2, 11, 30, 0, 0, time.UTC)},
			{time.Date(2000, 1, 1, 23, 59, 59, 999999999, time.Local), time.Date(2000, 3, 1, 11, 30, 0, 0, time.Local)},
			{time.Date(2000, 3, 1, 11, 30, 0, 0, time.Local), time.Date(2000, 3, 1, 12, 30, 0, 0, time.Local)},
			{time.Date(2000, 3, 1, 12, 30, 0, 0, time.Local), time.Date(2000, 3, 1, 13, 30, 0, 0, time.Local)},
			{time.Date(2000, 3, 1, 13, 30, 0, 0, time.Local), time.Date(2000, 3, 2, 11, 30, 0, 0, time.Local)},
		},
	}, {
		schedule: "0 0 * * MON-FRI", // Every work day at midnight
		events: [][2]time.Time{
			{time.Date(2000, 1, 1, 23, 59, 59, 999999999, time.UTC), time.Date(2000, 1, 3, 0, 0, 0, 0, time.UTC)},
			{time.Date(2000, 1, 3, 23, 59, 59, 999999999, time.UTC), time.Date(2000, 1, 4, 0, 0, 0, 0, time.UTC)},
			{time.Date(2000, 1, 4, 23, 59, 59, 999999999, time.UTC), time.Date(2000, 1, 5, 0, 0, 0, 0, time.UTC)},
			{time.Date(2000, 1, 5, 23, 59, 59, 999999999, time.UTC), time.Date(2000, 1, 6, 0, 0, 0, 0, time.UTC)},
			{time.Date(2000, 1, 6, 23, 59, 59, 999999999, time.UTC), time.Date(2000, 1, 7, 0, 0, 0, 0, time.UTC)},
			{time.Date(2000, 1, 7, 23, 59, 59, 999999999, time.UTC), time.Date(2000, 1, 10, 0, 0, 0, 0, time.UTC)},
			{time.Date(2000, 1, 1, 23, 59, 59, 999999999, time.Local), time.Date(2000, 1, 3, 0, 0, 0, 0, time.Local)},
			{time.Date(2000, 1, 3, 23, 59, 59, 999999999, time.Local), time.Date(2000, 1, 4, 0, 0, 0, 0, time.Local)},
			{time.Date(2000, 1, 4, 23, 59, 59, 999999999, time.Local), time.Date(2000, 1, 5, 0, 0, 0, 0, time.Local)},
			{time.Date(2000, 1, 5, 23, 59, 59, 999999999, time.Local), time.Date(2000, 1, 6, 0, 0, 0, 0, time.Local)},
			{time.Date(2000, 1, 6, 23, 59, 59, 999999999, time.Local), time.Date(2000, 1, 7, 0, 0, 0, 0, time.Local)},
			{time.Date(2000, 1, 7, 23, 59, 59, 999999999, time.Local), time.Date(2000, 1, 10, 0, 0, 0, 0, time.Local)},
		},
	}, {
		schedule: "0 0 29 2 *", // Feb 29th is only on leap year
		events: [][2]time.Time{
			{time.Date(2000, 1, 1, 0, 0, 0, 0, time.UTC), time.Date(2000, 2, 29, 0, 0, 0, 0, time.UTC)},
			{time.Date(2000, 2, 29, 0, 0, 0, 0, time.UTC), time.Date(2004, 2, 29, 0, 0, 0, 0, time.UTC)},
			{time.Date(2000, 1, 1, 0, 0, 0, 0, time.Local), time.Date(2000, 2, 29, 0, 0, 0, 0, time.Local)},
			{time.Date(2000, 2, 29, 0, 0, 0, 0, time.Local), time.Date(2004, 2, 29, 0, 0, 0, 0, time.Local)},
		},
	}, {
		schedule: "0 0 30,31 2 *", // Feb 30th or 31st does not exist
		events: [][2]time.Time{
			{time.Now(), time.Time{}},
		},
	}}

	for _, tt := range tests {
		s, err := ParseSchedule(tt.schedule)
		if gotFail := err != nil; gotFail != tt.wantFail {
			if gotFail {
				t.Errorf("ParseSchedule(%s) failure, want success", tt.schedule)
			} else {
				t.Errorf("ParseSchedule(%s) success, want failure", tt.schedule)
			}
			continue
		}
		for _, tc := range tt.events {
			in, want := tc[0], tc[1]
			if got := s.NextAfter(in); !got.Equal(want) {
				t.Errorf("ParseSchedule(%v).NextAfter(%v):\ngot  %v\nwant %v", s, in, got, want)
			}
		}
	}
}
