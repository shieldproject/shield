package fabric

import (
	"github.com/starkandwayne/shield/core/scheduler"
	"github.com/starkandwayne/shield/core/vault"
	"github.com/starkandwayne/shield/db"
)

type Fabric interface {
	/* back up a target to a store, encrypt it,
	   and optionally compress it. */
	Backup(*db.Task, vault.Parameters) scheduler.Chore

	/* restore an encrypted archive to a target. */
	Restore(*db.Task, vault.Parameters) scheduler.Chore

	/* check the status of the agent. */
	Status(*db.Task) scheduler.Chore

	/* purge an from cloud storage archive. */
	Purge(*db.Task) scheduler.Chore

	/* test the viability of a storage system. */
	TestStore(*db.Task) scheduler.Chore
}
