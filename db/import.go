package db

import (
	"encoding/json"
	"fmt"

	"github.com/jhunt/go-log"

	"github.com/shieldproject/shield/core/vault"
)

func (db *DB) importAgents(n uint, in *json.Decoder) {
	type agent struct {
		UUID          string `json:"uuid"`
		Name          string `json:"name"`
		Address       string `json:"address"`
		Version       string `json:"version"`
		Hidden        bool   `json:"hidden"`
		LastSeenAt    int64  `json:"last_seen_at"`
		LastCheckedAt int64  `json:"last_checked_at"`
		LastError     string `json:"last_error"`
		Status        string `json:"status"`
		Metadata      string `json:"metadata"`
	}

	for ; n > 0; n-- {
		var v agent
		if err := in.Decode(&v); err != nil {
			panic(err)
		}

		log.Infof("<<import>> inserting agent %s...", v.UUID)
		err := db.exec(`
		  INSERT INTO agents
		    (uuid, name, address, version, hidden,
		     last_seen_at, last_checked_at, last_error,
		     status, metadata)
		  VALUES
		    (?, ?, ?, ?, ?,
		     ?, ?, ?,
		     ?, ?)`,
			v.UUID, v.Name, v.Address, v.Version, v.Hidden,
			v.LastSeenAt, v.LastCheckedAt, v.LastError,
			v.Status, v.Metadata)
		if err != nil {
			panic(err)
		}
	}
}

func (db *DB) importArchives(n uint, in *json.Decoder, vlt *vault.Client) {
	type archive struct {
		UUID           string `json:"uuid"`
		TenantUUID     string `json:"tenant_uuid"`
		TargetUUID     string `json:"target_uuid"`
		StoreUUID      string `json:"store_uuid"`
		StoreKey       string `json:"store_key"`
		TakenAt        int    `json:"taken_at"`
		ExpiresAt      int    `json:"expires_at"`
		Notes          string `json:"notes"`
		PurgeReason    string `json:"purge_reason"`
		Status         string `json:"status"`
		Size           *int   `json:"size"`
		Job            string `json:"jobs"`
		EncryptionType string `json:"encryption_type"`
		Compression    string `json:"compression"`
		EncryptionKey  string `json:"encryption_key"`
		EncryptionIV   string `json:"encryption_iv"`
	}

	for ; n > 0; n-- {
		var v archive
		if err := in.Decode(&v); err != nil {
			panic(err)
		}

		log.Infof("<<import>> inserting archive %s...", v.UUID)
		err := db.exec(`
		  INSERT INTO archives
		    (uuid, tenant_uuid, target_uuid, store_uuid,
		     store_key, taken_at, expires_at, notes, purge_reason,
		     status, size, job, encryption_type, compression)
		  VALUES
		    (?, ?, ?, ?,
		     ?, ?, ?, ?, ?,
		     ?, ?, ?, ?, ?)`,
			v.UUID, v.TenantUUID, v.TargetUUID, v.StoreUUID,
			v.StoreKey, v.TakenAt, v.ExpiresAt, v.Notes, v.PurgeReason,
			v.Status, v.Size, v.Job, v.EncryptionType, v.Compression)
		if err != nil {
			panic(err)
		}

		err = vlt.Store(v.UUID, vault.Parameters{
			Key: v.EncryptionKey,
			IV: v.EncryptionIV,
			Type: v.EncryptionType,
		})
		if err != nil {
			panic(err)
		}
	}
}

func (db *DB) importFixups(n uint, in *json.Decoder) {
	type fixup struct {
		ID        string `json:"id"`
		Name      string `json:"name"`
		Summary   string `json:"summary"`
		CreatedAt int    `json:"created_at"`
		AppliedAt int    `json:"applied_at"`
	}

	for ; n > 0; n-- {
		var v fixup
		if err := in.Decode(&v); err != nil {
			panic(err)
		}

		log.Infof("<<import>> inserting fixup #%s...", v.ID)
		err := db.exec(`
		  INSERT INTO fixups
		    (id, name, summary, created_at, applied_at)
		  VALUES
		    (?, ?, ?, ?, ?)`,
			v.ID, v.Name, v.Summary, v.CreatedAt, v.AppliedAt)
		if err != nil {
			panic(err)
		}
	}
}

