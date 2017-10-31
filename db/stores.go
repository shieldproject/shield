package db

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/jhunt/go-log"
	"github.com/pborman/uuid"
)

type StoreConfigItem struct {
	Label string `json:"label"`
	Value string `json:"value"`
}

type Store struct {
	UUID    uuid.UUID `json:"uuid"`
	Name    string    `json:"name"`
	Summary string    `json:"summary"`
	Agent   string    `json:"agent"`
	Plugin  string    `json:"plugin"`
	Global  bool      `json:"global"`

	PublicConfig  string `json:"-"`
	PrivateConfig string `json:"-"`

	Config        map[string]interface{} `json:"config,omitempty"`
	DisplayConfig []StoreConfigItem      `json:"display_config,omitempty"`

	TenantUUID    uuid.UUID `json:"-"`
	DailyIncrease int64     `json:"daily_increase"`
	StorageUsed   int64     `json:"storage_used"`
	Threshold     int64     `json:"threshold"`
	ArchiveCount  int       `json:"archive_count"`

	Healthy          bool      `json:"healthy"`
	LastTestTaskUUID uuid.UUID `json:"last_test_task_uuid"`
}

type StoreStats struct {
	DailyIncrease int64 `json:"daily_increase"`
	StorageUsed   int64 `json:"storage_used"`
	ArchiveCount  int   `json:"archive_count"`
}

func (s *Store) DisplayPublic() error {
	if s.PublicConfig == "" {
		return nil
	}
	return json.Unmarshal([]byte(s.PublicConfig), &s.DisplayConfig)
}

func (s *Store) DisplayPrivate() error {
	if s.PrivateConfig == "" {
		return nil
	}
	return json.Unmarshal([]byte(s.PrivateConfig), &s.DisplayConfig)
}

func (s *Store) CacheConfigs(db *DB) error {
	if s.Config == nil {
		return nil
	}

	/* get the metadata from the agent, for the given plugin */
	meta, err := db.GetAgentPluginMetadata(s.Agent, s.Plugin)
	if meta == nil || err != nil {
		return nil
	}

	/* fashion two lists of key + value pairs, representing
	   the public and private configurations of this store.
	   public will show only non-sensitive credentials;
	   private will show all of them. */
	public := make([]StoreConfigItem, 0)
	private := make([]StoreConfigItem, 0)
	for _, field := range meta.Fields {
		if field.Mode == "target" {
			continue
		}

		vprivate := fmt.Sprintf("%v", s.Config[field.Name])
		if field.Type == "bool" {
			if s.Config[field.Name] == nil {
				vprivate = "no"
			} else {
				vprivate = "yes"
			}
		}

		vpublic := vprivate
		if field.Type == "password" {
			vpublic = "<em>REDACTED</em>"
		}

		public = append(public, StoreConfigItem{
			Label: field.Title,
			Value: vpublic,
		})
		private = append(private, StoreConfigItem{
			Label: field.Title,
			Value: vprivate,
		})
	}

	/* store the public config as a JSON string */
	b, err := json.Marshal(public)
	if err != nil {
		return err
	}
	s.PublicConfig = string(b)

	/* store the private config as a JSON string */
	b, err = json.Marshal(private)
	if err != nil {
		return err
	}
	s.PrivateConfig = string(b)

	return nil
}

type StoreFilter struct {
	SkipUsed   bool
	SkipUnused bool
	SearchName string
	ForPlugin  string
	ForTenant  string
	ExactMatch bool
}

