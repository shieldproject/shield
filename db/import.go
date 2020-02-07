package db

import (
	"encoding/json"
	"fmt"

	"github.com/jhunt/go-log"

	"github.com/shieldproject/shield/core/vault"
)

func (db *DB) importAgents(n uint, in *json.Decoder) error {
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
			return err
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
			return err
		}
	}
	return nil
}

func (db *DB) importArchives(n uint, in *json.Decoder, vlt *vault.Client) error {
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
			return err
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
			return err
		}

		err = vlt.Store(v.UUID, vault.Parameters{
			Key:  v.EncryptionKey,
			IV:   v.EncryptionIV,
			Type: v.EncryptionType,
		})
		if err != nil {
			return err
		}
	}
	return nil
}

func (db *DB) importFixups(n uint, in *json.Decoder) error {
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
			return err
		}

		log.Infof("<<import>> inserting fixup #%s...", v.ID)
		err := db.exec(`
		  INSERT INTO fixups
		    (id, name, summary, created_at, applied_at)
		  VALUES
		    (?, ?, ?, ?, ?)`,
			v.ID, v.Name, v.Summary, v.CreatedAt, v.AppliedAt)
		if err != nil {
			return err
		}
	}
	return nil
}

func (db *DB) importJobs(n uint, in *json.Decoder) error {
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
			return err
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
			return err
		}
	}
	return nil
}

func (db *DB) importMemberships(n uint, in *json.Decoder) error {
	type membership struct {
		TenantUUID string `json:"tenant_uuid"`
		UserUUID   string `json:"user_uuid"`
		Role       string `json:"role"`
	}

	for ; n > 0; n-- {
		var v membership
		if err := in.Decode(&v); err != nil {
			return err
		}

		err := db.exec(`
		  INSERT INTO memberships
		    (user_uuid, tenant_uuid, role)
		  VALUES
		    (?, ?, ?)`,
			v.UserUUID, v.TenantUUID, v.Role)
		if err != nil {
			return err
		}
	}
	return nil
}

func (db *DB) importStores(n uint, in *json.Decoder) error {
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
			return err
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
			return err
		}
	}
	return nil
}

func (db *DB) importTargets(n uint, in *json.Decoder) error {
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
			return err
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
			return err
		}
	}
	return nil
}

func (db *DB) importTasks(n uint, in *json.Decoder) error {
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
		TargetEndpoint string  `json:"target_endpoint"`
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
			return err
		}
		if v.Status == "done" || v.Status == "failed" {
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
				return err
			}
		} else {
			log.Infof("<<import>> skipping insert task %s...", v.UUID)
		}
	}
	return nil
}

func (db *DB) importTenants(n uint, in *json.Decoder) error {
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
			return err
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
			return err
		}
	}
	return nil
}

func (db *DB) importUsers(n uint, in *json.Decoder) error {
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
			return err
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
			return err
		}
	}
	return nil
}

func (db *DB) importSessions(n uint, in *json.Decoder) error {
	type Session struct {
		UUID           string `json:"uuid"`
		UserUUID       string `json:"user_uuid"`
		CreatedAt      int64  `json:"created_at"`
		LastSeen       int64  `json:"last_seen_at"`
		Token          string `json:"token_uuid"`
		Name           string `json:"name"`
		IP             string `json:"ip_addr"`
		UserAgent      string `json:"user_agent"`
		UserAccount    string `json:"user_account"`
		CurrentSession bool   `json:"current_session"`
	}

	for ; n > 0; n-- {
		var v Session
		if err := in.Decode(&v); err != nil {
			return err
		}

		log.Infof("<<import>> inserting session %s...", v.UUID)
		err := db.exec(`
          INSERT INTO sessions
            (uuid, user_uuid, created_at, last_seen,
             token, name, ip_addr, user_agent)
          VALUES
             (?, ?, ?, ?,
              ?, ?, ?, ?)`,
			v.UUID, v.UserUUID, v.CreatedAt, v.LastSeen,
			v.Token, v.Name, v.IP, v.UserAgent)
		if err != nil {
			return err
		}
	}
	return nil
}

func (db *DB) importErrors(in *json.Decoder) error {
	type fail struct {
		V    string `json:"v"`
		Type string `json:"type"`
		E    error  `json:"error"`
	}

	var v fail
	if err := in.Decode(&v); err != nil {
		return err
	}
	return (fmt.Errorf("Import of %s table has errors: %s", v.Type, v.E))
}

func (db *DB) clear(tables ...string) error {
	for _, t := range tables {
		log.Infof("<<import>> clearing table %s...", t)
		if err := db.exec(fmt.Sprintf("DELETE FROM %s", t)); err != nil {
			return fmt.Errorf("clear table failed: %s", err)
		}
	}
	return nil
}

func (db *DB) Import(in *json.Decoder, vault *vault.Client) error {
	var h header

	err := db.transactionally(func() error {
		err := db.clear("agents", "archives", "fixups", "jobs", "memberships", "stores")
		if err != nil {
			return err
		}
		err = db.clear("targets", "tasks", "tenants", "users")
		if err != nil {
			return err
		}
		err = db.clear("sessions")
		if err != nil {
			return err
		}

		for in.More() {
			if err := in.Decode(&h); err != nil {
				return err
			}
			if h.V != "v1" {
				return fmt.Errorf("unrecognized import header version '%s'", h.V)
			}

			switch h.Type {
			case "agents":
				err := db.importAgents(h.N, in)
				if err != nil {
					return err
				}
			case "archives":
				err := db.importArchives(h.N, in, vault)
				if err != nil {
					return err
				}
			case "fixups":
				err := db.importFixups(h.N, in)
				if err != nil {
					return err
				}
			case "jobs":
				err := db.importJobs(h.N, in)
				if err != nil {
					return err
				}
			case "memberships":
				err := db.importMemberships(h.N, in)
				if err != nil {
					return err
				}
			case "stores":
				err := db.importStores(h.N, in)
				if err != nil {
					return err
				}
			case "targets":
				err := db.importTargets(h.N, in)
				if err != nil {
					return err
				}
			case "tasks":
				err := db.importTasks(h.N, in)
				if err != nil {
					return err
				}
			case "tenants":
				err := db.importTenants(h.N, in)
				if err != nil {
					return err
				}
			case "users":
				err := db.importUsers(h.N, in)
				if err != nil {
					return err
				}
			case "sessions":
				err := db.importSessions(h.N, in)
				if err != nil {
					return err
				}
			case "errors":
				return db.importErrors(in)
			case "":
			default:
				return fmt.Errorf("unrecognized import header type '%s'", h.Type)
			}
		}

		return nil
	})
	if err != nil {
		return err
	}
	return nil
}
