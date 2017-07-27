package task_test

import (
	"time"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	boshuit "github.com/cloudfoundry/bosh-cli/ui/task"
)

var _ = Describe("Event", func() {
	Describe("sameness", func() {
		var e1, e2 boshuit.Event

		BeforeEach(func() {
			e1 = boshuit.Event{Stage: "stage", Task: "task", Tags: []string{"tag1", "tag2"}}
			e2 = boshuit.Event{Stage: "stage", Task: "task", Tags: []string{"tag1", "tag2"}}
		})

		Describe("IsSame", func() {
			It("returns true if stage, tags, and task are same", func() {
				Expect(e1.IsSame(e2)).To(BeTrue())
			})

			It("returns false if stage is different", func() {
				e2.Stage = "stage2"
				Expect(e1.IsSame(e2)).To(BeFalse())
			})

			It("returns false if tags are different", func() {
				e2.Tags = []string{"tag1"}
				Expect(e1.IsSame(e2)).To(BeFalse())
			})

			It("returns false if stage, and tags are same but task is different", func() {
				e2.Task = "barf"
				Expect(e1.IsSame(e2)).To(BeFalse())
			})

			It("returns true if stage is same and tags are empty", func() {
				e1.Tags = []string{}
				e2.Tags = []string{}
				Expect(e1.IsSame(e2)).To(BeTrue())
			})

			It("returns false if one is an error and one is not", func() {
				e2 := boshuit.Event{Error: &boshuit.EventError{Code: 42, Message: "nope nope nope"}}
				Expect(e1.IsSame(e2)).To(BeFalse())
			})
		})

		Describe("IsSameGroup", func() {
			It("returns false if both stage are empty", func() {
				e1.Stage = ""
				e2.Stage = ""
				Expect(e1.IsSame(e2)).To(BeFalse())
			})

			It("returns true if stage and tags are same", func() {
				Expect(e1.IsSameGroup(e2)).To(BeTrue())
			})

			It("returns false if stage is different", func() {
				e2.Stage = "Offstage"
				Expect(e1.IsSameGroup(e2)).To(BeFalse())
			})

			It("returns false if stage is same but tags are different", func() {
				e2.Tags = []string{"noththesamevalue"}
				Expect(e1.IsSameGroup(e2)).To(BeFalse())
			})

			It("returns true if stage is same and tags are empty", func() {
				e1.Tags = []string{}
				e2.Tags = []string{}
				Expect(e1.IsSameGroup(e2)).To(BeTrue())
			})
		})
	})

	Describe("Time", func() {
		It("returns formatted time string", func() {
			e := boshuit.Event{UnixTime: 3793593658}
			Expect(e.Time()).To(Equal(time.Date(2090, time.March, 19, 8, 0, 58, 0, time.UTC)))
		})
	})

	Describe("TimeAsStr", func() {
		It("returns formatted time string", func() {
			e := boshuit.Event{UnixTime: 3793593658}
			Expect(e.TimeAsStr()).To(Equal("Sun Mar 19 08:00:58 UTC 2090"))
		})
	})

	Describe("TimeAsHoursStr", func() {
		It("returns formatted hours string", func() {
			e := boshuit.Event{UnixTime: 100}
			Expect(e.TimeAsHoursStr()).To(Equal("00:01:40"))
		})
	})

	Describe("DurationAsStr", func() {
		It("returns formatted duration since given event's time", func() {
			start := boshuit.Event{UnixTime: 100}
			end := boshuit.Event{UnixTime: 200, StartEvent: &start}
			Expect(start.DurationAsStr(end)).To(Equal("00:01:40"))
		})
	})

	Describe("DurationSinceStartAsStr", func() {
		It("returns empty string if does not have a start", func() {
			Expect(boshuit.Event{}.DurationSinceStartAsStr()).To(Equal(""))
		})

		It("returns formatted duration since the start event's time", func() {
			start := boshuit.Event{UnixTime: 100}
			end := boshuit.Event{UnixTime: 200, StartEvent: &start}
			Expect(end.DurationSinceStartAsStr()).To(Equal("00:01:40"))
		})
	})

	Describe("IsWorthKeeping", func() {
		It("returns true if event is not a progress event", func() {
			Expect(boshuit.Event{State: boshuit.EventStateStarted}.IsWorthKeeping()).To(BeTrue())
			Expect(boshuit.Event{State: boshuit.EventStateInProgress}.IsWorthKeeping()).To(BeFalse())
		})
	})
})