func (f *StoreFilter) Query() (string, []interface{}) {
	wheres := []string{"s.uuid = s.uuid"}
	args := []interface{}{}
	if f.SearchName != "" {
		if f.ExactMatch {
			wheres = append(wheres, "s.name = ?")
			args = append(args, f.SearchName)
		} else {
			wheres = append(wheres, "s.name LIKE ?")
			args = append(args, Pattern(f.SearchName))
		}
	}
	if f.ForPlugin != "" {
		wheres = append(wheres, "s.plugin = ?")
		args = append(args, f.ForPlugin)
	}
	if f.ForTenant != "" {
		wheres = append(wheres, "s.tenant_uuid = ?")
		args = append(args, f.ForTenant)
	}

	if !f.SkipUsed && !f.SkipUnused {
		return `
		   SELECT s.uuid, s.name, s.summary, s.agent,
		          s.plugin, s.endpoint, s.tenant_uuid, -1 AS n,
		          s.private_config, s.public_config, s.daily_increase,
		          s.storage_used, s.archive_count, s.threshold,
		          s.healthy, s.last_test_task_uuid
		     FROM stores s
		    WHERE ` + strings.Join(wheres, " AND ") + `
		 ORDER BY s.name, s.uuid ASC`, args
	}

	having := `HAVING COUNT(j.uuid) = 0`
	if f.SkipUnused {
		having = `HAVING COUNT(j.uuid) > 0`
	}

	return `
	   SELECT DISTINCT s.uuid, s.name, s.summary, s.agent,
	                   s.plugin, s.endpoint, s.tenant_uuid, COUNT(j.uuid) AS n,
	                   s.private_config, s.public_config, s.daily_increase,
	                   s.storage_used, s.archive_count, s.threshold,
	                   s.healthy, s.last_test_task_uuid
	              FROM stores s
	         LEFT JOIN jobs j
	                ON j.store_uuid = s.uuid
	             WHERE ` + strings.Join(wheres, " AND ") + `
	             GROUP BY s.uuid
	             ` + having + `
	          ORDER BY s.name, s.uuid ASC`, args
}

func (db *DB) GetAllStores(filter *StoreFilter) ([]*Store, error) {
	if filter == nil {
		filter = &StoreFilter{}
	}

	l := []*Store{}
	query, args := filter.Query()
	r, err := db.Query(query, args...)
	if err != nil {
		return l, err
	}
	defer r.Close()

	for r.Next() {
		s := &Store{}
		var (
			this, tenant, lastTestTaskUUID NullUUID
			rawconfig                      []byte
			n                              int
			daily, used, threshold         *int64
			archives                       *int
			healthy                        bool
		)
		if err = r.Scan(&this, &s.Name, &s.Summary, &s.Agent, &s.Plugin, &rawconfig, &tenant, &n, &s.PrivateConfig,
			&s.PublicConfig, &daily, &used, &archives, &threshold, &healthy, &lastTestTaskUUID); err != nil {
			return l, err
		}
		s.UUID = this.UUID
		s.TenantUUID = tenant.UUID
		s.Healthy = healthy
		s.LastTestTaskUUID = lastTestTaskUUID.UUID

		if daily != nil {
			s.DailyIncrease = *daily
		}
		if archives != nil {
			s.ArchiveCount = *archives
		}
		if used != nil {
			s.StorageUsed = *used
		}
		if threshold != nil {
			s.Threshold = *threshold
		}
		if rawconfig != nil {
			if err := json.Unmarshal(rawconfig, &s.Config); err != nil {
				log.Warnf("failed to parse storage system endpoint json '%s': %s", rawconfig, err)
			}
		}

		s.Global = s.TenantUUID.String() == uuid.NIL.String()
		l = append(l, s)
	}

	return l, nil
}

