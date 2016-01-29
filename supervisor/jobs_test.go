package supervisor_test

import (
	"time"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	. "github.com/starkandwayne/shield/supervisor"
	"github.com/starkandwayne/shield/timespec"
)

var _ = Describe("Jobs", func() {
	Describe("Runnable()", func() {
		runnable := func(paused bool, offset int) bool {
			job := &Job{
				Paused:  paused,
				NextRun: time.Now().Add(time.Duration(offset) * time.Minute),
			}
			return job.Runnable()
		}

		Context("with unpaused jobs", func() {
			It("should see unpaused jobs with next run in the past as runnable", func() {
				Ω(runnable(false, -10)).Should(BeTrue())
			})
			It("should see unpaused jobs with next run right now as runnable", func() {
				Ω(runnable(false, 0)).Should(BeTrue())
			})
			It("should see unpaused jobs with next run in the future as not runnable", func() {
				Ω(runnable(false, 10)).Should(BeFalse())
			})
		})

		Context("with paused jobs", func() {
			It("should see paused jobs with next run in the past as not runnable", func() {
				Ω(runnable(true, -10)).Should(BeFalse())
			})
			It("should see paused jobs with next run right now as not runnable", func() {
				Ω(runnable(true, 0)).Should(BeFalse())
			})
			It("should see paused jobs with next run in the future as not runnable", func() {
				Ω(runnable(true, 10)).Should(BeFalse())
			})
		})
	})

	Describe("Reschedule()", func() {
		It("should set next run to a date in the future", func() {
			job := &Job{
				NextRun: time.Now().Add(-1 * time.Minute),
				Spec: &timespec.Spec{
					Interval:  timespec.Daily,
					TimeOfDay: 60,
				},
			}

			Ω(job.Reschedule()).Should(Succeed())
			Ω(job.NextRun.After(time.Now())).Should(BeTrue())
		})
	})
})
