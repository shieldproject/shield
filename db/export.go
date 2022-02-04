package db

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"time"

	"github.com/shieldproject/shield/core/vault"
)

const exportVersion = "v1"

type header struct {
	V    string `json:"v"`
	Type string `json:"type"`
	N    uint   `json:"n"`
}

type fail struct {
	E string `json:"error"`
}

type finalizer struct {
	Task           string `json:"task"`
	EncryptionType string `json:"encryption_type"`
	EncryptionKey  string `json:"encryption_key"`
	EncryptionIV   string `json:"encryption_iv"`
	TakenAt        int64  `json:"taken_at"`

	Error string `json:"error,omitempty"`
}

func (db *DB) exportHeader(out *json.Encoder, table string) error {
	n, err := db.count(fmt.Sprintf(`SELECT * FROM %s`, table))
	if err != nil {
		return err
	}

	out.Encode(header{
		V:    exportVersion,
		Type: table,
		N:    n,
	})
	return nil
}

func (db *DB) exportFooter(out *json.Encoder) {
	out.Encode(header{
		V:    exportVersion,
		Type: "",
		N:    0,
	})
}

func (db *DB) exportErrors(out *json.Encoder, err error) {
	out.Encode(fail{
		E: err.Error(),
	})
}

func (db *DB) exportAgents(out *json.Encoder) error {
	db.exportHeader(out, "agents")

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

	r, err := db.query(`
	  SELECT uuid, name, address, version,
	         hidden, last_seen_at, last_checked_at,
	         last_error, status, metadata
	    FROM agents`)
	if err != nil {
		return err
	}
	defer r.Close()

	for r.Next() {
		v := agent{}

		var seen, checked *int64
		if err = r.Scan(
			&v.UUID, &v.Name, &v.Address, &v.Version,
			&v.Hidden, &seen, &checked,
			&v.LastError, &v.Status, &v.Metadata); err != nil {

			return err
		}
		if seen != nil {
			v.LastSeenAt = *seen
		}
		if checked != nil {
			v.LastCheckedAt = *checked
		}

		out.Encode(&v)
	}
	return nil
}

func (db *DB) exportArchives(out *json.Encoder, vault *vault.Client) error {
	db.exportHeader(out, "archives")

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

	r, err := db.query(`
	  SELECT uuid, tenant_uuid, target_uuid, store_uuid,
	         store_key, taken_at, expires_at, notes, purge_reason,
	         status, size, job,compression
	    FROM archives`)
	if err != nil {
		return err
	}
	defer r.Close()

	for r.Next() {
		v := archive{}

		if err = r.Scan(
			&v.UUID, &v.TenantUUID, &v.TargetUUID, &v.StoreUUID,
			&v.StoreKey, &v.TakenAt, &v.ExpiresAt, &v.Notes, &v.PurgeReason,
			&v.Status, &v.Size, &v.Job, &v.Compression); err != nil {

			return err
		}

		if e, err := vault.Retrieve(v.UUID); err == nil {
			v.EncryptionKey = e.Key
			v.EncryptionIV = e.IV
			v.EncryptionType = e.Type
		} else {
			return err
		}

		out.Encode(&v)
	}
	return nil
}

func (db *DB) exportFixups(out *json.Encoder) error {
	db.exportHeader(out, "fixups")

	type fixup struct {
		ID        string `json:"id"`
		Name      string `json:"name"`
		Summary   string `json:"summary"`
		CreatedAt int    `json:"created_at"`
		AppliedAt int    `json:"applied_at"`
	}

	r, err := db.query(`
	  SELECT id, name, summary, created_at, applied_at
	    FROM fixups`)
	if err != nil {
		return err
	}
	defer r.Close()

	for r.Next() {
		v := fixup{}

		if err = r.Scan(
			&v.ID, &v.Name, &v.Summary, &v.CreatedAt, &v.AppliedAt); err != nil {

			return err
		}

		out.Encode(&v)
	}
	return nil
}

func (db *DB) exportJobs(out *json.Encoder) error {
	db.exportHeader(out, "jobs")

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
		Retries    int    `json:"retries"`
	}

	r, err := db.query(`
	  SELECT uuid, target_uuid, store_uuid, tenant_uuid,
	         name, summary, schedule, keep_n, keep_days,
	         next_run, priority, paused, fixed_key, healthy, retries
	    FROM jobs`)
	if err != nil {
		return err
	}
	defer r.Close()

	for r.Next() {
		v := job{}

		if err = r.Scan(
			&v.UUID, &v.TargetUUID, &v.StoreUUID, &v.TenantUUID,
			&v.Name, &v.Summary, &v.Schedule, &v.KeepN, &v.KeepDays,
			&v.NextRun, &v.Priority, &v.Paused, &v.FixedKey, &v.Healthy, &v.Retries); err != nil {

			return err
		}

		out.Encode(&v)
	}
	return nil
}