func (db *DB) importJobs(n uint, in *json.Decoder) {
	type job struct {
		UUID       string `json:"uuid"`
		TargetUUID string `json:"target_uuid"`
		StoreUUID  string `json:"store_uuid"`
		TenantUUID string `json:"tenant_uuid"`
		Name       string `json:"name"`
		Summary    string `json:"summary"`
		Schedule   string `json:"schedule"`
		KeepN      int    `json:"keep_n"`
		KeepDays   int    `json:"keep_days"`
		NextRun    int    `json:"next_run"`
		Priority   int    `json:"priority"`
		Paused     bool   `json:"paused"`
		FixedKey   bool   `json:"fixed_key"`
		Healthy    bool   `json:"healthy"`
	}

	for ; n > 0; n-- {
		var v job
		if err := in.Decode(&v); err != nil {
			panic(err)
		}

		log.Infof("<<import>> inserting job %s...", v.UUID)
		err := db.exec(`
		  INSERT INTO jobs
		    (uuid, target_uuid, store_uuid, tenant_uuid,
		     name, summary, schedule, keep_n, keep_days,
		     next_run, priority, paused, fixed_key, healthy)
		  VALUES
		    (?, ?, ?, ?,
		     ?, ?, ?, ?, ?,
		     ?, ?, ?, ?, ?)`,
			v.UUID, v.TargetUUID, v.StoreUUID, v.TenantUUID,
			v.Name, v.Summary, v.Schedule, v.KeepN, v.KeepDays,
			v.NextRun, v.Priority, v.Paused, v.FixedKey, v.Healthy)
		if err != nil {
			panic(err)
		}
	}
}

func (db *DB) importMemberships(n uint, in *json.Decoder) {
	type membership struct {
		TenantUUID string `json:"tenant_uuid"`
		UserUUID   string `json:"user_uuid"`
		Role       string `json:"role"`
	}

	for ; n > 0; n-- {
		var v membership
		if err := in.Decode(&v); err != nil {
			panic(err)
		}

		err := db.exec(`
		  INSERT INTO memberships
		    (tenant_uuid, user_uuid, role)
		  VALUES
		    (?, ?, ?)`,
			v.TenantUUID, v.UserUUID, v.Role)
		if err != nil {
			panic(err)
		}
	}
}

func (db *DB) importStores(n uint, in *json.Decoder) {
	type store struct {
		UUID             string `json:"uuid"`
		TenantUUID       string `json:"tenant_uuid"`
		Name             string `json:"name"`
		Summary          string `json:"summary"`
		Plugin           string `json:"plugin"`
		Endpoint         string `json:"endpoint"`
		Agent            string `json:"agent"`
		DailyIncrease    *int   `json:"daily_increase"`
		StorageUsed      *int   `json:"storage_used"`
		ArchiveCount     *int   `json:"archive_count"`
		Threshold        *int   `json:"threshold"`
		Healthy          bool   `json:"healthy"`
		LastTestTaskUUID string `json:"last_test_task_uuid"`
	}

	for ; n > 0; n-- {
		var v store
		if err := in.Decode(&v); err != nil {
			panic(err)
		}

		log.Infof("<<import>> inserting store %s...", v.UUID)
		err := db.exec(`
		  INSERT INTO stores
		    (uuid, tenant_uuid, name, summary,
		     plugin, endpoint, agent,
		     daily_increase, storage_used, archive_count,
		     threshold, healthy, last_test_task_uuid)
		  VALUES
		    (?, ?, ?, ?,
		     ?, ?, ?,
		     ?, ?, ?,
		     ?, ?, ?)`,
			v.UUID, v.TenantUUID, v.Name, v.Summary,
			v.Plugin, v.Endpoint, v.Agent,
			v.DailyIncrease, v.StorageUsed, v.ArchiveCount,
			v.Threshold, v.Healthy, v.LastTestTaskUUID)
		if err != nil {
			panic(err)
		}
	}
}

