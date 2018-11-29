package fabric

import (
	"time"

	"github.com/starkandwayne/shield/core/scheduler"
	"github.com/starkandwayne/shield/core/vault"
	"github.com/starkandwayne/shield/db"
)

func Dummy(delay int) DummyFabric {
	return DummyFabric{
		delay: delay,
	}
}

type DummyFabric struct {
	delay int
}

func (f DummyFabric) Sleep() {
	if f.delay > 0 {
		time.Sleep(time.Duration(f.delay) * time.Second)
	}
}

func (f DummyFabric) Backup(task *db.Task, encryption vault.Parameters) scheduler.Chore {
	return scheduler.NewChore(
		task.UUID,
		func(chore scheduler.Chore) {
			chore.Errorf("DUMMY> starting a backup operation; delay is %ds", f.delay)
			chore.Errorf("DUMMY>")
			chore.Errorf("DUMMY>   target plugin:   '%s'", task.TargetPlugin)
			chore.Errorf("DUMMY>   target endpoint: '%s'", task.TargetEndpoint)
			chore.Errorf("DUMMY>")
			chore.Errorf("DUMMY>   store plugin:    '%s'", task.StorePlugin)
			chore.Errorf("DUMMY>   store endpoint:  '%s'", task.StoreEndpoint)
			chore.Errorf("DUMMY>")
			chore.Errorf("DUMMY>   compression:     '%s'", task.Compression)
			chore.Errorf("DUMMY>")
			chore.Errorf("DUMMY>   encryption type: '%s'", encryption.Type)
			chore.Errorf("DUMMY>   encryption key:  '%s'", encryption.Key)
			chore.Errorf("DUMMY>   encryption iv:   '%s'", encryption.IV)
			f.Sleep()
			chore.Errorf("DUMMY>")
			chore.Errorf("DUMMY> backup operation complete.")
			chore.Infof(`{"key":"%s","archive_size":1337,"compression":"%s"}`,
				time.Now().Format("2006/01/02/15/04/05/2006-01-02T1504.archive"),
				task.Compression)
			chore.UnixExit(0)
			return
		})
}

func (f DummyFabric) Restore(task *db.Task, encryption vault.Parameters) scheduler.Chore {
	return scheduler.NewChore(
		task.UUID,
		func(chore scheduler.Chore) {
			chore.Errorf("DUMMY> starting a restore operation; delay is %ds", f.delay)
			chore.Errorf("DUMMY>")
			chore.Errorf("DUMMY>   restore key:     '%s'", task.RestoreKey)
			chore.Errorf("DUMMY>")
			chore.Errorf("DUMMY>   target plugin:   '%s'", task.TargetPlugin)
			chore.Errorf("DUMMY>   target endpoint: '%s'", task.TargetEndpoint)
			chore.Errorf("DUMMY>")
			chore.Errorf("DUMMY>   store plugin:    '%s'", task.StorePlugin)
			chore.Errorf("DUMMY>   store endpoint:  '%s'", task.StoreEndpoint)
			chore.Errorf("DUMMY>")
			chore.Errorf("DUMMY>   compression:     '%s'", task.Compression)
			chore.Errorf("DUMMY>")
			chore.Errorf("DUMMY>   encryption type: '%s'", encryption.Type)
			chore.Errorf("DUMMY>   encryption key:  '%s'", encryption.Key)
			chore.Errorf("DUMMY>   encryption iv:   '%s'", encryption.IV)
			f.Sleep()
			chore.Errorf("DUMMY>")
			chore.Errorf("DUMMY> restore operation complete.")
			chore.UnixExit(0)
			return
		})
}

func (f DummyFabric) Status(task *db.Task) scheduler.Chore {
	return scheduler.NewChore(
		task.UUID,
		func(chore scheduler.Chore) {
			chore.Errorf("DUMMY> starting an agent-status operation; delay is %ds", f.delay)
			f.Sleep()
			chore.Errorf("DUMMY>")
			chore.Errorf("DUMMY> (there is no status; this is a test/dev fabric...")
			chore.Errorf("DUMMY>")
			chore.Errorf("DUMMY> agent-status operation complete.")
			chore.UnixExit(0)
			return
		})
}

func (f DummyFabric) Purge(task *db.Task) scheduler.Chore {
	return scheduler.NewChore(
		task.UUID,
		func(chore scheduler.Chore) {
			chore.Errorf("DUMMY> starting an archive purge operation; delay is %ds", f.delay)
			chore.Errorf("DUMMY>")
			chore.Errorf("DUMMY>   archive key:     '%s'", task.RestoreKey)
			chore.Errorf("DUMMY>")
			chore.Errorf("DUMMY>   store plugin:    '%s'", task.StorePlugin)
			chore.Errorf("DUMMY>   store endpoint:  '%s'", task.StoreEndpoint)
			f.Sleep()
			chore.Errorf("DUMMY>")
			chore.Errorf("DUMMY> archive purge operation complete.")
			chore.UnixExit(0)
			return
		})
}

func (f DummyFabric) TestStore(task *db.Task) scheduler.Chore {
	return scheduler.NewChore(
		task.UUID,
		func(chore scheduler.Chore) {
			chore.Errorf("DUMMY> starting storage test operation; delay is %ds", f.delay)
			chore.Errorf("DUMMY>")
			chore.Errorf("DUMMY>   store plugin:    '%s'", task.StorePlugin)
			chore.Errorf("DUMMY>   store endpoint:  '%s'", task.StoreEndpoint)
			f.Sleep()
			chore.Errorf("DUMMY>")
			chore.Errorf("DUMMY> storage test operation complete.")
			chore.UnixExit(0)
			return
		})
}
