package db

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/jhunt/go-log"
)

type Target struct {
	UUID    string `json:"uuid"        mbus:"uuid"`
	Name    string `json:"name"        mbus:"name"`
	Summary string `json:"summary"     mbus:"summary"`
	Plugin  string `json:"plugin"      mbus:"plugin"`
	Agent   string `json:"agent"       mbus:"agent"`
	Healthy bool   `json:"healthy"     mbus:"healthy"`

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
	if err != nil {
		return nil, err
	}
	if meta == nil {
		return nil, fmt.Errorf("unable to retrieve target configuration: agent metadata not found in database")
	}

	return DisplayableConfig("target", meta, target.Config, private), nil
}

type TargetFilter struct {
	UUID       string
	SkipUsed   bool
	SkipUnused bool
	SearchName string
	ForPlugin  string
	ExactMatch bool
}

func (f *TargetFilter) Query() (string, []interface{}) {
	wheres := []string{}
	args := []interface{}{}

	if f.UUID != "" {
		if f.ExactMatch {
			wheres = append(wheres, "t.uuid::text = ?")
			args = append(args, f.UUID)
		} else {
			wheres = append(wheres, "t.uuid::text LIKE ?")
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
		wheres = []string{"true"}
	} else if len(wheres) > 1 {
		wheres = []string{strings.Join(wheres, " OR ")}
	}

	if f.ForPlugin != "" {
		wheres = append(wheres, "t.plugin LIKE ?")
		args = append(args, f.ForPlugin)
	}

	if !f.SkipUsed && !f.SkipUnused {
		return `
		   SELECT t.uuid, t.name, t.summary, t.plugin,
		          t.endpoint, t.agent, t.healthy, -1 AS n
		     FROM targets t
		    WHERE ` + strings.Join(wheres, " AND ") + `
		 ORDER BY t.name, t.uuid ASC`, args
	}

	having := `HAVING COUNT(j.uuid) = 0`
	if f.SkipUnused {
		having = `HAVING COUNT(j.uuid) > 0`
	}

	return `
	   SELECT DISTINCT t.uuid, t.name, t.summary, t.plugin,
	                   t.endpoint, t.agent, t.healthy, COUNT(j.uuid) AS n
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

	query, args := filter.Query()
	n, err := db.Count(query, args...)
	return int(n), err
}

func (db *DB) GetAllTargets(filter *TargetFilter) ([]*Target, error) {
	if filter == nil {
		filter = &TargetFilter{}
	}

	l := []*Target{}
	query, args := filter.Query()
	r, err := db.query(query, args...)
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
		if err = r.Scan(&t.UUID, &t.Name, &t.Summary, &t.Plugin, &rawconfig, &t.Agent, &t.Healthy, &n); err != nil {
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
	r, err := db.query(`
	    SELECT uuid, name, summary, plugin,
	           endpoint, agent, healthy

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
	if err = r.Scan(&t.UUID, &t.Name, &t.Summary, &t.Plugin,
		&rawconfig, &t.Agent, &t.Healthy); err != nil {
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
	target.Healthy = true
	err = db.Exec(`
		    INSERT INTO targets (uuid, name, summary, plugin,
		                         endpoint, agent, healthy)
		                 VALUES (?, ?, ?, ?,
		                         ?, ?, ?)`,
		target.UUID, target.Name, target.Summary, target.Plugin,
		string(rawconfig), target.Agent, target.Healthy)
	if err != nil {
		return nil, err
	}

	db.sendCreateObjectEvent(target, "*")
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
	         agent       = ?
	   WHERE uuid = ?`,
		target.Name, target.Summary, target.Plugin, string(rawconfig),
		target.Agent, target.UUID)
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

	db.sendUpdateObjectEvent(target, "*")
	return nil
}

func (db *DB) UpdateTargetHealth(id string, health bool) error {
	target, err := db.GetTarget(id)
	if err != nil {
		return fmt.Errorf("error when finding target with uuid `%s' to update health: %s", id, err)
	}
	if target == nil {
		return fmt.Errorf("no target with uuid `%s' was found to update health", id)
	}
	err = db.Exec(`
        UPDATE targets
            SET healthy = ?
        WHERE uuid = ?`,
		health,
		target.UUID)
	if err != nil {
		return err
	}

	db.sendHealthUpdateEvent(target, "*")
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

	n, err := db.Count(`SELECT uuid FROM jobs WHERE jobs.target_uuid = ?`, target.UUID)
	if n > 0 || err != nil {
		return false, err
	}

	err = db.Exec(`DELETE FROM targets WHERE uuid = ?`, id)
	if err != nil {
		return false, err
	}

	db.sendDeleteObjectEvent(target, "*")
	return true, nil
}

func (db *DB) targetShouldExist(uuid string) error {
	if ok, err := db.Exists(`SELECT uuid FROM targets WHERE uuid = ?`, uuid); err != nil {
		return fmt.Errorf("unable to look up target [%s]: %s", uuid, err)
	} else if !ok {
		return fmt.Errorf("target [%s] does not exist", uuid)
	}
	return nil
}