func (db *DB) importTargets(n uint, in *json.Decoder) {
	type target struct {
		UUID        string `json:"uuid"`
		TenantUUID  string `json:"tenant_uuid"`
		Name        string `json:"name"`
		Summary     string `json:"summary"`
		Plugin      string `json:"plugin"`
		Endpoint    string `json:"endpoint"`
		Agent       string `json:"agent"`
		Compression string `json:"compression"`
		Healthy     bool   `json:"healthy"`
	}

	for ; n > 0; n-- {
		var v target
		if err := in.Decode(&v); err != nil {
			panic(err)
		}

		log.Infof("<<import>> inserting target %s...", v.UUID)
		err := db.exec(`
		  INSERT INTO targets
		    (uuid, tenant_uuid, name, summary,
		     plugin, endpoint, agent, compression, healthy)
		  VALUES
		    (?, ?, ?, ?,
		     ?, ?, ?, ?, ?)`,
			v.UUID, v.TenantUUID, v.Name, v.Summary,
			v.Plugin, v.Endpoint, v.Agent, v.Compression, v.Healthy)
		if err != nil {
			panic(err)
		}
	}
}

func (db *DB) importTasks(n uint, in *json.Decoder) {
	type task struct {
		UUID           string  `json:"uuid"`
		Owner          string  `json:"owner"`
		Op             string  `json:"op"`
		TenantUUID     *string `json:"tenant_uuid"`
		JobUUID        *string `json:"job_uuid"`
		ArchiveUUID    *string `json:"archive_uuid"`
		TargetUUID     *string `json:"target_uuid"`
		StoreUUID      *string `json:"store_uuid"`
		Status         string  `json:"status"`
		RequestedAt    int     `json:"requested_at"`
		StartedAt      *int    `json:"started_at"`
		StoppedAt      *int    `json:"stopped_at"`
		TimeoutAt      *int    `json:"timeout_at"`
		Log            string  `json:"log"`
		Attempts       int     `json:"attempts"`
		Agent          string  `json:"agent"`
		FixedKey       string  `json:"fixed_key"`
		TargetPlugin   string  `json:"target_plugin"`
		TargetEndpoint string  `json;"target_endpoint"`
		StorePlugin    string  `json:"store_plugin"`
		StoreEndpoint  string  `json:"store_endpoint"`
		RestoreKey     string  `json:"restore_key"`
		OK             bool    `json:"ok"`
		Notes          string  `json:"notes"`
		Clear          string  `json:"clear"`
	}

	for ; n > 0; n-- {
		var v task
		if err := in.Decode(&v); err != nil {
			panic(err)
		}

		log.Infof("<<import>> inserting task %s...", v.UUID)
		err := db.exec(`
		  INSERT INTO tasks
		    (uuid, owner, op,
		     tenant_uuid, job_uuid, archive_uuid, target_uuid, store_uuid,
		     status, requested_at, started_at, stopped_at, timeout_at,
		     log, attempts, agent, fixed_key,
		     target_plugin, target_endpoint,
		     store_plugin, store_endpoint, restore_key,
		     ok, notes, clear)
		  VALUES
		    (?, ?, ?,
		     ?, ?, ?, ?, ?,
		     ?, ?, ?, ?, ?,
		     ?, ?, ?, ?,
		     ?, ?,
		     ?, ?, ?,
		     ?, ?, ?)`,
			v.UUID, v.Owner, v.Op,
			v.TenantUUID, v.JobUUID, v.ArchiveUUID, v.TargetUUID, v.StoreUUID,
			v.Status, v.RequestedAt, v.StartedAt, v.StoppedAt, v.TimeoutAt,
			v.Log, v.Attempts, v.Agent, v.FixedKey,
			v.TargetPlugin, v.TargetEndpoint,
			v.StorePlugin, v.StoreEndpoint, v.RestoreKey,
			v.OK, v.Notes, v.Clear)
		if err != nil {
			panic(err)
		}
	}
}