func (db *DB) GetStore(id uuid.UUID) (*Store, error) {
	r, err := db.Query(`
	   SELECT s.uuid, s.name, s.summary, s.agent,
	          s.plugin, s.endpoint, s.tenant_uuid,
	          s.private_config, s.public_config, s.daily_increase,
	          s.storage_used, s.archive_count, s.threshold,
	          s.healthy, s.last_test_task_uuid
	     FROM stores s
	LEFT JOIN jobs j
	       ON j.store_uuid = s.uuid
	    WHERE s.uuid = ?`, id.String())
	if err != nil {
		return nil, err
	}
	defer r.Close()

	if !r.Next() {
		return nil, nil
	}

	s := &Store{}
	var (
		this, tenant, lastTestTaskUUID NullUUID
		rawconfig                      []byte
		daily, used, threshold         *int64
		archives                       *int
		healthy                        bool
	)
	if err = r.Scan(&this, &s.Name, &s.Summary, &s.Agent, &s.Plugin, &rawconfig, &tenant, &s.PrivateConfig,
		&s.PublicConfig, &daily, &used, &archives, &threshold, &healthy, &lastTestTaskUUID); err != nil {
		return nil, err
	}
	s.UUID = this.UUID
	s.TenantUUID = tenant.UUID
	s.Healthy = healthy
	s.LastTestTaskUUID = lastTestTaskUUID.UUID

	if daily != nil {
		s.DailyIncrease = *daily
	}
	if archives != nil {
		s.ArchiveCount = *archives
	}
	if used != nil {
		s.StorageUsed = *used
	}
	if threshold != nil {
		s.Threshold = *threshold
	}

	if rawconfig != nil {
		if err := json.Unmarshal(rawconfig, &s.Config); err != nil {
			log.Warnf("failed to parse storage system endpoint json '%s': %s", rawconfig, err)
		}
	}

	s.Global = s.TenantUUID.String() == uuid.NIL.String()
	return s, nil
}

func (db *DB) CreateStore(s *Store) (*Store, error) {
	if err := s.CacheConfigs(db); err != nil {
		return nil, fmt.Errorf("unable to cache storage configs: %s", err)
	}

	rawconfig, err := json.Marshal(s.Config)
	if err != nil {
		return nil, fmt.Errorf("unable to marshal storage endpoint configs: %s", err)
	}

	s.UUID = uuid.NewRandom()
	return s, db.Exec(`
	   INSERT INTO stores (uuid, tenant_uuid, name, summary, agent, plugin, endpoint, private_config, public_config, threshold, healthy, last_test_task_uuid)
	               VALUES (?,    ?,           ?,    ?,       ?,     ?,      ?,        ?,              ?,              ?,        ?,       ?)`,
		s.UUID.String(), s.TenantUUID.String(), s.Name, s.Summary, s.Agent, s.Plugin, string(rawconfig), s.PrivateConfig, s.PublicConfig, s.Threshold, s.Healthy, s.LastTestTaskUUID)
}

func (db *DB) UpdateStore(s *Store) error {
	if err := s.CacheConfigs(db); err != nil {
		return fmt.Errorf("unable to cache storage configs: %s", err)
	}

	rawconfig, err := json.Marshal(s.Config)
	if err != nil {
		return fmt.Errorf("unable to marshal storage endpoint configs: %s", err)
	}

	return db.Exec(`
		UPDATE stores
	      SET name                    = ?,
	          summary                 = ?,
	          agent                   = ?,
	          plugin                  = ?,
	          endpoint                = ?,
	          private_config          = ?,
	          public_config           = ?,
	          daily_increase          = ?,
	          archive_count           = ?,
	          storage_used            = ?,
	          threshold               = ?,
	          healthy                 = ?,
	          last_test_task_uuid     = ?
		WHERE uuid = ?`, s.Name, s.Summary, s.Agent, s.Plugin, string(rawconfig), s.PrivateConfig, s.PublicConfig, s.DailyIncrease,
		s.ArchiveCount, s.StorageUsed, s.Threshold, s.Healthy, s.LastTestTaskUUID, s.UUID.String(),
	)
}

func (db *DB) DeleteStore(id uuid.UUID) (bool, error) {
	r, err := db.Query(
		`SELECT COUNT(uuid) FROM jobs WHERE jobs.store_uuid = ?`,
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
		return false, fmt.Errorf("Store %s is in used by %d (negative) Jobs", id.String(), numJobs)
	}
	if numJobs > 0 {
		return false, nil
	}

	r.Close()
	return true, db.Exec(
		`DELETE FROM stores WHERE uuid = ?`,
		id.String(),
	)
}

func (s Store) ConfigJSON() (string, error) {
	b, err := json.Marshal(s.Config)
	if err != nil {
		return "", err
	}
	return string(b), err
}
