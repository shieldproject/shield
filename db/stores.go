package db

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/jhunt/go-log"
)

type StoreConfigItem struct {
	Label string `json:"label"`
	Value string `json:"value"`
}

type Store struct {
	UUID       string `json:"uuid"    mbus:"uuid"`
	TenantUUID string `json:"-"       mbus:"tenant_uuid"`
	Name       string `json:"name"    mbus:"name"`
	Summary    string `json:"summary" mbus:"summary"`
	Agent      string `json:"agent"   mbus:"agent"`
	Plugin     string `json:"plugin"  mbus:"plugin"`
	Global     bool   `json:"global"  mbus:"global"`

	PublicConfig  string `json:"-"`
	PrivateConfig string `json:"-"`

	Config        map[string]interface{} `json:"config,omitempty"`
	DisplayConfig []StoreConfigItem      `json:"display_config,omitempty"`

	DailyIncrease int64 `json:"daily_increase"`
	StorageUsed   int64 `json:"storage_used"`
	Threshold     int64 `json:"threshold"`
	ArchiveCount  int   `json:"archive_count"`

	Healthy          bool   `json:"healthy"`
	LastTestTaskUUID string `json:"last_test_task_uuid"`
}

type StoreStats struct {
	DailyIncrease int64 `json:"daily_increase"`
	StorageUsed   int64 `json:"storage_used"`
	ArchiveCount  int   `json:"archive_count"`
}

func (store *Store) DisplayPublic() error {
	if store.PublicConfig == "" {
		return nil
	}
	return json.Unmarshal([]byte(store.PublicConfig), &store.DisplayConfig)
}

func (store *Store) DisplayPrivate() error {
	if store.PrivateConfig == "" {
		return nil
	}
	return json.Unmarshal([]byte(store.PrivateConfig), &store.DisplayConfig)
}

