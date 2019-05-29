package db

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/jhunt/go-log"
)

type Target struct {
	UUID        string `json:"uuid"        mbus:"uuid"`
	TenantUUID  string `json:"-"           mbus:"tenant_uuid"`
	Name        string `json:"name"        mbus:"name"`
	Summary     string `json:"summary"     mbus:"summary"`
	Plugin      string `json:"plugin"      mbus:"plugin"`
	Agent       string `json:"agent"       mbus:"agent"`
	Compression string `json:"compression" mbus:"compression"`

	Config map[string]interface{} `json:"config,omitempty"  mbus:"config"`
}

func (t Target) ConfigJSON() (string, error) {
	b, err := json.Marshal(t.Config)
	if err != nil {
		return "", err
	}
	return string(b), err
}

func (target *Target) Configuration(db *DB, private bool) ([]ConfigItem, error) {
	if target.Config == nil {
		return nil, nil
	}

	meta, err := db.GetAgentPluginMetadata(target.Agent, target.Plugin)
	if meta == nil || err != nil {
		return nil, err
	}

	return DisplayableConfig("target", meta, target.Config, private), nil
}

type TargetFilter struct {
	UUID       string
	SkipUsed   bool
	SkipUnused bool
	SearchName string
	ForTenant  string
	ForPlugin  string
	ExactMatch bool
}

func (f *TargetFilter) Query() (string, []interface{}) {
	wheres := []string{}
	args := []interface{}{}

	if f.UUID != "" {
		if f.ExactMatch {
			wheres = append(wheres, "t.uuid = ?")
			args = append(args, f.UUID)
		} else {
			wheres = append(wheres, "t.uuid LIKE ? ESCAPE '/'")
			args = append(args, PatternPrefix(f.UUID))
		}
	}

	if f.SearchName != "" {
		if f.ExactMatch {
			wheres = append(wheres, "t.name = ?")
			args = append(args, f.SearchName)
		} else {
			wheres = append(wheres, "t.name LIKE ?")
			args = append(args, Pattern(f.SearchName))
		}
	}

	if len(wheres) == 0 {
		wheres = []string{"1"}
	} else if len(wheres) > 1 {
		wheres = []string{strings.Join(wheres, " OR ")}
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
		          t.endpoint, t.agent, t.compression, -1 AS n
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
	                   t.endpoint, t.agent, t.compression, COUNT(j.uuid) AS n
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
			n         int
			rawconfig []byte
		)
		if err = r.Scan(&t.UUID, &t.TenantUUID, &t.Name, &t.Summary, &t.Plugin, &rawconfig, &t.Agent, &t.Compression, &n); err != nil {
			return l, err
		}
		if rawconfig != nil {
			if err := json.Unmarshal(rawconfig, &t.Config); err != nil {
				log.Warnf("failed to parse data system endpoint json '%s': %s", rawconfig, err)
			}
		}

		l = append(l, t)
	}

	return l, nil
}

func (db *DB) GetTarget(id string) (*Target, error) {
	r, err := db.Query(`
	    SELECT uuid, tenant_uuid, name, summary, plugin,
	           endpoint, agent, compression

	      FROM targets

	     WHERE uuid = ?`, id)
	if err != nil {
		return nil, err
	}
	defer r.Close()

	if !r.Next() {
		return nil, nil
	}

	t := &Target{}
	var (
		rawconfig []byte
	)
	if err = r.Scan(&t.UUID, &t.TenantUUID, &t.Name, &t.Summary, &t.Plugin,
		&rawconfig, &t.Agent, &t.Compression); err != nil {
		return nil, err
	}
	if rawconfig != nil {
		if err := json.Unmarshal(rawconfig, &t.Config); err != nil {
			log.Warnf("failed to parse data system endpoint json '%s': %s", rawconfig, err)
		}
	}

	return t, nil
}

func (db *DB) CreateTarget(target *Target) (*Target, error) {
	rawconfig, err := json.Marshal(target.Config)
	if err != nil {
		return nil, fmt.Errorf("unable to marshal target endpoint configs: %s", err)
	}

	target.UUID = RandomID()
	err = db.Exec(`
	    INSERT INTO targets (uuid, tenant_uuid, name, summary, plugin,
	                         endpoint, agent, compression)
	                 VALUES (?, ?, ?, ?, ?,
	                         ?, ?, ?)`,
		target.UUID, target.TenantUUID, target.Name, target.Summary, target.Plugin,
		string(rawconfig), target.Agent, target.Compression)
	if err != nil {
		return nil, err
	}

	db.sendCreateObjectEvent(target, "tenant:"+target.TenantUUID)
	return target, nil
}

func (db *DB) UpdateTarget(target *Target) error {
	rawconfig, err := json.Marshal(target.Config)
	if err != nil {
		return err
	}

	err = db.Exec(`
	  UPDATE targets
	     SET name        = ?,
	         summary     = ?,
	         plugin      = ?,
	         endpoint    = ?,
	         agent       = ?,
	         compression = ?
	   WHERE uuid = ?`,
		target.Name, target.Summary, target.Plugin, string(rawconfig),
		target.Agent, target.Compression, target.UUID)
	if err != nil {
		return err
	}

	update, err := db.GetTarget(target.UUID)
	if err != nil {
		return err
	}
	if update == nil {
		return fmt.Errorf("unable to retrieve target %s after update", target.UUID)
	}

	db.sendUpdateObjectEvent(target, "tenant:"+target.TenantUUID)
	return nil
}

func (db *DB) DeleteTarget(id string) (bool, error) {
	target, err := db.GetTarget(id)
	if err != nil {
		return false, err
	}

	if target == nil {
		/* already deleted */
		return true, nil
	}

	r, err := db.Query(`SELECT COUNT(uuid) FROM jobs WHERE jobs.target_uuid = ?`, target.UUID)
	if err != nil {
		return false, err
	}
	defer r.Close()

	if !r.Next() {
		/* already deleted (temporal anomaly detected) */
		return true, nil
	}

	var numJobs int
	if err = r.Scan(&numJobs); err != nil {
		return false, err
	}
	if numJobs < 0 {
		return false, fmt.Errorf("Target %s is in used by %d (negative) Jobs", id, numJobs)
	}
	if numJobs > 0 {
		return false, nil
	}
	r.Close()

	err = db.Exec(`DELETE FROM targets WHERE uuid = ?`, id)
	if err != nil {
		return false, err
	}

	db.sendDeleteObjectEvent(target, "tenant:"+target.TenantUUID)
	return true, nil
}

func (db *DB) CleanTargets() error {
	return db.Exec(`
	   DELETE FROM targets
	         WHERE uuid in (SELECT uuid
	                          FROM targets t
	                         WHERE tenant_uuid = ''
	                           AND (SELECT COUNT(*)
	                                  FROM archives a
	                                 WHERE a.target_uuid = t.uuid
	                                   AND a.status != 'purged') = 0)`)
}
