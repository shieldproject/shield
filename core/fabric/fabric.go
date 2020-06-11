package fabric

import (
	"github.com/shieldproject/shield/core/scheduler"
	"github.com/shieldproject/shield/db"
)

type Fabric interface {
	Backup(*db.Task) scheduler.Chore
	Restore(*db.Task) scheduler.Chore
	Status(*db.Task) scheduler.Chore
}
