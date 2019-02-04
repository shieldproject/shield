package db

import (
	"fmt"
	"strings"

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
	case Task, *Task:
		return "task"
	default:
		panic("SHIELD was unable to determine the type of thing, in order to craft a message bus event for it.  This is most certainly a bug in SHIELD itself.")
	}
}

func (db *DB) sendCreateObjectEvent(thing interface{}, queues ...string) {
	if db.bus != nil {
		fmt.Printf("sending %s to [%s] for %s\n", bus.CreateObjectEvent, strings.Join(queues, ", "), datatype(thing))
		db.bus.Send(bus.CreateObjectEvent, datatype(thing), thing, queues...)
	}
}

func (db *DB) sendUpdateObjectEvent(thing interface{}, queues ...string) {
	if db.bus != nil {
		db.bus.Send(bus.UpdateObjectEvent, datatype(thing), thing, queues...)
	}
}

func (db *DB) sendDeleteObjectEvent(thing interface{}, queues ...string) {
	if db.bus != nil {
		db.bus.Send(bus.DeleteObjectEvent, datatype(thing), thing, queues...)
	}
}

func (db *DB) sendTaskStatusUpdateEvent(task *Task, queues ...string) {
	if db.bus != nil {
		db.bus.Send(bus.TaskStatusUpdateEvent, "", map[string]interface{}{
			"uuid":       task.UUID,
			"status":     task.Status,
			"started_at": task.StartedAt,
			"stopped_at": task.StoppedAt,
			"ok":         task.OK,
		}, queues...)
	}
}

func (db *DB) sendTaskLogUpdateEvent(id, log string, queues ...string) {
	if db.bus != nil {
		db.bus.Send(bus.TaskLogUpdateEvent, "", map[string]interface{}{
			"uuid": id,
			"tail": log,
		}, queues...)
	}
}

func (db *DB) sendTenantInviteEvent(user, tenant, role string) {
	if db.bus != nil {
		db.bus.Send(bus.TenantInviteEvent, "", map[string]interface{}{
			"user_uuid":   user,
			"tenant_uuid": tenant,
			"role":        role,
		}, "user:"+user, "tenant:"+tenant, "admins")
	}
}

func (db *DB) sendTenantBanishEvent(user, tenant string) {
	if db.bus != nil {
		db.bus.Send(bus.TenantBanishEvent, "", map[string]interface{}{
			"user_uuid":   user,
			"tenant_uuid": tenant,
		}, "user:"+user, "tenant:"+tenant, "admins")
	}
}
