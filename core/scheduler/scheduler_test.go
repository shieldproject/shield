package scheduler_test

import (
	"sync"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"github.com/shieldproject/shield/core/scheduler"
)

var _ = Describe("Scheduler concurrency", func() {
	Context("Worker thread safety", func() {
		It("handles concurrent Reserve/Release and Available calls without races", func() {
			// The -race detector catches unsynchronised access across goroutines.
			d, _, err := setupDB()
			Expect(err).NotTo(HaveOccurred())
			defer d.Disconnect()

			worker := scheduler.NewWorker(d)

			const goroutines = 50
			var wg sync.WaitGroup
			wg.Add(goroutines)

			for i := 0; i < goroutines; i++ {
				go func(n int) {
					defer wg.Done()
					worker.Reserve("task-concurrent")
					_ = worker.Available()
					worker.Release()
				}(i)
			}

			wg.Wait()
		})

		It("Reserve makes worker unavailable atomically", func() {
			d, _, err := setupDB()
			Expect(err).NotTo(HaveOccurred())
			defer d.Disconnect()

			worker := scheduler.NewWorker(d)

			Expect(worker.Available()).To(BeTrue(),
				"new worker should be available")

			worker.Reserve("task-1")
			Expect(worker.Available()).To(BeFalse(),
				"worker should be unavailable after Reserve")

			worker.Release()
			Expect(worker.Available()).To(BeTrue(),
				"worker should be available again after Release")
		})
	})
})
