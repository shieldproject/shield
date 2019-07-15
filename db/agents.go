package db

import (
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/shieldproject/shield/plugin"
)

type Agent struct {
	UUID          string `json:"uuid"            mbus:"uuid"`
	Name          string `json:"name"            mbus:"name"`
	Address       string `json:"address"         mbus:"address"`
	Version       string `json:"version"         mbus:"version"`
	Hidden        bool   `json:"hidden"          mbus:"hidden"`
	LastSeenAt    int64  `json:"last_seen_at"    mbus:"last_seen_at"`
	LastCheckedAt int64  `json:"last_checked_at" mbus:"last_checked_at"`
	LastError     string `json:"last_error"      mbus:"last_error"`
	Status        string `json:"status"          mbus:"status"`

	Meta    map[string]interface{} `json:"metadata,omitempty"`
	RawMeta string                 `json:"-"`
}

func (a *Agent) Metadata() (map[string]interface{}, error) {
	raw := make(map[string]interface{})
	if a.RawMeta == "" {
		return raw, nil
	}
	return raw, json.Unmarshal([]byte(a.RawMeta), &raw)
}

type AgentFilter struct {
	UUID       string
	ExactMatch bool
	Address    string
	Name       string
	Status     string
	OnlyHidden bool
	SkipHidden bool

	InflateMetadata bool
}

func (f *AgentFilter) Query() (string, []interface{}) {
	wheres := []string{"a.uuid = a.uuid"}
	var args []interface{}

	if f.UUID != "" {
		if f.ExactMatch {
			wheres = []string{"a.uuid = ?"}
			args = []interface{}{f.UUID}
		} else {
			wheres = []string{"a.uuid LIKE ? ESCAPE '/'"}
			args = []interface{}{PatternPrefix(f.UUID)}
		}
	}

	if f.OnlyHidden {
		wheres = append(wheres, "a.hidden = ?")
		args = append(args, true)
	} else if f.SkipHidden {
		wheres = append(wheres, "a.hidden = ?")
		args = append(args, false)
	}

	if f.Status != "" {
		wheres = append(wheres, "a.status = ?")
		args = append(args, f.Status)
	}

	if f.Address != "" {
		wheres = append(wheres, "a.address = ?")
		args = append(args, f.Address)
	}

	if f.Name != "" {
		wheres = append(wheres, "a.name = ?")
		args = append(args, f.Name)
	}

	return `
	   SELECT a.uuid, a.name, a.address, a.version,
	          a.hidden, a.last_seen_at, a.last_checked_at, a.status,
	          a.metadata

	     FROM agents a

	    WHERE ` + strings.Join(wheres, " AND ") + `
	 ORDER BY a.name DESC, a.uuid ASC`, args
}

func (db *DB) GetAllAgents(filter *AgentFilter) ([]*Agent, error) {
	if filter == nil {
		filter = &AgentFilter{}
	}

	l := []*Agent{}
	query, args := filter.Query()
	r, err := db.Query(query, args...)
	if err != nil {
		return l, err
	}
	defer r.Close()

	for r.Next() {
		agent := &Agent{}

		var seen, checked *int64
		if err = r.Scan(
			&agent.UUID, &agent.Name, &agent.Address, &agent.Version,
			&agent.Hidden, &seen, &checked, &agent.Status, &agent.RawMeta); err != nil {
			return l, err
		}
		if seen != nil {
			agent.LastSeenAt = *seen
		}
		if checked != nil {
			agent.LastCheckedAt = *checked
		}

		if filter.InflateMetadata {
			agent.Meta, _ = agent.Metadata()
		}

		l = append(l, agent)
	}

	return l, nil
}

func (db *DB) GetAgent(id string) (*Agent, error) {
	r, err := db.Query(`
		SELECT a.uuid, a.name, a.address, a.version,
		       a.hidden, a.last_seen_at, a.last_checked_at, a.last_error, a.status,
		       a.metadata

		FROM agents a

		WHERE a.uuid = ?`, id)
	if err != nil {
		return nil, err
	}
	defer r.Close()

	if !r.Next() {
		return nil, nil
	}

	agent := &Agent{}

	var seen, checked *int64
	if err = r.Scan(
		&agent.UUID, &agent.Name, &agent.Address, &agent.Version,
		&agent.Hidden, &seen, &checked, &agent.LastError, &agent.Status,
		&agent.RawMeta); err != nil {

		return nil, err
	}
	if seen != nil {
		agent.LastSeenAt = *seen
	}
	if checked != nil {
		agent.LastCheckedAt = *checked
	}

	return agent, nil
}

func (db *DB) GetAgentPluginMetadata(addr, name string) (*plugin.PluginInfo, error) {
	r, err := db.Query(`SELECT metadata FROM agents WHERE address = ?`, addr)
	if err != nil {
		return nil, err
	}
	defer r.Close()

	if !r.Next() {
		return nil, nil
	}

	var (
		raw  []byte
		meta struct {
			Plugins map[string]plugin.PluginInfo `json:"plugins"`
		}
	)
	if err = r.Scan(&raw); err != nil {
		return nil, err
	}
	if err = json.Unmarshal(raw, &meta); err != nil {
		return nil, err
	}

	if p, ok := meta.Plugins[name]; ok {
		return &p, nil
	}
	return nil, nil
}

func (db *DB) PreRegisterAgent(host, name string, port int) error {
	address := fmt.Sprintf("%s:%d", host, port)
	existing, err := db.GetAllAgents(&AgentFilter{
		Name: name,
	})
	if err != nil {
		return err
	}

	if len(existing) > 0 {
		return db.Exec(`
		  UPDATE agents
		     SET address      = ?,
		         last_seen_at = ?
		   WHERE name = ?`,
			address, time.Now().Unix(), name,
		)
	}

	id := RandomID()
	err = db.Exec(`
	   INSERT INTO agents (uuid, name, address, hidden, status, last_seen_at)
	               VALUES (?,    ?,    ?,       ?,      ?,      ?)`,
		id, name, address, false, "pending", time.Now().Unix(),
	)
	if err != nil {
		return err
	}

	agent, err := db.GetAgent(id)
	if err != nil {
		return err
	}

	db.sendCreateObjectEvent(agent, "*")
	return nil
}

func (db *DB) UpdateAgent(agent *Agent) error {
	return db.Exec(
		`UPDATE agents SET name            = ?,
		                   address         = ?,
		                   version         = ?,
		                   status          = ?,
		                   hidden          = ?,
		                   metadata        = ?,
		                   last_checked_at = ?,
		                   last_error      = ?
		        WHERE uuid = ?`,
		agent.Name, agent.Address, agent.Version, agent.Status, agent.Hidden, agent.RawMeta,
		time.Now().Unix(), agent.LastError,
		agent.UUID)
}
