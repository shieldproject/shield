package db

import (
	"fmt"
	"strings"

	"github.com/jhunt/go-log"

	"github.com/starkandwayne/shield/core/bus"
)

func datauuid(thing interface{}) string {
	switch thing.(type) {
	case Agent:
		return fmt.Sprintf("agent [%s]", thing.(Agent).UUID)
	case *Agent:
		return fmt.Sprintf("agent [%s]", thing.(*Agent).UUID)

	case Job:
		return fmt.Sprintf("job [%s]", thing.(Job).UUID)
	case *Job:
		return fmt.Sprintf("job [%s]", thing.(*Job).UUID)

	case Store:
		return fmt.Sprintf("store [%s]", thing.(Store).UUID)
	case *Store:
		return fmt.Sprintf("store [%s]", thing.(*Store).UUID)

	case Target:
		return fmt.Sprintf("target [%s]", thing.(Target).UUID)
	case *Target:
		return fmt.Sprintf("target [%s]", thing.(*Target).UUID)

	case Tenant:
		return fmt.Sprintf("tenant [%s]", thing.(Tenant).UUID)
	case *Tenant:
		return fmt.Sprintf("tenant [%s]", thing.(*Tenant).UUID)

	case Task:
		return fmt.Sprintf("task [%s]", thing.(Task).UUID)
	case *Task:
		return fmt.Sprintf("task [%s]", thing.(*Task).UUID)

	default:
		panic("SHIELD was unable to determine the type of thing, in order to craft a message bus event for it.  This is most certainly a bug in SHIELD itself.")
	}
}

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
		log.Infof("sending %s to [%s] for %s", bus.CreateObjectEvent, strings.Join(queues, ", "), datauuid(thing))
		db.bus.Send(bus.CreateObjectEvent, datatype(thing), thing, queues...)
	}
}

func (db *DB) sendUpdateObjectEvent(thing interface{}, queues ...string) {
	if db.bus != nil {
		log.Infof("sending %s to [%s] for %s", bus.UpdateObjectEvent, strings.Join(queues, ", "), datauuid(thing))
		db.bus.Send(bus.UpdateObjectEvent, datatype(thing), thing, queues...)
	}
}

func (db *DB) sendDeleteObjectEvent(thing interface{}, queues ...string) {
	if db.bus != nil {
		log.Infof("sending %s to [%s] for %s", bus.DeleteObjectEvent, strings.Join(queues, ", "), datauuid(thing))
		db.bus.Send(bus.DeleteObjectEvent, datatype(thing), thing, queues...)
	}
}

func (db *DB) sendTaskStatusUpdateEvent(task *Task, queues ...string) {
	if db.bus != nil {
		log.Infof("sending %s to [%s] for task [%s]", bus.TaskStatusUpdateEvent, strings.Join(queues, ", "), task.UUID)
		db.bus.Send(bus.TaskStatusUpdateEvent, "", map[string]interface{}{
			"uuid":       task.UUID,
			"status":     task.Status,
			"started_at": task.StartedAt,
			"stopped_at": task.StoppedAt,
			"ok":         task.OK,
		}, queues...)
	}
}

func (db *DB) sendTaskLogUpdateEvent(id, msg string, queues ...string) {
	if db.bus != nil {
		log.Infof("sending %s to [%s] for task [%s]", bus.TaskLogUpdateEvent, strings.Join(queues, ", "), id)
		db.bus.Send(bus.TaskLogUpdateEvent, "", map[string]interface{}{
			"uuid": id,
			"tail": msg,
		}, queues...)
	}
}

func (db *DB) sendTenantInviteEvent(user, tenant, role string) {
	if db.bus != nil {
		db.bus.Send(bus.TenantInviteEvent, "", map[string]interface{}{
			"user_uuid":   user,
			"tenant_uuid": tenant,
			"role":        role,
		}, "user:"+user, "tenant:"+tenant)
	}
}

func (db *DB) sendTenantBanishEvent(user, tenant string) {
	if db.bus != nil {
		db.bus.Send(bus.TenantBanishEvent, "", map[string]interface{}{
			"user_uuid":   user,
			"tenant_uuid": tenant,
		}, "user:"+user, "tenant:"+tenant)
	}
}
