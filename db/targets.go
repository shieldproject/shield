package db

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/pborman/uuid"
	"github.com/jhunt/go-log"
)

type Target struct {
	TenantUUID uuid.UUID `json:"-"`

	UUID    uuid.UUID `json:"uuid"`
	Name    string    `json:"name"`
	Summary string    `json:"summary"`
	Plugin  string    `json:"plugin"`
	Agent   string    `json:"agent"`

	Config map[string]interface{} `json:"config,omitempty"`
}

func (t Target) ConfigJSON() (string, error) {
	b, err := json.Marshal(t.Config)
	if err != nil {
		return "", err
	}
	return string(b), err
}

type TargetFilter struct {
	SkipUsed   bool
	SkipUnused bool
	SearchName string
	ForTenant  string
	ForPlugin  string
	ExactMatch bool
}

func (f *TargetFilter) Query() (string, []interface{}) {
	wheres := []string{"t.uuid = t.uuid"}
	args := []interface{}{}

	if f.SearchName != "" {
		if f.ExactMatch {
			wheres = append(wheres, "t.name = ?")
			args = append(args, f.SearchName)
		} else {
			wheres = append(wheres, "t.name LIKE ?")
			args = append(args, Pattern(f.SearchName))
		}
	}
	if f.ForTenant != "" {
		wheres = append(wheres, "t.tenant_uuid = ?")
		args = append(args, f.ForTenant)
	}
	if f.ForPlugin != "" {
		wheres = append(wheres, "t.plugin LIKE ?")
		args = append(args, f.ForPlugin)
	}

	if !f.SkipUsed && !f.SkipUnused {
		return `
		   SELECT t.uuid, t.tenant_uuid, t.name, t.summary, t.plugin,
		          t.endpoint, t.agent, -1 AS n
		     FROM targets t
		    WHERE ` + strings.Join(wheres, " AND ") + `
		 ORDER BY t.name, t.uuid ASC`, args
	}

	having := `HAVING COUNT(j.uuid) = 0`
	if f.SkipUnused {
		having = `HAVING COUNT(j.uuid) > 0`
	}

	return `
	   SELECT DISTINCT t.uuid, t.tenant_uuid, t.name, t.summary, t.plugin,
	                   t.endpoint, t.agent, COUNT(j.uuid) AS n
	              FROM targets t
	         LEFT JOIN jobs j
	                ON j.target_uuid = t.uuid
	             WHERE ` + strings.Join(wheres, " AND ") + `
	          GROUP BY t.uuid
	          ` + having + `
	          ORDER BY t.name, t.uuid ASC`, args
}

func (db *DB) CountTargets(filter *TargetFilter) (int, error) {
	if filter == nil {
		filter = &TargetFilter{}
	}

	var i int
	query, args := filter.Query()
	r, err := db.Query(query, args...)
	if err != nil {
		return i, err
	}
	defer r.Close()

	for r.Next() {
		i++
	}

	return i, nil
}

func (db *DB) GetAllTargets(filter *TargetFilter) ([]*Target, error) {
	if filter == nil {
		filter = &TargetFilter{}
	}

	l := []*Target{}
	query, args := filter.Query()
	r, err := db.Query(query, args...)
	if err != nil {
		return l, err
	}
	defer r.Close()

	for r.Next() {
		t := &Target{}
		var (
			n            int
			this, tenant NullUUID
			rawconfig    []byte
		)
		if err = r.Scan(&this, &tenant, &t.Name, &t.Summary, &t.Plugin, &rawconfig, &t.Agent, &n); err != nil {
			return l, err
		}
		t.UUID = this.UUID
		t.TenantUUID = tenant.UUID

		if rawconfig != nil {
			if err := json.Unmarshal(rawconfig, &t.Config); err != nil {
				log.Warnf("failed to parse data system endpoint json '%s': %s", rawconfig, err)
			}
		}

		l = append(l, t)
	}

	return l, nil
}

func (db *DB) GetTarget(id uuid.UUID) (*Target, error) {
	r, err := db.Query(`
	  SELECT uuid, tenant_uuid, name, summary, plugin, endpoint, agent
	    FROM targets
	   WHERE uuid = ?`, id.String())
	if err != nil {
		return nil, err
	}
	defer r.Close()

	if !r.Next() {
		return nil, nil
	}

	t := &Target{}
	var (
		this, tenant NullUUID
		rawconfig    []byte
	)
	if err = r.Scan(&this, &tenant, &t.Name, &t.Summary, &t.Plugin, &rawconfig, &t.Agent); err != nil {
		return nil, err
	}
	t.UUID = this.UUID
	t.TenantUUID = tenant.UUID

	if rawconfig != nil {
		if err := json.Unmarshal(rawconfig, &t.Config); err != nil {
			log.Warnf("failed to parse data system endpoint json '%s': %s", rawconfig, err)
		}
	}

	return t, nil
}

func (db *DB) CreateTarget(in *Target) (*Target, error) {
	rawconfig, err := json.Marshal(in.Config)
	if err != nil {
		return nil, err
	}

	in.UUID = uuid.NewRandom()
	return in, db.Exec(`
	    INSERT INTO targets (uuid, tenant_uuid, name, summary, plugin, endpoint, agent)
	                 VALUES (?,    ?,           ?,    ?,       ?,      ?,        ?)`,
		in.UUID.String(), in.TenantUUID.String(), in.Name, in.Summary, in.Plugin, string(rawconfig), in.Agent)
}

func (db *DB) UpdateTarget(t *Target) error {
	rawconfig, err := json.Marshal(t.Config)
	if err != nil {
		return err
	}

	return db.Exec(`
	  UPDATE targets
	     SET name     = ?,
	         summary  = ?,
	         plugin   = ?,
	         endpoint = ?,
	         agent    = ?
	   WHERE uuid = ?`,
		t.Name, t.Summary, t.Plugin, string(rawconfig), t.Agent,
		t.UUID.String())
}

func (db *DB) DeleteTarget(id uuid.UUID) (bool, error) {
	r, err := db.Query(
		`SELECT COUNT(uuid) FROM jobs WHERE jobs.target_uuid = ?`,
		id.String(),
	)
	if err != nil {
		return false, err
	}
	defer r.Close()

	// already deleted
	if !r.Next() {
		return true, nil
	}

	var numJobs int
	if err = r.Scan(&numJobs); err != nil {
		return false, err
	}

	if numJobs < 0 {
		return false, fmt.Errorf("Target %s is in used by %d (negative) Jobs", id.String(), numJobs)
	}
	if numJobs > 0 {
		return false, nil
	}

	r.Close()
	return true, db.Exec(
		`DELETE FROM targets WHERE uuid = ?`,
		id.String(),
	)
}
