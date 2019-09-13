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

        /* Message bus tests for Agent */
        Context("when sending a createObject message for Agent", func() {
            BeforeEach(func() {
                /* Create Agent object */
                db.sendCreateObjectEvent(Agent{
                    UUID:     "foo",
                    Name:     "agent",
                    Address:  "addr",
                    Version:  "v1.0",
                    Hidden: false,
                    LastSeenAt: 2,
                    LastCheckedAt: 3,
                    LastError: "nil",
                    Status: "good",
                }, "*")
            })

            It("should receive a create-object message bus event for Agent, eventually", func(done Done) {
                /* this is executed in a goroutine */
                var a Agent

                /* Create agent object bus event*/
                receive(<-events, "create-object", "agent", &a)
                Ω(a.UUID).Should(Equal("foo"))
                Ω(a.Name).Should(Equal("agent"))
                Ω(a.Address).Should(Equal("addr"))
                Ω(a.Version).Should(Equal("v1.0"))
                Ω(a.Hidden).Should(Equal(false))
                Ω(a.LastSeenAt).Should(Equal(int64(2)))
                Ω(a.LastCheckedAt).Should(Equal(int64(3)))
                Ω(a.LastError).Should(Equal("nil"))
                Ω(a.Status).Should(Equal("good"))

                close(done)
            }, 2 /* timeout (in seconds) */)
        })

        Context("when sending an updateObject message for Agent", func() {
            BeforeEach(func() {
                /* Update Agent object*/
                db.sendUpdateObjectEvent(Agent{
                    UUID:     "foo",
                    Name:     "agent",
                    Address:  "addr",
                    Version:  "v1.1",
                    Hidden: true,
                    LastSeenAt: 3,
                    LastCheckedAt: 4,
                    LastError: "nil",
                    Status: "bad",
                }, "*")
            })

            It("should receive a update-object message bus event for Agent, eventually", func(done Done) {
                /* this is executed in a goroutine */
                var a Agent

                /* Update Agent object message bus event*/
                receive(<-events, "update-object", "agent", &a)
                Ω(a.UUID).Should(Equal("foo"))
                Ω(a.Name).Should(Equal("agent"))
                Ω(a.Address).Should(Equal("addr"))
                Ω(a.Version).Should(Equal("v1.1"))
                Ω(a.Hidden).Should(Equal(true))
                Ω(a.LastSeenAt).Should(Equal(int64(3)))
                Ω(a.LastCheckedAt).Should(Equal(int64(4)))
                Ω(a.LastError).Should(Equal("nil"))
                Ω(a.Status).Should(Equal("bad"))

                close(done)
            }, 2 /* timeout (in seconds) */)
        })

        /* Message bus tests for Job */
        Context("when sending a createObject message for Job", func() {
            BeforeEach(func() {
                /* Create Job object */
                db.sendCreateObjectEvent(Job{
                    UUID:     "foo",
                    Name:     "daily",
                    Summary:  "A Daily Backup",
                    KeepN:    2,
                    KeepDays: 2,
                    Schedule: "daily",
                    Paused:   true,
                    FixedKey: false,
                }, "*")
            })

            It("should receive a create-object message bus event for Job, eventually", func(done Done) {
                /* this is executed in a goroutine */
                var j Job

                /* Create job object bus event*/
                receive(<-events, "create-object", "job", &j)
                Ω(j.UUID).Should(Equal("foo"))
                Ω(j.Name).Should(Equal("daily"))
                Ω(j.Summary).Should(Equal("A Daily Backup"))
                Ω(j.KeepN).Should(Equal(2))
                Ω(j.KeepDays).Should(Equal(2))
                Ω(j.Schedule).Should(Equal("daily"))
                Ω(j.Paused).Should(Equal(true))
                Ω(j.FixedKey).Should(Equal(false))

                close(done)
            }, 2 /* timeout (in seconds) */)
        })

        Context("when sending an updateObject message for Job", func() {
            BeforeEach(func() {
                /* Update Job object*/
                db.sendUpdateObjectEvent(Job{
                    UUID:     "foo",
                    Name:     "weekly",
                    Summary:  "A Weekly Backup",
                    KeepN:    3,
                    KeepDays: 3,
                    Schedule: "weekly",
                    Paused:   true,
                    FixedKey: false,
                }, "*")
            })

            It("should receive a update-object message bus event for Job, eventually", func(done Done) {
                /* this is executed in a goroutine */
                var j Job

                /* Update Job object message bus event*/
                receive(<-events, "update-object", "job", &j)
                Ω(j.UUID).Should(Equal("foo"))
                Ω(j.Name).Should(Equal("weekly"))
                Ω(j.Summary).Should(Equal("A Weekly Backup"))
                Ω(j.KeepN).Should(Equal(3))
                Ω(j.KeepDays).Should(Equal(3))
                Ω(j.Schedule).Should(Equal("weekly"))
                Ω(j.Paused).Should(Equal(true))
                Ω(j.FixedKey).Should(Equal(false))
                /* etc. */

                close(done)
            }, 2 /* timeout (in seconds) */)
        })

        /* Message bus tests for Store */
        Context("when sending a createObject message for Store", func() {
            BeforeEach(func() {
                /* Create Store object */
                db.sendCreateObjectEvent(Store{
                    UUID:    "foo",
                    Name:    "Store",
                    Summary: "A Store Plugin",
                    Agent:   "agent",
                    Plugin:  "test plugin",
                    Global:  true,
                    Healthy: true,
                }, "*")
            })

            It("should receive a create-object message bus event for Store, eventually", func(done Done) {
                /* this is executed in a goroutine */
                var s Store

                /* Create store object bus event*/
                receive(<-events, "create-object", "store", &s)
                Ω(s.UUID).Should(Equal("foo"))
                Ω(s.Name).Should(Equal("Store"))
                Ω(s.Summary).Should(Equal("A Store Plugin"))
                Ω(s.Agent).Should(Equal("agent"))
                Ω(s.Plugin).Should(Equal("test plugin"))
                Ω(s.Global).Should(Equal(true))
                Ω(s.Healthy).Should(Equal(true))

                close(done)
            }, 2 /* timeout (in seconds) */)
        })

        Context("when sending an updateObject message for Store", func() {
            BeforeEach(func() {
                /* Update Store object*/
                db.sendUpdateObjectEvent(Store{
                    UUID:    "foo",
                    Name:    "weekly",
                    Summary: "A Store plugin",
                    Agent:   "Agent",
                    Plugin:  "plugin",
                    Global:  false,
                    Healthy: false,
                }, "*")
            })

            It("should receive a update-object message bus event for Store, eventually", func(done Done) {
                /* this is executed in a goroutine */
                var s Store

                /* Update Store object message bus event*/
                receive(<-events, "update-object", "store", &s)
                Ω(s.UUID).Should(Equal("foo"))
                Ω(s.Name).Should(Equal("weekly"))
                Ω(s.Summary).Should(Equal("A Store plugin"))
                Ω(s.Agent).Should(Equal("Agent"))
                Ω(s.Plugin).Should(Equal("plugin"))
                Ω(s.Global).Should(Equal(false))
                Ω(s.Healthy).Should(Equal(false))

                close(done)
            }, 2 /* timeout (in seconds) */)
        })

        /* Message bus tests for Target */
        Context("when sending a createObject message for Target", func() {
            BeforeEach(func() {
                /* Create Target object */
                db.sendCreateObjectEvent(Target{
                    UUID:        "foo",
                    Name:        "target",
                    Summary:     "A Target Plugin",
                    Agent:       "agent",
                    Plugin:      "test plugin",
                    Compression: "zip",
                }, "*")
            })

            It("should receive a create-object message bus event for Target, eventually", func(done Done) {
                /* this is executed in a goroutine */
                var t Target

                /* Create Target object bus event*/
                receive(<-events, "create-object", "target", &t)
                Ω(t.UUID).Should(Equal("foo"))
                Ω(t.Name).Should(Equal("target"))
                Ω(t.Summary).Should(Equal("A Target Plugin"))
                Ω(t.Agent).Should(Equal("agent"))
                Ω(t.Plugin).Should(Equal("test plugin"))
                Ω(t.Compression).Should(Equal("zip"))

                close(done)
            }, 2 /* timeout (in seconds) */)
        })

        Context("when sending an updateObject message for Target", func() {
            BeforeEach(func() {
                /* Update Target object*/
                db.sendUpdateObjectEvent(Target{
                    UUID:        "foo",
                    Name:        "weekly",
                    Summary:     "A Target plugin",
                    Agent:       "Agent",
                    Plugin:      "plugin",
                    Compression: "zip",
                }, "*")
            })

            It("should receive a update-object message bus event for Target, eventually", func(done Done) {
                /* this is executed in a goroutine */
                var t Target

                /* Update Target object message bus event*/
                receive(<-events, "update-object", "target", &t)
                Ω(t.UUID).Should(Equal("foo"))
                Ω(t.Name).Should(Equal("weekly"))
                Ω(t.Summary).Should(Equal("A Target plugin"))
                Ω(t.Agent).Should(Equal("Agent"))
                Ω(t.Plugin).Should(Equal("plugin"))
                Ω(t.Compression).Should(Equal("zip"))

                close(done)
            }, 2 /* timeout (in seconds) */)
        })

        /* Message bus tests for Tenants */
        Context("when sending a createObject message for Tenant", func() {
            BeforeEach(func() {
                /* Create Tenants object */
                db.sendCreateObjectEvent(Tenant{
                    UUID:          "foo",
                    Name:          "tenants",
                    DailyIncrease: 2,
                    StorageUsed:   2,
                    ArchiveCount:  1,
                }, "*")
            })

            It("should receive a create-object message bus event for Tenant, eventually", func(done Done) {
                /* this is executed in a goroutine */
                var t Tenant

                /* Create Target object bus event*/
                receive(<-events, "create-object", "tenant", &t)
                Ω(t.UUID).Should(Equal("foo"))
                Ω(t.Name).Should(Equal("tenants"))
                Ω(t.DailyIncrease).Should(Equal(int64(2)))
                Ω(t.StorageUsed).Should(Equal(int64(2)))
                Ω(t.ArchiveCount).Should(Equal(1))

                close(done)
            }, 2 /* timeout (in seconds) */)
        })

        Context("when sending an updateObject message for Tenant", func() {
            BeforeEach(func() {
                /* Update Target object*/
                db.sendUpdateObjectEvent(Tenant{
                    UUID:          "foo",
                    Name:          "tenants",
                    DailyIncrease: 3,
                    StorageUsed:   3,
                    ArchiveCount:  2,
                }, "*")
            })

            It("should receive a update-object message bus event for Tenant, eventually", func(done Done) {
                /* this is executed in a goroutine */
                var t Tenant

                /* Update Target object message bus event*/
                receive(<-events, "update-object", "tenant", &t)
                Ω(t.UUID).Should(Equal("foo"))
                Ω(t.Name).Should(Equal("tenants"))
                Ω(t.DailyIncrease).Should(Equal(int64(3)))
                Ω(t.StorageUsed).Should(Equal(int64(3)))
                Ω(t.ArchiveCount).Should(Equal(2))

                close(done)
            }, 2 /* timeout (in seconds) */)
        })

        /* Message bus tests for Task */
        Context("when sending a createObject message for Task", func() {
            BeforeEach(func() {
                /* Create Task object */
                db.sendCreateObjectEvent(Task{
                    UUID:        "foo",
                    TenantUUID:  "bar",
                    Owner:       "Admin",
                    Op:          "type",
                    JobUUID:     "job_foo",
                    ArchiveUUID: "archive_foo",
                    StoreUUID:   "store_foo",
                    TargetUUID:  "target_foo",
                    Status:      "running",
                    RequestedAt: 2,
                    StartedAt:   2,
                    StoppedAt:   3,
                    OK:          true,
                    Notes:       "notes",
                    Clear:       "clear",
                }, "*")
            })

            It("should receive a create-object message bus event for Task, eventually", func(done Done) {
                /* this is executed in a goroutine */
                var t Task

                /* Create Target object bus event*/
                receive(<-events, "create-object", "task", &t)
                Ω(t.UUID).Should(Equal("foo"))
                Ω(t.TenantUUID).Should(Equal("bar"))
                Ω(t.Owner).Should(Equal("Admin"))
                /* Fixme: Op won't serialize properly because jason name is of type
                Ω(t.Op).Should(Equal("type")) */
                Ω(t.JobUUID).Should(Equal("job_foo"))
                Ω(t.ArchiveUUID).Should(Equal("archive_foo"))
                Ω(t.StoreUUID).Should(Equal("store_foo"))
                Ω(t.TargetUUID).Should(Equal("target_foo"))
                Ω(t.Status).Should(Equal("running"))
                Ω(t.RequestedAt).Should(Equal(int64(2)))
                Ω(t.StartedAt).Should(Equal(int64(2)))
                Ω(t.StoppedAt).Should(Equal(int64(3)))
                Ω(t.OK).Should(Equal(true))
                Ω(t.Notes).Should(Equal("notes"))
                Ω(t.Clear).Should(Equal("clear"))

                close(done)
            }, 2 /* timeout (in seconds) */)
        })

        Context("when sending an updateObject message for Task", func() {
            BeforeEach(func() {
                /* Update Target object*/
                db.sendUpdateObjectEvent(Task{
                    UUID:        "foo",
                    TenantUUID:  "bar",
                    Owner:       "Admin",
                    Op:          "type",
                    JobUUID:     "job_bar",
                    ArchiveUUID: "archive_bar",
                    StoreUUID:   "store_bar",
                    TargetUUID:  "target_bar",
                    Status:      "stopped",
                    RequestedAt: 3,
                    StartedAt:   3,
                    StoppedAt:   4,
                    OK:          false,
                    Notes:       "notes",
                    Clear:       "clear",
                }, "*")
            })

            It("should receive a update-object message bus event for Task, eventually", func(done Done) {
                /* this is executed in a goroutine */
                var t Task

                /* Update Target object message bus event*/
                receive(<-events, "update-object", "task", &t)
                Ω(t.UUID).Should(Equal("foo"))
                Ω(t.TenantUUID).Should(Equal("bar"))
                Ω(t.Owner).Should(Equal("Admin"))
                /* Fixme: Op won't serialize properly because jason name is of type
                Ω(t.Op).Should(Equal("type")) */
                Ω(t.JobUUID).Should(Equal("job_bar"))
                Ω(t.ArchiveUUID).Should(Equal("archive_bar"))
                Ω(t.StoreUUID).Should(Equal("store_bar"))
                Ω(t.TargetUUID).Should(Equal("target_bar"))
                Ω(t.Status).Should(Equal("stopped"))
                Ω(t.RequestedAt).Should(Equal(int64(3)))
                Ω(t.StartedAt).Should(Equal(int64(3)))
                Ω(t.StoppedAt).Should(Equal(int64(4)))
                Ω(t.OK).Should(Equal(false))
                Ω(t.Notes).Should(Equal("notes"))
                Ω(t.Clear).Should(Equal("clear"))

                close(done)
            }, 2 /* timeout (in seconds) */)
        })

        /* Message bus tests for Archive */
        Context("when sending a createObject message for Archive", func() {
            BeforeEach(func() {
                /* Create Archive object */
                db.sendCreateObjectEvent(Archive{
                    UUID:           "foo",
                    TenantUUID:     "bar",
                    StoreUUID:      "store_foo",
                    TargetUUID:     "target_foo",
                    StoreKey:       "key1",
                    TakenAt:        2,
                    ExpiresAt:      2,
                    Notes:          "notes",
                    Status:         "running",
                    PurgeReason:    "some reason",
                    EncryptionType: "rsa",
                    Compression:    "zip",
                    Size:           2,
                }, "*")
            })

            It("should receive a create-object message bus event for Archive, eventually", func(done Done) {
                /* this is executed in a goroutine */
                var a Archive

                /* Create Archive object bus event*/
                receive(<-events, "create-object", "archive", &a)
                Ω(a.UUID).Should(Equal("foo"))
                Ω(a.TenantUUID).Should(Equal("bar"))
                Ω(a.StoreUUID).Should(Equal("store_foo"))
                Ω(a.TargetUUID).Should(Equal("target_foo"))
                Ω(a.StoreKey).Should(Equal("key1"))
                Ω(a.TakenAt).Should(Equal(int64(2)))
                Ω(a.ExpiresAt).Should(Equal(int64(2)))
                Ω(a.Notes).Should(Equal("notes"))
                Ω(a.Status).Should(Equal("running"))
                Ω(a.PurgeReason).Should(Equal("some reason"))
                Ω(a.EncryptionType).Should(Equal("rsa"))
                Ω(a.Compression).Should(Equal("zip"))
                Ω(a.Size).Should(Equal(int64(2)))

                close(done)
            }, 2 /* timeout (in seconds) */)
        })

        Context("when sending an updateObject message for Archive", func() {
            BeforeEach(func() {
                /* Update Archive object*/
                db.sendUpdateObjectEvent(Archive{
                    UUID:           "foo",
                    TenantUUID:     "bar",
                    StoreUUID:      "store_bar",
                    TargetUUID:     "target_bar",
                    StoreKey:       "key12",
                    TakenAt:        3,
                    ExpiresAt:      4,
                    Notes:          "notes",
                    Status:         "stopped",
                    PurgeReason:    "some reason",
                    EncryptionType: "rsa",
                    Compression:    "zip",
                    Size:           2,
                }, "*")
            })

            It("should receive a update-object message bus event for Archive, eventually", func(done Done) {
                /* this is executed in a goroutine */
                var a Archive

                /* Update Archive object message bus event*/
                receive(<-events, "update-object", "archive", &a)
                Ω(a.UUID).Should(Equal("foo"))
                Ω(a.TenantUUID).Should(Equal("bar"))
                Ω(a.StoreUUID).Should(Equal("store_bar"))
                Ω(a.TargetUUID).Should(Equal("target_bar"))
                Ω(a.StoreKey).Should(Equal("key12"))
                Ω(a.TakenAt).Should(Equal(int64(3)))
                Ω(a.ExpiresAt).Should(Equal(int64(4)))
                Ω(a.Notes).Should(Equal("notes"))
                Ω(a.Status).Should(Equal("stopped"))
                Ω(a.PurgeReason).Should(Equal("some reason"))
                Ω(a.EncryptionType).Should(Equal("rsa"))
                Ω(a.Compression).Should(Equal("zip"))
                Ω(a.Size).Should(Equal(int64(2)))

                close(done)
            }, 2 /* timeout (in seconds) */)
        })
    })
})