func (db *DB) importTenants(n uint, in *json.Decoder) {
	type tenant struct {
		UUID          string `json:"uuid"`
		Name          string `json:"name"`
		DailyIncrease *int   `json:"daily_increase"`
		StorageUsed   *int   `json:"storage_used"`
		ArchiveCount  *int   `json:"archive_count"`
	}

	for ; n > 0; n-- {
		var v tenant
		if err := in.Decode(&v); err != nil {
			panic(err)
		}

		log.Infof("<<import>> inserting tenant %s...", v.UUID)
		err := db.exec(`
		  INSERT INTO tenants
		    (uuid, name,
		     daily_increase, storage_used, archive_count)
		  VALUES
		    (?, ?,
		     ?, ?, ?)`,
			v.UUID, v.Name,
			v.DailyIncrease, v.StorageUsed, v.ArchiveCount)
		if err != nil {
			panic(err)
		}
	}
}

func (db *DB) importUsers(n uint, in *json.Decoder) {
	type user struct {
		UUID          string `json:"uuid"`
		Name          string `json:"name"`
		Account       string `json:"account"`
		Backend       string `json:"backend"`
		PasswordHash  string `json:"password_hash"`
		SystemRole    string `json:"system_role"`
		DefaultTenant string `json:"default_tenant"`
	}

	for ; n > 0; n-- {
		var v user
		if err := in.Decode(&v); err != nil {
			panic(err)
		}

		log.Infof("<<import>> inserting user %s...", v.UUID)
		err := db.exec(`
		  INSERT INTO users
		    (uuid, name, account, backend,
		     pwhash, sysrole, default_tenant)
		  VALUES
		    (?, ?, ?, ?,
		     ?, ?, ?)`,
			v.UUID, v.Name, v.Account, v.Backend,
			v.PasswordHash, v.SystemRole, v.DefaultTenant)
		if err != nil {
			panic(err)
		}
	}
}

func (db *DB) clear(tables ...string) {
	for _, t := range tables {
		log.Infof("<<import>> clearing table %s...", t)
		if err := db.exec(fmt.Sprintf("DELETE FROM %s", t)); err != nil {
			panic(fmt.Errorf("clear table failed: %s", err))
		}
	}
}

func (db *DB) Import(in *json.Decoder, vault *vault.Client) {
	var h header

	db.transactionally(func() error {
		db.clear("agents", "archives", "fixups", "jobs", "memberships", "stores")
		db.clear("targets", "tasks", "tenants", "users")
		db.clear("sessions")

		for in.More() {
			if err := in.Decode(&h); err != nil {
				panic(err)
			}
			if h.V != "v1" {
				panic(fmt.Errorf("unrecognized import header version '%s'", h.V))
			}

			switch h.Type {
			case "agents":
				db.importAgents(h.N, in)
			case "archives":
				db.importArchives(h.N, in, vault)
			case "fixups":
				db.importFixups(h.N, in)
			case "jobs":
				db.importJobs(h.N, in)
			case "memberships":
				db.importMemberships(h.N, in)
			case "stores":
				db.importStores(h.N, in)
			case "targets":
				db.importTargets(h.N, in)
			case "tasks":
				db.importTasks(h.N, in)
			case "tenants":
				db.importTenants(h.N, in)
			case "users":
				db.importUsers(h.N, in)

			case "":
			default:
				panic(fmt.Errorf("unrecognized import header type '%s'", h.Type))
			}
		}

		return nil
	})
}