func (db *DB) exportMemberships(out *json.Encoder) error {
	db.exportHeader(out, "memberships")

	type membership struct {
		TenantUUID string `json:"tenant_uuid"`
		UserUUID   string `json:"user_uuid"`
		Role       string `json:"role"`
	}

	r, err := db.query(`
	  SELECT user_uuid, tenant_uuid, role
	    FROM memberships`)
	if err != nil {
		return err
	}
	defer r.Close()

	for r.Next() {
		v := membership{}

		if err = r.Scan(
			&v.UserUUID, &v.TenantUUID, &v.Role); err != nil {

			return err
		}

		out.Encode(&v)
	}
	return nil
}

func (db *DB) exportStores(out *json.Encoder) error {
	db.exportHeader(out, "stores")

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

	r, err := db.query(`
	  SELECT uuid, tenant_uuid, name, summary, plugin, endpoint,
	         agent, daily_increase, storage_used, archive_count,
	         threshold, healthy, last_test_task_uuid
	    FROM stores`)
	if err != nil {
		return err
	}
	defer r.Close()

	for r.Next() {
		v := store{}

		if err = r.Scan(
			&v.UUID, &v.TenantUUID, &v.Name, &v.Summary, &v.Plugin, &v.Endpoint,
			&v.Agent, &v.DailyIncrease, &v.StorageUsed, &v.ArchiveCount,
			&v.Threshold, &v.Healthy, &v.LastTestTaskUUID); err != nil {

			return err
		}

		out.Encode(&v)
	}
	return nil
}

func (db *DB) exportTargets(out *json.Encoder) error {
	db.exportHeader(out, "targets")

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

	r, err := db.query(`
	  SELECT uuid, tenant_uuid, name, summary, plugin, endpoint,
	         agent, compression, healthy
	    FROM targets`)
	if err != nil {
		return err
	}
	defer r.Close()

	for r.Next() {
		v := target{}

		if err = r.Scan(
			&v.UUID, &v.TenantUUID, &v.Name, &v.Summary, &v.Plugin, &v.Endpoint,
			&v.Agent, &v.Compression, &v.Healthy); err != nil {

			return err
		}

		out.Encode(&v)
	}
	return nil
}

func (db *DB) exportTasks(out *json.Encoder, task_uuid string, vault *vault.Client) (*finalizer, error) {
	var final *finalizer = nil

	db.exportHeader(out, "tasks")

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
		StoppedAt      *int64  `json:"stopped_at"`
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
		Compression    string  `json:"compression"`
		//Retries        string  `json:"retries"`
	}

	r, err := db.query(`
	  SELECT uuid, owner, op, tenant_uuid, job_uuid, archive_uuid, target_uuid, store_uuid,
	         status, requested_at, started_at, stopped_at, timeout_at,
	         log, attempts, agent, fixed_key, compression,
	         target_plugin, target_endpoint, store_plugin, store_endpoint,
	         restore_key, ok, notes, clear
	    FROM tasks`)
	if err != nil {
		return nil, err
	}
	defer r.Close()

	for r.Next() {
		v := task{}

		if err = r.Scan(
			&v.UUID, &v.Owner, &v.Op, &v.TenantUUID, &v.JobUUID, &v.ArchiveUUID, &v.TargetUUID, &v.StoreUUID,
			&v.Status, &v.RequestedAt, &v.StartedAt, &v.StoppedAt, &v.TimeoutAt,
			&v.Log, &v.Attempts, &v.Agent, &v.FixedKey, &v.Compression,
			&v.TargetPlugin, &v.TargetEndpoint, &v.StorePlugin, &v.StoreEndpoint,
			&v.RestoreKey, &v.OK, &v.Notes, &v.Clear); err != nil {

			return nil, err
		}

		if v.UUID == task_uuid && v.Status == "running" {
			enc, err := vault.Retrieve(*v.ArchiveUUID)
			if err != nil {
				return nil, err
			}

			final = &finalizer{
				Task:           v.UUID,
				EncryptionKey:  enc.Key,
				EncryptionIV:   enc.IV,
				EncryptionType: enc.Type,
				TakenAt:        0, // to be updated later
			}
		}

		out.Encode(&v)
	}

	return final, nil
}