func (store *Store) CacheConfigs(db *DB) error {
	if store.Config == nil {
		return nil
	}

	/* get the metadata from the agent, for the given plugin */
	meta, err := db.GetAgentPluginMetadata(store.Agent, store.Plugin)
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

		vprivate := fmt.Sprintf("%v", store.Config[field.Name])
		if field.Type == "bool" {
			if store.Config[field.Name] == nil {
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
	store.PublicConfig = string(b)

	/* store the private config as a JSON string */
	b, err = json.Marshal(private)
	if err != nil {
		return err
	}
	store.PrivateConfig = string(b)

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
	r, err := db.query(query, args...)
	if err != nil {
		return l, err
	}
	defer r.Close()

	for r.Next() {
		store := &Store{}

		var (
			rawconfig              []byte
			n                      int
			daily, used, threshold *int64
			archives               *int
			healthy                bool
			ltt                    sql.NullString
		)
		if err = r.Scan(&store.UUID, &store.Name, &store.Summary, &store.Agent,
			&store.Plugin, &rawconfig, &store.TenantUUID, &n,
			&store.PrivateConfig, &store.PublicConfig, &daily,
			&used, &archives, &threshold,
			&healthy, &ltt); err != nil {
			return l, err
		}
		store.Healthy = healthy
		store.Global = store.TenantUUID == GlobalTenantUUID
		if ltt.Valid {
			store.LastTestTaskUUID = ltt.String
		}
		if daily != nil {
			store.DailyIncrease = *daily
		}
		if archives != nil {
			store.ArchiveCount = *archives
		}
		if used != nil {
			store.StorageUsed = *used
		}
		if threshold != nil {
			store.Threshold = *threshold
		}
		if rawconfig != nil {
			if err := json.Unmarshal(rawconfig, &store.Config); err != nil {
				log.Warnf("failed to parse storage system endpoint json '%s': %s", rawconfig, err)
			}
		}

		l = append(l, store)
	}

	return l, nil
}

func (db *DB) GetStore(id string) (*Store, error) {
	r, err := db.query(`
	       SELECT s.uuid, s.name, s.summary, s.agent,
	              s.plugin, s.endpoint, s.tenant_uuid,
	              s.private_config, s.public_config, s.daily_increase,
	              s.storage_used, s.archive_count, s.threshold,
	              s.healthy, s.last_test_task_uuid

	         FROM stores s
	    LEFT JOIN jobs j ON j.store_uuid = s.uuid

	        WHERE s.uuid = ?`, id)
	if err != nil {
		return nil, err
	}
	defer r.Close()

	if !r.Next() {
		return nil, nil
	}

	store := &Store{}
	var (
		rawconfig              []byte
		daily, used, threshold *int64
		archives               *int
		healthy                bool
		ltt                    sql.NullString
	)
	if err = r.Scan(&store.UUID, &store.Name, &store.Summary, &store.Agent,
		&store.Plugin, &rawconfig, &store.TenantUUID,
		&store.PrivateConfig, &store.PublicConfig, &daily,
		&used, &archives, &threshold,
		&healthy, &ltt); err != nil {
		return nil, err
	}
	store.Global = store.TenantUUID == GlobalTenantUUID
	store.Healthy = healthy
	if ltt.Valid {
		store.LastTestTaskUUID = ltt.String
	}
	if daily != nil {
		store.DailyIncrease = *daily
	}
	if archives != nil {
		store.ArchiveCount = *archives
	}
	if used != nil {
		store.StorageUsed = *used
	}
	if threshold != nil {
		store.Threshold = *threshold
	}
	if rawconfig != nil {
		if err := json.Unmarshal(rawconfig, &store.Config); err != nil {
			log.Warnf("failed to parse storage system endpoint json '%s': %s", rawconfig, err)
		}
	}

	return store, nil
}

func (db *DB) CreateStore(store *Store) (*Store, error) {
	if err := store.CacheConfigs(db); err != nil {
		return nil, fmt.Errorf("unable to cache storage configs: %s", err)
	}

	rawconfig, err := json.Marshal(store.Config)
	if err != nil {
		return nil, fmt.Errorf("unable to marshal storage endpoint configs: %s", err)
	}

	store.UUID = randomID()
	err = db.exec(`
	   INSERT INTO stores (uuid, tenant_uuid, name, summary, agent,
	                       plugin, endpoint, private_config, public_config,
	                       threshold, healthy, last_test_task_uuid)
	               VALUES (?, ?, ?, ?, ?,
	                       ?, ?, ?, ?,
	                       ?, ?, ?)`,
		store.UUID, store.TenantUUID, store.Name, store.Summary, store.Agent,
		store.Plugin, string(rawconfig), store.PrivateConfig, store.PublicConfig,
		store.Threshold, store.Healthy, store.LastTestTaskUUID)
	if err != nil {
		return nil, err
	}

	db.sendCreateObjectEvent(store, "tenant:"+store.TenantUUID)
	return store, nil
}

func (db *DB) UpdateStore(store *Store) error {
	if err := store.CacheConfigs(db); err != nil {
		return fmt.Errorf("unable to cache storage configs: %s", err)
	}

	rawconfig, err := json.Marshal(store.Config)
	if err != nil {
		return fmt.Errorf("unable to marshal storage endpoint configs: %s", err)
	}

	err = db.exec(`
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
	    WHERE uuid = ?`,
		store.Name, store.Summary, store.Agent, store.Plugin,
		string(rawconfig), store.PrivateConfig, store.PublicConfig,
		store.DailyIncrease, store.ArchiveCount, store.StorageUsed,
		store.Threshold, store.Healthy, store.LastTestTaskUUID,
		store.UUID)
	if err != nil {
		return err
	}

	update, err := db.GetStore(store.UUID)
	if err != nil {
		return err
	}
	if update == nil {
		return fmt.Errorf("unable to retrieve store %s after update", store.UUID)
	}

	db.sendUpdateObjectEvent(store, "tenant:"+store.TenantUUID)
	return nil
}

func (db *DB) DeleteStore(id string) (bool, error) {
	store, err := db.GetStore(id)
	if err != nil {
		return false, err
	}

	if store == nil {
		/* already deleted */
		return true, nil
	}

	r, err := db.query(`SELECT COUNT(uuid) FROM jobs WHERE jobs.store_uuid = ?`, store.UUID)
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
		return false, fmt.Errorf("Store %s is in used by %d (negative) Jobs", id, numJobs)
	}
	if numJobs > 0 {
		return false, nil
	}
	r.Close()

	err = db.exec(`DELETE FROM stores WHERE uuid = ?`, store.UUID)
	if err != nil {
		return false, err
	}

	db.sendDeleteObjectEvent(store, "tenant:"+store.TenantUUID)
	return true, nil
}

func (store Store) ConfigJSON() (string, error) {
	b, err := json.Marshal(store.Config)
	if err != nil {
		return "", err
	}
	return string(b), err
}
