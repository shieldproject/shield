package fabric

import (
	"github.com/shieldproject/shield/core/scheduler"
	"github.com/shieldproject/shield/db"
)

type Fabric interface {
	/* back up a target to a store, encrypt it,
	   and optionally compress it. */
	Backup(*db.Task) scheduler.Chore

	/* restore an encrypted archive to a target. */
	Restore(*db.Task) scheduler.Chore

	/* check the status of the agent. */
	Status(*db.Task) scheduler.Chore

	/* purge an from cloud storage archive. */
	Purge(*db.Task) scheduler.Chore
}
