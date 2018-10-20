package db

import (
	"fmt"

	"github.com/pborman/uuid"

	"github.com/starkandwayne/shield/core/bus"
)

func datatype(thing interface{}) string {
	switch thing.(type) {
	case Agent, *Agent:
		return "agent"
	case Job, *Job:
		return "job"
	case Store, *Store:
		return "store"
	case Target, *Target:
		return "target"
	case Tenant, *Tenant:
		return "tenant"
	default:
		panic("SHIELD was unable to determine the type of thing, in order to craft a message bus event for it.  This is most certainly a bug in SHIELD itself.")
	}
}

func toTenant(id uuid.UUID) string {
	return id.String()
}

func toAdmins() string {
	return "admin"
}

func toAll() string {
	return "*"
}

func (db *DB) sendCreateObjectEvent(to string, thing interface{}) {
	fmt.Printf("sending %s to %s for %s\n", bus.CreateObjectEvent, to, datatype(thing))
	db.bus.Send(bus.CreateObjectEvent, to, datatype(thing), thing)
}

func (db *DB) sendUpdateObjectEvent(to string, thing interface{}) {
	db.bus.Send(bus.UpdateObjectEvent, to, datatype(thing), thing)
}

func (db *DB) sendDeleteObjectEvent(to string, thing interface{}) {
	db.bus.Send(bus.DeleteObjectEvent, to, datatype(thing), thing)
}

func (db *DB) sendTaskStatusUpdateEvent(to string, task *Task) {
	db.bus.Send(bus.TaskStatusUpdateEvent, to, "", map[string]interface{}{
		"uuid":   task.UUID.String(),
		"status": task.Status,
	})
}

func (db *DB) sendTaskLogUpdateEvent(to string, task *Task, log string) {
	db.bus.Send(bus.TaskLogUpdateEvent, to, "", map[string]interface{}{
		"uuid": task.UUID.String(),
		"tail": log,
	})
}
