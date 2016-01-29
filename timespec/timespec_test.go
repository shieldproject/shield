package timespec_test

import (
	"time"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	. "github.com/starkandwayne/shield/timespec"
)

var _ = Describe("Timespec", func() {
	inMinutes := func(hours uint, minutes uint) int {
		return int(hours*60 + minutes)
	}

	Describe("Stringification", func() {
		It("can stringify hourly specs", func() {
			spec := &Spec{
				Interval:   Hourly,
				TimeOfHour: 45,
			}

			Ω(spec.String()).Should(Equal("hourly at 45 after"))
		})

		It("can stringify daily specs", func() {
			spec := &Spec{
				Interval:  Daily,
				TimeOfDay: inMinutes(16, 00),
			}

			Ω(spec.String()).Should(Equal("daily at 16:00"))
		})

		It("can stringify weekly specs", func() {
			spec := &Spec{
				Interval:  Weekly,
				DayOfWeek: time.Thursday,
				TimeOfDay: inMinutes(23, 35),
			}

			Ω(spec.String()).Should(Equal("thursdays at 23:35"))
		})

		It("can stringify monthly (nth week) specs", func() {
			spec := &Spec{
				Interval:  Monthly,
				DayOfWeek: time.Tuesday,
				Week:      3,
				TimeOfDay: inMinutes(02, 05),
			}

			Ω(spec.String()).Should(Equal("3rd tuesday at 2:05"))
		})

		It("can stringify monthly (day of month) specs", func() {
			spec := &Spec{
				Interval:   Monthly,
				DayOfMonth: 14,
				TimeOfDay:  inMinutes(02, 05),
			}

			Ω(spec.String()).Should(Equal("monthly at 2:05 on 14th"))
		})
	})

	Describe("Determining the next timestamp from a spec object", func() {
		// August 6th, 1991, at about 11:15 in the morning: start up Internet thing
		tz := time.Now().Location()
		now := time.Date(1991, 8, 6, 11, 15, 42, 100203, tz)

		Context("with an hourly interval", func() {
			It("can handle hourly at 30 after", func() {
				spec := &Spec{
					Interval:   Hourly,
					TimeOfHour: 30,
				}

				Ω(spec.Next(now)).Should(Equal(
					time.Date(1991, 8, 6, 11, 30, 00, 00, tz)))
			})

			It("can handle hourly at 7 after (before now)", func() {
				spec := &Spec{
					Interval:   Hourly,
					TimeOfHour: 7,
				}

				Ω(spec.Next(now)).Should(Equal(
					time.Date(1991, 8, 6, 12, 7, 00, 00, tz)))
			})

			It("throws errors for minute offsets that are inconceivable large", func() {
				spec := &Spec{
					Interval:   Hourly,
					TimeOfHour: 900,
				}

				_, err := spec.Next(now)
				Ω(err).Should(HaveOccurred())
			})
		})

		Context("with a daily interval", func() {
			It("can handle daily at 4pm", func() {
				spec := &Spec{
					Interval:  Daily,
					TimeOfDay: inMinutes(16, 00),
				}

				Ω(spec.Next(now)).Should(Equal(
					time.Date(1991, 8, 6, 16, 00, 00, 00, tz)))
			})

			It("does not affect the original time.Time passed in", func() {
				spec := &Spec{
					Interval:  Daily,
					TimeOfDay: inMinutes(16, 00),
				}

				spec.Next(now)
				Ω(now.Second()).Should(Equal(42))
			})

			It("can handle daily at 4:30pm", func() {
				spec := &Spec{
					Interval:  Daily,
					TimeOfDay: inMinutes(16, 30),
				}

				Ω(spec.Next(now)).Should(Equal(
					time.Date(1991, 8, 6, 16, 30, 00, 00, tz)))
			})

			It("can handle daily at 6:07am (next is tomorrow)", func() {
				spec := &Spec{
					Interval:  Daily,
					TimeOfDay: inMinutes(6, 07),
				}

				Ω(spec.Next(now)).Should(Equal(
					time.Date(1991, 8, 6+1, 6, 07, 00, 00, tz)))
			})

			It("can handle daily at 11:15am (next is tomorrow)", func() {
				spec := &Spec{
					Interval:  Daily,
					TimeOfDay: inMinutes(11, 15),
				}

				Ω(spec.Next(now)).Should(Equal(
					time.Date(1991, 8, 6+1, 11, 15, 00, 00, tz)))
			})

			It("can handle daily at 11:16am (next is today)", func() {
				spec := &Spec{
					Interval:  Daily,
					TimeOfDay: inMinutes(11, 16),
				}

				Ω(spec.Next(now)).Should(Equal(
					time.Date(1991, 8, 6, 11, 16, 00, 00, tz)))
			})
		})

		Context("with a weekly interval", func() {
			/*
			   Remember: Aug 6, 1991 was a Tuesday.
			   The best things start on Tuesdays...

			          August 1991
			      Su Mo Tu We Th Fr Sa
			                   1  2  3
			       4  5  6* 7  8  9 10
			      11 12 13 14 15 16 17
			      18 19 20 21 22 23 24
			      25 26 27 28 29 30 31

			*/

			It("handles the next timestamp being later in the week", func() {
				spec := &Spec{
					Interval:  Weekly,
					DayOfWeek: time.Saturday,
					TimeOfDay: inMinutes(7, 00),
				}

				Ω(spec.Next(now)).Should(Equal(
					time.Date(1991, 8, 10, 7, 00, 00, 00, tz)))
			})

			It("handles the next timestamp being early next week", func() {
				spec := &Spec{
					Interval:  Weekly,
					DayOfWeek: time.Monday,
					TimeOfDay: inMinutes(19, 45),
				}

				Ω(spec.Next(now)).Should(Equal(
					time.Date(1991, 8, 12, 19, 45, 00, 00, tz)))
			})

			It("handles the next timestamp being later that day", func() {
				spec := &Spec{
					Interval:  Weekly,
					DayOfWeek: time.Tuesday,
					TimeOfDay: inMinutes(23, 55),
				}

				Ω(spec.Next(now)).Should(Equal(
					time.Date(1991, 8, 6, 23, 55, 00, 00, tz)))
			})
		})

		Context("with a monthly (nth week) spec", func() {
			/*
			   Fall of '91 was a wonderful time, I recall it fondly.

			             August              September
			      Su Mo Tu We Th Fr Sa  Su Mo Tu We Th Fr Sa
			                   1  2  3   1  2  3  4  5  6  7
			       4  5  6* 7  8  9 10   8  9 10 11 12 13 14
			      11 12 13 14 15 16 17  15 16 17 18 19 20 21
			      18 19 20 21 22 23 24  22 23 24 25 26 27 28
			      25 26 27 28 29 30 31  29 30

			*/

			It("handles the next timestamp being later in the same month", func() {
				spec := &Spec{
					Interval:  Monthly,
					DayOfWeek: time.Thursday,
					Week:      3,
					TimeOfDay: inMinutes(1, 00),
				}

				Ω(spec.Next(now)).Should(Equal(
					time.Date(1991, 8, 15, 1, 00, 00, 00, tz)))
			})

			It("handles the next timestamp being early in the next month", func() {
				spec := &Spec{
					Interval:  Monthly,
					DayOfWeek: time.Monday,
					Week:      1,
					TimeOfDay: inMinutes(2, 30),
				}

				Ω(spec.Next(now)).Should(Equal(
					time.Date(1991, 9, 2, 2, 30, 00, 00, tz)))
			})

			It("handles the next timestamp being later in the same day", func() {
				spec := &Spec{
					Interval:  Monthly,
					DayOfWeek: time.Tuesday,
					Week:      1,
					TimeOfDay: inMinutes(14, 10),
				}

				Ω(spec.Next(now)).Should(Equal(
					time.Date(1991, 8, 6, 14, 10, 00, 00, tz)))
			})

			It("handles having just missed this month", func() {
				spec := &Spec{
					Interval:  Monthly,
					DayOfWeek: time.Tuesday,
					Week:      1,
					TimeOfDay: inMinutes(10, 05),
				}

				Ω(spec.Next(now)).Should(Equal(
					time.Date(1991, 9, 3, 10, 05, 00, 00, tz)))
			})

			It("throws errors for week offsets that are inconceivable large", func() {
				spec := &Spec{
					Interval:  Monthly,
					DayOfWeek: time.Tuesday,
					Week:      6,
					TimeOfDay: inMinutes(17, 55),
				}

				_, err := spec.Next(now)
				Ω(err).Should(HaveOccurred())
			})

			It("throws errors for week offsets that are negative", func() {
				spec := &Spec{
					Interval:  Monthly,
					DayOfWeek: time.Tuesday,
					Week:      -1,
					TimeOfDay: inMinutes(17, 55),
				}

				_, err := spec.Next(now)
				Ω(err).Should(HaveOccurred())
			})
		})

		Context("with a monthly (day of month) spec", func() {
			/*
			   Fall of '91, the leaves were so beautiful...

			             August              September
			      Su Mo Tu We Th Fr Sa  Su Mo Tu We Th Fr Sa
			                   1  2  3   1  2  3  4  5  6  7
			       4  5  6* 7  8  9 10   8  9 10 11 12 13 14
			      11 12 13 14 15 16 17  15 16 17 18 19 20 21
			      18 19 20 21 22 23 24  22 23 24 25 26 27 28
			      25 26 27 28 29 30 31  29 30

			*/

			It("handles the next timestamp being later in the same month", func() {
				spec := &Spec{
					Interval:   Monthly,
					DayOfMonth: 15,
					TimeOfDay:  inMinutes(1, 00),
				}

				Ω(spec.Next(now)).Should(Equal(
					time.Date(1991, 8, 15, 1, 00, 00, 00, tz)))
			})

			It("handles the next timestamp being early in the next month", func() {
				spec := &Spec{
					Interval:   Monthly,
					DayOfMonth: 2,
					TimeOfDay:  inMinutes(2, 30),
				}

				Ω(spec.Next(now)).Should(Equal(
					time.Date(1991, 9, 2, 2, 30, 00, 00, tz)))
			})

			It("handles the next timestamp being later in the same day", func() {
				spec := &Spec{
					Interval:   Monthly,
					DayOfMonth: 6,
					TimeOfDay:  inMinutes(14, 10),
				}

				Ω(spec.Next(now)).Should(Equal(
					time.Date(1991, 8, 6, 14, 10, 00, 00, tz)))
			})

			It("handles having just missed this month", func() {
				spec := &Spec{
					Interval:   Monthly,
					DayOfMonth: 6,
					TimeOfDay:  inMinutes(10, 05),
				}

				Ω(spec.Next(now)).Should(Equal(
					time.Date(1991, 9, 6, 10, 05, 00, 00, tz)))
			})

			It("throws errors for month days that are inconceivable large", func() {
				spec := &Spec{
					Interval:   Monthly,
					DayOfWeek:  time.Tuesday,
					DayOfMonth: 32,
					TimeOfDay:  inMinutes(17, 55),
				}

				_, err := spec.Next(now)
				Ω(err).Should(HaveOccurred())
			})

			It("throws errors for month days that are negative", func() {
				spec := &Spec{
					Interval:   Monthly,
					DayOfWeek:  time.Tuesday,
					DayOfMonth: -1,
					TimeOfDay:  inMinutes(17, 55),
				}

				_, err := spec.Next(now)
				Ω(err).Should(HaveOccurred())
			})
		})
	})

	Describe("spec parser", func() {
		Context("for hourly specs", func() {
			specOK := func(spec string, m int) {
				s, err := Parse(spec)
				Ω(err).ShouldNot(HaveOccurred())
				Ω(s).ShouldNot(BeNil())
				Ω(s.Interval).Should(Equal(Hourly))
				Ω(s.TimeOfHour).Should(Equal(m))
			}

			It("handles proper time formats", func() {
				specOK("hourly at 30", 30)
				specOK("hourly at :30", 30)
				specOK("hourly at x:30", 30)
				specOK("hourly at *:30", 30)
				specOK("hourly at h:30", 30)
				specOK("hourly at 15", 15)

				specOK("hourly at quarter after", 15)
				specOK("hourly at quarter til", 45)
				specOK("hourly at 23 after", 23)
				specOK("hourly at 23 past", 23)
				specOK("hourly at 10 til", 50)
				specOK("hourly at 10 until", 50)
				specOK("hourly at half past", 30)
			})

			It("is case insensitive", func() {
				specOK("Hourly at :30", 30)
				specOK("hoUrly At X:30", 30)
				specOK("Every HOUR at H:30", 30)
			})

			It("handles a missing 'at' keyword", func() {
				specOK("hourly 10", 10)
			})

			It("handles the 'every hour ' variant", func() {
				specOK("every hour at h:30", 30)
				specOK("every hour at x:30", 30)
				specOK("every hour at *:30", 30)
				specOK("every hour at 30", 30)
				specOK("every         hour 45", 45)
			})

			It("handles weird and extraneous whitespace", func() {
				specOK("     hourly\t\tat\n\n\n  \r\t\r\t16\t\t", 16)
			})
		})

		Context("for daily specs", func() {
			specOK := func(spec string, h int, m int) {
				s, err := Parse(spec)
				Ω(err).ShouldNot(HaveOccurred())
				Ω(s).ShouldNot(BeNil())
				Ω(s.Interval).Should(Equal(Daily))
				Ω(s.TimeOfDay).Should(Equal(h*60 + m))
			}

			It("handles proper time formats", func() {
				specOK("daily at 2:30", 2, 30)
				specOK("daily at 14:30", 14, 30)
				specOK("daily at 2am", 2, 00)
				specOK("daily at 2:30am", 2, 30)
				specOK("daily at 2:30pm", 14, 30)
				specOK("daily at 14:30am", 14, 30)
				specOK("daily at 14:30pm", 14, 30)
				specOK("daily at 14am", 14, 00)
			})

			It("is case insensitive", func() {
				specOK("Daily at 2:30", 2, 30)
				specOK("daIly At 2:30PM", 14, 30)
				specOK("Every Day at 2:30", 2, 30)
			})

			It("Allows spaces between hour + am/pm", func() {
				specOK("daily at 2:30 pm", 14, 30)
				specOK("daily at 2 pm", 14, 00)
			})

			It("handles a missing 'at' keyword", func() {
				specOK("daily 4am", 4, 00)
			})

			It("handles the 'every day' variant", func() {
				specOK("every day at 2:30", 2, 30)
				specOK("every day at 14:30", 14, 30)
				specOK("every day at 2am", 2, 00)
				specOK("every day at 2:30am", 2, 30)
				specOK("every day at 2:30pm", 14, 30)
				specOK("every day at 14:30am", 14, 30)
				specOK("every day at 14:30pm", 14, 30)
				specOK("every day at 14am", 14, 00)

				specOK("every day 4am", 4, 00)

				specOK("every         day 4am", 4, 00)
			})

			It("handles weird and extraneous whitespace", func() {
				specOK("     daily\t\tat\n\n\n  \r\t\r\t4    pm\t\t", 16, 00)
			})
		})

		Context("for weekly specs", func() {
			specOK := func(spec string, weekday time.Weekday, h int, m int) {
				s, err := Parse(spec)
				Ω(err).ShouldNot(HaveOccurred())
				Ω(s).ShouldNot(BeNil())
				Ω(s.Interval).Should(Equal(Weekly))
				Ω(s.TimeOfDay).Should(Equal(h*60 + m))
				Ω(s.DayOfWeek).Should(Equal(weekday))
			}

			It("handles 'weekly at <time> on <day>' forms", func() {
				specOK("weekly at 5:35pm on sun", time.Sunday, 17, 35)
				specOK("weekly at 5:35pm on sunday", time.Sunday, 17, 35)
				specOK("weekly at 5:35pm on sundays", time.Sunday, 17, 35)

				specOK("weekly at 5:35pm on mon", time.Monday, 17, 35)
				specOK("weekly at 5:35pm on monday", time.Monday, 17, 35)
				specOK("weekly at 5:35pm on mondays", time.Monday, 17, 35)

				specOK("weekly at 5:35pm on tue", time.Tuesday, 17, 35)
				specOK("weekly at 5:35pm on tues", time.Tuesday, 17, 35)
				specOK("weekly at 5:35pm on tuesday", time.Tuesday, 17, 35)
				specOK("weekly at 5:35pm on tuesdays", time.Tuesday, 17, 35)

				specOK("weekly at 5:35pm on wed", time.Wednesday, 17, 35)
				specOK("weekly at 5:35pm on wednesday", time.Wednesday, 17, 35)
				specOK("weekly at 5:35pm on wednesdays", time.Wednesday, 17, 35)

				specOK("weekly at 5:35pm on thu", time.Thursday, 17, 35)
				specOK("weekly at 5:35pm on thur", time.Thursday, 17, 35)
				specOK("weekly at 5:35pm on thurs", time.Thursday, 17, 35)
				specOK("weekly at 5:35pm on thursday", time.Thursday, 17, 35)
				specOK("weekly at 5:35pm on thursdays", time.Thursday, 17, 35)

				specOK("weekly at 5:35pm on fri", time.Friday, 17, 35)
				specOK("weekly at 5:35pm on friday", time.Friday, 17, 35)
				specOK("weekly at 5:35pm on fridays", time.Friday, 17, 35)

				specOK("weekly at 5:35pm on sat", time.Saturday, 17, 35)
				specOK("weekly at 5:35pm on saturday", time.Saturday, 17, 35)
				specOK("weekly at 5:35pm on saturdays", time.Saturday, 17, 35)
			})

			It("handles `<day> at <time>' forms", func() {
				specOK("sun at 5:35pm", time.Sunday, 17, 35)
				specOK("sunday at 5:35pm", time.Sunday, 17, 35)
				specOK("sundays at 5:35pm", time.Sunday, 17, 35)

				specOK("mon at 5:35pm", time.Monday, 17, 35)
				specOK("monday at 5:35pm", time.Monday, 17, 35)
				specOK("mondays at 5:35pm", time.Monday, 17, 35)

				specOK("tue at 5:35pm", time.Tuesday, 17, 35)
				specOK("tues at 5:35pm", time.Tuesday, 17, 35)
				specOK("tuesday at 5:35pm", time.Tuesday, 17, 35)
				specOK("tuesdays at 5:35pm", time.Tuesday, 17, 35)

				specOK("wed at 5:35pm", time.Wednesday, 17, 35)
				specOK("wednesday at 5:35pm", time.Wednesday, 17, 35)
				specOK("wednesdays at 5:35pm", time.Wednesday, 17, 35)

				specOK("thu at 5:35pm", time.Thursday, 17, 35)
				specOK("thur at 5:35pm", time.Thursday, 17, 35)
				specOK("thurs at 5:35pm", time.Thursday, 17, 35)
				specOK("thursday at 5:35pm", time.Thursday, 17, 35)
				specOK("thursdays at 5:35pm", time.Thursday, 17, 35)

				specOK("fri at 5:35pm", time.Friday, 17, 35)
				specOK("friday at 5:35pm", time.Friday, 17, 35)
				specOK("fridays at 5:35pm", time.Friday, 17, 35)

				specOK("sat at 5:35pm", time.Saturday, 17, 35)
				specOK("saturday at 5:35pm", time.Saturday, 17, 35)
				specOK("saturdays at 5:35pm", time.Saturday, 17, 35)
			})

			It("Is case insensitive", func() {
				specOK("Weekly at 2:30 on Sat", time.Saturday, 2, 30)
				specOK("Weekly at 2:30 on Sunday", time.Sunday, 2, 30)
				specOK("Weekly at 2:30 on Mon", time.Monday, 2, 30)
				specOK("Weekly at 2:30 on Tuesdays", time.Tuesday, 2, 30)
				specOK("Weekly at 2:30 on Wed", time.Wednesday, 2, 30)
				specOK("Weekly at 2:30 on thu", time.Thursday, 2, 30)
				specOK("Weekly at 2:30 on Friday", time.Friday, 2, 30)
				specOK("Sat at 2:30", time.Saturday, 2, 30)
				specOK("Sun at 2:30", time.Sunday, 2, 30)
				specOK("Mon at 2:30", time.Monday, 2, 30)
				specOK("Tue at 2:30", time.Tuesday, 2, 30)
				specOK("Wed at 2:30", time.Wednesday, 2, 30)
				specOK("Thu at 2:30", time.Thursday, 2, 30)
				specOK("Fri at 2:30", time.Friday, 2, 30)
				specOK("Saturday at 2:30", time.Saturday, 2, 30)
				specOK("Sunday at 2:30", time.Sunday, 2, 30)
				specOK("Monday at 2:30", time.Monday, 2, 30)
				specOK("TuesDay at 2:30", time.Tuesday, 2, 30)
				specOK("Wednesday at 2:30", time.Wednesday, 2, 30)
				specOK("Thursday at 2:30", time.Thursday, 2, 30)
				specOK("Friday at 2:30", time.Friday, 2, 30)
			})

			It("can skip the  'at' and 'on' keywords", func() {
				specOK("weekly 5:35pm on sat", time.Saturday, 17, 35)
				specOK("weekly at 5:35pm sat", time.Saturday, 17, 35)
				specOK("weekly 5:35pm sat", time.Saturday, 17, 35)

				specOK("thu 5:35pm", time.Thursday, 17, 35)
			})
		})

		Context("for monthly specs (day-of-week flavor)", func() {
			specOK := func(spec string, week int, weekday time.Weekday, h int, m int) {
				s, err := Parse(spec)
				Ω(err).ShouldNot(HaveOccurred())
				Ω(s).ShouldNot(BeNil())
				Ω(s.Interval).Should(Equal(Monthly))
				Ω(s.TimeOfDay).Should(Equal(h*60 + m))
				Ω(s.DayOfWeek).Should(Equal(weekday))
				Ω(s.Week).Should(Equal(week))
				Ω(s.DayOfMonth).Should(Equal(0))
			}

			It("just works", func() {
				specOK("2st tuesday at 23:15", 2, time.Tuesday, 23, 15)
				specOK("2nd tuesday at 23:15", 2, time.Tuesday, 23, 15)
				specOK("2rd tuesday at 23:15", 2, time.Tuesday, 23, 15)
				specOK("2th tuesday at 23:15", 2, time.Tuesday, 23, 15)

				specOK("4th fridays at 8am", 4, time.Friday, 8, 00)
				specOK("4th fridays 8am", 4, time.Friday, 8, 00)
			})
		})

		Context("for monthly specs (day-of-month flavor)", func() {
			specOK := func(spec string, d int, h int, m int) {
				s, err := Parse(spec)
				Ω(err).ShouldNot(HaveOccurred())
				Ω(s).ShouldNot(BeNil())
				Ω(s.Interval).Should(Equal(Monthly))
				Ω(s.TimeOfDay).Should(Equal(h*60 + m))
				Ω(s.Week).Should(Equal(0))
				Ω(s.DayOfMonth).Should(Equal(d))
			}

			It("just works", func() {
				specOK("monthly at 11:01pm on 4th", 4, 23, 01)
				specOK("monthly at 11:01pm on 19st", 19, 23, 01)
				specOK("monthly 11:01pm 19st", 19, 23, 01)
				specOK("monthly at 11:01pm 19st", 19, 23, 01)
				specOK("monthly 11:01pm on 19st", 19, 23, 01)
			})
		})
	})
})
