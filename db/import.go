package db

import (
	"encoding/json"
	"fmt"
	"time"

	"github.com/jhunt/go-log"
)

const MetaPluginName = "metashield"

type preimport struct {
	RestoreTask *Task
	Archive     *Archive
}

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
		Error         string `json:"error"`
	}

	for ; n > 0; n-- {
		var v agent
		if err := in.Decode(&v); err != nil {
			return err
		}

		if v.Error != "" {
			return fmt.Errorf(v.Error)
		}

		log.Infof("IMPORT: inserting agent %s...", v.UUID)
		err := db.Exec(`
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

func (db *DB) importArchives(n uint, in *json.Decoder) error {
	type archive struct {
		UUID        string `json:"uuid"`
		TargetUUID  string `json:"target_uuid"`
		StoreKey    string `json:"store_key"`
		TakenAt     int    `json:"taken_at"`
		ExpiresAt   int    `json:"expires_at"`
		Notes       string `json:"notes"`
		PurgeReason string `json:"purge_reason"`
		Status      string `json:"status"`
		Size        *int   `json:"size"`
		Job         string `json:"jobs"`
		Error       string `json:"error"`
	}

	for ; n > 0; n-- {
		var v archive
		if err := in.Decode(&v); err != nil {
			return err
		}

		if v.Error != "" {
			return fmt.Errorf(v.Error)
		}

		log.Infof("IMPORT: inserting archive %s...", v.UUID)
		err := db.Exec(`
		  INSERT INTO archives
		    (uuid, target_uuid,
		     store_key, taken_at, expires_at, notes, purge_reason,
		     status, size, job)
		  VALUES
		    (?, ?,
		     ?, ?, ?, ?, ?,
		     ?, ?, ?)`,
			v.UUID, v.TargetUUID,
			v.StoreKey, v.TakenAt, v.ExpiresAt, v.Notes, v.PurgeReason,
			v.Status, v.Size, v.Job)
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
		Error     string `json:"error"`
	}

	for ; n > 0; n-- {
		var v fixup
		if err := in.Decode(&v); err != nil {
			return err
		}

		if v.Error != "" {
			return fmt.Errorf(v.Error)
		}

		log.Infof("IMPORT: inserting fixup #%s...", v.ID)
		err := db.Exec(`
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
		Name       string `json:"name"`
		Summary    string `json:"summary"`
		Schedule   string `json:"schedule"`
		KeepN      int    `json:"keep_n"`
		KeepDays   int    `json:"keep_days"`
		Bucket     string `json:"bucket"`
		NextRun    int    `json:"next_run"`
		Priority   int    `json:"priority"`
		Paused     bool   `json:"paused"`
		Healthy    bool   `json:"healthy"`
		Error      string `json:"error"`
	}

	for ; n > 0; n-- {
		var v job
		if err := in.Decode(&v); err != nil {
			return err
		}

		if v.Error != "" {
			return fmt.Errorf(v.Error)
		}

		log.Infof("IMPORT: inserting job %s...", v.UUID)
		err := db.Exec(`
		  INSERT INTO jobs
		    (uuid, target_uuid, bucket,
		     name, summary, schedule, keep_n, keep_days,
		     next_run, priority, paused, healthy)
		  VALUES
		    (?, ?, ?,
		     ?, ?, ?, ?, ?,
		     ?, ?, ?, ?)`,
			v.UUID, v.TargetUUID, v.Bucket,
			v.Name, v.Summary, v.Schedule, v.KeepN, v.KeepDays,
			v.NextRun, v.Priority, v.Paused, v.Healthy)
		if err != nil {
			return err
		}
	}
	return nil
}

func (db *DB) importTargets(n uint, in *json.Decoder) error {
	type target struct {
		UUID     string `json:"uuid"`
		Name     string `json:"name"`
		Summary  string `json:"summary"`
		Plugin   string `json:"plugin"`
		Endpoint string `json:"endpoint"`
		Agent    string `json:"agent"`
		Healthy  bool   `json:"healthy"`
		Error    string `json:"error"`
	}

	for ; n > 0; n-- {
		var v target
		if err := in.Decode(&v); err != nil {
			return err
		}

		if v.Error != "" {
			return fmt.Errorf(v.Error)
		}

		log.Infof("IMPORT: inserting target %s...", v.UUID)
		err := db.Exec(`
		  INSERT INTO targets
		    (uuid, name, summary,
		     plugin, endpoint, agent, healthy)
		  VALUES
		    (?, ?, ?,
		     ?, ?, ?, ?)`,
			v.UUID, v.Name, v.Summary,
			v.Plugin, v.Endpoint, v.Agent, v.Healthy)
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
		JobUUID        *string `json:"job_uuid"`
		ArchiveUUID    *string `json:"archive_uuid"`
		TargetUUID     *string `json:"target_uuid"`
		Status         string  `json:"status"`
		RequestedAt    int     `json:"requested_at"`
		StartedAt      *int    `json:"started_at"`
		StoppedAt      *int64  `json:"stopped_at"`
		TimeoutAt      *int    `json:"timeout_at"`
		Log            string  `json:"log"`
		Attempts       int     `json:"attempts"`
		Agent          string  `json:"agent"`
		TargetPlugin   string  `json:"target_plugin"`
		TargetEndpoint string  `json:"target_endpoint"`
		RestoreKey     string  `json:"restore_key"`
		OK             bool    `json:"ok"`
		Notes          string  `json:"notes"`
		Clear          string  `json:"clear"`
		Error          string  `json:"error"`
	}

	for ; n > 0; n-- {
		var v task
		if err := in.Decode(&v); err != nil {
			return err
		}

		if v.Error != "" {
			return fmt.Errorf(v.Error)
		}

		if v.TargetPlugin == MetaPluginName && v.Op == "backup" && v.Status == "running" {
			log.Infof("IMPORT: inserting task %s (as done)...", v.UUID)
			v.Status = "done"
			v.OK = true
			at := time.Now().Unix()
			v.StoppedAt = &at
		}
		if v.Status == "done" || v.Status == "failed" || v.Status == "canceled" {
			log.Infof("IMPORT: inserting task %s... ", v.UUID)
			err := db.Exec(`
            INSERT INTO tasks
                (uuid, owner, op,
                job_uuid, archive_uuid, target_uuid,
                status, requested_at, started_at, stopped_at, timeout_at,
                log, attempts, agent,
                target_plugin, target_endpoint,
                restore_key,
                ok, notes, clear)
            VALUES
                (?, ?, ?,
                ?, ?, ?,
                ?, ?, ?, ?, ?,
                ?, ?, ?,
                ?, ?,
                ?,
                ?, ?, ?)`,
				v.UUID, v.Owner, v.Op,
				v.JobUUID, v.ArchiveUUID, v.TargetUUID,
				v.Status, v.RequestedAt, v.StartedAt, v.StoppedAt, v.TimeoutAt,
				v.Log, v.Attempts, v.Agent,
				v.TargetPlugin, v.TargetEndpoint,
				v.RestoreKey,
				v.OK, v.Notes, v.Clear)
			if err != nil {
				return err
			}
		} else {
			log.Infof("IMPORT: skipping insert task %s...", v.UUID)
		}
	}
	return nil
}

