package scheduler_test

import (
	"sync/atomic"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	// sqlite3 driver required for in-memory test DB
	_ "github.com/mattn/go-sqlite3"

	"github.com/shieldproject/shield/core/scheduler"
	"github.com/shieldproject/shield/db"
)

// setupDB creates a minimal in-memory database with a tenant fixture.
func setupDB() (*db.DB, string, error) {
	tenantUUID := db.RandomID()
	d, err := db.Connect(":memory:")
	if err != nil {
		return nil, "", err
	}
	if _, err := d.Setup(0); err != nil {
		d.Disconnect()
		return nil, "", err
	}
	if err := d.Exec(
		`INSERT INTO tenants (uuid, name) VALUES (?, ?)`,
		tenantUUID, "test-tenant",
	); err != nil {
		d.Disconnect()
		return nil, "", err
	}
	return d, tenantUUID, nil
}

var _ = Describe("Worker.Execute retry logic", func() {
	// Describe the loop condition logic in isolation.
	//
	// The bug at chore.go:146:
	//   for i := 0; i < retries || rc == 0; i++ { ... }
	//
	// When rc==0 (success exit code), "rc == 0" is always true,
	// making the loop condition i < retries || true = always true => infinite.
	//
	// Fix: for i := 0; i < retries && rc != 0; i++ { ... }
	Describe("retry loop condition", func() {
		// loopIterations runs the loop condition until it terminates or maxIter
		// is reached, returning the number of iterations executed.
		loopIterations := func(condition func(i, retries, rc int) bool, retries, rc, maxIter int) int {
			count := 0
			for i := 0; i < maxIter; i++ {
				if !condition(i, retries, rc) {
					break
				}
				count++
			}
			return count
		}

		buggyCondition := func(i, retries, rc int) bool { return i < retries || rc == 0 }
		fixedCondition := func(i, retries, rc int) bool { return i < retries && rc != 0 }

		Context("buggy OR condition (i < retries || rc == 0)", func() {
			It("never terminates when rc=0 regardless of retries count", func() {
				// This proves the bug: rc=0 makes the condition always true.
				iters := loopIterations(buggyCondition, 3, 0, 10000)
				Expect(iters).To(Equal(10000),
					"buggy condition runs forever when rc=0")
			})

			It("terminates correctly when rc!=0", func() {
				iters := loopIterations(buggyCondition, 3, 1, 10000)
				Expect(iters).To(Equal(3))
			})
		})

		Context("fixed AND condition (i < retries && rc != 0)", func() {
			It("does not iterate when rc=0 (success — no retry needed)", func() {
				iters := loopIterations(fixedCondition, 3, 0, 10000)
				Expect(iters).To(Equal(0))
			})

			It("retries exactly the configured number of times when rc!=0", func() {
				iters := loopIterations(fixedCondition, 3, 1, 10000)
				Expect(iters).To(Equal(3))
			})
		})
	})

	// Test Worker.Execute directly: when a chore exits with rc=0, Execute must
	// complete promptly.  With the buggy || condition the retry loop is infinite
	// when the outer if rc != 0 guard is bypassed by the data race; with the
	// correct && condition no infinite loop is possible for rc=0.
	Describe("Worker.Execute with successful chore (rc=0)", func() {
		It("completes without hanging", func() {
			d, tenantUUID, err := setupDB()
			Expect(err).NotTo(HaveOccurred())
			defer d.Disconnect()

			task, err := d.CreateInternalTask("test-owner", db.RestoreOperation, tenantUUID)
			Expect(err).NotTo(HaveOccurred())
			Expect(task).NotTo(BeNil())

			var callCount int32
			chore := scheduler.NewChore(task.UUID, func(c scheduler.Chore) {
				atomic.AddInt32(&callCount, 1)
				c.UnixExit(0)
			})

			worker := scheduler.NewWorker(d)

			done := make(chan struct{})
			go func() {
				defer close(done)
				worker.Execute(chore)
			}()

			select {
			case <-done:
				Expect(atomic.LoadInt32(&callCount)).To(Equal(int32(1)),
					"chore Do function must be called exactly once on success")
			case <-time.After(5 * time.Second):
				Fail("Worker.Execute did not complete within 5 seconds — infinite loop detected")
			}
		})
	})
})