func (db *DB) exportTenants(out *json.Encoder) error {
	db.exportHeader(out, "tenants")

	type tenant struct {
		UUID          string `json:"uuid"`
		Name          string `json:"name"`
		DailyIncrease *int   `json:"daily_increase"`
		StorageUsed   *int   `json:"storage_used"`
		ArchiveCount  *int   `json:"archive_count"`
	}

	r, err := db.query(`
	  SELECT uuid, name, daily_increase, storage_used, archive_count
	    FROM tenants`)
	if err != nil {
		return err
	}
	defer r.Close()

	for r.Next() {
		v := tenant{}

		if err = r.Scan(
			&v.UUID, &v.Name, &v.DailyIncrease, &v.StorageUsed, &v.ArchiveCount); err != nil {

			return err
		}

		out.Encode(&v)
	}
	return nil
}

func (db *DB) exportUsers(out *json.Encoder) error {
	db.exportHeader(out, "users")

	type user struct {
		UUID          string `json:"uuid"`
		Name          string `json:"name"`
		Account       string `json:"account"`
		Backend       string `json:"backend"`
		PasswordHash  string `json:"password_hash"`
		SystemRole    string `json:"system_role"`
		DefaultTenant string `json:"default_tenant"`
	}

	r, err := db.query(`
	  SELECT uuid, name, account, backend,
	         pwhash, sysrole, default_tenant
	    FROM users`)
	if err != nil {
		return err
	}
	defer r.Close()

	for r.Next() {
		v := user{}

		if err = r.Scan(
			&v.UUID, &v.Name, &v.Account, &v.Backend,
			&v.PasswordHash, &v.SystemRole, &v.DefaultTenant); err != nil {

			return err
		}

		out.Encode(&v)
	}
	return nil
}

func (db *DB) exportSessions(out *json.Encoder) error {
	db.exportHeader(out, "sessions")

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

	r, err := db.query(`
        SELECT uuid, user_uuid, created_at, last_seen,
                token, name, ip_addr, user_agent
            FROM sessions`)
	if err != nil {
		return err
	}
	defer r.Close()

	for r.Next() {
		s := &Session{}

		var (
			last  *int64
			token sql.NullString
		)
		if err := r.Scan(&s.UUID, &s.UserUUID, &s.CreatedAt, &last, &token, &s.Name, &s.IP, &s.UserAgent); err != nil {
			return err
		}
		if last != nil {
			s.LastSeen = *last
		}
		if token.Valid {
			s.Token = token.String
		}
		out.Encode(&s)
	}
	return nil
}

func (db *DB) exportFinalizer(out *json.Encoder, vault *vault.Client, fin *finalizer) error {
	if fin == nil {
		return nil
	}

	out.Encode(header{
		V:    exportVersion,
		Type: "finalizer",
		N:    1,
	})

	at := time.Now()
	fin.TakenAt = effectively(at)
	out.Encode(&fin)

	return nil
}

func (db *DB) Export(out *json.Encoder, vault *vault.Client, task_uuid string) {
	db.exclusively(func() error {
		err := db.exportAgents(out)
		if err != nil {
			db.exportErrors(out, err)
		}

		err = db.exportFixups(out)
		if err != nil {
			db.exportErrors(out, err)
		}

		err = db.exportTenants(out)
		if err != nil {
			db.exportErrors(out, err)
		}

		err = db.exportStores(out)
		if err != nil {
			db.exportErrors(out, err)
		}

		err = db.exportTargets(out)
		if err != nil {
			db.exportErrors(out, err)
		}

		err = db.exportJobs(out)
		if err != nil {
			db.exportErrors(out, err)
		}

		err = db.exportArchives(out, vault)
		if err != nil {
			db.exportErrors(out, err)
		}

		// we might get some additional information out
		// of exportTasks, based on our current task_uuid
		// and the running tasks.
		//
		// we call that a "finalizer", and we tack it onto
		// the end of the export stream.
		finalizer, err := db.exportTasks(out, task_uuid, vault)
		if err != nil {
			db.exportErrors(out, err)
		}

		err = db.exportUsers(out)
		if err != nil {
			db.exportErrors(out, err)
		}

		err = db.exportMemberships(out)
		if err != nil {
			db.exportErrors(out, err)
		}

		err = db.exportSessions(out)
		if err != nil {
			db.exportErrors(out, err)
		}

		// if we got a finalizer out of our exportTasks()
		// call, above, let's tack it onto the end, so
		// that the import half of this equation can make
		// use of it.
		err = db.exportFinalizer(out, vault, finalizer)
		if err != nil {
			db.exportErrors(out, err)
		}

		db.exportFooter(out)
		return nil
	})
}