func (db *DB) importUsers(n uint, in *json.Decoder) error {
	type user struct {
		UUID         string `json:"uuid"`
		Name         string `json:"name"`
		Account      string `json:"account"`
		Backend      string `json:"backend"`
		PasswordHash string `json:"password_hash"`
		SystemRole   string `json:"system_role"`
		Error        string `json:"error"`
	}

	for ; n > 0; n-- {
		var v user
		if err := in.Decode(&v); err != nil {
			return err
		}

		if v.Error != "" {
			return fmt.Errorf(v.Error)
		}

		log.Infof("IMPORT: inserting user %s...", v.UUID)
		err := db.Exec(`
		  INSERT INTO users
		    (uuid, name, account, backend,
		     pwhash, sysrole)
		  VALUES
		    (?, ?, ?, ?,
		     ?, ?)`,
			v.UUID, v.Name, v.Account, v.Backend,
			v.PasswordHash, v.SystemRole)
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
		Error          string `json:"error"`
	}

	for ; n > 0; n-- {
		var v Session
		if err := in.Decode(&v); err != nil {
			return err
		}

		if v.Error != "" {
			return fmt.Errorf(v.Error)
		}

		log.Infof("IMPORT: inserting session %s...", v.UUID)
		err := db.Exec(`
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

func (db *DB) preimportState(restoreKey, uuid string) (*preimport, error) {
	ctx := preimport{}

	archives, err := db.GetAllArchives(&ArchiveFilter{
		ForStoreKey: restoreKey,
	})
	if err != nil {
		return nil, err
	}

	if len(archives) == 1 {
		ctx.Archive = archives[0]
	}

	task, err := db.GetTask(uuid)
	if err != nil {
		return nil, err
	}
	ctx.RestoreTask = task
	return &ctx, nil
}

func (db *DB) clear(tables ...string) error {
	for _, t := range tables {
		log.Infof("IMPORT: clearing table %s...", t)
		if err := db.Exec(fmt.Sprintf("DELETE FROM %s", t)); err != nil {
			return fmt.Errorf("clear table failed: %s", err)
		}
	}
	return nil
}

func (db *DB) Import(in *json.Decoder, restoreKey, uuid string) error {
	var h header

	ctx, err := db.preimportState(restoreKey, uuid)
	if err != nil {
		return err
	}

	err = db.transactionally(func() error {
		err = db.clear("agents", "archives", "fixups", "jobs")
		if err != nil {
			return err
		}
		err = db.clear("targets", "tasks", "users")
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
				if err := db.importAgents(h.N, in); err != nil {
					return err
				}

			case "archives":
				if err := db.importArchives(h.N, in); err != nil {
					return err
				}

			case "fixups":
				if err := db.importFixups(h.N, in); err != nil {
					return err
				}

			case "jobs":
				if err := db.importJobs(h.N, in); err != nil {
					return err
				}

			case "targets":
				if err := db.importTargets(h.N, in); err != nil {
					return err
				}

			case "tasks":
				if err := db.importTasks(h.N, in); err != nil {
					return err
				}

			case "users":
				if err := db.importUsers(h.N, in); err != nil {
					return err
				}

			case "sessions":
				if err := db.importSessions(h.N, in); err != nil {
					return err
				}

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

	if ctx.RestoreTask == nil {
		log.Errorf("IMPORT: unable to locate the restore task we are currently running; skipping finalization...")
	} else {
		t := ctx.RestoreTask
		log.Infof("IMPORT: re-inserting stored restore task [%s] for continuity's sake", t.UUID)

		err = db.ReinsertTask(t)
		if err != nil {
			return err
		}
	}

	return nil
}
