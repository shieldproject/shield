package db

import (
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/pborman/uuid"
	. "github.com/starkandwayne/goutils/timestamp"

	"github.com/starkandwayne/shield/plugin"
)

type Agent struct {
	UUID       uuid.UUID `json:"uuid"`
	Name       string    `json:"name"`
	Address    string    `json:"address"`
	Version    string    `json:"version"`
	Hidden     bool      `json:"hidden"`
	LastSeenAt Timestamp `json:"last_seen_at"`
	LastError  string    `json:"last_error"`
	Status     string    `json:"status"`
	Metadata   string    `json:"-"`
}

type AgentFilter struct {
	Address    string
	Name       string
	OnlyHidden bool
	SkipHidden bool
}

func (f *AgentFilter) Query() (string, []interface{}) {
	wheres := []string{"a.uuid = a.uuid"}
	var args []interface{}

	if f.OnlyHidden {
		wheres = append(wheres, "a.hidden = ?")
		args = append(args, true)
	} else if f.SkipHidden {
		wheres = append(wheres, "a.hidden = ?")
		args = append(args, false)
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
		       a.hidden, a.last_seen_at, a.status,
		       a.metadata

		FROM agents a

		WHERE ` + strings.Join(wheres, " AND ") + `
		ORDER BY a.name DESC, a.uuid ASC
	`, args
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

		var lastSeenAt *int64
		var hidden *bool
		var this NullUUID
		if err = r.Scan(
			&this, &agent.Name, &agent.Address, &agent.Version,
			&hidden, &lastSeenAt, &agent.Status,
			&agent.Metadata); err != nil {

			return l, err
		}
		agent.UUID = this.UUID
		if lastSeenAt != nil {
			agent.LastSeenAt = parseEpochTime(*lastSeenAt)
		}

		l = append(l, agent)
	}

	return l, nil
}

func (db *DB) GetAgent(id uuid.UUID) (*Agent, error) {
	r, err := db.Query(`
		SELECT a.uuid, a.name, a.address, a.version,
		       a.hidden, a.last_seen_at, a.last_error, a.status,
		       a.metadata

		FROM agents a

		WHERE a.uuid = ?`, id.String())
	if err != nil {
		return nil, err
	}
	defer r.Close()

	if !r.Next() {
		return nil, nil
	}

	agent := &Agent{}

	var lastSeenAt *int64
	var hidden *bool
	var this NullUUID

	if err = r.Scan(
		&this, &agent.Name, &agent.Address, &agent.Version,
		&hidden, &lastSeenAt, &agent.LastError, &agent.Status,
		&agent.Metadata); err != nil {

		return nil, err
	}
	agent.UUID = this.UUID
	if lastSeenAt != nil {
		agent.LastSeenAt = parseEpochTime(*lastSeenAt)
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

	fmt.Printf("start of plugin list\n")
	for n := range meta.Plugins {
		fmt.Printf("found plugin %s\n", n)
	}
	fmt.Printf("end of plugin list\n")
	if p, ok := meta.Plugins[name]; ok {
		return &p, nil
	}
	return nil, nil
}

func (db *DB) PreRegisterAgent(host, name string, port int) error {
	address := fmt.Sprintf("%s:%d", host, port)
	existing, err := db.GetAllAgents(&AgentFilter{
		Address: address,
		Name:    name,
	})
	if err != nil {
		return err
	}

	if len(existing) > 0 {
		// already pre-registered
		return nil
	}

	id := uuid.NewRandom()
	return db.Exec(
		`INSERT INTO agents (uuid, name, address, status, last_seen_at)
		             VALUES (?, ?, ?, ?, ?)`,
		id.String(), name, address, "pending", time.Now().Unix(),
	)
}

func (db *DB) UpdateAgent(agent *Agent) error {
	return db.Exec(
		`UPDATE agents SET name         = ?,
		                   address      = ?,
		                   version      = ?,
		                   status       = ?,
		                   metadata     = ?,
		                   last_seen_at = ?,
		                   last_error   = ?
		        WHERE uuid = ?`,
		agent.Name, agent.Address, agent.Version, agent.Status, agent.Metadata,
		time.Now().Unix(), agent.LastError,
		agent.UUID.String())
}
