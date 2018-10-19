package fabric

import (
	"github.com/starkandwayne/shield/core/scheduler"
	"github.com/starkandwayne/shield/core/vault"
	"github.com/starkandwayne/shield/db"
)

type Fabric interface {
	/* back up a target to a store, encrypt it,
	   and optionally compress it. */
	Backup(*db.Target, *db.Store, string, *vault.Parameters) scheduler.Chore

	/* restore an encrypted archive to a target. */
	Restore(*db.Archive, *db.Target, *vault.Parameters) scheduler.Chore

	/* check the status of the agent. */
	Status() scheduler.Chore

	/* purge an from cloud storage archive. */
	Purge(*db.Archive) scheduler.Chore

	/* test the viability of a storage system. */
	TestStore(*db.Store) scheduler.Chore
}
