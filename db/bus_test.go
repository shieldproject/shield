package db

import (
	"encoding/json"
	"fmt"

	//"database/sql"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	// sql drivers
	_ "github.com/mattn/go-sqlite3"

	"github.com/shieldproject/shield/core/bus"
)

var _ = Describe("MessageBus Database Integration", func() {
	var db *DB

	BeforeEach(func() {
		var err error
		db, err = Database()
		Ω(err).ShouldNot(HaveOccurred())
		Ω(db).ShouldNot(BeNil())
		fmt.Printf("db: %v\n", db)
	})

	Context("datauuid / datatype detection", func() {
		check := func(typ, id string, thing, ptr interface{}) {
			It(fmt.Sprintf("should understand how to generate a datatype for %s objects", typ), func() {
				Ω(datatype(thing)).Should(Equal(typ))
			})
			It(fmt.Sprintf("should understand how to generate a datauuid for %s objects", typ), func() {
				Ω(datauuid(thing)).Should(Equal(fmt.Sprintf("%s [%s]", typ, id)))
			})
			It(fmt.Sprintf("should understand how to generate a datatype for %s pointer objects", typ), func() {
				Ω(datatype(ptr)).Should(Equal(typ))
			})
			It(fmt.Sprintf("should understand how to generate a datauuid for %s pointer objects", typ), func() {
				Ω(datauuid(ptr)).Should(Equal(fmt.Sprintf("%s [%s]", typ, id)))
			})
		}
		check("agent", "foo", Agent{UUID: "foo"}, &Agent{UUID: "foo"})
		check("job", "foo", Job{UUID: "foo"}, &Job{UUID: "foo"})
		check("store", "foo", Store{UUID: "foo"}, &Store{UUID: "foo"})
		check("target", "foo", Target{UUID: "foo"}, &Target{UUID: "foo"})
		check("tenant", "foo", Tenant{UUID: "foo"}, &Tenant{UUID: "foo"})
		check("task", "foo", Task{UUID: "foo"}, &Task{UUID: "foo"})
		check("archive", "foo", Archive{UUID: "foo"}, &Archive{UUID: "foo"})
	})

	Context("with a (local) messagebus", func() {
		var events chan bus.Event

		BeforeEach(func() {
			db.bus = bus.New(1, 1) /* 1 slot, 1 backlog; highly concurrent :) */
			Ω(db.bus).ShouldNot(BeNil())

			var err error
			events, _, err = db.bus.Register([]string{"*"})
			Ω(err).ShouldNot(HaveOccurred())
			Ω(events).ShouldNot(BeNil())
		})

		receive := func(e bus.Event, event, typ string, out interface{}) {
			Ω(e).ShouldNot(BeNil())
			Ω(e.Event).Should(Equal(event))
			Ω(e.Type).Should(Equal(typ))

			b, err := json.Marshal(e.Data)
			Ω(err).ShouldNot(HaveOccurred())

			/* NOTE:
			   some mbus attributes will NOT make the cut
			   from mbus -> JSON string -> data object
			   because they have nil JSON tags in the go
			   structure.  See Job.TenantUUID for example. */
			err = json.Unmarshal(b, &out)
			Ω(err).ShouldNot(HaveOccurred())
		}

		Context("when sending a createObject message", func() {
			BeforeEach(func() {
				db.sendCreateObjectEvent(Job{
					UUID:    "foo",
					Name:    "daily",
					Summary: "A Daily Backup",
				}, "*")
			})

			It("should receive a create-object message bus event, eventually", func(done Done) {
				/* this is executed in a goroutine */
				var j Job
				receive(<-events, "create-object", "job", &j)
				Ω(j.UUID).Should(Equal("foo"))
				Ω(j.Name).Should(Equal("daily"))
				Ω(j.Summary).Should(Equal("A Daily Backup"))
				/* etc. */

				close(done)
			}, 2 /* timeout (in seconds) */)
		})

		Context("when sending an updateObject message", func() {
			BeforeEach(func() {
				db.sendUpdateObjectEvent(Job{UUID: "foo", Name: "weekly"}, "*")
			})

			It("should receive a update-object message bus event, eventually", func(done Done) {
				/* this is executed in a goroutine */
				var j Job
				receive(<-events, "update-object", "job", &j)
				Ω(j.UUID).Should(Equal("foo"))
				Ω(j.Name).Should(Equal("weekly"))
				/* etc. */

				close(done)
			}, 2 /* timeout (in seconds) */)
		})
	})
})
